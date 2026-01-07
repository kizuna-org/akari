package infrastructure_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kizuna-org/akari/gen/kiseki"
	"github.com/kizuna-org/akari/pkg/kiseki/infrastructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ErrorTransport struct{}

func (t *ErrorTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return nil, errors.New("simulated network error")
}

func createTestServer(statusCode int, delay time.Duration, body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, r *http.Request) {
		if delay > 0 {
			time.Sleep(delay)
		}

		if statusCode == http.StatusOK && body != "" {
			writer.Header().Set("Content-Type", "application/json")
		}

		writer.WriteHeader(statusCode)

		if body != "" {
			_, _ = writer.Write([]byte(body))
		}
	}))
}

func TestHTTPClient_HealthCheck(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		statusCode     int
		serverDelay    time.Duration
		contextTimeout time.Duration
		expectError    bool
	}{
		{"success", http.StatusOK, 0, 5 * time.Second, false},
		{"server error", http.StatusInternalServerError, 0, 5 * time.Second, true},
		{"context timeout", http.StatusOK, 100 * time.Millisecond, 10 * time.Millisecond, true},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			server := createTestServer(testCase.statusCode, testCase.serverDelay, "")
			defer server.Close()
			client := infrastructure.NewHTTPClient(server.URL)

			ctx, cancel := context.WithTimeout(t.Context(), testCase.contextTimeout)
			defer cancel()

			err := client.HealthCheck(ctx)
			if testCase.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestHTTPClient_PostMemoryPolling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		statusCode     int
		serverDelay    time.Duration
		contextTimeout time.Duration
		characterID    string
		responseBody   string
		expectError    bool
	}{
		{
			name:           "success",
			statusCode:     http.StatusOK,
			contextTimeout: 5 * time.Second,
			characterID:    "550e8400-e29b-41d4-a716-446655440000",
			responseBody: `{"items":[
				{"t_type":"task-type-1","items":[{"task_id":"task-001","d_type":"type1","data":{}}]},
				{"t_type":"task-type-2","items":[{"task_id":"task-002","d_type":"type2","data":{}}]}
			]}`,
			expectError: false,
		},
		{
			name:           "invalid JSON response",
			statusCode:     http.StatusOK,
			contextTimeout: 5 * time.Second,
			characterID:    "550e8400-e29b-41d4-a716-446655440001",
			responseBody:   `{invalid json}`,
			expectError:    true,
		},
		{
			name:           "bad request status",
			statusCode:     http.StatusBadRequest,
			contextTimeout: 5 * time.Second,
			characterID:    "550e8400-e29b-41d4-a716-446655440002",
			responseBody:   "Bad Request",
			expectError:    true,
		},
		{
			name:           "context timeout",
			statusCode:     http.StatusOK,
			serverDelay:    100 * time.Millisecond,
			contextTimeout: 10 * time.Millisecond,
			characterID:    "550e8400-e29b-41d4-a716-446655440003",
			expectError:    true,
		},
		{
			name:           "invalid character ID format",
			statusCode:     http.StatusOK,
			contextTimeout: 5 * time.Second,
			characterID:    "char-001",
			responseBody:   "{}",
			expectError:    true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			server := createTestServer(testCase.statusCode, testCase.serverDelay, testCase.responseBody)
			defer server.Close()
			client := infrastructure.NewHTTPClient(server.URL)

			ctx, cancel := context.WithTimeout(t.Context(), testCase.contextTimeout)
			defer cancel()

			req := &kiseki.MemoryPollingRequest{Items: []kiseki.PollingRequestItem{}}
			resp, err := client.PostMemoryPolling(ctx, testCase.characterID, req)

			if testCase.expectError {
				require.Error(t, err)
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, resp)
			}
		})
	}
}
