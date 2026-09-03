package identity

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

const (
	RefreshCookieName            = "__Host-refresh_token"
	DevelopmentRefreshCookieName = "refresh_token"
)

type AuthHandler struct {
	svc *AuthService
}

func NewAuthHandler(svc *AuthService) http.Handler {
	h := &AuthHandler{svc: svc}

	r := chi.NewRouter()
	r.Post("/register", h.register)
	r.Post("/login", h.login)
	r.Get("/me", h.me)
	r.Post("/logout", h.logout)
	r.Post("/logout-all", h.logoutAll)
	r.Post("/refresh", h.refresh)
	r.Get("/sessions", h.listSessions)
	r.Delete("/sessions/{sessionID}", h.revokeSession)
	r.Post("/email/verify", h.emailVerify)
	r.Post("/email/resend", h.emailResend)
	r.Post("/password/forgot", h.passwordForgot)
	r.Post("/password/reset", h.passwordReset)
	r.Post("/mfa/totp/setup", h.mfaSetup)
	r.Post("/mfa/totp/confirm", h.mfaConfirm)
	r.Post("/mfa/totp/disable", h.mfaDisable)
	r.Post("/account/deletion/request", h.requestDeletion)
	r.Post("/account/deletion/cancel", h.cancelDeletion)
	return r
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userResponse struct {
	ID     string `json:"id"`
	Email  string `json:"email"`
	Status string `json:"status"`
}

type tokenResponse struct {
	Status      string `json:"status"`
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

func (h *AuthHandler) register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	u, err := h.svc.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, ErrEmailTaken):
			writeError(w, http.StatusConflict, "EMAIL_TAKEN", "email already registered")
		case errors.Is(err, ErrEmptyEmail), errors.Is(err, ErrInvalidEmail):
			writeError(w, http.StatusBadRequest, "INVALID_EMAIL", "invalid email")
		case errors.Is(err, ErrEmptyPassword):
			writeError(w, http.StatusBadRequest, "INVALID_PASSWORD", "invalid password")
		default:
			writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		}
		return
	}

	writeJSON(w, http.StatusCreated, userResponse{
		ID:     u.ID,
		Email:  u.Email,
		Status: u.Status,
	})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	sess, err := h.svc.LoginWithMetadata(r.Context(), req.Email, req.Password, SessionMetadata{
		UserAgentSummary: summarizeUserAgent(r.UserAgent()),
		SafeIPMetadata:   safeIPMetadata(r.RemoteAddr),
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrAccountSuspended):
			writeError(w, http.StatusForbidden, "ACCOUNT_SUSPENDED", "account suspended")
		case errors.Is(err, ErrInvalidCredentials):
			writeError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid credentials")
		default:
			writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		}
		return
	}

	access, err := h.svc.IssueAccessToken(sess)
	if err != nil {
		_ = h.svc.store.RevokeSession(r.Context(), sess.RefreshTokenHash)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	setRefreshCookie(w, r, sess)
	writeTokenResponse(w, http.StatusOK, "ok", access)
}

func (h *AuthHandler) me(w http.ResponseWriter, r *http.Request) {
	principal, user, err := h.authenticatedUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
		return
	}
	roles, _ := h.svc.store.GetUserRoles(r.Context(), principal.UserID)
	mfaEnabled := false
	if method, err := h.svc.store.GetMFAMethod(r.Context(), principal.UserID); err == nil {
		mfaEnabled = method.Confirmed && !method.Disabled
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id": user.ID, "email": user.Email, "status": user.Status,
		"email_verified": user.EmailVerifiedAt != "", "roles": roles,
		"mfa_enabled": mfaEnabled,
	})
}

func (h *AuthHandler) logout(w http.ResponseWriter, r *http.Request) {
	if principal, _, err := h.svc.AuthenticateRequest(r.Context(), r); err == nil {
		_ = h.svc.RevokeSessionByID(r.Context(), principal, principal.SessionID)
	} else if token, cookieErr := refreshTokenFromRequest(r); cookieErr == nil {
		// Logout remains safe when the short-lived access token has already expired:
		// the refresh credential can revoke itself but is never accepted as normal
		// application authorization.
		_ = h.svc.store.RevokeSession(r.Context(), HashToken(token))
	}
	clearRefreshCookies(w)
	writeJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}

func (h *AuthHandler) logoutAll(w http.ResponseWriter, r *http.Request) {
	principal, _, err := h.authenticatedUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
		return
	}
	if err := h.svc.RevokeAllSessions(r.Context(), principal.UserID); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	clearRefreshCookies(w)
	writeJSON(w, http.StatusOK, map[string]string{"status": "logged_out_all"})
}

func (h *AuthHandler) refresh(w http.ResponseWriter, r *http.Request) {
	token, err := refreshTokenFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
		return
	}

	sess, err := h.svc.RefreshSession(r.Context(), token)
	if err != nil {
		clearRefreshCookies(w)
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
		return
	}
	access, err := h.svc.IssueAccessToken(sess)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	setRefreshCookie(w, r, sess)
	writeTokenResponse(w, http.StatusOK, "refreshed", access)
}

type sessionResponse struct {
	ID               string `json:"id"`
	Current          bool   `json:"current"`
	CreatedAt        string `json:"created_at"`
	LastUsedAt       string `json:"last_used_at"`
	ExpiresAt        string `json:"expires_at"`
	UserAgentSummary string `json:"user_agent_summary,omitempty"`
	SafeIPMetadata   string `json:"safe_ip_metadata,omitempty"`
}

func (h *AuthHandler) listSessions(w http.ResponseWriter, r *http.Request) {
	principal, _, err := h.authenticatedUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
		return
	}
	sessions, err := h.svc.ListSessions(r.Context(), principal)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	items := make([]sessionResponse, 0, len(sessions))
	for _, sess := range sessions {
		items = append(items, sessionResponse{
			ID:               sess.ID,
			Current:          sess.ID == principal.SessionID,
			CreatedAt:        formatSessionTime(sess.CreatedAt),
			LastUsedAt:       formatSessionTime(sess.LastUsedAt),
			ExpiresAt:        formatSessionTime(sess.ExpiresAt),
			UserAgentSummary: sess.UserAgentSummary,
			SafeIPMetadata:   sess.SafeIPMetadata,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *AuthHandler) revokeSession(w http.ResponseWriter, r *http.Request) {
	principal, _, err := h.authenticatedUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
		return
	}
	sessionID := chi.URLParam(r, "sessionID")
	if err := h.svc.RevokeSessionByID(r.Context(), principal, sessionID); err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			writeError(w, http.StatusNotFound, "SESSION_NOT_FOUND", "session not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	if sessionID == principal.SessionID {
		clearRefreshCookies(w)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

func setRefreshCookie(w http.ResponseWriter, r *http.Request, sess Session) {
	secure := isSecureRequest(r)
	name := RefreshCookieName
	if !secure {
		name = DevelopmentRefreshCookieName
	}
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    sess.RefreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  sess.ExpiresAt,
	})
}

func clearRefreshCookies(w http.ResponseWriter) {
	clearRefreshCookie(w, RefreshCookieName)
	clearRefreshCookie(w, DevelopmentRefreshCookieName)
}

func clearRefreshCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name: name, Value: "", Path: "/", HttpOnly: true,
		Secure: name == RefreshCookieName, SameSite: http.SameSiteLaxMode, MaxAge: -1,
	})
}

func (h *AuthHandler) authenticatedUser(r *http.Request) (Principal, User, error) {
	return h.svc.AuthenticateRequest(r.Context(), r)
}

func refreshTokenFromRequest(r *http.Request) (string, error) {
	names := []string{RefreshCookieName, DevelopmentRefreshCookieName}
	if !isSecureRequest(r) {
		names = []string{DevelopmentRefreshCookieName, RefreshCookieName}
	}
	for _, name := range names {
		cookie, err := r.Cookie(name)
		if err == nil && cookie.Value != "" {
			return cookie.Value, nil
		}
	}
	return "", ErrUnauthenticated
}

func isSecureRequest(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return r.Header.Get("X-Forwarded-Proto") == "https"
}

func writeTokenResponse(w http.ResponseWriter, status int, state string, access AccessToken) {
	expiresIn := int64(time.Until(access.ExpiresAt).Seconds())
	if expiresIn < 0 {
		expiresIn = 0
	}
	writeJSON(w, status, tokenResponse{
		Status:      state,
		AccessToken: access.Token,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
	})
}

func formatSessionTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func summarizeUserAgent(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 160 {
		value = value[:160]
	}
	return value
}

func safeIPMetadata(remoteAddr string) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(remoteAddr))
	if err != nil {
		host = strings.TrimSpace(remoteAddr)
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return ""
	}
	if v4 := ip.To4(); v4 != nil {
		return net.IPv4(v4[0], v4[1], v4[2], 0).String() + "/24"
	}
	masked := ip.Mask(net.CIDRMask(48, 128))
	if masked == nil {
		return ""
	}
	return masked.String() + "/48"
}

type emailVerifyRequest struct {
	Email string `json:"email"`
	Token string `json:"token"`
}

func (h *AuthHandler) emailVerify(w http.ResponseWriter, r *http.Request) {
	var req emailVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	if err := h.svc.VerifyEmailByEmail(r.Context(), req.Email, req.Token); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_TOKEN", "invalid or expired token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type emailResendRequest struct {
	Email string `json:"email"`
}

func (h *AuthHandler) emailResend(w http.ResponseWriter, r *http.Request) {
	var req emailResendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	if _, err := h.svc.ResendEmailVerification(r.Context(), req.Email); err != nil {
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

type passwordForgotRequest struct {
	Email string `json:"email"`
}

func (h *AuthHandler) passwordForgot(w http.ResponseWriter, r *http.Request) {
	var req passwordForgotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	_, _ = h.svc.RequestPasswordResetByEmail(r.Context(), req.Email)
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

type passwordResetRequest struct {
	Email       string `json:"email"`
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

func (h *AuthHandler) passwordReset(w http.ResponseWriter, r *http.Request) {
	var req passwordResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	if err := h.svc.ResetPasswordByEmail(r.Context(), req.Email, req.Token, req.NewPassword); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_TOKEN", "invalid or expired token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *AuthHandler) mfaSetup(w http.ResponseWriter, r *http.Request) {
	principal, _, err := h.authenticatedUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
		return
	}

	secret, err := h.svc.SetupTOTP(r.Context(), principal.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_USER", "user not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"secret": secret})
}

type mfaConfirmRequest struct {
	Code string `json:"code"`
}

func (h *AuthHandler) mfaConfirm(w http.ResponseWriter, r *http.Request) {
	var req mfaConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	principal, _, err := h.authenticatedUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
		return
	}

	codes, err := h.svc.ConfirmTOTP(r.Context(), principal.UserID, req.Code, TOTPTimeStep(0))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_TOKEN", "invalid or expired code")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"recovery_codes": codes})
}

func (h *AuthHandler) mfaDisable(w http.ResponseWriter, r *http.Request) {
	principal, _, err := h.authenticatedUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
		return
	}

	if err := h.svc.DisableTOTP(r.Context(), principal.UserID); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_USER", "user not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *AuthHandler) requestDeletion(w http.ResponseWriter, r *http.Request) {
	principal, _, err := h.authenticatedUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
		return
	}

	if err := h.svc.RequestAccountDeletion(r.Context(), principal.UserID); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			writeError(w, http.StatusNotFound, "USER_NOT_FOUND", "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	clearRefreshCookies(w)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deactivated"})
}

func (h *AuthHandler) cancelDeletion(w http.ResponseWriter, r *http.Request) {
	principal, _, err := h.authenticatedUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
		return
	}

	if err := h.svc.CancelAccountDeletion(r.Context(), principal.UserID); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			writeError(w, http.StatusNotFound, "USER_NOT_FOUND", "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "active"})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
