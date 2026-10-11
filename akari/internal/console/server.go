package console

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type Server struct {
	http    *http.Server
	done    chan struct{}
	err     error
	address string
}

// Start binds the separate management listener and rotates its private per-process access code.
func Start(ctx context.Context, address, directory string, reader Reader) (*Server, error) {
	listener, err := new(net.ListenConfig).Listen(ctx, "tcp", address)
	if err != nil {
		return nil, err
	}

	token := rand.Text()

	err = writeToken(directory, token)
	if err != nil {
		_ = listener.Close()

		return nil, err
	}

	const timeout = 5 * time.Second

	transport := new(http.Server)
	transport.Handler = NewHandler(token, reader)
	transport.ReadHeaderTimeout = timeout
	transport.ReadTimeout = timeout
	transport.WriteTimeout = timeout
	transport.IdleTimeout = timeout

	server := &Server{http: transport, done: make(chan struct{}), err: nil,
		address: listener.Addr().String()}

	go func() {
		server.err = server.http.Serve(listener)
		close(server.done)
	}()

	return server, nil
}

func (server *Server) Done() <-chan struct{} { return server.done }

func (server *Server) Address() string { return server.address }

func (server *Server) Close(ctx context.Context) error {
	shutdownErr := server.http.Shutdown(ctx)
	closeErr := server.http.Close()
	<-server.done

	serveErr := server.err
	if errors.Is(serveErr, http.ErrServerClosed) {
		serveErr = nil
	}

	return errors.Join(shutdownErr, closeErr, serveErr)
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
