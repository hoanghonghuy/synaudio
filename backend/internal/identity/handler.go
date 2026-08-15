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
	r.Post("/api/v1/auth/register", h.register)
	r.Post("/api/v1/auth/login", h.login)
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
