package docrepo

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestGitDocumentLifecycle(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name string
		seed []Document
	}{
		{name: "create edit commit reopen", seed: testDocuments()},
		{name: "docs-only repository", seed: testDocuments()[:1]},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			directory := filepath.Join(t.TempDir(), "repository")
			seed := test.seed

			repository, err := Open(t.Context(), directory, true, seed)
			must(t, err)
			library, err := repository.List(t.Context())
			must(t, err)

			if len(library) != len(seed) || library[0].Revision == "" {
				t.Fatal("Git seed documents missing")
			}

			document, err := repository.Read(t.Context(), "docs/README.md")
			must(t, err)

			initial := document.Revision
			document.Content = "# Updated\n\nAPI edit.\n"
			changed, err := repository.Write(t.Context(), document)
			must(t, err)

			if changed.Revision == initial {
				t.Fatal("document update did not make a Git commit")
			}

			same, err := repository.Write(t.Context(), changed)
			must(t, err)

			if same.Revision != changed.Revision {
				t.Fatal("no-op edit made another commit")
			}

			reopened, err := Open(t.Context(), directory, false, nil)
			must(t, err)
			restored, err := reopened.Read(t.Context(), document.Path)
			must(t, err)

			if restored.Content != changed.Content || restored.Revision != changed.Revision {
				t.Fatal("committed edit was not restored")
			}

			previous, err := repository.git(t.Context(), "show", initial+":"+document.Path)
			must(t, err)

			if !strings.Contains(previous, "Original") {
				t.Fatal("previous Markdown disappeared from Git history")
			}

			created, err := repository.Write(t.Context(), Document{Path: "design/new.md", Content: "# New\n",
				Revision: changed.Revision})
			must(t, err)

			if created.Revision == changed.Revision {
				t.Fatal("new document was not committed")
			}
		})
	}
}

func TestGitDocumentRejectsConflictsAndPaths(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name string
		path string
		prep func(*testing.T, *Repository)
		want error
	}{
		{name: "traversal", path: "docs/../../private.md", prep: nil, want: ErrInvalid},
		{name: "Git metadata", path: ".git/config", prep: nil, want: ErrInvalid},
		{name: "absolute", path: "/tmp/file.md", prep: nil, want: ErrInvalid},
		{name: "non Markdown", path: "docs/script.sh", prep: nil, want: ErrInvalid},
		{name: "dirty worktree", path: "docs/README.md", want: ErrConflict,
			prep: func(t *testing.T, r *Repository) {
				t.Helper()
				must(t, os.WriteFile(filepath.Join(r.directory, "unrelated"), []byte("local work"), privateFile))
			}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			repository, err := Open(t.Context(), t.TempDir(), true, testDocuments())
			must(t, err)
			head, err := repository.head(t.Context())
			must(t, err)

			if test.prep != nil {
				test.prep(t, repository)
			}

			_, err = repository.Write(t.Context(), Document{Path: test.path, Content: "# Edited\n", Revision: head})
			if !errors.Is(err, test.want) {
				t.Fatalf("got %v, want %v", err, test.want)
			}

			after, err := repository.head(t.Context())
			must(t, err)

			if head != after {
				t.Fatal("rejected write made a Git commit")
			}
		})
	}
}

func TestConcurrentGitEdits(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"one revision accepts one writer"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			repository, err := Open(t.Context(), t.TempDir(), true, testDocuments())
			must(t, err)
			document, err := repository.Read(t.Context(), "docs/README.md")
			must(t, err)

			start := make(chan struct{})
			results := make(chan error, 2)
			group := new(sync.WaitGroup)

			for _, content := range []string{"# First\n", "# Second\n"} {
				group.Go(func() {
					<-start

					_, writeErr := repository.Write(t.Context(), Document{Path: document.Path,
						Content: content, Revision: document.Revision})
					results <- writeErr
				})
			}

			close(start)
			group.Wait()

			first, second := <-results, <-results

			if (first != nil || !errors.Is(second, ErrConflict)) && (second != nil || !errors.Is(first, ErrConflict)) {
				t.Fatal("concurrent edit was lost or both writers failed")
			}
		})
	}
}

func TestGitRejectsInvalidRepository(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"not a repo", "nonempty seed directory", "invalid seed", "missing document"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			directory := t.TempDir()
			seed := testDocuments()

			if name == "not a repo" {
				_, err := Open(t.Context(), directory, false, nil)
				if err == nil {
					t.Fatal("non-repository was accepted")
				}

				return
			}

			if name == "nonempty seed directory" {
				must(t, os.WriteFile(filepath.Join(directory, "existing"), nil, privateFile))
			}

			if name == "invalid seed" {
				seed[0].Path = "../outside.md"
			}

			repository, err := Open(t.Context(), directory, true, seed)
			if name != "missing document" {
				if err == nil {
					t.Fatal("unsafe initialization succeeded")
				}

				return
			}

			must(t, err)
			_, err = repository.Read(t.Context(), "docs/missing.md")

			if !errors.Is(err, ErrAbsent) {
				t.Fatal("missing document was accepted")
			}
		})
	}
}

func testDocuments() []Document {
	return []Document{{Path: "docs/README.md", Content: "# Original\n", Revision: ""},
		{Path: "design/README.md", Content: "# Design\n", Revision: ""}}
}

func must(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatal(err)
	}
}
