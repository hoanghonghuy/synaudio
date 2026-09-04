package identity_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/identity"
)

type fakeTransactionalEmail struct {
	verificationEmail string
	verificationToken string
	resetEmail        string
	resetToken        string
	err               error
}

func (f *fakeTransactionalEmail) QueueVerification(_ context.Context, email, token string) error {
	if f.err != nil {
		return f.err
	}
	f.verificationEmail = email
	f.verificationToken = token
	return nil
}

func (f *fakeTransactionalEmail) QueuePasswordReset(_ context.Context, email, token string) error {
	if f.err != nil {
		return f.err
	}
	f.resetEmail = email
	f.resetToken = token
	return nil
}

func directIdentityBoundary(_ context.Context, run func(context.Context) error) error {
	return run(context.Background())
}

func TestTransactionalEmailRegistrationCreatesHashedTokenAndDurableIntent(t *testing.T) {
	store := newFakeStore()
	svc := identity.NewAuthService(store)
	emails := &fakeTransactionalEmail{}
	h := identity.WrapTransactionalEmail(identity.NewAuthHandler(svc), svc, emails, directIdentityBoundary)

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{"email":"Reader@Example.com","password":"correct password"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("register status=%d body=%s", rec.Code, rec.Body.String())
	}
	if emails.verificationEmail != "reader@example.com" || emails.verificationToken == "" {
		t.Fatalf("verification intent not queued: email=%q tokenPresent=%v", emails.verificationEmail, emails.verificationToken != "")
	}
	u, err := store.GetUserByEmail(context.Background(), "reader@example.com")
	if err != nil {
		t.Fatal(err)
	}
	hash := store.verificationTokens[u.ID]
	if hash == "" || hash == emails.verificationToken {
		t.Fatal("verification token must be stored as a hash, never raw")
	}
	if !identity.VerifyTokenHash(hash, emails.verificationToken) {
		t.Fatal("queued verification token does not match durable token hash")
	}
}

func TestTransactionalEmailRegistrationDoesNotReturnSuccessWhenIntentCannotPersist(t *testing.T) {
	store := newFakeStore()
	svc := identity.NewAuthService(store)
	emails := &fakeTransactionalEmail{err: errors.New("outbox unavailable")}
	h := identity.WrapTransactionalEmail(identity.NewAuthHandler(svc), svc, emails, directIdentityBoundary)

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{"email":"reader@example.com","password":"correct password"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected non-success response when durable email intent fails, got %d", rec.Code)
	}
}

func TestPasswordForgotAndResendRemainEnumerationSafe(t *testing.T) {
	store := newFakeStore()
	svc := identity.NewAuthService(store)
	if _, err := svc.Register(context.Background(), "known@example.com", "correct password"); err != nil {
		t.Fatal(err)
	}
	emails := &fakeTransactionalEmail{}
	h := identity.WrapTransactionalEmail(identity.NewAuthHandler(svc), svc, emails, directIdentityBoundary)

	for _, tc := range []struct {
		name string
		path string
	}{
		{name: "resend", path: "/email/resend"},
		{name: "forgot", path: "/password/forgot"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			known := httptest.NewRecorder()
			h.ServeHTTP(known, httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(`{"email":"known@example.com"}`)))
			unknown := httptest.NewRecorder()
			h.ServeHTTP(unknown, httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(`{"email":"unknown@example.com"}`)))
			if known.Code != http.StatusAccepted || unknown.Code != http.StatusAccepted {
				t.Fatalf("enumeration-safe status mismatch known=%d unknown=%d", known.Code, unknown.Code)
			}
			if known.Body.String() != unknown.Body.String() {
				t.Fatalf("enumeration-safe response body mismatch known=%q unknown=%q", known.Body.String(), unknown.Body.String())
			}
		})
	}
	if emails.verificationToken == "" || emails.resetToken == "" {
		t.Fatal("known account did not enqueue both transactional-email intents")
	}
}
