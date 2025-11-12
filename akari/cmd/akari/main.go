package main

import (
	"context"
	"flag"
	"log/slog"

	adapter "github.com/kizuna-org/akari/internal/app/adapter/discord"
	"github.com/kizuna-org/akari/internal/app/infrastructure/logger"
	"github.com/kizuna-org/akari/pkg/config"
)

const version = "0.1.0"

func main() {
	showVersion := flag.Bool("version", false, "Show akari version")
	flag.Parse()

	if *showVersion {
		slog.Info("akari version", "version", version)

		return
	}

	configRepo := config.NewConfigRepository()
	cfg := configRepo.GetConfig()

	logger.SetupLogger(cfg.EnvMode)

	ctx := context.Background()
	if err := adapter.RunDiscordBot(ctx); err != nil {
		slog.Error("discord mode failed", "error", err)
	}
}
