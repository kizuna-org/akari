package console

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kizuna-org/akari/internal/development"
	"github.com/kizuna-org/akari/internal/docrepo"
)

func TestDevelopmentHTTPBoundary(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name, method, path, origin, token, match, body string
		want                                           int
	}{
		{name: "tasks require auth", method: http.MethodGet, path: taskAPI, origin: "",
			token: "", match: "",
			body: "",
			want: 401},
		{name: "list tasks", method: http.MethodGet, path: taskAPI,
			token: testToken, origin: "", match: "",
			body: "",
			want: 200},
		{name: "read task", method: http.MethodGet, path: taskAPI + "/BASE-01",
			token: testToken, origin: "", match: "",
			body: "",
			want: 200},
		{name: "missing task", method: http.MethodGet, path: taskAPI + "/absent",
			token: testToken, origin: "", match: "",
			body: "",
			want: 404},
		{name: "cross origin", method: http.MethodPost, path: taskAPI,
			token: testToken, origin: hostileOrigin, match: "1",
			body: "",
			want: 403},
		{name: "missing origin", method: http.MethodPost, path: taskAPI,
			token: testToken, match: "1", origin: "",
			body: "",
			want: 403},
		{name: "missing precondition", method: http.MethodPost, path: taskAPI,
			token: testToken, origin: testOrigin, match: "",
			body: "",
			want: 409},
		{name: "invalid JSON", method: http.MethodPost, path: taskAPI,
			token: testToken, origin: testOrigin, match: "1",
			body: "{",
			want: 400},
		{name: "invalid fields", method: http.MethodPost, path: taskAPI,
			token: testToken, origin: testOrigin, match: "1",
			body: `{}`,
			want: 400},
		{name: "MCP requires auth", method: http.MethodPost, path: "/mcp", origin: "",
			token: "", match: "",
			body: "",
			want: 401},
		{name: "MCP rejects origin", method: http.MethodPost, path: "/mcp",
			token: testToken, origin: hostileOrigin, match: "",
			body: "",
			want: 403},
		{name: "docs reject origin", method: http.MethodPut, path: "/api/docs",
			token: testToken, origin: "", match: "",
			body: "",
			want: 403},
		{name: "docs reject JSON", method: http.MethodPut, path: "/api/docs",
			token: testToken, origin: testOrigin,
			body: "{", match: "",
			want: 400},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			work := testWorkspace(t)
			handler := newHandler(testToken, staticReader{frame: emptyFrame()}, false, work)
			request := httptest.NewRequestWithContext(t.Context(), test.method, test.path, strings.NewReader(test.body))
			request.Header.Set("Authorization", "Bearer "+test.token)
			request.Header.Set("Origin", test.origin)
			request.Header.Set("If-Match", test.match)
			request.Header.Set("Content-Type", "application/json")

			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			if response.Code != test.want {
				t.Fatalf("got %d want %d: %s", response.Code, test.want, response.Body.String())
			}
		})
	}
}

func TestHTTPAndMCPShareTaskVersions(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"REST task version guards"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			work := testWorkspace(t)
			handler := newHandler(testToken, staticReader{frame: emptyFrame()}, false, work)
			board, err := work.tasks.Snapshot(t.Context())
			require(t, err)

			task := board.Tasks[0]
			task.Status = "in_progress"
			data, err := json.Marshal(task)
			require(t, err)

			for index, want := range []int{http.StatusOK, http.StatusConflict} {
				request := httptest.NewRequestWithContext(t.Context(), http.MethodPut,
					taskAPI+"/"+task.ID, strings.NewReader(string(data)))
				request.Header.Set("Authorization", "Bearer "+testToken)
				request.Header.Set("Origin", testOrigin)
				request.Header.Set("If-Match", `"1"`)
				request.Header.Set("Content-Type", "application/json")

				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)

				if response.Code != want {
					t.Fatalf("write %d: got %d, want %d", index, response.Code, want)
				}
			}
		})
	}
}

func testWorkspace(t *testing.T) workspace {
	t.Helper()

	store, err := development.Open(t.Context(), t.TempDir())
	require(t, err)
	t.Cleanup(func() { require(t, store.Close()) })

	library, err := docrepo.Open(t.Context(), filepath.Join(t.TempDir(), "repository"), true,
		[]docrepo.Document{{Path: "docs/README.md", Content: "# Docs\n", Revision: ""},
			{Path: "design/README.md", Content: "# Design\n", Revision: ""}})
	require(t, err)

	return workspace{tasks: store, documents: library}
}
