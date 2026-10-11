package console

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServerRotatesCodeAndStops(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name                   string
		previous, authDisabled bool
	}{
		{name: "new code", previous: false, authDisabled: false},
		{name: "rotate code", previous: true, authDisabled: false},
		{name: "authentication disabled", previous: true, authDisabled: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			directory := t.TempDir()
			if test.previous {
				//nolint:gosec // Deliberately insecure old test file; rotation must replace it with a private inode.
				require(t, os.WriteFile(filepath.Join(directory, "admin.token"), []byte(testToken), 0o644))
			}

			server, err := Start(t.Context(), "127.0.0.1:0", directory,
				staticReader{frame: emptyFrame()}, test.authDisabled)

			require(t, err)

			defer func() { require(t, server.Close(t.Context())) }()

			var token string

			if test.authDisabled {
				_, statErr := os.Stat(filepath.Join(directory, "admin.token"))
				if !os.IsNotExist(statErr) {
					t.Fatal("disabled authentication left an active-looking management code")
				}
			} else {
				//nolint:gosec // Private test directory and constant filename; never print the access code.
				data, err := os.ReadFile(filepath.Join(directory, "admin.token"))
				require(t, err)
				info, err := os.Stat(filepath.Join(directory, "admin.token"))
				require(t, err)

				if strings.TrimSpace(string(data)) == testToken || len(data) < len(testToken) ||
					info.Mode().Perm() != 0o600 {
					t.Fatal("access code was not rotated privately")
				}

				token = strings.TrimSpace(string(data))
			}

			request, err := http.NewRequestWithContext(t.Context(), http.MethodGet,
				"http://"+server.Address()+"/api/status", nil)
			require(t, err)

			if token != "" {
				request.Header.Set("Authorization", "Bearer "+token)
			}

			client := new(http.Client)
			response, err := client.Do(request)
			require(t, err)

			defer func() { require(t, response.Body.Close()) }()

			if response.StatusCode != http.StatusOK {
				t.Fatal("management listener did not provide state")
			}

			require(t, server.Close(t.Context()))
			<-server.Done()
		})
	}
}

func TestServerRejectsFailedStartup(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name, address string
		canceled      bool
	}{
		{name: "bind", address: "invalid", canceled: false},
		{name: "missing directory", address: "127.0.0.1:0", canceled: false},
		{name: "canceled", address: "127.0.0.1:0", canceled: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			if test.canceled {
				cancel()
			}

			server, err := Start(ctx, test.address, filepath.Join(t.TempDir(), "missing"),
				staticReader{frame: emptyFrame()}, false)
			if err == nil || server != nil {
				t.Fatal("failed startup succeeded")
			}
		})
	}
}

func TestCodeRejectsNonemptyDirectory(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"admin.token"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			directory := t.TempDir()
			path := filepath.Join(directory, name)
			require(t, os.Mkdir(path, 0o700))
			require(t, os.WriteFile(filepath.Join(path, "child"), nil, 0o600))

			if writeToken(directory, testToken) == nil {
				t.Fatal("nonempty directory was removed to write a code")
			}
		})
	}
}

func require(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatal(err)
	}
}
