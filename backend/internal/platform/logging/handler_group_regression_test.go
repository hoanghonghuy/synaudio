package logging_test

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/platform/logging"
)

func TestLoggerRedactsPreBoundAttrsUnderSensitiveWithGroup(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(logging.WrapHandler(slog.NewJSONHandler(&buf, nil)))

	logger.
		WithGroup("password").
		With("value", "prebound-group-secret").
		Info("event")

	raw := buf.String()
	if strings.Contains(raw, "prebound-group-secret") {
		t.Fatalf("emitted record leaked pre-bound attr under sensitive group: %s", raw)
	}
	if !strings.Contains(raw, "[REDACTED]") {
		t.Fatalf("expected redacted output, got %s", raw)
	}
}
