package logging

import (
	"context"
	"log/slog"
	"regexp"
	"strings"
)

const maxCorrelationIDLen = 128

var correlationIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

type correlationContextKey struct{}

// WithCorrelation stores a validated correlation identifier on the context.
func WithCorrelation(ctx context.Context, correlationID string) context.Context {
	correlationID = strings.TrimSpace(correlationID)
	if correlationID == "" {
		return ctx
	}
	return context.WithValue(ctx, correlationContextKey{}, correlationID)
}

// CorrelationID returns the request correlation identifier when present.
func CorrelationID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value, _ := ctx.Value(correlationContextKey{}).(string)
	return strings.TrimSpace(value)
}

// ResolveCorrelationID validates an inbound X-Correlation-ID header and falls
// back to the generated request ID when the header is missing or unsafe.
func ResolveCorrelationID(headerValue, requestID string) string {
	headerValue = strings.TrimSpace(headerValue)
	requestID = strings.TrimSpace(requestID)
	if headerValue == "" {
		return requestID
	}
	if len(headerValue) > maxCorrelationIDLen || !correlationIDPattern.MatchString(headerValue) {
		return requestID
	}
	return headerValue
}

// FromContext returns a logger enriched with the correlation identifier when
// one is present on the context.
func FromContext(ctx context.Context, base *slog.Logger) *slog.Logger {
	if base == nil {
		return slog.Default()
	}
	if id := CorrelationID(ctx); id != "" {
		return base.With("correlation_id", id)
	}
	return base
}
