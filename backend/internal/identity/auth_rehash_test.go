package identity_test

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/identity"
	"golang.org/x/crypto/argon2"
)

type rehashStore struct {
	*fakeStore
	updatePasswordErr   error
	updatePasswordCalls int
	createSessionCalls  int
}

func newRehashStore() *rehashStore {
	return &rehashStore{fakeStore: newFakeStore()}
}

func (s *rehashStore) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	s.updatePasswordCalls++
	if s.updatePasswordErr != nil {
		return s.updatePasswordErr
	}
	return s.fakeStore.UpdatePassword(ctx, userID, passwordHash)
}

func (s *rehashStore) CreateSession(ctx context.Context, sess identity.Session) error {
	s.createSessionCalls++
	return s.fakeStore.CreateSession(ctx, sess)
}

func argon2HashForTest(password string, memory, timeCost uint32, threads uint8) string {
	salt := []byte("0123456789abcdef")
	key := argon2.IDKey([]byte(password), salt, timeCost, memory, threads, 32)
	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		memory,
		timeCost,
		threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	)
}

func seedPasswordUser(store *rehashStore, hash string) identity.User {
	u := identity.User{
		ID:           "user-legacy",
		Email:        "legacy@example.com",
		PasswordHash: hash,
		Status:       identity.StatusActive,
	}
	store.users[u.Email] = u
	return u
}

func TestLoginRehashesAcceptedLegacyPasswordBeforeCreatingSession(t *testing.T) {
	const password = "legacy password"
	store := newRehashStore()
	u := seedPasswordUser(store, argon2HashForTest(password, 32*1024, 2, 2))
	svc := identity.NewAuthService(store)

	if _, err := svc.Login(context.Background(), u.Email, password); err != nil {
		t.Fatalf("login: %v", err)
	}
	if store.updatePasswordCalls != 1 {
		t.Fatalf("expected one password upgrade, got %d", store.updatePasswordCalls)
	}
	if store.createSessionCalls != 1 {
		t.Fatalf("expected session after upgrade, got %d", store.createSessionCalls)
	}

	upgraded := store.users[u.Email].PasswordHash
	if upgraded == u.PasswordHash {
		t.Fatal("expected persisted password hash to change")
	}
	matched, needsRehash := identity.VerifyPasswordPolicy(upgraded, password)
	if !matched || needsRehash {
		t.Fatalf("expected upgraded hash to match current policy, matched=%v needsRehash=%v", matched, needsRehash)
	}
}

func TestLoginFailsWithoutSessionWhenLegacyRehashPersistenceFails(t *testing.T) {
	const password = "legacy password"
	store := newRehashStore()
	u := seedPasswordUser(store, argon2HashForTest(password, 32*1024, 2, 2))
	persistErr := errors.New("password persistence unavailable")
	store.updatePasswordErr = persistErr
	svc := identity.NewAuthService(store)

	if _, err := svc.Login(context.Background(), u.Email, password); !errors.Is(err, persistErr) {
		t.Fatalf("expected persistence error, got %v", err)
	}
	if store.updatePasswordCalls != 1 {
		t.Fatalf("expected one attempted password upgrade, got %d", store.updatePasswordCalls)
	}
	if store.createSessionCalls != 0 || len(store.sessions) != 0 {
		t.Fatalf("expected no session after failed upgrade, calls=%d sessions=%d", store.createSessionCalls, len(store.sessions))
	}
	if got := store.users[u.Email].PasswordHash; got != u.PasswordHash {
		t.Fatal("failed persistence must not mutate stored password hash")
	}
}

func TestLoginWrongPasswordDoesNotRehashOrCreateSession(t *testing.T) {
	store := newRehashStore()
	u := seedPasswordUser(store, argon2HashForTest("correct password", 32*1024, 2, 2))
	svc := identity.NewAuthService(store)

	if _, err := svc.Login(context.Background(), u.Email, "wrong password"); !errors.Is(err, identity.ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
	if store.updatePasswordCalls != 0 || store.createSessionCalls != 0 {
		t.Fatalf("wrong password must not rehash or create session, updates=%d sessions=%d", store.updatePasswordCalls, store.createSessionCalls)
	}
}

func TestLoginDoesNotDowngradeMixedHistoricalArgon2Profile(t *testing.T) {
	const password = "mixed profile password"
	store := newRehashStore()
	// Lower memory but higher time cost is intentionally incomparable with the
	// current write policy and therefore must not be rewritten downward.
	u := seedPasswordUser(store, argon2HashForTest(password, 32*1024, 4, 4))
	svc := identity.NewAuthService(store)

	if _, err := svc.Login(context.Background(), u.Email, password); err != nil {
		t.Fatalf("login: %v", err)
	}
	if store.updatePasswordCalls != 0 {
		t.Fatalf("mixed historical profile must not be rewritten, got %d updates", store.updatePasswordCalls)
	}
	if store.createSessionCalls != 1 {
		t.Fatalf("expected normal session creation, got %d", store.createSessionCalls)
	}
}

func TestLoginCurrentPolicyDoesNotRewritePassword(t *testing.T) {
	const password = "current password"
	store := newRehashStore()
	svc := identity.NewAuthService(store)

	u, err := svc.Register(context.Background(), "current@example.com", password)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	store.updatePasswordCalls = 0

	if _, err := svc.Login(context.Background(), u.Email, password); err != nil {
		t.Fatalf("login: %v", err)
	}
	if store.updatePasswordCalls != 0 {
		t.Fatalf("current policy password must not be rewritten, got %d updates", store.updatePasswordCalls)
	}
	if store.createSessionCalls != 1 {
		t.Fatalf("expected one session, got %d", store.createSessionCalls)
	}
}
