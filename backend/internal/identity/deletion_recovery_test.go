package identity_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/synaudio/synaudio/backend/internal/identity"
)

type deletionRecoveryFakeStore struct {
	*fakeStore
	tokenHash string
	expiresAt time.Time
	used      bool
}

func newDeletionRecoveryFakeStore() *deletionRecoveryFakeStore {
	return &deletionRecoveryFakeStore{fakeStore: newFakeStore()}
}

func (s *deletionRecoveryFakeStore) ReplaceDeletionRecoveryToken(_ context.Context, userID, tokenHash string, expiresAt time.Time) error {
	u, err := s.GetUserByID(context.Background(), userID)
	if err != nil || u.Status != identity.StatusDeactivated {
		return identity.ErrInvalidToken
	}
	s.tokenHash = tokenHash
	s.expiresAt = expiresAt
	s.used = false
	return nil
}

func (s *deletionRecoveryFakeStore) CancelAccountDeletionWithToken(_ context.Context, userID, tokenHash string, now time.Time) error {
	if s.used || s.tokenHash == "" || s.tokenHash != tokenHash || !s.expiresAt.After(now) {
		return identity.ErrInvalidToken
	}
	u, err := s.GetUserByID(context.Background(), userID)
	if err != nil || u.Status != identity.StatusDeactivated {
		return identity.ErrInvalidToken
	}
	s.used = true
	return s.ReactivateUser(context.Background(), userID)
}

func TestDeletionRecoveryReactivatesDeactivatedAccountWithOneTimeToken(t *testing.T) {
	store := newDeletionRecoveryFakeStore()
	svc := identity.NewAuthService(store)
	u, err := svc.Register(context.Background(), "reader@example.com", "correct password")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.DeactivateUser(context.Background(), u.ID); err != nil {
		t.Fatal(err)
	}

	token, err := svc.RequestAccountDeletionRecovery(context.Background(), u.Email)
	if err != nil {
		t.Fatalf("request recovery: %v", err)
	}
	if token == "" || store.tokenHash == token {
		t.Fatal("expected opaque raw token with only its hash persisted")
	}
	if err := svc.CancelAccountDeletionByEmail(context.Background(), u.Email, token); err != nil {
		t.Fatalf("confirm recovery: %v", err)
	}

	reactivated, err := store.GetUserByEmail(context.Background(), u.Email)
	if err != nil {
		t.Fatal(err)
	}
	if reactivated.Status != identity.StatusActive {
		t.Fatalf("expected ACTIVE after recovery, got %s", reactivated.Status)
	}
	if err := svc.CancelAccountDeletionByEmail(context.Background(), u.Email, token); !errors.Is(err, identity.ErrInvalidToken) {
		t.Fatalf("expected replay rejection, got %v", err)
	}
}

func TestDeletionRecoveryRejectsExpiredToken(t *testing.T) {
	store := newDeletionRecoveryFakeStore()
	svc := identity.NewAuthService(store)
	u, _ := svc.Register(context.Background(), "reader@example.com", "correct password")
	_ = store.DeactivateUser(context.Background(), u.ID)

	token, err := svc.RequestAccountDeletionRecovery(context.Background(), u.Email)
	if err != nil {
		t.Fatal(err)
	}
	store.expiresAt = time.Now().UTC().Add(-time.Minute)
	if err := svc.CancelAccountDeletionByEmail(context.Background(), u.Email, token); !errors.Is(err, identity.ErrInvalidToken) {
		t.Fatalf("expected expired token rejection, got %v", err)
	}
}

func TestDeletionRecoveryDoesNotIssueTokenForActiveAccount(t *testing.T) {
	store := newDeletionRecoveryFakeStore()
	svc := identity.NewAuthService(store)
	_, _ = svc.Register(context.Background(), "reader@example.com", "correct password")

	if _, err := svc.RequestAccountDeletionRecovery(context.Background(), "reader@example.com"); !errors.Is(err, identity.ErrUserNotFound) {
		t.Fatalf("expected enumeration-safe ineligible result, got %v", err)
	}
}
