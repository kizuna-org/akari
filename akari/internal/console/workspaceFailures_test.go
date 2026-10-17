package console

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/kizuna-org/akari/internal/docrepo"
)

const blockedTokenCase = "blocked token removal"

func TestDevelopmentFailuresDoNotStartAHost(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"bad database", "existing nonrepo documents", blockedTokenCase} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			directory := t.TempDir()

			switch name {
			case "bad database":
				require(t, os.WriteFile(filepath.Join(directory, "development.db"), []byte("corrupt"), 0o600))
			case "existing nonrepo documents":
				require(t, os.Mkdir(filepath.Join(directory, "documentation"), 0o700))
				require(t, os.WriteFile(filepath.Join(directory, "documentation", "existing.md"), nil, 0o600))
			case blockedTokenCase:
				require(t, os.Mkdir(filepath.Join(directory, "admin.token"), 0o700))
				require(t, os.WriteFile(filepath.Join(directory, "admin.token", "existing"), nil, 0o600))
			}

			server, err := Start(t.Context(), "127.0.0.1:0", directory, staticReader{frame: emptyFrame()},
				name == blockedTokenCase)
			if err == nil || server != nil {
				t.Fatal("failed development workspace was reported as running")
			}
		})
	}
}

func TestConfiguredGitRepositoryIsTheDocumentSource(t *testing.T) {
	// Environment configuration is process-wide, so this test deliberately runs serially.
	for _, name := range []string{"existing repository"} {
		t.Run(name, func(t *testing.T) {
			directory := filepath.Join(t.TempDir(), "repository")
			_, err := docrepo.Open(t.Context(), directory, true, []docrepo.Document{
				{Path: "docs/README.md", Content: "# Configured repository\n", Revision: ""},
				{Path: "design/README.md", Content: "# Local design\n", Revision: ""},
			})
			require(t, err)
			t.Setenv("AKARI_DOCS_REPO", directory)
			library, err := openDocuments(t.Context(), t.TempDir())
			require(t, err)
			document, err := library.Read(t.Context(), "docs/README.md")
			require(t, err)

			if document.Content != "# Configured repository\n" {
				t.Fatal("configured repository was replaced by the bundled seed")
			}
		})
	}
}

func TestUnavailableWorkspaceAndDisabledSession(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		method, path string
		disabled     bool
		want         int
	}{
		{method: http.MethodGet, path: taskAPI, disabled: false, want: 503},
		{method: http.MethodPost, path: "/api/session", disabled: true, want: 204},
		{method: http.MethodDelete, path: "/api/session", disabled: true, want: 204},
	} {
		t.Run(test.method+test.path, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequestWithContext(t.Context(), test.method, test.path, nil)
			request.Header.Set("Authorization", "Bearer "+testToken)

			response := httptest.NewRecorder()
			NewHandler(testToken, staticReader{frame: emptyFrame()}, test.disabled).ServeHTTP(response, request)

			if response.Code != test.want {
				t.Fatalf("got %d want %d", response.Code, test.want)
			}
		})
	}
}
