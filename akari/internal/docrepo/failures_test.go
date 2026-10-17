package docrepo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestIgnoredDraftIsNotOverwritten(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"ignored local draft"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			repository, err := Open(t.Context(), t.TempDir(), true, testDocuments())
			must(t, err)
			must(t, os.WriteFile(filepath.Join(repository.directory, ".git", "info", "exclude"),
				[]byte("docs/draft.md\n"), privateFile))
			must(t, repository.writeFile(Document{Path: "docs/draft.md", Content: "# Local work\n", Revision: ""}))
			head, err := repository.head(t.Context())
			must(t, err)
			_, err = repository.Write(t.Context(), Document{Path: "docs/draft.md", Content: "# Replacement\n", Revision: head})

			if !errors.Is(err, ErrConflict) {
				t.Fatal("existing ignored draft was treated as a new document")
			}

			root, err := os.OpenRoot(repository.directory)

			must(t, err)

			defer func() { must(t, root.Close()) }()

			content, err := root.ReadFile("docs/draft.md")
			must(t, err)

			if string(content) != "# Local work\n" {
				t.Fatal("local draft was overwritten")
			}
		})
	}
}

func TestUnsafeSymlinkCannotModifyExternalMarkdown(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"document symlink", "directory symlink"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			repository, err := Open(t.Context(), t.TempDir(), true, testDocuments())
			must(t, err)
			outside := filepath.Join(t.TempDir(), "private.md")
			must(t, os.WriteFile(outside, []byte("# Private\n"), privateFile))

			if name == "document symlink" {
				must(t, os.Symlink(outside, filepath.Join(repository.directory, "docs", "linked.md")))
			} else {
				must(t, os.RemoveAll(filepath.Join(repository.directory, "docs")))
				must(t, os.Symlink(filepath.Dir(outside), filepath.Join(repository.directory, "docs")))
			}

			_, err = repository.git(t.Context(), "add", "--", "docs")
			must(t, err)
			_, err = repository.git(t.Context(), "commit", "-m", "test linked Markdown")
			must(t, err)
			head, err := repository.head(t.Context())
			must(t, err)

			_, err = repository.Write(t.Context(), Document{Path: "docs/linked.md", Content: "# Changed\n", Revision: head})
			if err == nil {
				t.Fatal("linked Markdown was written")
			}

			//nolint:gosec // Private test fixture outside the doc repo; verifies that a symlink could not overwrite it.
			content, err := os.ReadFile(outside)
			must(t, err)

			if string(content) != "# Private\n" {
				t.Fatal("document update escaped its repository")
			}
		})
	}
}

func TestGitFailureKeepsCommittedDocument(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"index locked", "canceled read", "repository unavailable"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			directory := filepath.Join(t.TempDir(), "repository")
			repository, err := Open(t.Context(), directory, true, testDocuments())
			must(t, err)
			document, err := repository.Read(t.Context(), "docs/README.md")
			must(t, err)

			switch name {
			case "index locked":
				must(t, os.WriteFile(filepath.Join(directory, ".git", "index.lock"), nil, privateFile))

				document.Content = "# New content must not be reported as committed\n"

				_, err = repository.Write(t.Context(), document)
				if err == nil {
					t.Fatal("failed Git commit was reported as saved")
				}

				actual, readErr := repository.Read(t.Context(), document.Path)
				must(t, readErr)

				if actual.Revision != document.Revision || actual.Content == document.Content {
					t.Fatal("uncommitted Markdown was exposed as the source")
				}
			case "canceled read":
				ctx, cancel := context.WithCancel(t.Context())
				cancel()

				_, err = repository.Read(ctx, document.Path)

				if !errors.Is(err, context.Canceled) {
					t.Fatal("canceled Git read continued")
				}
			case "repository unavailable":
				must(t, os.RemoveAll(directory))

				_, err = repository.Write(t.Context(), document)
				if err == nil {
					t.Fatal("missing repository was treated as saved")
				}
			}
		})
	}
}
