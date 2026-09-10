package logging

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

// SafeError returns a redacted error summary safe for production application
// logs. Raw upstream bodies, credentials, presigned URLs, and action links are
// never emitted.
func SafeError(err error) string {
	if err == nil {
		return ""
	}
	message := strings.TrimSpace(err.Error())
	if message == "" {
		return "error"
	}
	message = RedactString(message)
	if len(message) > 500 {
		message = message[:500]
	}
	return message
}

// ErrAttr returns a slog attribute with a redacted error value.
func ErrAttr(err error) slog.Attr {
	return slog.String("error", SafeError(err))
}

type classifiedError interface {
	error
	ErrorClass() string
	ErrorCode() string
}

// SafeFailureFields returns bounded error_class/error_code fields when the
// error implements a classified failure contract; otherwise it returns a single
// redacted error string attribute.
func SafeFailureFields(err error) []any {
	if err == nil {
		return nil
	}
	var classified classifiedError
	if errors.As(err, &classified) {
		class := strings.TrimSpace(classified.ErrorClass())
		code := strings.TrimSpace(classified.ErrorCode())
		if class != "" && code != "" {
			return []any{"error_class", class, "error_code", code}
		}
	}
	return []any{"error", SafeError(err)}
}

// ProviderHTTPError formats a provider failure without upstream response text.
func ProviderHTTPError(provider, operation string, status int) error {
	provider = strings.TrimSpace(provider)
	operation = strings.TrimSpace(operation)
	if provider == "" {
		provider = "provider"
	}
	if operation == "" {
		operation = "request"
	}
	return fmt.Errorf("%s %s failed with HTTP %d", provider, operation, status)
}
