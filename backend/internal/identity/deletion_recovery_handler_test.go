package identity_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/identity"
)

type deletionRecoveryEmailFake struct {
	email string
	token string
}

func (f *deletionRecoveryEmailFake) QueueVerification(context.Context, string, string) error { return nil }
func (f *deletionRecoveryEmailFake) QueuePasswordReset(context.Context, string, string) error { return nil }
func (f *deletionRecoveryEmailFake) QueueAccountDeletionRecovery(_ context.Context, email, token string) error {
	f.email = email
	f.token = token
	return nil
}

func deletionRecoveryBoundary(ctx context.Context, run func(context.Context) error) error {
	return run(ctx)
}

func TestDeletionRecoveryRequestIsEnumerationSafeAndConfirmConsumesToken(t *testing.T) {
	store := newDeletionRecoveryFakeStore()
	svc := identity.NewAuthService(store)
	u, err := svc.Register(context.Background(), "reader@example.com", "correct password")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.DeactivateUser(context.Background(), u.ID); err != nil {
		t.Fatal(err)
	}

	emails := &deletionRecoveryEmailFake{}
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unexpected delegate", http.StatusNotFound)
	})
	h := identity.WrapTransactionalEmail(next, svc, emails, deletionRecoveryBoundary)

	known := httptest.NewRecorder()
	h.ServeHTTP(known, jsonRequest(t, "/api/v1/auth/account/deletion/recovery/request", map[string]string{"email": u.Email}))
	if known.Code != http.StatusAccepted {
		t.Fatalf("known request status = %d, want 202", known.Code)
	}
	if emails.email != u.Email || emails.token == "" {
		t.Fatal("eligible account must queue a recovery credential")
	}

	unknown := httptest.NewRecorder()
	h.ServeHTTP(unknown, jsonRequest(t, "/api/v1/auth/account/deletion/recovery/request", map[string]string{"email": "missing@example.com"}))
	if unknown.Code != http.StatusAccepted {
		t.Fatalf("unknown request status = %d, want same 202", unknown.Code)
	}

	confirm := httptest.NewRecorder()
	h.ServeHTTP(confirm, jsonRequest(t, "/api/v1/auth/account/deletion/recovery/confirm", map[string]string{"email": u.Email, "token": emails.token}))
	if confirm.Code != http.StatusOK {
		t.Fatalf("confirm status = %d, want 200; body=%s", confirm.Code, confirm.Body.String())
	}

	replay := httptest.NewRecorder()
	h.ServeHTTP(replay, jsonRequest(t, "/api/v1/auth/account/deletion/recovery/confirm", map[string]string{"email": u.Email, "token": emails.token}))
	if replay.Code != http.StatusBadRequest {
		t.Fatalf("replay status = %d, want 400", replay.Code)
	}
}

func jsonRequest(t *testing.T, path string, body map[string]string) *http.Request {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(string(encoded)))
	req.Header.Set("Content-Type", "application/json")
	return req
}
