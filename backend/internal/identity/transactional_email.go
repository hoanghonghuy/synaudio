package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

type TransactionalEmail interface {
	QueueVerification(ctx context.Context, email, token string) error
	QueuePasswordReset(ctx context.Context, email, token string) error
}

type accountDeletionRecoveryEmail interface {
	QueueAccountDeletionRecovery(ctx context.Context, email, token string) error
}

type TransactionBoundary func(context.Context, func(context.Context) error) error

// WrapTransactionalEmail makes token creation and durable notification intent
// persistence part of one identity transaction. Raw one-time credentials only
// exist in memory long enough to encrypt the outbox payload.
func WrapTransactionalEmail(next http.Handler, svc *AuthService, emails TransactionalEmail, boundary TransactionBoundary) http.Handler {
	if next == nil || svc == nil || emails == nil || boundary == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			next.ServeHTTP(w, r)
			return
		}
		switch {
		case strings.HasSuffix(r.URL.Path, "/register"):
			handleRegistrationEmail(w, r, next, svc, emails, boundary)
		case strings.HasSuffix(r.URL.Path, "/email/resend"):
			handleResendEmail(w, r, svc, emails, boundary)
		case strings.HasSuffix(r.URL.Path, "/password/forgot"):
			handleForgotEmail(w, r, svc, emails, boundary)
		case strings.HasSuffix(r.URL.Path, "/account/deletion/recovery/request"):
			handleDeletionRecoveryRequest(w, r, svc, emails, boundary)
		case strings.HasSuffix(r.URL.Path, "/account/deletion/recovery/confirm"):
			handleDeletionRecoveryConfirm(w, r, svc)
		default:
			next.ServeHTTP(w, r)
		}
	})
}

func handleRegistrationEmail(w http.ResponseWriter, r *http.Request, next http.Handler, svc *AuthService, emails TransactionalEmail, boundary TransactionBoundary) {
	body, err := readAuthBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	var req registerRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	response := newEmailBufferedResponse()
	err = boundary(r.Context(), func(txCtx context.Context) error {
		txReq := r.Clone(txCtx)
		txReq.Body = io.NopCloser(bytes.NewReader(body))
		next.ServeHTTP(response, txReq)
		if response.statusCode() != http.StatusCreated {
			return nil
		}

		u, err := svc.store.GetUserByEmail(txCtx, NormalizeEmail(req.Email))
		if err != nil {
			return err
		}
		token, err := svc.RequestEmailVerification(txCtx, u.ID)
		if err != nil {
			return err
		}
		return emails.QueueVerification(txCtx, u.Email, token)
	})
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "EMAIL_DELIVERY_UNAVAILABLE", "registration could not be completed")
		return
	}
	response.flush(w)
}

func handleResendEmail(w http.ResponseWriter, r *http.Request, svc *AuthService, emails TransactionalEmail, boundary TransactionBoundary) {
	var req emailResendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	_ = boundary(r.Context(), func(txCtx context.Context) error {
		token, err := svc.ResendEmailVerification(txCtx, req.Email)
		if err != nil {
			if errors.Is(err, ErrUserNotFound) {
				return nil
			}
			return err
		}
		u, err := svc.store.GetUserByEmail(txCtx, NormalizeEmail(req.Email))
		if err != nil {
			return err
		}
		return emails.QueueVerification(txCtx, u.Email, token)
	})
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

func handleForgotEmail(w http.ResponseWriter, r *http.Request, svc *AuthService, emails TransactionalEmail, boundary TransactionBoundary) {
	var req passwordForgotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	_ = boundary(r.Context(), func(txCtx context.Context) error {
		token, err := svc.RequestPasswordResetByEmail(txCtx, req.Email)
		if err != nil {
			if errors.Is(err, ErrUserNotFound) {
				return nil
			}
			return err
		}
		u, err := svc.store.GetUserByEmail(txCtx, NormalizeEmail(req.Email))
		if err != nil {
			return err
		}
		return emails.QueuePasswordReset(txCtx, u.Email, token)
	})
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

type deletionRecoveryRequest struct {
	Email string `json:"email"`
}

type deletionRecoveryConfirmRequest struct {
	Email string `json:"email"`
	Token string `json:"token"`
}

func handleDeletionRecoveryRequest(w http.ResponseWriter, r *http.Request, svc *AuthService, emails TransactionalEmail, boundary TransactionBoundary) {
	var req deletionRecoveryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	recoveryEmail, ok := emails.(accountDeletionRecoveryEmail)
	if !ok {
		writeError(w, http.StatusServiceUnavailable, "EMAIL_DELIVERY_UNAVAILABLE", "account recovery is unavailable")
		return
	}

	// Known eligible and unknown/ineligible accounts return the same Accepted
	// response. For a real pending deletion, token persistence and durable email
	// intent are committed in one transaction.
	_ = boundary(r.Context(), func(txCtx context.Context) error {
		token, err := svc.RequestAccountDeletionRecovery(txCtx, req.Email)
		if err != nil {
			if errors.Is(err, ErrUserNotFound) || errors.Is(err, ErrInvalidToken) {
				return nil
			}
			return err
		}
		u, err := svc.store.GetUserByEmail(txCtx, NormalizeEmail(req.Email))
		if err != nil {
			return err
		}
		return recoveryEmail.QueueAccountDeletionRecovery(txCtx, u.Email, token)
	})
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

func handleDeletionRecoveryConfirm(w http.ResponseWriter, r *http.Request, svc *AuthService) {
	var req deletionRecoveryConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if err := svc.CancelAccountDeletionByEmail(r.Context(), req.Email, req.Token); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_TOKEN", "invalid or expired recovery token")
		return
	}
	clearRefreshCookies(w)
	writeJSON(w, http.StatusOK, map[string]string{"status": "active"})
}

func readAuthBody(r *http.Request) ([]byte, error) {
	defer r.Body.Close()
	return io.ReadAll(io.LimitReader(r.Body, 1<<20))
}

type emailBufferedResponse struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func newEmailBufferedResponse() *emailBufferedResponse {
	return &emailBufferedResponse{header: make(http.Header)}
}

func (b *emailBufferedResponse) Header() http.Header { return b.header }
func (b *emailBufferedResponse) WriteHeader(status int) {
	if b.status == 0 {
		b.status = status
	}
}
func (b *emailBufferedResponse) Write(p []byte) (int, error) {
	if b.status == 0 {
		b.status = http.StatusOK
	}
	return b.body.Write(p)
}
func (b *emailBufferedResponse) statusCode() int {
	if b.status == 0 {
		return http.StatusOK
	}
	return b.status
}
func (b *emailBufferedResponse) flush(w http.ResponseWriter) {
	for key, values := range b.header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(b.statusCode())
	_, _ = w.Write(b.body.Bytes())
}
