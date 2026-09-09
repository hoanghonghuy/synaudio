package httpapi

import (
	"bytes"
	"errors"
	"io"
	"net/http"

	"github.com/synaudio/synaudio/backend/internal/platform/httpserver"
)

// DefaultMaxRequestBodyBytes is the authoritative JSON body limit applied by the
// public API router.
const DefaultMaxRequestBodyBytes = httpserver.PublicMaxRequestBodyBytes

// LimitRequestBody rejects oversized client-controlled request bodies before
// handler/service work begins. GET/HEAD/OPTIONS requests skip buffering.
func LimitRequestBody(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
				next.ServeHTTP(w, r)
				return
			}

			if r.Body == nil || r.Body == http.NoBody {
				next.ServeHTTP(w, r)
				return
			}

			if r.ContentLength > maxBytes {
				writeError(w, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "request body too large")
				return
			}

			limited := http.MaxBytesReader(w, r.Body, maxBytes)
			body, err := io.ReadAll(limited)
			if err != nil {
				var maxErr *http.MaxBytesError
				if errors.As(err, &maxErr) {
					writeError(w, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "request body too large")
					return
				}
				writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
				return
			}

			r.Body = io.NopCloser(bytes.NewReader(body))
			next.ServeHTTP(w, r)
		})
	}
}
