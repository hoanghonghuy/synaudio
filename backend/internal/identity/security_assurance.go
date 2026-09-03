package identity

import (
	"bytes"
	"net/http"
	"strings"
)

// WrapSecurityAssurance binds a successful TOTP confirmation to the exact
// authenticated session before exposing success/recovery codes to the caller.
// If assurance persistence fails, the response fails closed and no privileged
// capability is granted to the password-only session.
func WrapSecurityAssurance(next http.Handler, svc *AuthService) http.Handler {
	if next == nil || svc == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/mfa/totp/confirm") {
			next.ServeHTTP(w, r)
			return
		}

		buffered := newSecurityBufferedResponse()
		next.ServeHTTP(buffered, r)
		if buffered.statusCode() != http.StatusOK {
			buffered.flush(w)
			return
		}

		principal, _, err := svc.AuthenticateRequest(r.Context(), r)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
			return
		}
		if err := svc.MarkSessionMFAAndRecentAuth(r.Context(), principal); err != nil {
			writeError(w, http.StatusServiceUnavailable, "SECURITY_ASSURANCE_UNAVAILABLE", "security assurance could not be persisted")
			return
		}
		buffered.flush(w)
	})
}

type securityBufferedResponse struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func newSecurityBufferedResponse() *securityBufferedResponse {
	return &securityBufferedResponse{header: make(http.Header)}
}

func (b *securityBufferedResponse) Header() http.Header { return b.header }
func (b *securityBufferedResponse) WriteHeader(status int) {
	if b.status == 0 {
		b.status = status
	}
}
func (b *securityBufferedResponse) Write(p []byte) (int, error) {
	if b.status == 0 {
		b.status = http.StatusOK
	}
	return b.body.Write(p)
}
func (b *securityBufferedResponse) statusCode() int {
	if b.status == 0 {
		return http.StatusOK
	}
	return b.status
}
func (b *securityBufferedResponse) flush(w http.ResponseWriter) {
	for key, values := range b.header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(b.statusCode())
	_, _ = w.Write(b.body.Bytes())
}
