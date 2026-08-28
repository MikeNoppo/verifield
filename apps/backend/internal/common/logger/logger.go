// Package logger menyediakan structured logger berbasis log/slog (stdlib).
package logger

import (
	"log/slog"
	"os"
)

// New membuat logger: JSON di production, text yang enak dibaca di development.
// Logger yang dikembalikan juga dipasang sebagai slog default.
func New(env string, debug bool) *slog.Logger {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if env == "production" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	log := slog.New(handler)
	slog.SetDefault(log)
	return log
}
