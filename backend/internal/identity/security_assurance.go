package identity

import (
	"encoding/json"
	"net/http"
	"strings"
)

// WrapSecurityAssurance owns the TOTP-confirm endpoint so confirmation,
// recovery-code rotation, and exact-session assurance are committed atomically
// before any plaintext recovery code is exposed. Other auth routes pass through.
func WrapSecurityAssurance(next http.Handler, svc *AuthService) http.Handler {
	if next == nil || svc == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/mfa/totp/confirm") {
			next.ServeHTTP(w, r)
			return
		}

		principal, _, err := svc.AuthenticateRequest(r.Context(), r)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
			return
		}
		var req mfaConfirmRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
			return
		}
		codes, err := svc.ConfirmTOTPForSession(r.Context(), principal, req.Code, TOTPTimeStep(0))
		if err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_TOKEN", "invalid or expired code")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"recovery_codes": codes})
	})
}
