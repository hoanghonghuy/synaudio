package identity_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/synaudio/synaudio/backend/internal/identity"
)

func newTimedAuthService(store *fakeStore, now *time.Time, accessTTL, absoluteTTL, idleTTL time.Duration) *identity.AuthService {
	return identity.NewAuthService(store, identity.WithAuthSettings(identity.AuthSettings{
		AccessTokenSecret:     "test-access-token-secret-that-is-longer-than-32-bytes",
		AccessTokenTTL:        accessTTL,
		RefreshSessionTTL:     absoluteTTL,
		RefreshSessionIdleTTL: idleTTL,
		Now: func() time.Time {
			return *now
		},
	}))
}

func registerAndLogin(t *testing.T, svc *identity.AuthService, email string) (identity.User, identity.Session, identity.AccessToken) {
	t.Helper()
	u, err := svc.Register(context.Background(), email, "correct password")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	sess, err := svc.Login(context.Background(), email, "correct password")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	access, err := svc.IssueAccessToken(sess)
	if err != nil {
		t.Fatalf("issue access token: %v", err)
	}
	return u, sess, access
}

func TestAccessTokenExpiresAtConfiguredTTL(t *testing.T) {
	now := time.Date(2026, 9, 3, 10, 0, 0, 0, time.UTC)
	store := newFakeStore()
	svc := newTimedAuthService(store, &now, time.Minute, 24*time.Hour, 12*time.Hour)
	_, _, access := registerAndLogin(t, svc, "expiry@example.com")

	if _, _, err := svc.AuthenticateAccessToken(context.Background(), access.Token); err != nil {
		t.Fatalf("expected fresh access token to authenticate: %v", err)
	}
	now = now.Add(time.Minute + time.Second)
	if _, _, err := svc.AuthenticateAccessToken(context.Background(), access.Token); !errors.Is(err, identity.ErrUnauthenticated) {
		t.Fatalf("expected expired access token rejection, got %v", err)
	}
}

func TestIdleExpiredSessionRejectsOtherwiseValidAccessToken(t *testing.T) {
	now := time.Date(2026, 9, 3, 10, 0, 0, 0, time.UTC)
	store := newFakeStore()
	svc := newTimedAuthService(store, &now, 30*24*time.Hour, 30*24*time.Hour, 7*24*time.Hour)
	_, _, access := registerAndLogin(t, svc, "idle@example.com")

	now = now.Add(8 * 24 * time.Hour)
	if _, _, err := svc.AuthenticateAccessToken(context.Background(), access.Token); !errors.Is(err, identity.ErrUnauthenticated) {
		t.Fatalf("expected idle-expired session rejection, got %v", err)
	}
}

func TestAbsoluteExpiredSessionRejectsOtherwiseValidAccessToken(t *testing.T) {
	now := time.Date(2026, 9, 3, 10, 0, 0, 0, time.UTC)
	store := newFakeStore()
	svc := newTimedAuthService(store, &now, 30*24*time.Hour, 24*time.Hour, 24*time.Hour)
	_, _, access := registerAndLogin(t, svc, "absolute@example.com")

	now = now.Add(24*time.Hour + time.Second)
	if _, _, err := svc.AuthenticateAccessToken(context.Background(), access.Token); !errors.Is(err, identity.ErrUnauthenticated) {
		t.Fatalf("expected absolute-expired session rejection, got %v", err)
	}
}

func TestSuspendedAccountInvalidatesStaleAccessToken(t *testing.T) {
	now := time.Date(2026, 9, 3, 10, 0, 0, 0, time.UTC)
	store := newFakeStore()
	svc := newTimedAuthService(store, &now, 15*time.Minute, 30*24*time.Hour, 7*24*time.Hour)
	u, sess, access := registerAndLogin(t, svc, "suspended@example.com")

	u.Status = identity.StatusSuspended
	store.users[u.Email] = u
	if _, _, err := svc.AuthenticateAccessToken(context.Background(), access.Token); !errors.Is(err, identity.ErrUnauthenticated) {
		t.Fatalf("expected suspended account rejection, got %v", err)
	}
	if stored := store.sessions[sess.ID]; stored.RevokedAt.IsZero() {
		t.Fatal("expected suspended account session to be revoked")
	}
}

func TestRefreshCredentialCanOnlyRotateOnce(t *testing.T) {
	now := time.Date(2026, 9, 3, 10, 0, 0, 0, time.UTC)
	store := newFakeStore()
	svc := newTimedAuthService(store, &now, 15*time.Minute, 30*24*time.Hour, 7*24*time.Hour)
	_, sess, _ := registerAndLogin(t, svc, "rotate@example.com")

	rotated, err := svc.RefreshSession(context.Background(), sess.RefreshToken)
	if err != nil {
		t.Fatalf("first refresh: %v", err)
	}
	if rotated.RefreshToken == sess.RefreshToken {
		t.Fatal("expected refresh token rotation")
	}
	if _, err := svc.RefreshSession(context.Background(), sess.RefreshToken); !errors.Is(err, identity.ErrUnauthenticated) {
		t.Fatalf("expected replayed refresh credential rejection, got %v", err)
	}
}
