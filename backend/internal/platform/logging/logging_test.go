package logging_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/generation"
	"github.com/synaudio/synaudio/backend/internal/platform/logging"
)

func TestRedactStringRedactsSensitiveValues(t *testing.T) {
	cases := []struct {
		input    string
		mustNot  []string
		mustHave string
	}{
		{
			input:   "Bearer secret-token-123",
			mustNot: []string{"secret-token-123"},
			mustHave: "[REDACTED]",
		},
		{
			input:   "postgres://user:pass@db.example:5432/app",
			mustNot: []string{"user:pass"},
			mustHave: "[REDACTED]",
		},
		{
			input:   "https://cdn.example/object?X-Amz-Signature=abc",
			mustNot: []string{"X-Amz-Signature=abc"},
			mustHave: "[REDACTED]",
		},
		{
			input:   "https://app.example/reset?token=abc123",
			mustNot: []string{"abc123"},
			mustHave: "[REDACTED]",
		},
		{
			input:   "x-goog-api-key=super-secret",
			mustNot: []string{"super-secret"},
			mustHave: "[REDACTED]",
		},
	}
	for _, tc := range cases {
		got := logging.RedactString(tc.input)
		for _, needle := range tc.mustNot {
			if strings.Contains(got, needle) {
				t.Fatalf("input %q still contains %q: %q", tc.input, needle, got)
			}
		}
		if !strings.Contains(got, tc.mustHave) {
			t.Fatalf("input %q => %q, expected %q", tc.input, got, tc.mustHave)
		}
	}
}

func TestLoggerRedactsSensitiveMessageContent(t *testing.T) {
	cases := []struct {
		name    string
		message string
		mustNot []string
	}{
		{
			name:    "bearer token in message",
			message: "provider failed: Bearer secret-token",
			mustNot: []string{"secret-token"},
		},
		{
			name:    "database url credential in message",
			message: "database connect failed: postgres://user:secret@db.example:5432/app",
			mustNot: []string{"user:secret", "secret@db"},
		},
		{
			name:    "presigned url in message",
			message: "upload failed for https://cdn.example/object?X-Amz-Signature=abc123",
			mustNot: []string{"X-Amz-Signature=abc123", "abc123"},
		},
		{
			name:    "action link query token in message",
			message: "email dispatch failed: https://app.example/reset?token=reset-secret",
			mustNot: []string{"reset-secret"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(logging.WrapHandler(slog.NewJSONHandler(&buf, nil)))
			logger.Error(tc.message)

			entry := decodeLog(t, buf.Bytes())
			msg, ok := entry["msg"].(string)
			if !ok {
				t.Fatalf("expected string msg field, got %T (%v)", entry["msg"], entry["msg"])
			}
			for _, needle := range tc.mustNot {
				if strings.Contains(msg, needle) {
					t.Fatalf("emitted message still contains %q: %q", needle, msg)
				}
			}
			if !strings.Contains(msg, "[REDACTED]") {
				t.Fatalf("expected redacted message, got %q", msg)
			}
		})
	}
}

func TestLoggerPreservesSafeStaticMessages(t *testing.T) {
	safeMessages := []string{
		"provider request failed",
		"chapter generation started",
		"request completed",
	}
	for _, message := range safeMessages {
		t.Run(message, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(logging.WrapHandler(slog.NewJSONHandler(&buf, nil)))
			logger.Info(message)

			entry := decodeLog(t, buf.Bytes())
			if got, ok := entry["msg"].(string); !ok || got != message {
				t.Fatalf("expected safe message preserved, got %v", entry["msg"])
			}
		})
	}
}

func TestLoggerRedactsSensitiveGroupWithGenericChildKey(t *testing.T) {
	cases := []struct {
		name    string
		logFn   func(logger *slog.Logger)
		mustNot []string
	}{
		{
			name: "sensitive group with generic child key",
			logFn: func(logger *slog.Logger) {
				logger.Info("event",
					slog.Group("password",
						slog.String("value", "secret-value"),
					),
				)
			},
			mustNot: []string{"secret-value"},
		},
		{
			name: "sensitive authorization group with generic child key",
			logFn: func(logger *slog.Logger) {
				logger.Info("event",
					slog.Group("authorization",
						slog.String("value", "Bearer secret-token"),
					),
				)
			},
			mustNot: []string{"secret-token", "Bearer secret-token"},
		},
		{
			name: "nested sensitive group under safe parent",
			logFn: func(logger *slog.Logger) {
				logger.Info("event",
					slog.Group("request",
						slog.Group("access_token",
							slog.String("payload", "jwt-secret"),
						),
					),
				)
			},
			mustNot: []string{"jwt-secret"},
		},
		{
			name: "WithAttrs sensitive group",
			logFn: func(logger *slog.Logger) {
				logger.With(
					slog.Group("password",
						slog.String("value", "withattrs-secret"),
					),
				).Info("event")
			},
			mustNot: []string{"withattrs-secret"},
		},
		{
			name: "WithGroup sensitive parent",
			logFn: func(logger *slog.Logger) {
				logger.WithGroup("password").Info("event", "value", "withgroup-secret")
			},
			mustNot: []string{"withgroup-secret"},
		},
		{
			name: "nested WithGroup with sensitive inner group",
			logFn: func(logger *slog.Logger) {
				logger.WithGroup("request").WithGroup("access_token").Info("event", "value", "nested-withgroup-secret")
			},
			mustNot: []string{"nested-withgroup-secret"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(logging.WrapHandler(slog.NewJSONHandler(&buf, nil)))
			tc.logFn(logger)

			raw := buf.String()
			for _, needle := range tc.mustNot {
				if strings.Contains(raw, needle) {
					t.Fatalf("emitted record still contains %q: %s", needle, raw)
				}
			}
			if !strings.Contains(raw, "[REDACTED]") {
				t.Fatalf("expected redacted output, got %s", raw)
			}
		})
	}
}

func TestLoggerRedactsSensitiveAttributeKeys(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(logging.WrapHandler(slog.NewJSONHandler(&buf, nil)))

	logger.Info("event",
		"access_token", "jwt-value",
		"password", "plaintext",
		"request_id", "req-1",
	)

	entry := decodeLog(t, buf.Bytes())
	if entry["access_token"] != "[REDACTED]" {
		t.Fatalf("expected redacted access_token, got %v", entry["access_token"])
	}
	if entry["password"] != "[REDACTED]" {
		t.Fatalf("expected redacted password, got %v", entry["password"])
	}
	if entry["request_id"] != "req-1" {
		t.Fatalf("expected safe request_id preserved, got %v", entry["request_id"])
	}
}

func TestSafeErrorRedactsWrappedSecrets(t *testing.T) {
	err := errors.New("database connect failed: postgres://user:secret@db:5432/app")
	got := logging.SafeError(err)
	if strings.Contains(got, "secret") {
		t.Fatalf("safe error leaked secret: %q", got)
	}
}

func TestSafeFailureFieldsUsesClassifiedError(t *testing.T) {
	err := &generation.ClassifiedError{Class: "TRANSIENT", Code: "PROVIDER_TIMEOUT", Err: errors.New("upstream timeout with secret-token")}
	fields := logging.SafeFailureFields(err)
	if len(fields) != 4 || fields[0] != "error_class" || fields[2] != "error_code" {
		t.Fatalf("unexpected fields: %v", fields)
	}
	if fields[1] != "TRANSIENT" || fields[3] != "PROVIDER_TIMEOUT" {
		t.Fatalf("unexpected class/code: %v", fields)
	}
}

func TestResolveCorrelationIDRejectsUnsafeValues(t *testing.T) {
	requestID := "generated-request-id"
	if got := logging.ResolveCorrelationID("", requestID); got != requestID {
		t.Fatalf("expected fallback to request id, got %q", got)
	}
	if got := logging.ResolveCorrelationID("valid-correlation_01", requestID); got != "valid-correlation_01" {
		t.Fatalf("expected valid correlation id, got %q", got)
	}
	if got := logging.ResolveCorrelationID("bad correlation id!", requestID); got != requestID {
		t.Fatalf("expected unsafe correlation to fall back, got %q", got)
	}
}

func decodeLog(t *testing.T, raw []byte) map[string]any {
	var entry map[string]any
	if err := json.Unmarshal(raw, &entry); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, raw)
	}
	return entry
}
