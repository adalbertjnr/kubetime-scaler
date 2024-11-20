package logger

import (
	"log/slog"
	"os"
	"sync"
)

var (
	instance *slog.Logger
	once     sync.Once
)

type Config struct {
	Level      slog.Level
	OutputJSON bool
}

func Initialize(c Config) {
	once.Do(func() {
		var handler slog.Handler
		if c.OutputJSON {
			handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: c.Level})
		} else {
			handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: c.Level})
		}

		instance = slog.New(handler).With(
			"controller", "downscaler",
			"controllerGroup", "downscaler.go",
		)
	})
}

func GetLogger() *slog.Logger {
	return instance
}
