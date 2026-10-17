package console

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/kizuna-org/akari/internal/development"
	"github.com/kizuna-org/akari/internal/docrepo"
)

type Server struct {
	http    *http.Server
	done    chan struct{}
	err     error
	address string
	tasks   *development.Store
}

// Start binds the separate management listener and configures its per-process access code.
func Start(ctx context.Context, address, directory string, reader Reader, authDisabled bool) (*Server, error) {
	accessVerifier, _, err := newAccessJWTVerifierFromEnvironment()
	if err != nil {
		return nil, err
	}

	listener, err := new(net.ListenConfig).Listen(ctx, "tcp", address)
	if err != nil {
		return nil, err
	}

	token := ""

	if authDisabled {
		err = removeToken(directory)
	} else {
		token = rand.Text()
		err = writeToken(directory, token)
	}

	if err != nil {
		_ = listener.Close()

		return nil, err
	}

	const timeout = 5 * time.Second

	tasks, err := development.Open(ctx, directory)
	if err != nil {
		_ = listener.Close()

		return nil, fmt.Errorf("console: open development tasks: %w", err)
	}

	library, err := openDocuments(ctx, directory)
	if err != nil {
		_ = listener.Close()
		_ = tasks.Close()

		return nil, err
	}

	transport := new(http.Server)
	transport.Handler = newHandlerWithAccess(token, reader, authDisabled,
		workspace{tasks: tasks, documents: library}, accessVerifier)
	transport.ReadHeaderTimeout = timeout
	transport.ReadTimeout = timeout
	transport.WriteTimeout = timeout
	transport.IdleTimeout = timeout

	server := &Server{http: transport, done: make(chan struct{}), err: nil,
		address: listener.Addr().String(), tasks: tasks}

	go func() {
		server.err = server.http.Serve(listener)
		close(server.done)
	}()

	return server, nil
}

func removeToken(directory string) error {
	err := os.Remove(filepath.Join(directory, "admin.token"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	return err
}

func (server *Server) Done() <-chan struct{} { return server.done }

func (server *Server) Address() string { return server.address }

func (server *Server) Close(ctx context.Context) error {
	shutdownErr := server.http.Shutdown(ctx)
	if shutdownErr != nil {
		// The process must not exit and release its lease while Git writes still run.
		_ = server.http.Shutdown(context.WithoutCancel(ctx))
	}

	closeErr := server.http.Close()
	<-server.done

	serveErr := server.err
	if errors.Is(serveErr, http.ErrServerClosed) {
		serveErr = nil
	}

	return errors.Join(shutdownErr, closeErr, serveErr, server.tasks.Close())
}

func openDocuments(ctx context.Context, directory string) (*docrepo.Repository, error) {
	var seed []docrepo.Document

	err := json.Unmarshal(documents, &seed)
	if err != nil {
		return nil, err
	}

	configured := os.Getenv("AKARI_DOCS_REPO")
	if configured != "" {
		return docrepo.Open(ctx, configured, false, nil)
	}

	return docrepo.Open(ctx, filepath.Join(directory, "documentation"), true, seed)
}

func writeToken(directory, token string) error {
	// Remove the previous inode so an existing file or symlink cannot change the new token's permissions.
	path := filepath.Join(directory, "admin.token")

	err := os.Remove(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	const permission = 0o600

	//nolint:gosec // Installation-owned directory and fixed filename; O_EXCL rejects replacement symlinks.
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, permission)
	if err != nil {
		return err
	}

	_, writeErr := fmt.Fprintln(file, token)

	return errors.Join(writeErr, file.Close())
}
