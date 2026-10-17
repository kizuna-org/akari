package console

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kizuna-org/akari/internal/docrepo"
)

const hostileOrigin = "https://attacker.invalid"

func TestGitDocumentHTTP(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name string
		want int
	}{
		{name: "save", want: http.StatusOK},
		{name: "stale", want: http.StatusConflict},
		{name: "invalid path", want: http.StatusBadRequest},
		{name: "canceled request", want: http.StatusServiceUnavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			work := testWorkspace(t)
			document, err := work.documents.Read(t.Context(), "docs/README.md")
			require(t, err)

			if test.name == "stale" {
				other := document
				other.Content = "# Other client\n"
				_, err = work.documents.Write(t.Context(), other)
				require(t, err)
			}

			if test.name == "invalid path" {
				document.Path = "../private.md"
			}

			document.Content = "# Updated through REST\n"
			data, err := json.Marshal(document)
			require(t, err)

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			if test.name == "canceled request" {
				cancel()
			}

			request := httptest.NewRequestWithContext(ctx, http.MethodPut, "/api/docs", bytes.NewReader(data))
			request.Header.Set("Origin", testOrigin)
			request.Header.Set("Authorization", "Bearer "+testToken)

			response := httptest.NewRecorder()
			newHandler(testToken, staticReader{frame: emptyFrame()}, false, work).ServeHTTP(response, request)

			if response.Code != test.want {
				t.Fatalf("got %d want %d", response.Code, test.want)
			}

			if test.name == "save" {
				var updated docrepo.Document

				require(t, json.Unmarshal(response.Body.Bytes(), &updated))

				if updated.Revision == document.Revision || updated.Content != document.Content {
					t.Fatal("HTTP save did not create a Git revision")
				}
			}
		})
	}
}

func TestMCPAuthenticationBoundary(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name, origin, bearer string
		cookie, disabled     bool
		want                 int
	}{
		{name: "bearer CLI", origin: "", bearer: testToken, cookie: false, disabled: false, want: 200},
		{name: "same origin cookie", origin: testOrigin, bearer: "", cookie: true, disabled: false, want: 200},
		{name: "cookie without origin", origin: "", bearer: "", cookie: true, disabled: false, want: 403},
		{name: "disabled CLI", origin: "", bearer: "", cookie: false, disabled: true, want: 200},
		{name: "disabled hostile origin", origin: hostileOrigin, bearer: "", cookie: false,
			disabled: true, want: 403},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			body := []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{
"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`)
			request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/mcp", bytes.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Accept", "application/json, text/event-stream")
			request.Header.Set("Origin", test.origin)

			if test.bearer != "" {
				request.Header.Set("Authorization", "Bearer "+test.bearer)
			}

			if test.cookie {
				request.AddCookie(sessionCookie(testToken, false, 0))
			}

			response := httptest.NewRecorder()
			handler := newHandler(testToken, staticReader{frame: emptyFrame()}, test.disabled, testWorkspace(t))
			handler.ServeHTTP(response, request)

			if response.Code != test.want {
				t.Fatalf("got %d want %d", response.Code, test.want)
			}
		})
	}
}
