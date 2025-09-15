package log

import (
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
)

// ConfigureLogger sets up the default structured logger to use tint on stderr
func ConfigureInteractiveLogger() {
	w := os.Stderr

	defaultLevel := slog.LevelInfo
	if os.Getenv("DEBUG") == "1" {
		defaultLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(
		tint.NewHandler(w, &tint.Options{
			Level:      defaultLevel,
			TimeFormat: time.TimeOnly,
		}),
	))
}
