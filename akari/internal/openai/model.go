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
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/kizuna-org/akari/internal/emotion"
	"github.com/kizuna-org/akari/internal/goal"
	"github.com/kizuna-org/akari/internal/interest"
	"github.com/kizuna-org/akari/internal/meaning"
	"github.com/kizuna-org/akari/internal/memory"
	"github.com/kizuna-org/akari/internal/thought"
)

const (
	defaultModel = "gpt-6-luna"
	defaultURL   = "https://api.openai.com/v1"
	maxResponse  = 1 << 20
	inputGuide   = "Interpret the selected Akari experience. Return JSON with appraisal, interests, memory, goals. " +
		"Do not infer execution permissions. Input: "
)

var (
	ErrConfig   = errors.New("OpenAI model requires an API key and valid configuration")
	ErrResponse = errors.New("OpenAI Responses API returned an invalid response")
	ErrStatus   = errors.New("OpenAI Responses API returned a non-success status")
)

var credentialMarker = regexp.MustCompile(
	`(?i)(sk-[a-z0-9_-]{8,}|bearer\s+[a-z0-9._-]{12,}|` +
		`(?:api[_ -]?key|password|secret)(?:\\?["'])?\s*(?:\\?["'])?\s*[:=]\s*(?:\\?["'])?\S+)`,
)

// Config contains deployment-specific API settings. APIKey is never included
// in errors or serialized request bodies.
type Config struct {
	Model   string
	APIKey  string
	BaseURL string
}

// Model translates only the selected thought request into a Responses API call.
type Model struct {
	model  string
	key    string
	url    string
	client *http.Client
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
		client = http.DefaultClient
	}

	return &Model{model: model, key: key, url: baseURL + "/responses", client: client}, nil
}

func endpoint(configured string) (string, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(configured), "/")
	if baseURL == "" {
		baseURL = defaultURL
	}
	parsedURL, err := url.ParseRequestURI(baseURL)
	if err != nil || (parsedURL.Scheme != "https" && parsedURL.Scheme != "http") || parsedURL.Host == "" ||
		parsedURL.User != nil || parsedURL.RawQuery != "" || parsedURL.Fragment != "" {
		return "", ErrConfig
	}

	return baseURL, nil
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
	Type string `json:"type"`
}

type responseEnvelope struct {
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
	Emotion    emotion.State    `json:"emotion"`
	Interests  []interest.Focus `json:"interests"`
	Goals      goal.State       `json:"goals"`
	Fatigue    float64          `json:"fatigue"`
	Recalled   []recalledInput  `json:"recalled"`
	Prediction bool             `json:"prediction"`
}

type recalledInput struct {
	Meaning  meaning.Point `json:"meaning"`
	Emotion  float64       `json:"emotion"`
	Will     float64       `json:"will"`
	Strength float64       `json:"strength"`
}

type interpretationJSON struct {
	Appraisal *emotion.Appraisal `json:"appraisal"`
	Interests []interest.Focus   `json:"interests"`
	Memory    *memory.Change     `json:"memory"`
	Goals     *goal.Change       `json:"goals"`
}

// Interpret calls Responses with store=false and sends only the supplied
// experience and recalled fragments. The API key is carried only in the
// Authorization header and is never included in model input or errors.
func (model *Model) Interpret(ctx context.Context, request thought.Request) (thought.Interpretation, error) {
	if request.Experience == nil {
		return thought.Interpretation{}, ErrResponse
	}
	recalled := make([]recalledInput, 0, len(request.Recalled))
	for _, fragment := range request.Recalled {
		recalled = append(recalled, recalledInput{Meaning: fragment.Meaning, Emotion: fragment.Emotion,
			Will: fragment.Will, Strength: fragment.Strength})
	}
	input, err := json.Marshal(modelInput{
		Message: request.Query.Cue.Text, Emotion: request.Experience.Emotion,
		Interests: request.Experience.Interests, Goals: request.Experience.Goals,
		Fatigue: request.Experience.Fatigue, Recalled: recalled, Prediction: request.Prediction,
	})
	if err != nil {
		return thought.Interpretation{}, ErrResponse
	}
	if credentialMarker.Match(input) {
		return thought.Interpretation{}, ErrResponse
	}

	body := responseRequest{
		Model: model.model,
		Input: inputGuide + string(input),
		Store: false,
		Text:  responseText{Format: responseFormat{Type: "json_object"}},
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
		return thought.Interpretation{}, fmt.Errorf("OpenAI Responses request failed: %w", err)
	}
	defer func() { _ = result.Body.Close() }()
	if result.StatusCode < 200 || result.StatusCode >= 300 {
		return thought.Interpretation{}, fmt.Errorf("%w: %d", ErrStatus, result.StatusCode)
	}

	return decodeResponse(io.LimitReader(result.Body, maxResponse))
}

func decodeResponse(reader io.Reader) (thought.Interpretation, error) {
	var envelope responseEnvelope
	if json.NewDecoder(reader).Decode(&envelope) != nil {
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
			//nolint:musttag // Wire keys are tagged; nested domain structs intentionally retain their Go field names.
			if json.Unmarshal([]byte(content.Text), &payload) != nil {
				return thought.Interpretation{}, ErrResponse
			}

			return thought.Interpretation{Appraisal: payload.Appraisal, Interests: payload.Interests,
				Memory: payload.Memory, Goals: payload.Goals}, nil
		}
	}

	return thought.Interpretation{}, ErrResponse
}
