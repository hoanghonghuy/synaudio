package identity

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// NewAdminSecurityHandler exposes the narrow privileged identity mutations
// implemented by AuthService. The platform router remains authoritative for
// session assurance, operation-specific permission and recent-auth gates; the
// service repeats the permission check as defense in depth.
func NewAdminSecurityHandler(svc *AuthService) http.Handler {
	h := &adminSecurityHandler{svc: svc}
	r := chi.NewRouter()
	r.Post("/admin/users/{userID}/roles/admin", h.grantAdmin)
	r.Delete("/admin/users/{userID}/roles/admin", h.revokeAdmin)
	r.Patch("/admin/users/{userID}/status", h.setStatus)
	return r
}

type adminSecurityHandler struct {
	svc *AuthService
}

type adminStatusRequest struct {
	Status string `json:"status"`
}

func (h *adminSecurityHandler) grantAdmin(w http.ResponseWriter, r *http.Request) {
	actorID, err := h.svc.ResolveUserID(r.Context(), r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
		return
	}
	if err := h.svc.GrantAdmin(r.Context(), actorID, chi.URLParam(r, "userID")); err != nil {
		writeAdminSecurityError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "granted"})
}

func (h *adminSecurityHandler) revokeAdmin(w http.ResponseWriter, r *http.Request) {
	actorID, err := h.svc.ResolveUserID(r.Context(), r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
		return
	}
	if err := h.svc.RevokeAdmin(r.Context(), actorID, chi.URLParam(r, "userID")); err != nil {
		writeAdminSecurityError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

func (h *adminSecurityHandler) setStatus(w http.ResponseWriter, r *http.Request) {
	actorID, err := h.svc.ResolveUserID(r.Context(), r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
		return
	}
	var req adminStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if err := h.svc.SetUserStatusAsAdmin(r.Context(), actorID, chi.URLParam(r, "userID"), req.Status); err != nil {
		writeAdminSecurityError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": req.Status})
}

func writeAdminSecurityError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrForbidden):
		writeError(w, http.StatusForbidden, "FORBIDDEN", "permission required")
	case errors.Is(err, ErrLastAdmin):
		writeError(w, http.StatusConflict, "LAST_ACTIVE_ADMIN", "cannot remove last active admin")
	case err != nil && err.Error() == "invalid account status":
		writeError(w, http.StatusBadRequest, "INVALID_STATUS", "invalid account status")
	default:
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
	}
}
