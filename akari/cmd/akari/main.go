package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/kizuna-org/akari/internal/host"
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

	err := host.Run(ctx, address)
	if err != nil {
		slog.Error("foundation host stopped", "error", err)

		return 1
	}

	return 0
}
