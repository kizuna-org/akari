// Package docrepo manages Markdown in a local Git repository without pushing or fetching.
package docrepo

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

type repositoryError string

func (err repositoryError) Error() string { return string(err) }

const (
	ErrInvalid        repositoryError = "docrepo: invalid Markdown path or content"
	ErrConflict       repositoryError = "docrepo: repository changed or has uncommitted changes"
	ErrAbsent         repositoryError = "docrepo: document not found"
	maxContent                        = 1 << 20
	privateDirectory                  = 0o700
	privateFile                       = 0o600
	gitTimeout                        = 5 * time.Second
	gitFixedArguments                 = 10
)

type Document struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	Revision string `json:"revision"`
}

type Repository struct {
	mu        sync.Mutex
	directory string
}

// Open seeds a dedicated local repository once. An explicitly configured path
// must already be a Git worktree. No remote access or automatic push is performed.
func Open(ctx context.Context, directory string, initialize bool, seed []Document) (*Repository, error) {
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return nil, err
	}

	repository := &Repository{mu: sync.Mutex{}, directory: absolute}
	if initialize {
		err = repository.initialize(ctx, seed)
		if err != nil {
			return nil, err
		}
	}

	root, err := repository.git(ctx, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(root) != absolute {
		return nil, ErrInvalid
	}

	_, err = repository.git(ctx, "rev-parse", "HEAD")

	return repository, err
}

func (repository *Repository) List(ctx context.Context) ([]Document, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()

	head, err := repository.head(ctx)
	if err != nil {
		return nil, err
	}

	paths, err := repository.git(ctx, "ls-tree", "-r", "--name-only", head, "--", "docs", "design")
	if err != nil {
		return nil, err
	}

	documents := []Document{}

	for path := range strings.SplitSeq(strings.TrimSpace(paths), "\n") {
		if !validPath(path) {
			continue
		}

		content, readErr := repository.git(ctx, "show", head+":"+path)
		if readErr != nil {
			return nil, readErr
		}

		if len(content) > maxContent {
			return nil, ErrInvalid
		}

		documents = append(documents, Document{Path: path, Content: content, Revision: head})
	}

	return documents, nil
}

func (repository *Repository) Read(ctx context.Context, path string) (Document, error) {
	if !validPath(path) {
		return Document{}, ErrInvalid
	}

	documents, err := repository.List(ctx)
	if err != nil {
		return Document{}, err
	}

	for _, document := range documents {
		if document.Path == path {
			return document, nil
		}
	}

	return Document{}, ErrAbsent
}

func (repository *Repository) Write(ctx context.Context, document Document) (Document, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()

	if !validDocument(document) || (len(document.Revision) != 40 && len(document.Revision) != 64) {
		return Document{}, ErrInvalid
	}

	head, err := repository.head(ctx)
	if err != nil {
		return Document{}, err
	}

	status, err := repository.git(ctx, "status", "--porcelain")
	if err != nil {
		return Document{}, err
	}

	if head != document.Revision || status != "" {
		return Document{}, ErrConflict
	}

	return repository.commitDocument(ctx, document, head)
}

func (repository *Repository) initialize(ctx context.Context, seed []Document) error {
	_, err := os.Stat(filepath.Join(repository.directory, ".git"))
	if err == nil {
		return nil
	}

	if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	err = os.MkdirAll(repository.directory, privateDirectory)
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(repository.directory)
	if err != nil {
		return err
	}

	if len(entries) != 0 {
		return ErrConflict
	}

	_, err = repository.git(ctx, "init", "--initial-branch=main")
	if err != nil {
		return err
	}

	return repository.seed(ctx, seed)
}

func (repository *Repository) seed(ctx context.Context, seed []Document) error {
	for _, document := range seed {
		if !validDocument(document) {
			return ErrInvalid
		}

		writeErr := repository.writeFile(document)
		if writeErr != nil {
			return writeErr
		}
	}

	_, err := repository.git(ctx, "add", "--", "docs", "design")
	if err != nil {
		return err
	}

	_, err = repository.git(ctx, "commit", "-m", "docs: initialize Akari documentation workspace")

	return err
}

func (repository *Repository) commitDocument(ctx context.Context, document Document, head string) (Document, error) {
	previous, err := repository.git(ctx, "show", head+":"+document.Path)
	if err == nil && previous == document.Content {
		return document, nil
	}

	writeErr := repository.writeFile(document)
	if writeErr != nil {
		return Document{}, writeErr
	}

	_, err = repository.git(ctx, "add", "--", document.Path)
	if err != nil {
		return Document{}, err
	}

	_, err = repository.git(ctx, "commit", "--only", "-m", "docs: update "+document.Path, "--", document.Path)
	if err != nil {
		return Document{}, err
	}

	document.Revision, err = repository.head(ctx)

	return document, err
}

func (repository *Repository) head(ctx context.Context) (string, error) {
	head, err := repository.git(ctx, "rev-parse", "HEAD")

	return strings.TrimSpace(head), err
}

func validPath(path string) bool {
	return regexp.MustCompile(`^(docs|design)/[A-Za-z0-9_-]+\.md$`).MatchString(path)
}

func validDocument(document Document) bool {
	return validPath(document.Path) && len(document.Content) <= maxContent &&
		strings.TrimSpace(document.Content) != "" && !strings.ContainsRune(document.Content, '\x00')
}

func (repository *Repository) writeFile(document Document) error {
	root, err := os.OpenRoot(repository.directory)
	if err != nil {
		return err
	}

	defer func() { _ = root.Close() }()

	group := strings.Split(document.Path, "/")[0]

	err = root.MkdirAll(group, privateDirectory)
	if err != nil {
		return err
	}

	for _, path := range []string{group, document.Path} {
		info, statErr := root.Lstat(path)
		if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
			return statErr
		}

		if statErr == nil && info.Mode()&os.ModeSymlink != 0 {
			return ErrInvalid
		}
	}

	// Root prevents a symlink in a configured worktree from escaping the repository.
	return root.WriteFile(document.Path, []byte(document.Content), privateFile)
}

func (repository *Repository) git(ctx context.Context, arguments ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, gitTimeout)
	defer cancel()

	options := make([]string, 0, gitFixedArguments+len(arguments))
	options = append(options, "-c", "core.hooksPath=/dev/null", "-c", "commit.gpgsign=false",
		"-c", "user.name=Akari Docs", "-c", "user.email=akari@localhost", "-C", repository.directory)

	//nolint:gosec // Fixed Git binary, explicit arguments and validated Markdown paths; no shell or remote commands.
	command := exec.CommandContext(ctx, "git", append(options, arguments...)...)

	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("docrepo: git %s: %w", arguments[0], err)
	}

	return string(output), nil
}
