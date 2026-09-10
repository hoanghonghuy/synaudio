package httpapi

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/synaudio/synaudio/backend/internal/platform/logging"
)

// responseRecorder captures the status code written by a handler.
type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// WithRequestLogger wraps a handler with structured JSON request logging and
// panic recovery. Every request, including recovered panics, emits a bounded
// structured record with validated correlation identifiers. Sensitive headers,
// bodies, panic values, and private payloads are never logged.
func WithRequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &responseRecorder{ResponseWriter: w, status: http.StatusOK}

			requestID := middleware.GetReqID(r.Context())
			correlationID := logging.ResolveCorrelationID(r.Header.Get("X-Correlation-ID"), requestID)
			ctx := logging.WithCorrelation(r.Context(), correlationID)
			r = r.WithContext(ctx)

			panicRecovered := false
			defer func() {
				route := routePattern(r)
				latency := time.Since(start).Milliseconds()
				fields := []any{
					"method", r.Method,
					"route", route,
					"status", rec.status,
					"latency_ms", latency,
					"request_id", requestID,
					"correlation_id", correlationID,
				}
				if panicRecovered {
					logger.Error("request panic recovered", fields...)
					return
				}
				logger.Info("request", fields...)
			}()

			defer func() {
				if recover() != nil {
					panicRecovered = true
					rec.status = http.StatusInternalServerError
					http.Error(rec, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(rec, r)
		})
	}
}

func routePattern(r *http.Request) string {
	if rc := chi.RouteContext(r.Context()); rc != nil {
		if pattern := strings.TrimSpace(rc.RoutePattern()); pattern != "" {
			return pattern
		}
	}
	return "unmatched"
}
