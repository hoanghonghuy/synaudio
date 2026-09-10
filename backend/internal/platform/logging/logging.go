package logging

import (
	"log/slog"
	"os"
	"strings"
)

// New returns a production JSON logger with repository-owned redaction and
// configurable level semantics via LOG_LEVEL (debug, info, warn, error).
func New(service string) *slog.Logger {
	level := parseLevel(os.Getenv("LOG_LEVEL"))
	inner := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	handler := newRedactingHandler(inner)
	return slog.New(handler).With("service", service)
}

func parseLevel(raw string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
