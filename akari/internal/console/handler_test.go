package console

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kizuna-org/akari/internal/action"
	"github.com/kizuna-org/akari/internal/continuity"
	"github.com/kizuna-org/akari/internal/mind"
)

const (
	testToken  = "test-only-management-code"
	statusPath = "/api/status"
	docsPath   = "/api/docs"
	testOrigin = "http://example.com"
)

type staticReader struct{ frame continuity.Frame }

func (reader staticReader) Export() continuity.Frame { return reader.frame }

func TestConsoleRoutesAndRedaction(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		path string
		auth string
		want int
	}{
		{path: "/", auth: "", want: http.StatusOK},
		{path: "/assets/app.js", auth: "", want: http.StatusOK},
		{path: "/assets/style.css", auth: "", want: http.StatusOK},
		{path: "/assets/mermaid-11.12.2.license", auth: "", want: http.StatusOK},
		{path: "/assets/missing", auth: "", want: http.StatusNotFound},
		{path: "/assets/.hidden", auth: "", want: http.StatusNotFound},
		{path: docsPath, auth: "", want: http.StatusUnauthorized},
		{path: statusPath, auth: "wrong", want: http.StatusUnauthorized},
		{path: docsPath, auth: testToken, want: http.StatusOK},
		{path: statusPath, auth: testToken, want: http.StatusOK},
	} {
		t.Run(test.path+test.auth, func(t *testing.T) {
			t.Parallel()

			var checkpoint mind.Checkpoint

			checkpoint.Snapshot.Experience = new(mind.Experience)
			checkpoint.Snapshot.Items = map[string]mind.Item{
				"private-context": {Content: "private-content", Version: 1},
			}

			frame := continuity.Frame{Schema: 1, Identity: "private-persona", Sequence: 7, Checkpoint: checkpoint,
				Outbox: map[string]continuity.Record{
					"private-task": {ID: "private-task", Revision: 1, Stage: continuity.Queued, Approved: false,
						Command: action.Command{Tool: "private-tool", Arguments: []byte("private-arguments"),
							Impact: action.Irreversible, Destination: "private-recipient", ConflictKey: "private-key",
							Sources: []string{"private-source"}, NeedsApproval: true}, Data: []byte("private-result")},
				}}
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, test.path, nil)
			request.Header.Set("Authorization", "Bearer "+test.auth)

			response := httptest.NewRecorder()
			NewHandler(testToken, staticReader{frame: frame}).ServeHTTP(response, request)

			if response.Code != test.want || response.Header().Get("Cache-Control") != "no-store" ||
				response.Header().Get("Content-Security-Policy") == "" {
				t.Fatalf("unexpected route response: %d", response.Code)
			}

			checkResponse(t, response, test.path, test.want)
		})
	}
}

func TestAuthenticationAndOrigin(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name   string
		method string
		body   string
		origin string
		tls    bool
		want   int
	}{
		{name: "login", method: http.MethodPost, body: `{"token":"` + testToken + `"}`,
			origin: testOrigin, tls: false, want: http.StatusNoContent},
		{name: "TLS", method: http.MethodPost, body: `{"token":"` + testToken + `"}`,
			origin: "https://example.com", tls: true, want: http.StatusNoContent},
		{name: "wrong token", method: http.MethodPost, body: `{"token":"wrong"}`,
			origin: testOrigin, tls: false, want: http.StatusUnauthorized},
		{name: "invalid JSON", method: http.MethodPost, body: `{`,
			origin: testOrigin, tls: false, want: http.StatusUnauthorized},
		{name: "oversize", method: http.MethodPost, body: strings.Repeat("x", 2048),
			origin: testOrigin, tls: false, want: http.StatusUnauthorized},
		{name: "foreign login", method: http.MethodPost, body: "", origin: "http://foreign.example",
			tls: false, want: http.StatusForbidden},
		{name: "logout", method: http.MethodDelete, body: "", origin: testOrigin,
			tls: false, want: http.StatusNoContent},
		{name: "foreign logout", method: http.MethodDelete, body: "", origin: "",
			tls: false, want: http.StatusForbidden},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequestWithContext(t.Context(), test.method, "http://example.com/api/session",
				strings.NewReader(test.body))
			request.Header.Set("Origin", test.origin)

			if test.tls {
				request.TLS = new(tls.ConnectionState)
			}

			response := httptest.NewRecorder()
			NewHandler(testToken, staticReader{frame: emptyFrame()}).ServeHTTP(response, request)

			if response.Code != test.want {
				t.Fatalf("code = %d, want %d", response.Code, test.want)
			}

			if test.want == http.StatusNoContent {
				result := response.Result()
				defer func() { require(t, result.Body.Close()) }()

				checkCookie(t, result.Cookies()[0], test.method, test.tls)
			}
		})
	}
}

func TestCookieAndEmptyCode(t *testing.T) {
	t.Parallel()

	for _, token := range []string{"", testToken} {
		t.Run("code="+token, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, statusPath, nil)
			request.AddCookie(sessionCookie(testToken, false, 0))

			response := httptest.NewRecorder()
			NewHandler(token, staticReader{frame: emptyFrame()}).ServeHTTP(response, request)

			if (response.Code == http.StatusOK) != (token != "") {
				t.Fatal("empty code allowed administrative access or valid cookie was rejected")
			}
		})
	}
}

func checkCookie(t *testing.T, cookie *http.Cookie, method string, secure bool) {
	t.Helper()

	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode || cookie.Secure != secure ||
		(method == http.MethodDelete && cookie.MaxAge != -1) {
		t.Fatal("cookie permissions or logout are incorrect")
	}
}

func checkResponse(t *testing.T, response *httptest.ResponseRecorder, path string, want int) {
	t.Helper()

	if path == statusPath && want == http.StatusOK {
		var data status

		err := json.Unmarshal(response.Body.Bytes(), &data)
		require(t, err)

		if data.Sequence != uint64(7) || data.AwaitingApproval != 1 || data.SharedItems != 1 ||
			strings.Contains(response.Body.String(), "private-") {
			t.Fatal("status projection leaked private content or lost counters")
		}
	}

	if path == docsPath && want == http.StatusOK &&
		!strings.Contains(response.Body.String(), "docs/06-architecture.md") {
		t.Fatal("embedded source documents are absent")
	}
}

func emptyFrame() continuity.Frame {
	var frame continuity.Frame

	return frame
}
