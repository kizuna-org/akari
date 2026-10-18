//nolint:cyclop,exhaustruct,gocognit,wsl,wsl_v5 // Cases stay grouped to keep protocol assertions together.
package openai

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kizuna-org/akari/internal/meaning"
	"github.com/kizuna-org/akari/internal/memory"
	"github.com/kizuna-org/akari/internal/mind"
	"github.com/kizuna-org/akari/internal/thought"
)

const testKey = "test-secret-key"

func TestNewConfiguration(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{name: "defaults", config: Config{APIKey: testKey}},
		{name: "custom model and URL", config: Config{APIKey: testKey, Model: "model-x",
			BaseURL: "http://localhost:8080/v1/"}},
		{name: "missing key", config: Config{}, wantErr: true},
		{name: "invalid URL", config: Config{APIKey: testKey, BaseURL: "example.invalid"}, wantErr: true},
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
			if test.name == "defaults" && (model.model != defaultModel || model.url != defaultURL+"/responses") {
				t.Fatalf("defaults = model %q URL %q", model.model, model.url)
			}
		})
	}
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
			response: `{"output":[{"type":"message","content":[{"type":"output_text","text":"{}"}]}]}`},
		{name: "server error", status: http.StatusUnauthorized, response: `{"error":"not included"}`, wantErr: true},
		{name: "malformed envelope", status: http.StatusOK, response: `{`, wantErr: true},
		{name: "missing output text", status: http.StatusOK, response: `{"output":[]}`, wantErr: true},
		{name: "invalid interpretation", status: http.StatusOK,
			response: `{"output":[{"type":"message","content":[{"type":"output_text","text":"not-json"}]}]}`, wantErr: true},
		{name: "credential marker rejected", status: http.StatusOK,
			message: `{"password":"do-not-send-this"}`, wantErr: true},
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
				}
				err := json.NewDecoder(request.Body).Decode(&body)
				if err != nil {
					t.Errorf("decode body: %v", err)
				}
				if body.Model != defaultModel || body.Store == nil || *body.Store || strings.Contains(body.Input, testKey) {
					t.Errorf("unsafe request body: %+v", body)
				}
				if strings.Contains(body.Input, "unrelated-local-data") || !strings.Contains(body.Input, "hello") {
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
			request := thought.Request{
				Experience: &mind.Experience{},
				Query:      memory.Query{Cue: meaning.Point{Text: "hello", Space: "test", Vector: []float64{1}}},
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
		{name: "non-message", body: `{"output":[{"type":"reasoning","content":[]}]}`},
		{name: "non-text content", body: `{"output":[{"type":"message","content":[{"type":"refusal"}]}]}`},
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
