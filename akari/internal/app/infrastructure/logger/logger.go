package logger

import (
	"log/slog"
	"os"

	"github.com/kizuna-org/akari/pkg/config"
)

func SetupLogger(envMode config.EnvMode) {
	var handler slog.Handler

	switch envMode {
	case config.EnvModeDevelopment:
		opts := &slog.HandlerOptions{
			Level:       slog.LevelDebug,
			AddSource:   false,
			ReplaceAttr: nil,
		}
		handler = slog.NewTextHandler(os.Stdout, opts)
	case config.EnvModeProduction:
		opts := &slog.HandlerOptions{
			Level:       slog.LevelInfo,
			AddSource:   false,
			ReplaceAttr: nil,
		}
		handler = slog.NewJSONHandler(os.Stdout, opts)
	case config.EnvModeTest:
		opts := &slog.HandlerOptions{
			Level:       slog.LevelDebug,
			AddSource:   false,
			ReplaceAttr: nil,
		}
		handler = slog.NewTextHandler(os.Stdout, opts)
	default:
		opts := &slog.HandlerOptions{
			Level:       slog.LevelInfo,
			AddSource:   false,
			ReplaceAttr: nil,
		}
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	slog.SetDefault(slog.New(handler))
}
