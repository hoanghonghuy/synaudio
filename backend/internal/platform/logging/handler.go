package logging

import (
	"context"
	"log/slog"
)

type redactingHandler struct {
	inner  slog.Handler
	groups []string
}

func newRedactingHandler(inner slog.Handler) *redactingHandler {
	return &redactingHandler{inner: inner}
}

func (h *redactingHandler) clone(inner slog.Handler, groups []string) *redactingHandler {
	return &redactingHandler{inner: inner, groups: groups}
}

// WrapHandler applies the repository-owned redaction policy to an slog handler.
func WrapHandler(inner slog.Handler) slog.Handler {
	return newRedactingHandler(inner)
}

func (h *redactingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *redactingHandler) Handle(ctx context.Context, record slog.Record) error {
	attrs := make([]slog.Attr, 0, record.NumAttrs())
	record.Attrs(func(attr slog.Attr) bool {
		attrs = append(attrs, h.redactRecordAttr(attr))
		return true
	})
	record = slog.NewRecord(record.Time, record.Level, RedactString(record.Message), record.PC)
	record.AddAttrs(attrs...)
	return h.inner.Handle(ctx, record)
}

func (h *redactingHandler) redactRecordAttr(attr slog.Attr) slog.Attr {
	for i := len(h.groups) - 1; i >= 0; i-- {
		if sensitiveAttributeKey(h.groups[i]) {
			return slog.Any(attr.Key, redacted)
		}
	}
	return redactAttr(attr)
}

func (h *redactingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	redacted := make([]slog.Attr, len(attrs))
	for i, attr := range attrs {
		// WithAttrs values are bound before Handle and will not be revisited there,
		// so ancestor WithGroup context must be applied at bind time as well.
		redacted[i] = h.redactRecordAttr(attr)
	}
	return h.clone(h.inner.WithAttrs(redacted), h.groups)
}

func (h *redactingHandler) WithGroup(name string) slog.Handler {
	groups := append(append([]string(nil), h.groups...), name)
	return h.clone(h.inner.WithGroup(name), groups)
}
