package identity

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
)

const RefreshCookieName = "__Host-refresh_token"

type AuthHandler struct {
	svc *AuthService
}

func NewAuthHandler(svc *AuthService) http.Handler {
	h := &AuthHandler{svc: svc}

	r := chi.NewRouter()
	r.Post("/register", h.register)
	r.Post("/login", h.login)
	r.Post("/email/verify", h.emailVerify)
	r.Post("/email/resend", h.emailResend)
	r.Post("/password/forgot", h.passwordForgot)
	r.Post("/password/reset", h.passwordReset)
	r.Post("/mfa/totp/setup", h.mfaSetup)
	r.Post("/mfa/totp/confirm", h.mfaConfirm)
	r.Post("/mfa/totp/disable", h.mfaDisable)
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

	sess, err := h.svc.Login(r.Context(), req.Email, req.Password)
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

	http.SetCookie(w, &http.Cookie{
		Name:     RefreshCookieName,
		Value:    sess.RefreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Expires:  sess.ExpiresAt,
	})

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
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
		// Do not reveal whether the email exists.
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

	// Always return 202 to avoid revealing whether the email exists.
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

type mfaSetupRequest struct {
	UserID string `json:"user_id"`
}

func (h *AuthHandler) mfaSetup(w http.ResponseWriter, r *http.Request) {
	var req mfaSetupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	secret, err := h.svc.SetupTOTP(r.Context(), req.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_USER", "user not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"secret": secret})
}

type mfaConfirmRequest struct {
	UserID string `json:"user_id"`
	Code   string `json:"code"`
}

func (h *AuthHandler) mfaConfirm(w http.ResponseWriter, r *http.Request) {
	var req mfaConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	codes, err := h.svc.ConfirmTOTP(r.Context(), req.UserID, req.Code, TOTPTimeStep(0))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_TOKEN", "invalid or expired code")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"recovery_codes": codes})
}

type mfaDisableRequest struct {
	UserID string `json:"user_id"`
}

func (h *AuthHandler) mfaDisable(w http.ResponseWriter, r *http.Request) {
	var req mfaDisableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	if err := h.svc.DisableTOTP(r.Context(), req.UserID); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_USER", "user not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
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
