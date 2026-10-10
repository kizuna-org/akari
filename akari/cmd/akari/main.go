package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/kizuna-org/akari/internal/action"
	"github.com/kizuna-org/akari/internal/host"
	"github.com/kizuna-org/akari/internal/lifecycle"
	"github.com/kizuna-org/akari/internal/persistence"
)

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

	err := host.RunWith(ctx, address, func(ctx context.Context) (host.Service, error) {
		store, err := persistence.New(directory)
		if err != nil {
			return nil, err
		}

		gateway, err := action.New(nil, 1)
		if err != nil {
			return nil, err
		}

		return lifecycle.Open(ctx, store, gateway, lifecycle.FoundationConfig())
	})
	if err != nil {
		slog.Error("foundation host stopped", "error", err)

		return 1
	}

	return 0
}
