// Package openai implements the provider-neutral thought.Model port with the
// OpenAI Responses API.
//
//nolint:wsl,wsl_v5 // Request construction follows the HTTP protocol sequence.
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/kizuna-org/akari/internal/emotion"
	"github.com/kizuna-org/akari/internal/goal"
	"github.com/kizuna-org/akari/internal/interest"
	"github.com/kizuna-org/akari/internal/meaning"
	"github.com/kizuna-org/akari/internal/memory"
	"github.com/kizuna-org/akari/internal/thought"
)

const (
	defaultModel   = "gpt-6-luna"
	defaultURL     = "https://api.openai.com/v1"
	maxResponse    = 1 << 20
	requestTimeout = 45 * time.Second
	inputGuide     = "Interpret the selected Akari experience. Return JSON with appraisal, interests, memory, goals. " +
		"Do not infer execution permissions. Semantic points must contain text only; " +
		"preserve IDs when copying supplied memory fragments. " +
		"Return all four top-level properties. Input: "
	interpretationSchema = `{"type":"object","properties":{"appraisal":{"anyOf":[{"type":"null"},{"type":"ob` +
		`ject","properties":{"feelings":{"type":"array","items":{"$ref":"#/$defs/feeling"}},"othe` +
		`rs":{"type":"array","items":{"$ref":"#/$defs/feeling"}},"readiness":{"type":"array","ite` +
		`ms":{"$ref":"#/$defs/feeling"}},"moodDelta":{"type":"array","items":{"type":"number"},"m` +
		`inItems":3,"maxItems":3}},"required":["feelings","others","readiness","moodDelta"],"addi` +
		`tionalProperties":false}]},"interests":{"type":"array","items":{"type":"object","propert` +
		`ies":{"id":{"type":"string"},"target":{"$ref":"#/$defs/point"},"affinity":{"type":"numbe` +
		`r"}},"required":["id","target","affinity"],"additionalProperties":false}},"memory":{"any` +
		`Of":[{"type":"null"},{"type":"object","properties":{"context":{"anyOf":[{"type":"null"},` +
		`{"type":"array","items":{"$ref":"#/$defs/fragment"}}]},"working":{"anyOf":[{"type":"null` +
		`"},{"type":"array","items":{"$ref":"#/$defs/fragment"}}]},"remember":{"type":"array","it` +
		`ems":{"$ref":"#/$defs/fragment"}},"recalled":{"type":"array","items":{"type":"string"}}}` +
		`,"required":["context","working","remember","recalled"],"additionalProperties":false}]},` +
		`"goals":{"anyOf":[{"type":"null"},{"type":"object","properties":{"desires":{"type":"arra` +
		`y","items":{"type":"object","properties":{"id":{"type":"string"},"meaning":{"$ref":"#/$d` +
		`efs/point"}},"required":["id","meaning"],"additionalProperties":false}},"adopt":{"type":` +
		`"array","items":{"type":"object","properties":{"id":{"type":"string"},"intention":{"$ref` +
		`":"#/$defs/intention"}},"required":["id","intention"],"additionalProperties":false}},"en` +
		`d":{"type":"array","items":{"type":"string"}}},"required":["desires","adopt","end"],"add` +
		`itionalProperties":false}]}},"required":["appraisal","interests","memory","goals"],"addi` +
		`tionalProperties":false,"$defs":{"point":{"type":"object","properties":{"text":{"type":"` +
		`string"}},"required":["text"],"additionalProperties":false},"feeling":{"type":"object","` +
		`properties":{"meaning":{"$ref":"#/$defs/point"},"strength":{"type":"number"}},"required"` +
		`:["meaning","strength"],"additionalProperties":false},"fragment":{"type":"object","prope` +
		`rties":{"id":{"type":"string"},"meaning":{"$ref":"#/$defs/point"},"occurredAt":{"type":"` +
		`string","format":"date-time"},"emotion":{"type":"number"},"will":{"type":"number"},"stre` +
		`ngth":{"type":"number"}},"required":["id","meaning","occurredAt","emotion","will","stren` +
		`gth"],"additionalProperties":false},"intention":{"type":"object","properties":{"meaning"` +
		`:{"$ref":"#/$defs/point"},"claims":{"type":"array","items":{"type":"object","properties"` +
		`:{"target":{"type":"string"},"value":{"type":"string"}},"required":["target","value"],"a` +
		`dditionalProperties":false}},"steps":{"type":"array","items":{"$ref":"#/$defs/point"}},"` +
		`next":{"type":"integer","minimum":0},"paused":{"type":"boolean"}},"required":["meaning",` +
		`"claims","steps","next","paused"],"additionalProperties":false}}}`
)

var (
	ErrConfig   = errors.New("openai: model requires an API key and valid configuration")
	ErrResponse = errors.New("openai: Responses API returned an invalid response")
	ErrStatus   = errors.New("openai: Responses API returned a non-success status")
)

var credentialMarker = regexp.MustCompile(
	`(?i)(sk-[a-z0-9_-]{8,}|bearer[[:space:]]+[a-z0-9._-]{12,}|` +
		`(?:api[_ -]?key|password|secret)["']?[[:space:]]*[:=][[:space:]]*\S+)`,
)

type DisclosurePolicy interface {
	AllowsDisclosure(sources []string, recipient string) bool
}

// Config contains deployment-specific API settings. APIKey is never included
// in errors or serialized request bodies.
type Config struct {
	Model      string
	APIKey     string
	BaseURL    string
	Disclosure DisclosurePolicy
	Encoder    meaning.Encoder
}

// Model translates only the selected thought request into a Responses API call.
type Model struct {
	model      string
	key        string
	url        string
	client     *http.Client
	disclosure DisclosurePolicy
	encoder    meaning.Encoder
}

func New(config Config, client *http.Client) (*Model, error) {
	key := strings.TrimSpace(config.APIKey)
	if key == "" {
		return nil, ErrConfig
	}

	model := strings.TrimSpace(config.Model)
	if model == "" {
		model = defaultModel
	}
	baseURL, err := endpoint(config.BaseURL)
	if err != nil {
		return nil, err
	}
	if client == nil {
		defaultClient := *http.DefaultClient
		defaultClient.Timeout = requestTimeout
		client = &defaultClient
	}
	clientCopy := *client
	clientCopy.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	client = &clientCopy

	return &Model{model: model, key: key, url: baseURL + "/responses", client: client,
		disclosure: config.Disclosure, encoder: config.Encoder}, nil
}

func endpoint(configured string) (string, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(configured), "/")
	if baseURL == "" {
		baseURL = defaultURL
	}
	parsedURL, err := url.ParseRequestURI(baseURL)
	if err != nil || parsedURL.Host == "" ||
		parsedURL.User != nil || parsedURL.RawQuery != "" || parsedURL.Fragment != "" {
		return "", ErrConfig
	}
	if parsedURL.Scheme != "https" && (parsedURL.Scheme != "http" || !loopback(parsedURL.Hostname())) {
		return "", ErrConfig
	}

	return baseURL, nil
}

func loopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	address := net.ParseIP(host)

	return address != nil && address.IsLoopback()
}

var _ thought.Model = (*Model)(nil)

type responseRequest struct {
	Model string       `json:"model"`
	Input string       `json:"input"`
	Store bool         `json:"store"`
	Text  responseText `json:"text"`
}

type responseText struct {
	Format responseFormat `json:"format"`
}

type responseFormat struct {
	Type   string          `json:"type"`
	Name   string          `json:"name"`
	Strict bool            `json:"strict"`
	Schema json.RawMessage `json:"schema"`
}

type responseEnvelope struct {
	Status string `json:"status"`
	Output []struct {
		Type    string `json:"type"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
}

type modelInput struct {
	Message    string           `json:"message"`
	At         time.Time        `json:"at"`
	Context    []shortTermInput `json:"context"`
	Working    []shortTermInput `json:"working"`
	Emotion    emotionInput     `json:"emotion"`
	Interests  []focusInput     `json:"interests"`
	Goals      goalInput        `json:"goals"`
	Fatigue    float64          `json:"fatigue"`
	Recalled   []recalledInput  `json:"recalled"`
	Prediction bool             `json:"prediction"`
}

type shortTermInput struct {
	ID         string     `json:"id"`
	Meaning    pointInput `json:"meaning"`
	OccurredAt time.Time  `json:"occurredAt"`
}

type pointInput struct {
	Text string `json:"text"`
}

type feelingInput struct {
	Meaning  pointInput `json:"meaning"`
	Strength float64    `json:"strength"`
}

type emotionInput struct {
	Feelings  []feelingInput `json:"feelings"`
	Readiness []feelingInput `json:"readiness"`
	Mood      emotion.Mood   `json:"mood"`
}

type focusInput struct {
	ID       string     `json:"id"`
	Target   pointInput `json:"target"`
	Affinity float64    `json:"affinity"`
}

type intentionInput struct {
	Meaning pointInput        `json:"meaning"`
	Claims  map[string]string `json:"claims"`
	Steps   []pointInput      `json:"steps"`
	Next    int               `json:"next"`
	Paused  bool              `json:"paused"`
}

type goalInput struct {
	Desires    map[string]pointInput     `json:"desires"`
	Intentions map[string]intentionInput `json:"intentions"`
}

type recalledInput struct {
	ID         string     `json:"id"`
	Meaning    pointInput `json:"meaning"`
	OccurredAt time.Time  `json:"occurredAt"`
	Emotion    float64    `json:"emotion"`
	Will       float64    `json:"will"`
	Strength   float64    `json:"strength"`
}

type interpretationJSON struct {
	Appraisal *emotion.Appraisal `json:"appraisal"`
	Interests []interest.Focus   `json:"interests"`
	Memory    *memory.Change     `json:"memory"`
	Goals     *goalsJSON         `json:"goals"`
}

type goalsJSON struct {
	Desires []desireJSON `json:"desires"`
	Adopt   []adoptJSON  `json:"adopt"`
	End     []string     `json:"end"`
}

type desireJSON struct {
	ID      string        `json:"id"`
	Meaning meaning.Point `json:"meaning"`
}

type adoptJSON struct {
	ID        string        `json:"id"`
	Intention intentionJSON `json:"intention"`
}

type intentionJSON struct {
	Meaning meaning.Point   `json:"meaning"`
	Claims  []claimJSON     `json:"claims"`
	Steps   []meaning.Point `json:"steps"`
	Next    int             `json:"next"`
	Paused  bool            `json:"paused"`
}

type claimJSON struct {
	Target string `json:"target"`
	Value  string `json:"value"`
}

func projectEmotion(state emotion.State) emotionInput {
	return emotionInput{
		Feelings: projectFeelings(state.Feelings), Readiness: projectFeelings(state.Readiness), Mood: state.Mood,
	}
}

func projectFeelings(feelings []emotion.Feeling) []feelingInput {
	projected := make([]feelingInput, len(feelings))
	for index, feeling := range feelings {
		projected[index] = feelingInput{
			Meaning: pointInput{Text: feeling.Meaning.Text}, Strength: feeling.Strength,
		}
	}

	return projected
}

func projectFocus(interests []interest.Focus) []focusInput {
	projected := make([]focusInput, len(interests))
	for index, focus := range interests {
		projected[index] = focusInput{
			ID: focus.ID, Target: pointInput{Text: focus.Target.Text}, Affinity: focus.Affinity,
		}
	}

	return projected
}

func projectGoals(state goal.State) goalInput {
	projected := goalInput{
		Desires:    make(map[string]pointInput, len(state.Desires)),
		Intentions: make(map[string]intentionInput, len(state.Intentions)),
	}
	for id, desire := range state.Desires {
		projected.Desires[id] = pointInput{Text: desire.Text}
	}
	for id, intention := range state.Intentions {
		steps := make([]pointInput, len(intention.Steps))
		for index, step := range intention.Steps {
			steps[index] = pointInput{Text: step.Text}
		}
		projected.Intentions[id] = intentionInput{
			Meaning: pointInput{Text: intention.Meaning.Text}, Claims: intention.Claims,
			Steps: steps, Next: intention.Next, Paused: intention.Paused,
		}
	}

	return projected
}

// Interpret calls Responses with store=false and sends only the supplied
// experience and recalled fragments. The API key is carried only in the
// Authorization header and is never included in model input or errors.
//
//nolint:cyclop,funlen,gocognit // Keep request and security checks ordered.
func (model *Model) Interpret(ctx context.Context, request thought.Request) (thought.Interpretation, error) {
	if request.Experience == nil {
		return thought.Interpretation{}, ErrResponse
	}
	recalled := make([]recalledInput, 0, len(request.Recalled))
	allowedIDs := make(map[string]struct{}, len(request.Recalled))
	context, allowedContext := projectShortTerm(request.Experience.Context, model.disclosure)
	working, allowedWorking := projectShortTerm(request.Experience.Working, model.disclosure)
	for _, fragment := range request.Recalled {
		if len(fragment.Sources) > 0 &&
			(model.disclosure == nil || !model.disclosure.AllowsDisclosure(slices.Clone(fragment.Sources), "openai")) {
			continue
		}
		recalled = append(recalled, recalledInput{ID: fragment.ID,
			Meaning: pointInput{Text: fragment.Meaning.Text}, OccurredAt: fragment.OccurredAt, Emotion: fragment.Emotion,
			Will: fragment.Will, Strength: fragment.Strength})
		allowedIDs[fragment.ID] = struct{}{}
	}
	input, err := json.Marshal(modelInput{
		Message: request.Query.Cue.Text, At: request.Query.At, Emotion: projectEmotion(request.Experience.Emotion),
		Context: context, Working: working,
		Interests: projectFocus(request.Experience.Interests), Goals: projectGoals(request.Experience.Goals),
		Fatigue: request.Experience.Fatigue, Recalled: recalled, Prediction: request.Prediction,
	})
	if err != nil {
		return thought.Interpretation{}, ErrResponse
	}
	if credentialMarker.Match(input) || containsCredentialString(input, model.key) {
		return thought.Interpretation{}, ErrResponse
	}
	// Recheck immediately before transmission because sharing policy can change
	// while the local request is being prepared.
	for groupIndex, fragments := range [][]memory.Fragment{request.Experience.Context, request.Experience.Working} {
		allowed := allowedContext
		if groupIndex == 1 {
			allowed = allowedWorking
		}
		for _, fragment := range fragments {
			if slices.Contains(allowed, fragment.ID) && len(fragment.Sources) > 0 &&
				(model.disclosure == nil || !model.disclosure.AllowsDisclosure(slices.Clone(fragment.Sources), "openai")) {
				return thought.Interpretation{}, ErrResponse
			}
		}
	}
	for _, fragment := range request.Recalled {
		if _, included := allowedIDs[fragment.ID]; included && len(fragment.Sources) > 0 &&
			(model.disclosure == nil || !model.disclosure.AllowsDisclosure(slices.Clone(fragment.Sources), "openai")) {
			return thought.Interpretation{}, ErrResponse
		}
	}

	body := responseRequest{
		Model: model.model,
		Input: inputGuide + string(input),
		Store: false,
		Text: responseText{Format: responseFormat{
			Type: "json_schema", Name: "akari_interpretation", Strict: true, Schema: json.RawMessage(interpretationSchema),
		}},
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return thought.Interpretation{}, ErrResponse
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, model.url, bytes.NewReader(encoded))
	if err != nil {
		return thought.Interpretation{}, ErrResponse
	}
	httpRequest.Header.Set("Authorization", "Bearer "+model.key)
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Accept", "application/json")

	result, err := model.client.Do(httpRequest)
	if err != nil {
		return thought.Interpretation{}, fmt.Errorf("openai: Responses request failed: %w", err)
	}
	defer func() { _ = result.Body.Close() }()
	if result.StatusCode < 200 || result.StatusCode >= 300 {
		return thought.Interpretation{}, fmt.Errorf("%w: %d", ErrStatus, result.StatusCode)
	}

	interpretation, err := decodeResponse(io.LimitReader(result.Body, maxResponse))
	if err != nil {
		return thought.Interpretation{}, err
	}
	if !validRecalledIDs(interpretation.Memory, allowedIDs) || model.encodeInterpretation(ctx, &interpretation) != nil {
		return thought.Interpretation{}, ErrResponse
	}
	restoreMemoryMetadata(interpretation.Memory, request, allowedContext, allowedWorking, allowedIDs)

	return interpretation, nil
}

func projectShortTerm(fragments []memory.Fragment, disclosure DisclosurePolicy) ([]shortTermInput, []string) {
	projected := make([]shortTermInput, 0, len(fragments))
	allowed := make([]string, 0, len(fragments))
	for _, fragment := range fragments {
		if len(fragment.Sources) > 0 &&
			(disclosure == nil || !disclosure.AllowsDisclosure(slices.Clone(fragment.Sources), "openai")) {
			continue
		}
		projected = append(projected, shortTermInput{ID: fragment.ID,
			Meaning: pointInput{Text: fragment.Meaning.Text}, OccurredAt: fragment.OccurredAt})
		allowed = append(allowed, fragment.ID)
	}

	return projected, allowed
}

func decodeResponse(reader io.Reader) (thought.Interpretation, error) {
	var envelope responseEnvelope
	if json.NewDecoder(reader).Decode(&envelope) != nil || envelope.Status != "completed" {
		return thought.Interpretation{}, ErrResponse
	}
	for _, item := range envelope.Output {
		if item.Type != "message" {
			continue
		}
		for _, content := range item.Content {
			if content.Type != "output_text" {
				continue
			}
			var payload interpretationJSON
			decoder := json.NewDecoder(strings.NewReader(content.Text))
			decoder.DisallowUnknownFields()
			//nolint:musttag // Every field in interpretationJSON has an explicit JSON tag.
			if decoder.Decode(&payload) != nil {
				return thought.Interpretation{}, ErrResponse
			}
			goals := convertGoals(payload.Goals)
			if payload.Goals != nil && goals == nil {
				return thought.Interpretation{}, ErrResponse
			}

			return thought.Interpretation{Appraisal: payload.Appraisal, Interests: payload.Interests,
				Memory: payload.Memory, Goals: goals}, nil
		}
	}

	return thought.Interpretation{}, ErrResponse
}

//nolint:cyclop // Validate duplicate and empty identities while translating each keyed entry.
func convertGoals(output *goalsJSON) *goal.Change {
	if output == nil {
		return nil
	}

	change := &goal.Change{
		Desires: make(map[string]meaning.Point, len(output.Desires)),
		Adopt:   make(map[string]goal.Intention, len(output.Adopt)), End: output.End,
	}
	for _, desire := range output.Desires {
		if desire.ID == "" {
			return nil
		}
		if _, exists := change.Desires[desire.ID]; exists {
			return nil
		}
		change.Desires[desire.ID] = desire.Meaning
	}
	for _, adopted := range output.Adopt {
		if adopted.ID == "" {
			return nil
		}
		if _, exists := change.Adopt[adopted.ID]; exists {
			return nil
		}
		claims := make(map[string]string, len(adopted.Intention.Claims))
		for _, claim := range adopted.Intention.Claims {
			if claim.Target == "" || claim.Value == "" {
				return nil
			}
			if _, exists := claims[claim.Target]; exists {
				return nil
			}
			claims[claim.Target] = claim.Value
		}
		change.Adopt[adopted.ID] = goal.Intention{
			Meaning: adopted.Intention.Meaning, Claims: claims,
			Steps: adopted.Intention.Steps, Next: adopted.Intention.Next, Paused: adopted.Intention.Paused,
		}
	}

	return change
}

func validRecalledIDs(change *memory.Change, allowed map[string]struct{}) bool {
	if change == nil {
		return true
	}
	for _, id := range change.Recalled {
		if _, ok := allowed[id]; !ok {
			return false
		}
	}

	return true
}

func restoreMemoryMetadata(change *memory.Change, request thought.Request, allowedContext, allowedWorking []string,
	allowedRecalled map[string]struct{}) {
	if change == nil {
		return
	}
	capacity := len(request.Experience.Context) + len(request.Experience.Working) + len(request.Recalled)
	trusted := make([]memory.Fragment, 0, capacity)
	trusted = appendAllowedFragments(trusted, request.Experience.Context, allowedContext)
	trusted = appendAllowedFragments(trusted, request.Experience.Working, allowedWorking)
	for _, fragment := range request.Recalled {
		if _, allowed := allowedRecalled[fragment.ID]; allowed {
			trusted = append(trusted, fragment)
		}
	}
	change.Context = filterWithheldCopies(change.Context, request.Experience.Context, allowedContext,
		request.Experience.Working, allowedWorking, request.Recalled, allowedRecalled)
	change.Working = filterWithheldCopies(change.Working, request.Experience.Context, allowedContext,
		request.Experience.Working, allowedWorking, request.Recalled, allowedRecalled)
	change.Remember = filterWithheldCopies(change.Remember, request.Experience.Context, allowedContext,
		request.Experience.Working, allowedWorking, request.Recalled, allowedRecalled)
	metadataByID := make(map[string]memory.Fragment, len(trusted))
	metadataByText := make(map[string][]memory.Fragment, len(trusted))
	for _, fragment := range trusted {
		metadataByID[fragment.ID] = fragment
		metadataByText[fragment.Meaning.Text] = append(metadataByText[fragment.Meaning.Text], fragment)
	}
	instant := trustedTime(request)
	restoreMemoryGroup(change.Context, metadataByID, metadataByText, instant)
	restoreMemoryGroup(change.Working, metadataByID, metadataByText, instant)
	restoreMemoryGroup(change.Remember, metadataByID, metadataByText, instant)
	change.Context = preserveWithheldFragments(change.Context, request.Experience.Context, allowedContext)
	change.Working = preserveWithheldFragments(change.Working, request.Experience.Working, allowedWorking)
}

func trustedTime(request thought.Request) time.Time {
	if !request.Query.At.IsZero() {
		return request.Query.At
	}
	if !request.Experience.Emotion.At.IsZero() {
		return request.Experience.Emotion.At
	}

	return time.Now().UTC()
}

func restoreMemoryGroup(output []memory.Fragment, metadataByID map[string]memory.Fragment,
	metadataByText map[string][]memory.Fragment, instant time.Time) {
	for index := range output {
		restoreFragmentMetadata(&output[index], metadataByID, metadataByText, instant)
	}
}

func restoreFragmentMetadata(fragment *memory.Fragment, metadataByID map[string]memory.Fragment,
	metadataByText map[string][]memory.Fragment, instant time.Time) {
	if original, exists := metadataByID[fragment.ID]; exists {
		fragment.OccurredAt = original.OccurredAt
		fragment.Sources = mergeSources(append([]memory.Fragment{original}, metadataByText[fragment.Meaning.Text]...))
		fragment.Accesses = original.Accesses

		return
	}
	matches := metadataByText[fragment.Meaning.Text]
	if len(matches) == 0 {
		fragment.OccurredAt = instant

		return
	}
	fragment.OccurredAt = matches[0].OccurredAt
	for _, original := range matches[1:] {
		if original.OccurredAt.Before(fragment.OccurredAt) {
			fragment.OccurredAt = original.OccurredAt
		}
	}
	fragment.Sources = mergeSources(matches)
}

func appendAllowedFragments(destination, fragments []memory.Fragment, allowedIDs []string) []memory.Fragment {
	for _, fragment := range fragments {
		if slices.Contains(allowedIDs, fragment.ID) {
			destination = append(destination, fragment)
		}
	}

	return destination
}

func preserveWithheldFragments(output, original []memory.Fragment, allowedIDs []string) []memory.Fragment {
	withheld, withheldText := findWithheldFragments(original, allowedIDs)
	if output == nil {
		return nil
	}
	result := memory.CloneFragments(output)
	seen := make(map[string]struct{}, len(output)+len(withheld))
	filtered := make([]memory.Fragment, 0, len(result)+len(withheld))
	for _, fragment := range result {
		if original, exists := withheld[fragment.ID]; exists {
			filtered = append(filtered, memory.CloneFragments([]memory.Fragment{original})[0])
			seen[original.ID] = struct{}{}

			continue
		}
		if _, privateCopy := withheldText[fragment.Meaning.Text]; privateCopy {
			continue
		}
		filtered = append(filtered, fragment)
	}
	for id, fragment := range withheld {
		if _, exists := seen[id]; !exists {
			filtered = append(filtered, memory.CloneFragments([]memory.Fragment{fragment})[0])
			seen[fragment.ID] = struct{}{}
		}
	}

	return filtered
}

func filterWithheldCopies(
	output, originalContext []memory.Fragment, allowedContext []string,
	originalWorking []memory.Fragment, allowedWorking []string,
	recalled []memory.Fragment, allowedRecalled map[string]struct{},
) []memory.Fragment {
	if output == nil {
		return nil
	}
	withheldIDs, withheldText := withheldMemory(originalContext, allowedContext, originalWorking, allowedWorking,
		recalled, allowedRecalled)
	filtered := make([]memory.Fragment, 0, len(output))
	for _, fragment := range output {
		if _, privateID := withheldIDs[fragment.ID]; privateID {
			continue
		}
		if _, privateText := withheldText[fragment.Meaning.Text]; privateText {
			continue
		}
		filtered = append(filtered, fragment)
	}

	return filtered
}

func withheldMemory(originalContext []memory.Fragment, allowedContext []string, originalWorking []memory.Fragment,
	allowedWorking []string, recalled []memory.Fragment, allowedRecalled map[string]struct{},
) (map[string]memory.Fragment, map[string]struct{}) {
	withheldIDs, withheldText := findWithheldFragments(originalContext, allowedContext)
	workingIDs, workingText := findWithheldFragments(originalWorking, allowedWorking)
	mergeWithheld(withheldIDs, withheldText, workingIDs, workingText)
	allowedRecallIDs := make([]string, 0, len(allowedRecalled))
	for id := range allowedRecalled {
		allowedRecallIDs = append(allowedRecallIDs, id)
	}
	recalledIDs, recalledText := findWithheldFragments(recalled, allowedRecallIDs)
	mergeWithheld(withheldIDs, withheldText, recalledIDs, recalledText)
	disclosedText := make(map[string]struct{})
	collectDisclosedText(disclosedText, originalContext, allowedContext)
	collectDisclosedText(disclosedText, originalWorking, allowedWorking)
	for _, fragment := range recalled {
		if _, allowed := allowedRecalled[fragment.ID]; allowed {
			disclosedText[fragment.Meaning.Text] = struct{}{}
		}
	}
	for text := range disclosedText {
		delete(withheldText, text)
	}

	return withheldIDs, withheldText
}

func collectDisclosedText(destination map[string]struct{}, fragments []memory.Fragment, allowedIDs []string) {
	for _, fragment := range fragments {
		if slices.Contains(allowedIDs, fragment.ID) {
			destination[fragment.Meaning.Text] = struct{}{}
		}
	}
}

func mergeWithheld(
	destinationIDs map[string]memory.Fragment, destinationText map[string]struct{},
	sourceIDs map[string]memory.Fragment, sourceText map[string]struct{},
) {
	maps.Copy(destinationIDs, sourceIDs)
	maps.Copy(destinationText, sourceText)
}

func findWithheldFragments(
	original []memory.Fragment, allowedIDs []string,
) (map[string]memory.Fragment, map[string]struct{}) {
	allowed := make(map[string]struct{}, len(allowedIDs))
	for _, id := range allowedIDs {
		allowed[id] = struct{}{}
	}
	withheld := make(map[string]memory.Fragment)
	withheldText := make(map[string]struct{})
	allowedText := make(map[string]struct{})
	for _, fragment := range original {
		if _, sent := allowed[fragment.ID]; sent {
			allowedText[fragment.Meaning.Text] = struct{}{}
		}
	}
	for _, fragment := range original {
		if _, sent := allowed[fragment.ID]; sent {
			continue
		}
		withheld[fragment.ID] = fragment
		if _, disclosedCopy := allowedText[fragment.Meaning.Text]; !disclosedCopy {
			withheldText[fragment.Meaning.Text] = struct{}{}
		}
	}

	return withheld, withheldText
}

func mergeSources(matches []memory.Fragment) []string {
	sources := make([]string, 0)
	for _, fragment := range matches {
		for _, source := range fragment.Sources {
			if !slices.Contains(sources, source) {
				sources = append(sources, source)
			}
		}
	}

	return sources
}

//nolint:cyclop,gocognit // It visits each semantic point-bearing field once by domain group.
func (model *Model) encodeInterpretation(ctx context.Context, interpretation *thought.Interpretation) error {
	if interpretation.Appraisal != nil {
		for _, feelings := range [][]emotion.Feeling{
			interpretation.Appraisal.Feelings, interpretation.Appraisal.Others, interpretation.Appraisal.Readiness,
		} {
			for index := range feelings {
				err := model.encodePoint(ctx, &feelings[index].Meaning)
				if err != nil {
					return err
				}
			}
		}
	}
	for index := range interpretation.Interests {
		err := model.encodePoint(ctx, &interpretation.Interests[index].Target)
		if err != nil {
			return err
		}
	}
	if interpretation.Memory != nil {
		for _, fragments := range [][]memory.Fragment{
			interpretation.Memory.Context, interpretation.Memory.Working, interpretation.Memory.Remember,
		} {
			for index := range fragments {
				err := model.encodePoint(ctx, &fragments[index].Meaning)
				if err != nil {
					return err
				}
			}
		}
	}
	if interpretation.Goals != nil {
		for goalID, point := range interpretation.Goals.Desires {
			err := model.encodePoint(ctx, &point)
			if err != nil {
				return err
			}
			interpretation.Goals.Desires[goalID] = point
		}
		for goalID, intention := range interpretation.Goals.Adopt {
			err := model.encodePoint(ctx, &intention.Meaning)
			if err != nil {
				return err
			}
			for index := range intention.Steps {
				err := model.encodePoint(ctx, &intention.Steps[index])
				if err != nil {
					return err
				}
			}
			interpretation.Goals.Adopt[goalID] = intention
		}
	}

	return nil
}

func (model *Model) encodePoint(ctx context.Context, point *meaning.Point) error {
	if point.Text == "" || model.encoder == nil {
		return ErrResponse
	}
	encoded, err := model.encoder.Encode(ctx, point.Text)
	if err != nil || !encoded.Valid() {
		return ErrResponse
	}
	point.Space = encoded.Space
	point.Vector = slices.Clone(encoded.Vector)

	return nil
}

type disclosureFunc func(sources []string, recipient string) bool

func (policy disclosureFunc) AllowsDisclosure(sources []string, recipient string) bool {
	return policy(sources, recipient)
}

type encoderFunc func(context.Context, string) (meaning.Point, error)

func (encoder encoderFunc) Encode(ctx context.Context, text string) (meaning.Point, error) {
	return encoder(ctx, text)
}

func containsCredentialString(encoded []byte, configuredKeys ...string) bool {
	decoder := json.NewDecoder(strings.NewReader(string(encoded)))
	decoder.UseNumber()
	var input any
	if decoder.Decode(&input) != nil {
		return true
	}

	return scanCredentialValue(input, configuredKeys...)
}

//nolint:cyclop // The recursive walk checks both values and object keys.
func scanCredentialValue(value any, configuredKeys ...string) bool {
	switch item := value.(type) {
	case string:
		return credentialMarker.MatchString(item) || containsConfiguredKey(item, configuredKeys)
	case []any:
		for _, nested := range item {
			if scanCredentialValue(nested, configuredKeys...) {
				return true
			}
		}
	case map[string]any:
		for key, nested := range item {
			if credentialMarker.MatchString(key) || containsConfiguredKey(key, configuredKeys) {
				return true
			}
			if scanCredentialValue(nested, configuredKeys...) {
				return true
			}
		}
	}

	return false
}

func containsConfiguredKey(value string, configuredKeys []string) bool {
	for _, key := range configuredKeys {
		if key != "" && strings.Contains(value, key) {
			return true
		}
	}

	return false
}
