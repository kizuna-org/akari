//nolint:cyclop,exhaustruct,gocognit,wsl,wsl_v5 // Cases stay grouped to keep protocol assertions together.
package openai

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/kizuna-org/akari/internal/emotion"
	"github.com/kizuna-org/akari/internal/goal"
	"github.com/kizuna-org/akari/internal/interest"
	"github.com/kizuna-org/akari/internal/meaning"
	"github.com/kizuna-org/akari/internal/memory"
	"github.com/kizuna-org/akari/internal/mind"
	"github.com/kizuna-org/akari/internal/thought"
)

const defaultCaseName = "defaults"
const testKey = "test-secret-key"
const testFragmentID = "fragment-1"
const trustedSpace = "trusted-v1"
const semanticText = "meaning"
const testGoalID = "goal"
const unrelatedLocalData = "unrelated-local-data"
const tabBearer = "Bearer\tabcdefghijklmnop"
const testCustomKey = "custom-key-123"
const conversationSource = "conversation"
const openaiRecipient = "openai"
const jsonAppraisal = "appraisal"
const jsonContext = "context"
const jsonGoals = "goals"
const jsonInterests = "interests"
const jsonMemory = "memory"
const jsonRecalled = "recalled"
const jsonRemember = "remember"
const jsonWorking = "working"
const validInterpretation = `{"appraisal":null,"interests":[],"memory":null,"goals":null}`

func TestNewConfiguration(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{name: defaultCaseName, config: Config{APIKey: testKey}},
		{name: "custom model and loopback HTTP", config: Config{APIKey: testKey, Model: "model-x",
			BaseURL: "http://localhost:8080/v1/"}},
		{name: "loopback IPv6", config: Config{APIKey: testKey, BaseURL: "http://[::1]:8080/v1"}},
		{name: "HTTPS remote", config: Config{APIKey: testKey, BaseURL: "https://api.example.com/v1"}},
		{name: "missing key", config: Config{}, wantErr: true},
		{name: "invalid URL", config: Config{APIKey: testKey, BaseURL: "example.invalid"}, wantErr: true},
		{name: "remote HTTP", config: Config{APIKey: testKey, BaseURL: "http://api.example.com/v1"}, wantErr: true},
		{name: "lookalike localhost", config: Config{APIKey: testKey, BaseURL: "http://localhost.evil/v1"}, wantErr: true},
		{name: "missing URL host", config: Config{APIKey: testKey, BaseURL: "https://"}, wantErr: true},
		{name: "URL user info", config: Config{APIKey: testKey, BaseURL: "https://user@example.invalid"}, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			model, err := New(test.config, nil)
			if test.wantErr {
				if model != nil || !errors.Is(err, ErrConfig) {
					t.Fatalf("New() = %v, %v; want config error", model, err)
				}

				return
			}
			if err != nil || model == nil {
				t.Fatalf("New() = %v, %v; want model", model, err)
			}
			if test.name == defaultCaseName && (model.model != defaultModel || model.url != defaultURL+"/responses") {
				t.Fatalf("defaults = model %q URL %q", model.model, model.url)
			}
			if test.name == defaultCaseName && model.client.Timeout != requestTimeout {
				t.Fatalf("default client timeout = %s, want %s", model.client.Timeout, requestTimeout)
			}
		})
	}
}

func TestInterpretDisclosureAndRecalledIDs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		policy       DisclosurePolicy
		wantRecalled []string
		wantInputID  bool
	}{
		{name: "no policy omits sourced memory"},
		{name: "approved policy sends source and preserves ID",
			policy: disclosureFunc(func(sources []string, recipient string) bool {
				return recipient == openaiRecipient && strings.Contains(strings.Join(sources, ","), conversationSource)
			}), wantRecalled: []string{testFragmentID}, wantInputID: true},
		{name: "denied policy omits sourced memory", policy: disclosureFunc(func([]string, string) bool { return false })},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var input string
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				var body responseRequest
				decodeErr := json.NewDecoder(request.Body).Decode(&body)
				if decodeErr != nil {
					t.Errorf("decode request: %v", decodeErr)
				}
				input = body.Input
				output, marshalErr := json.Marshal(map[string]any{
					jsonAppraisal: nil, jsonInterests: []any{},
					jsonMemory: map[string]any{
						jsonContext: nil, jsonWorking: nil, jsonRemember: []any{}, jsonRecalled: test.wantRecalled,
					},
					jsonGoals: nil,
				})
				if marshalErr != nil {
					t.Errorf("marshal response: %v", marshalErr)
				}
				_, _ = writer.Write([]byte(responseWithStatus("completed", string(output))))
			}))
			defer server.Close()
			model, err := New(Config{APIKey: testKey, BaseURL: server.URL, Disclosure: test.policy}, server.Client())
			if err != nil {
				t.Fatal(err)
			}
			request := thought.Request{Experience: &mind.Experience{}, Recalled: []memory.Fragment{{
				ID: testFragmentID, Meaning: meaning.Point{Text: "private detail"}, Sources: []string{conversationSource},
			}}}
			result, err := model.Interpret(context.Background(), request)
			if err != nil {
				t.Fatal(err)
			}
			gotID := strings.Contains(input, testFragmentID)
			if gotID != test.wantInputID {
				t.Fatalf("input ID presence = %t, input %q", strings.Contains(input, testFragmentID), input)
			}
			if result.Memory == nil || strings.Join(result.Memory.Recalled, ",") != strings.Join(test.wantRecalled, ",") {
				t.Fatalf("recalled IDs = %#v, want %#v", result.Memory, test.wantRecalled)
			}
		})
	}
}

func TestInterpretRejectsUntrustedRecalledIDs(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		payload, marshalErr := json.Marshal(map[string]any{
			jsonAppraisal: nil, "interests": []any{}, "memory": map[string]any{
				jsonContext: nil, "working": nil, jsonRemember: []any{}, jsonRecalled: []string{"not-recalled"},
			}, jsonGoals: nil,
		})
		if marshalErr != nil {
			t.Errorf("marshal response: %v", marshalErr)
		}
		_, _ = writer.Write([]byte(responseWithStatus("completed", string(payload))))
	}))
	defer server.Close()
	model, err := New(Config{APIKey: testKey, BaseURL: server.URL}, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	_, err = model.Interpret(context.Background(), thought.Request{Experience: &mind.Experience{}})
	if !errors.Is(err, ErrResponse) {
		t.Fatalf("Interpret() error = %v, want %v", err, ErrResponse)
	}
}

func TestInterpretRechecksDisclosureBeforeSending(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		decisions   []bool
		wantRequest bool
		wantError   bool
	}{
		{name: "sharing remains allowed", decisions: []bool{true, true}, wantRequest: true},
		{name: "sharing revoked before send", decisions: []bool{true, false}, wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var policyCalls int
			policy := disclosureFunc(func([]string, string) bool {
				decision := test.decisions[policyCalls]
				policyCalls++

				return decision
			})
			var serverCalled bool
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				serverCalled = true
				_, _ = writer.Write([]byte(responseWithStatus("completed", validInterpretation)))
			}))
			defer server.Close()
			model, err := New(Config{APIKey: testKey, BaseURL: server.URL, Disclosure: policy}, server.Client())
			if err != nil {
				t.Fatal(err)
			}
			request := thought.Request{Experience: &mind.Experience{}, Recalled: []memory.Fragment{{
				ID: testFragmentID, Meaning: meaning.Point{Text: "private detail"}, Sources: []string{"conversation"},
			}}}
			_, err = model.Interpret(context.Background(), request)
			if (err != nil) != test.wantError {
				t.Fatalf("Interpret() error = %v, wantError %t", err, test.wantError)
			}
			if serverCalled != test.wantRequest || policyCalls != len(test.decisions) {
				t.Fatalf("serverCalled = %t, policyCalls = %d", serverCalled, policyCalls)
			}
		})
	}
}

func TestInterpretRejectsConfiguredAPIKeyInCue(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		key  string
		cue  string
	}{
		{name: "key alone", key: testCustomKey, cue: testCustomKey},
		{name: "key embedded", key: testCustomKey, cue: "please repeat " + testCustomKey + " now"},
		{name: "JSON escaped key", key: `custom"key-123`, cue: `please repeat custom"key-123`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			called := false
			server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
			defer server.Close()
			model, err := New(Config{APIKey: test.key, BaseURL: server.URL}, server.Client())
			if err != nil {
				t.Fatal(err)
			}
			request := thought.Request{Experience: &mind.Experience{}, Query: memory.Query{Cue: meaning.Point{Text: test.cue}}}
			_, err = model.Interpret(context.Background(), request)
			if !errors.Is(err, ErrResponse) || called {
				t.Fatalf("Interpret error=%v called=%t", err, called)
			}
		})
	}
}

func TestInterpretRestoresSourcesForKnownMemoryCopies(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		group string
	}{
		{name: jsonWorking, group: jsonWorking},
		{name: jsonContext, group: jsonContext},
		{name: jsonRemember, group: jsonRemember},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			fragment := memory.Fragment{ID: testFragmentID, Meaning: meaning.Point{Text: "known detail"},
				OccurredAt: time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC), Strength: 0.5, Sources: []string{conversationSource}}
			var policyAllows = true
			policy := disclosureFunc(func(sources []string, recipient string) bool {
				return policyAllows && recipient == openaiRecipient && slices.Equal(sources, []string{conversationSource})
			})
			var sentInput string
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				var body responseRequest
				decodeErr := json.NewDecoder(request.Body).Decode(&body)
				if decodeErr != nil {
					t.Errorf("decode request: %v", decodeErr)
				}
				sentInput = body.Input
				fragments := []any{map[string]any{"id": fragment.ID, "meaning": map[string]any{"text": fragment.Meaning.Text},
					"occurredAt": fragment.OccurredAt.Format(time.RFC3339), "emotion": 0, "will": 0, "strength": fragment.Strength}}
				memoryOutput := map[string]any{jsonContext: nil, jsonWorking: nil, jsonRemember: []any{}, jsonRecalled: []string{}}
				memoryOutput[test.group] = fragments
				payload, marshalErr := json.Marshal(map[string]any{
					jsonAppraisal: nil, "interests": []any{}, "memory": memoryOutput, jsonGoals: nil,
				})
				if marshalErr != nil {
					t.Errorf("marshal response: %v", marshalErr)
				}
				_, _ = writer.Write([]byte(responseWithStatus("completed", string(payload))))
			}))
			defer server.Close()
			model, err := New(Config{APIKey: testKey, BaseURL: server.URL, Disclosure: policy,
				Encoder: encoderFunc(func(_ context.Context, text string) (meaning.Point, error) {
					return meaning.Point{Text: text, Space: trustedSpace, Vector: []float64{1}}, nil
				})}, server.Client())
			if err != nil {
				t.Fatal(err)
			}
			experience := &mind.Experience{}
			experience.Working = []memory.Fragment{fragment}
			result, err := model.Interpret(context.Background(), thought.Request{Experience: experience})
			if err != nil {
				t.Fatal(err)
			}
			var copies []memory.Fragment
			switch test.group {
			case "working":
				copies = result.Memory.Working
			case "context":
				copies = result.Memory.Context
			case "remember":
				copies = result.Memory.Remember
			}
			if len(copies) != 1 || !slices.Equal(copies[0].Sources, fragment.Sources) {
				t.Fatalf("copied sources = %#v", copies)
			}
			state, err := memory.Apply(memory.State{}, *result.Memory)
			if err != nil {
				t.Fatal(err)
			}
			policyAllows = false
			_, err = model.Interpret(context.Background(), thought.Request{Experience: &mind.Experience{Working: state.Working}})
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(sentInput, "known detail") {
				t.Fatal("revoked memory was sent")
			}
		})
	}
}

func TestInterpretEncodesSemanticPointsLocally(t *testing.T) {
	t.Parallel()
	encoder := encoderFunc(func(_ context.Context, text string) (meaning.Point, error) {
		return meaning.Point{Text: text, Space: trustedSpace, Vector: []float64{0.5, 0.5}}, nil
	})
	model := &Model{encoder: encoder}
	point := meaning.Point{Text: semanticText, Space: "model-invented", Vector: []float64{99}}
	interpretation := thought.Interpretation{
		Appraisal: &emotion.Appraisal{
			Feelings: []emotion.Feeling{{Meaning: point}}, Others: []emotion.Feeling{{Meaning: point}},
			Readiness: []emotion.Feeling{{Meaning: point}},
		},
		Interests: []interest.Focus{{Target: point}},
		Memory: &memory.Change{
			Context: []memory.Fragment{{Meaning: point}}, Working: []memory.Fragment{{Meaning: point}},
			Remember: []memory.Fragment{{Meaning: point}},
		},
		Goals: &goal.Change{
			Desires: map[string]meaning.Point{"desire": point},
			Adopt:   map[string]goal.Intention{"intention": {Meaning: point, Steps: []meaning.Point{point}}},
		},
	}
	err := model.encodeInterpretation(context.Background(), &interpretation)
	if err != nil {
		t.Fatal(err)
	}
	if interpretation.Interests[0].Target.Space != trustedSpace || len(interpretation.Interests[0].Target.Vector) != 2 {
		t.Fatalf("semantic point was not replaced by encoder output: %+v", interpretation.Interests[0].Target)
	}
	if interpretation.Goals.Desires["desire"].Space != trustedSpace ||
		interpretation.Goals.Adopt["intention"].Steps[0].Space != trustedSpace {
		t.Fatalf("goal semantic points were not encoded: %+v", interpretation.Goals)
	}
}

func TestEncodeInterpretationRejectsMissingOrInvalidEncoder(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		encoder meaning.Encoder
	}{
		{name: "missing"},
		{name: "invalid vector", encoder: encoderFunc(func(_ context.Context, text string) (meaning.Point, error) {
			return meaning.Point{Text: text, Space: "empty", Vector: []float64{0}}, nil
		})},
		{name: "encoder error", encoder: encoderFunc(func(context.Context, string) (meaning.Point, error) {
			return meaning.Point{}, ErrResponse
		})},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			model := &Model{encoder: test.encoder}
			interpretation := thought.Interpretation{Interests: []interest.Focus{{Target: meaning.Point{Text: semanticText}}}}
			err := model.encodeInterpretation(context.Background(), &interpretation)
			if !errors.Is(err, ErrResponse) {
				t.Fatalf("encodeInterpretation() error = %v, want %v", err, ErrResponse)
			}
		})
	}
}

func TestEncodeInterpretationRejectsEncoderFailureInEveryDomain(t *testing.T) {
	t.Parallel()
	point := meaning.Point{Text: semanticText}
	tests := []struct {
		name           string
		interpretation thought.Interpretation
		failAfterFirst bool
	}{
		{name: "appraisal feeling", interpretation: thought.Interpretation{
			Appraisal: &emotion.Appraisal{Feelings: []emotion.Feeling{{Meaning: point}}},
		}},
		{name: "appraisal other", interpretation: thought.Interpretation{
			Appraisal: &emotion.Appraisal{Others: []emotion.Feeling{{Meaning: point}}},
		}},
		{name: "appraisal readiness", interpretation: thought.Interpretation{
			Appraisal: &emotion.Appraisal{Readiness: []emotion.Feeling{{Meaning: point}}},
		}},
		{name: "memory context", interpretation: thought.Interpretation{
			Memory: &memory.Change{Context: []memory.Fragment{{Meaning: point}}},
		}},
		{name: "memory working", interpretation: thought.Interpretation{
			Memory: &memory.Change{Working: []memory.Fragment{{Meaning: point}}},
		}},
		{name: "memory remember", interpretation: thought.Interpretation{
			Memory: &memory.Change{Remember: []memory.Fragment{{Meaning: point}}},
		}},
		{name: "goal desire", interpretation: thought.Interpretation{
			Goals: &goal.Change{Desires: map[string]meaning.Point{"desire": point}},
		}},
		{name: "goal adopted meaning", interpretation: thought.Interpretation{
			Goals: &goal.Change{Adopt: map[string]goal.Intention{testGoalID: {Meaning: point}}},
		}},
		{name: "goal adopted step", interpretation: thought.Interpretation{
			Goals: &goal.Change{Adopt: map[string]goal.Intention{testGoalID: {
				Meaning: point, Steps: []meaning.Point{{}},
			}}},
		}, failAfterFirst: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			calls := 0
			model := &Model{encoder: encoderFunc(func(context.Context, string) (meaning.Point, error) {
				calls++
				if test.failAfterFirst && calls == 1 {
					return meaning.Point{Text: semanticText, Space: trustedSpace, Vector: []float64{1}}, nil
				}

				return meaning.Point{}, ErrResponse
			})}
			err := model.encodeInterpretation(context.Background(), &test.interpretation)
			if !errors.Is(err, ErrResponse) {
				t.Fatalf("encodeInterpretation() error = %v, want %v", err, ErrResponse)
			}
		})
	}
}

func TestCredentialScanner(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		json []byte
		want bool
	}{
		{name: "malformed JSON", json: []byte("{"), want: true},
		{name: "safe nested values", json: []byte(`{"items":[1,true,{"note":"ordinary text"}]}`)},
		{name: "credential assignment", json: []byte(`{"note":"password\n= private"}`), want: true},
		{name: "credential marker in decoded map key", json: []byte("{\"Bearer\\tabcdefghijklmnop\":\"value\"}"), want: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := containsCredentialString(test.json); got != test.want {
				t.Fatalf("containsCredentialString() = %t, want %t", got, test.want)
			}
		})
	}
	if !scanCredentialValue([]any{"ordinary text", map[string]any{"token": tabBearer}}) {
		t.Fatal("scanCredentialValue() missed nested credential")
	}
	if scanCredentialValue(float64(42)) {
		t.Fatal("scanCredentialValue() treated a number as a credential")
	}
}

func TestInterpretShortTermContextAndRedirectProtection(t *testing.T) {
	t.Parallel()
	t.Run("short term context is text only and disclosure filtered", func(t *testing.T) {
		t.Parallel()
		var captured string
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			var body responseRequest
			decodeErr := json.NewDecoder(request.Body).Decode(&body)
			if decodeErr != nil {
				t.Error(decodeErr)
			}
			captured = body.Input
			_, _ = writer.Write([]byte(responseWithStatus("completed", validInterpretation)))
		}))
		defer server.Close()
		policy := disclosureFunc(func(sources []string, recipient string) bool {
			return recipient == "openai" && slices.Contains(sources, "approved")
		})
		model, err := New(Config{APIKey: testKey, BaseURL: server.URL, Disclosure: policy}, server.Client())
		if err != nil {
			t.Fatal(err)
		}
		experience := &mind.Experience{
			Context: []memory.Fragment{{ID: "ctx", Meaning: meaning.Point{Text: "project is a garden"},
				Sources: []string{"approved"}}},
			Working: []memory.Fragment{{ID: "work", Meaning: meaning.Point{Text: "continue planting"},
				Sources: []string{"denied"}}},
		}
		_, err = model.Interpret(context.Background(), thought.Request{Experience: experience,
			Query: memory.Query{Cue: meaning.Point{Text: "continue"}}})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(captured, "project is a garden") || strings.Contains(captured, "continue planting") ||
			strings.Contains(captured, `"id":"ctx"`) || strings.Contains(captured, `"sources"`) {
			t.Fatalf("unexpected short-term projection: %s", captured)
		}
	})
	t.Run("rejects redirects without mutating caller client", func(t *testing.T) {
		t.Parallel()
		for _, status := range []int{http.StatusTemporaryRedirect, http.StatusPermanentRedirect} {
			t.Run(http.StatusText(status), func(t *testing.T) {
				t.Parallel()
				targetCalled := false
				target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { targetCalled = true }))
				defer target.Close()
				redirect := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
					http.Redirect(writer, request, target.URL, status)
				}))
				defer redirect.Close()
				client := redirect.Client()
				model, err := New(Config{APIKey: testKey, BaseURL: redirect.URL}, client)
				if err != nil {
					t.Fatal(err)
				}
				_, err = model.Interpret(context.Background(), thought.Request{Experience: &mind.Experience{}})
				if err == nil || targetCalled {
					t.Fatalf("Interpret error=%v targetCalled=%t", err, targetCalled)
				}
				if model.client == client || client.CheckRedirect != nil {
					t.Fatal("New did not keep redirect policy on a private client copy")
				}
			})
		}
	})
	t.Run("rejects HTTPS to HTTP redirect", func(t *testing.T) {
		t.Parallel()
		targetCalled := false
		target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { targetCalled = true }))
		defer target.Close()
		secure := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			http.Redirect(writer, request, target.URL, http.StatusTemporaryRedirect)
		}))
		defer secure.Close()
		model, err := New(Config{APIKey: testKey, BaseURL: secure.URL}, secure.Client())
		if err != nil {
			t.Fatal(err)
		}
		_, err = model.Interpret(context.Background(), thought.Request{Experience: &mind.Experience{}})
		if err == nil || targetCalled {
			t.Fatalf("Interpret error=%v targetCalled=%t", err, targetCalled)
		}
	})
	t.Run("rejects credential in short term context", func(t *testing.T) {
		t.Parallel()
		called := false
		server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
		defer server.Close()
		model, err := New(Config{APIKey: testKey, BaseURL: server.URL}, server.Client())
		if err != nil {
			t.Fatal(err)
		}
		experience := &mind.Experience{Context: []memory.Fragment{{ID: "ctx",
			Meaning: meaning.Point{Text: tabBearer}}}}
		_, err = model.Interpret(context.Background(), thought.Request{Experience: experience})
		if !errors.Is(err, ErrResponse) || called {
			t.Fatalf("Interpret error=%v called=%t", err, called)
		}
	})
}

func TestInterpretResponses(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		status   int
		response string
		message  string
		wantErr  bool
	}{
		{name: "interprets selected context", status: http.StatusOK,
			response: responseWithStatus("completed", validInterpretation)},
		{name: "incomplete response", status: http.StatusOK,
			response: responseWithStatus("incomplete", validInterpretation), wantErr: true},
		{name: "server error", status: http.StatusUnauthorized, response: `{"error":"not included"}`, wantErr: true},
		{name: "malformed envelope", status: http.StatusOK, response: `{`, wantErr: true},
		{name: "missing output text", status: http.StatusOK, response: `{"status":"completed","output":[]}`, wantErr: true},
		{name: "invalid interpretation", status: http.StatusOK,
			response: responseWithStatus("completed", "not-json"), wantErr: true},
		{name: "credential marker rejected", status: http.StatusOK,
			message: `{"password":"do-not-send-this"}`, wantErr: true},
		{name: "tab bearer token rejected", status: http.StatusOK,
			message: "Bearer\tabcdefghijklmnop", wantErr: true},
		{name: "newline password rejected", status: http.StatusOK,
			message: "password\n= do-not-send-this", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var called bool
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				called = true
				if got := request.Header.Get("Authorization"); got != "Bearer "+testKey {
					t.Errorf("authorization = %q", got)
				}
				var body struct {
					Model string `json:"model"`
					Input string `json:"input"`
					Store *bool  `json:"store"`
					Text  struct {
						Format struct {
							Type   string          `json:"type"`
							Strict bool            `json:"strict"`
							Name   string          `json:"name"`
							Schema json.RawMessage `json:"schema"`
						} `json:"format"`
					} `json:"text"`
				}
				err := json.NewDecoder(request.Body).Decode(&body)
				if err != nil {
					t.Errorf("decode body: %v", err)
				}
				if body.Model != defaultModel || body.Store == nil || *body.Store || strings.Contains(body.Input, testKey) {
					t.Errorf("unsafe request body: %+v", body)
				}
				if body.Text.Format.Type != "json_schema" || !body.Text.Format.Strict ||
					body.Text.Format.Name != "akari_interpretation" ||
					!json.Valid(body.Text.Format.Schema) {
					t.Errorf("invalid output contract: %+v", body.Text.Format)
				}
				var schema struct {
					Required             []string                   `json:"required"`
					AdditionalProperties bool                       `json:"additionalProperties"`
					Properties           map[string]json.RawMessage `json:"properties"`
				}
				schemaErr := json.Unmarshal(body.Text.Format.Schema, &schema)
				if schemaErr != nil {
					t.Errorf("decode output schema: %v", schemaErr)
				}
				if strings.Join(schema.Required, ",") != "appraisal,interests,memory,goals" ||
					schema.AdditionalProperties || len(schema.Properties) != 4 {
					t.Errorf("output schema does not specify the complete strict shape: %+v", schema)
				}
				var schemaTree any
				treeErr := json.Unmarshal(body.Text.Format.Schema, &schemaTree)
				if treeErr != nil || !strictSchemaObjects(schemaTree) {
					t.Errorf("output schema contains a non-strict object: %v", treeErr)
				}
				if strings.Contains(body.Input, unrelatedLocalData) || strings.Contains(body.Input, `"vector"`) ||
					strings.Contains(body.Input, `"space"`) || !strings.Contains(body.Input, "hello") {
					t.Errorf("unexpected input projection: %s", body.Input)
				}
				writer.WriteHeader(test.status)
				_, _ = writer.Write([]byte(test.response))
			}))
			defer server.Close()
			model, err := New(Config{APIKey: testKey, BaseURL: server.URL}, server.Client())
			if err != nil {
				t.Fatal(err)
			}
			experience := &mind.Experience{}
			experience.Emotion.Feelings = []emotion.Feeling{{
				Meaning: meaning.Point{Text: "calm", Space: unrelatedLocalData, Vector: []float64{1}}, Strength: 0.8,
			}}
			experience.Interests = []interest.Focus{{
				ID: "focus", Target: meaning.Point{Text: "music", Space: unrelatedLocalData, Vector: []float64{1}},
			}}
			experience.Goals.Desires = map[string]meaning.Point{
				"goal": {Text: "finish", Space: unrelatedLocalData, Vector: []float64{1}},
			}
			experience.Goals.Intentions = map[string]goal.Intention{
				testGoalID: {
					Meaning: meaning.Point{Text: "plan", Space: unrelatedLocalData, Vector: []float64{1}},
					Steps:   []meaning.Point{{Text: "step", Space: unrelatedLocalData, Vector: []float64{1}}},
				},
			}
			request := thought.Request{
				Experience: experience,
				Recalled: []memory.Fragment{{ID: testFragmentID,
					Meaning: meaning.Point{Text: "remembered", Space: unrelatedLocalData, Vector: []float64{1}}}},
				Query: memory.Query{Cue: meaning.Point{Text: "hello", Space: "test", Vector: []float64{1}}},
			}
			if test.message != "" {
				request.Query.Cue.Text = test.message
			}
			_, err = model.Interpret(context.Background(), request)
			if (err != nil) != test.wantErr {
				t.Fatalf("Interpret() error = %v, wantErr %t", err, test.wantErr)
			}
			if test.wantErr && strings.Contains(err.Error(), testKey) {
				t.Fatalf("error exposed credential: %v", err)
			}
			if test.message != "" && called {
				t.Fatal("credential-marked input was sent")
			}
		})
	}
}

func strictSchemaObjects(value any) bool {
	switch item := value.(type) {
	case map[string]any:
		if item["type"] == "object" {
			additional, ok := item["additionalProperties"].(bool)
			properties, propsOK := item["properties"].(map[string]any)
			required, requiredOK := item["required"].([]any)
			if !ok || additional || !propsOK || !requiredOK || len(properties) != len(required) {
				return false
			}
			for name := range properties {
				found := false
				for _, requiredName := range required {
					if requiredName == name {
						found = true

						break
					}
				}
				if !found {
					return false
				}
			}
		}
		for _, nested := range item {
			if !strictSchemaObjects(nested) {
				return false
			}
		}
	case []any:
		for _, nested := range item {
			if !strictSchemaObjects(nested) {
				return false
			}
		}
	}

	return true
}

func responseWithStatus(status, output string) string {
	encoded, err := json.Marshal(output)
	if err != nil {
		return ""
	}

	return `{"status":"` + status + `","output":[{"type":"message","content":[{"type":"output_text","text":` +
		string(encoded) + `}]}]}`
}

func TestInterpretRejectsInvalidInputAndCanceledRequest(t *testing.T) {
	t.Parallel()
	model, err := New(Config{APIKey: testKey, BaseURL: "http://127.0.0.1:1"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		request thought.Request
		cancel  bool
	}{
		{name: "missing experience", request: thought.Request{}},
		{name: "canceled", request: thought.Request{Experience: &mind.Experience{}}, cancel: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			if test.cancel {
				canceled, cancel := context.WithCancel(ctx)
				cancel()
				ctx = canceled
			}
			_, err := model.Interpret(ctx, test.request)
			if err == nil {
				t.Fatal("Interpret() unexpectedly succeeded")
			}
		})
	}
}

func TestDecodeResponseSkipsNonTextItems(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
	}{
		{name: "non-message", body: `{"status":"completed","output":[{"type":"reasoning","content":[]}]}`},
		{name: "non-text content", body: `{"status":"completed","output":[{"type":"message","content":[{"type":"refusal"}]}]}`}, //nolint:lll
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := decodeResponse(strings.NewReader(test.body))
			if !errors.Is(err, ErrResponse) {
				t.Fatalf("decodeResponse() error = %v, want %v", err, ErrResponse)
			}
		})
	}
}

func TestDecodeResponseConvertsStrictGoalEntries(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		goals   string
		wantErr bool
	}{
		{
			name: "entries convert to domain maps",
			goals: `{"desires":[{"id":"wish","meaning":{"text":"want"}}],` +
				`"adopt":[{"id":"plan","intention":{"meaning":{"text":"do"},` +
				`"claims":[{"target":"thing","value":"useful"}],"steps":[{"text":"step"}],` +
				`"next":0,"paused":false}}],"end":[]}`,
		},
		{
			name: "duplicate desire rejected",
			goals: `{"desires":[{"id":"wish","meaning":{"text":"one"}},` +
				`{"id":"wish","meaning":{"text":"two"}}],"adopt":[],"end":[]}`,
			wantErr: true,
		},
		{
			name: "empty desire ID rejected",
			goals: `{"desires":[{"id":"","meaning":{"text":"one"}}],` +
				`"adopt":[],"end":[]}`,
			wantErr: true,
		},
		{
			name: "empty adopted ID rejected",
			goals: `{"desires":[],"adopt":[{"id":"","intention":{"meaning":{"text":"do"},` +
				`"claims":[],"steps":[],"next":0,"paused":false}}],"end":[]}`,
			wantErr: true,
		},
		{
			name: "duplicate adopted ID rejected",
			goals: `{"desires":[],"adopt":[{"id":"plan","intention":{"meaning":{"text":"one"},` +
				`"claims":[],"steps":[],"next":0,"paused":false}},` +
				`{"id":"plan","intention":{"meaning":{"text":"two"},"claims":[],` +
				`"steps":[],"next":0,"paused":false}}],"end":[]}`,
			wantErr: true,
		},
		{
			name: "duplicate claim rejected",
			goals: `{"desires":[],"adopt":[{"id":"plan","intention":` +
				`{"meaning":{"text":"do"},"claims":[{"target":"thing","value":"one"},` +
				`{"target":"thing","value":"two"}],"steps":[],"next":0,"paused":false}}],"end":[]}`,
			wantErr: true,
		},
		{
			name: "empty claim target rejected",
			goals: `{"desires":[],"adopt":[{"id":"plan","intention":` +
				`{"meaning":{"text":"do"},"claims":[{"target":"","value":"useful"}],` +
				`"steps":[],"next":0,"paused":false}}],"end":[]}`,
			wantErr: true,
		},
		{
			name: "empty claim value rejected",
			goals: `{"desires":[],"adopt":[{"id":"plan","intention":` +
				`{"meaning":{"text":"do"},"claims":[{"target":"thing","value":""}],` +
				`"steps":[],"next":0,"paused":false}}],"end":[]}`,
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			output := `{"appraisal":null,"interests":[],"memory":null,"goals":` + test.goals + `}`
			result, err := decodeResponse(strings.NewReader(responseWithStatus("completed", output)))
			if (err != nil) != test.wantErr {
				t.Fatalf("decodeResponse() error = %v, wantErr %t", err, test.wantErr)
			}
			if test.wantErr {
				return
			}
			if result.Goals == nil || result.Goals.Desires["wish"].Text != "want" ||
				result.Goals.Adopt["plan"].Claims["thing"] != "useful" ||
				result.Goals.Adopt["plan"].Steps[0].Text != "step" {
				t.Fatalf("decoded goals = %+v", result.Goals)
			}
		})
	}
}

func TestInterpretRejectsUnserializableOrInvalidEndpoint(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		invalidURL   bool
		invalidState bool
	}{
		{name: "invalid semantic value", invalidState: true},
		{name: "invalid endpoint", invalidURL: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			model, err := New(Config{APIKey: testKey}, nil)
			if err != nil {
				t.Fatal(err)
			}
			if test.invalidURL {
				model.url = ":// invalid"
			}
			request := thought.Request{Experience: &mind.Experience{},
				Query: memory.Query{Cue: meaning.Point{Text: "hello", Space: "test", Vector: []float64{1}}}}
			if test.invalidState {
				request.Experience.Emotion.Mood[0] = math.NaN()
			}
			_, err = model.Interpret(context.Background(), request)
			if err == nil {
				t.Fatal("Interpret() unexpectedly succeeded")
			}
		})
	}
}
