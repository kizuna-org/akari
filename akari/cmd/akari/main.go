package main

import (
	"flag"
	"log/slog"

	"github.com/kizuna-org/akari/internal/di"
	"github.com/kizuna-org/akari/pkg/discord/adapter"
	"go.uber.org/fx"
)

const version = "0.1.0"

func main() {
	showVersion := flag.Bool("version", false, "Show akari version")
	flag.Parse()

	if *showVersion {
		slog.Info("akari version", "version", version)

		return
	}

	app := fx.New(
		di.NewModule(),
		fx.Invoke(runBot),
	)

	app.Run()
}

func runBot(lc fx.Lifecycle, botRunner *adapter.BotRunner) {
	botRunner.RegisterLifecycle(lc)
}
