package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/kizuna-org/akari/internal/action"
	"github.com/kizuna-org/akari/internal/console"
	"github.com/kizuna-org/akari/internal/host"
	"github.com/kizuna-org/akari/internal/lifecycle"
	"github.com/kizuna-org/akari/internal/persistence"
)

type managed struct {
	session *lifecycle.Session
	console *console.Server
}

func main() {
	os.Exit(run())
}

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	address := os.Getenv("AKARI_ADDR")
	if address == "" {
		address = "127.0.0.1:8080"
	}

	directory := os.Getenv("AKARI_DATA_DIR")
	if directory == "" {
		directory = "data"
	}

	adminAddress := os.Getenv("AKARI_ADMIN_ADDR")
	if adminAddress == "" {
		adminAddress = "127.0.0.1:8082"
	}

	authDisabled, err := adminAuthDisabled()
	if err != nil {
		slog.Error("invalid AKARI_ADMIN_AUTH_DISABLED value", "error", err)

		return 1
	}

	err = host.RunWith(ctx, address, func(ctx context.Context) (host.Service, error) {
		return startManaged(ctx, adminAddress, directory, authDisabled, stop)
	})
	if err != nil {
		slog.Error("foundation host stopped", "error", err)

		return 1
	}

	return 0
}

func adminAuthDisabled() (bool, error) {
	value := os.Getenv("AKARI_ADMIN_AUTH_DISABLED")
	if value == "" {
		return false, nil
	}

	return strconv.ParseBool(value)
}

func (service managed) Close(ctx context.Context) error {
	// Stop development writes before the session releases this installation's lease.
	consoleErr := service.console.Close(ctx)
	stateErr := service.session.Close(ctx)

	return errors.Join(stateErr, consoleErr)
}

func startManaged(ctx context.Context, address, directory string, authDisabled bool,
	stop context.CancelFunc) (host.Service, error) {
	store, err := persistence.New(directory)
	if err != nil {
		return nil, err
	}

	gateway, err := action.New(nil, 1)
	if err != nil {
		return nil, err
	}

	session, err := lifecycle.Open(ctx, store, gateway, lifecycle.FoundationConfig())
	if err != nil {
		return nil, err
	}

	admin, err := console.Start(ctx, address, directory, session, authDisabled)
	if err != nil {
		const cleanupTimeout = 5 * time.Second

		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), cleanupTimeout)
		defer cancel()

		return nil, errors.Join(err, session.Close(cleanup))
	}

	go func() {
		select {
		case <-admin.Done():
			stop()
		case <-ctx.Done():
		}
	}()

	slog.Info("management console listening", "address", admin.Address(), "permission", "observer")

	return managed{session: session, console: admin}, nil
}
