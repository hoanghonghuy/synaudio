package identity_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/identity"
)

func TestNewRefreshTokenIsOpaqueAndUnique(t *testing.T) {
	a, err := identity.NewRefreshToken()
	if err != nil {
		t.Fatalf("new refresh token: %v", err)
	}
	b, err := identity.NewRefreshToken()
	if err != nil {
		t.Fatalf("new refresh token: %v", err)
	}
	if a == b {
		t.Fatal("expected distinct refresh tokens")
	}
	if len(a) < 32 {
		t.Fatalf("expected token length >= 32, got %d", len(a))
	}
}

func TestHashTokenVerifies(t *testing.T) {
	token, _ := identity.NewRefreshToken()
	hash := identity.HashToken(token)

	if hash == token {
		t.Fatal("hash must not equal raw token")
	}
	if !identity.VerifyTokenHash(hash, token) {
		t.Fatal("expected token to verify against its hash")
	}
	if identity.VerifyTokenHash(hash, "other-token") {
		t.Fatal("expected different token to fail verification")
	}
}

type fakeStore struct {
	users              map[string]identity.User
	sessions           map[string]identity.Session
	verificationTokens map[string]string
	resetTokens        map[string]string
	sessionsRevoked    bool
	mfaMethods         map[string]*identity.MFAMethod
	userRoles          map[string][]string
	rolePermissions    map[string][]string
	nextID             int
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		users:              map[string]identity.User{},
		sessions:           map[string]identity.Session{},
		verificationTokens: map[string]string{},
		resetTokens:        map[string]string{},
		mfaMethods:         map[string]*identity.MFAMethod{},
		userRoles:          map[string][]string{},
		rolePermissions:    map[string][]string{},
	}
}

func (s *fakeStore) CreateUser(_ context.Context, u identity.User) (identity.User, error) {
	if _, exists := s.users[u.Email]; exists {
		return identity.User{}, identity.ErrEmailTaken
	}
	if u.ID == "" {
		s.nextID++
		u.ID = fmt.Sprintf("user-%d", s.nextID)
	}
	s.users[u.Email] = u
	return u, nil
}

func (s *fakeStore) GetUserByEmail(_ context.Context, email string) (identity.User, error) {
	u, ok := s.users[email]
	if !ok {
		return identity.User{}, identity.ErrUserNotFound
	}
	return u, nil
}

func (s *fakeStore) CreateSession(_ context.Context, sess identity.Session) error {
	s.sessions[sess.ID] = sess
	return nil
}

func (s *fakeStore) GetUserByID(_ context.Context, id string) (identity.User, error) {
	for _, u := range s.users {
		if u.ID == id {
			return u, nil
		}
	}
	return identity.User{}, identity.ErrUserNotFound
}

func (s *fakeStore) StoreVerificationToken(_ context.Context, userID, tokenHash string) error {
	s.verificationTokens[userID] = tokenHash
	return nil
}

func (s *fakeStore) GetVerificationToken(_ context.Context, userID string) (string, error) {
	h, ok := s.verificationTokens[userID]
	if !ok {
		return "", identity.ErrInvalidToken
	}
	return h, nil
}

func (s *fakeStore) MarkEmailVerified(_ context.Context, userID string) error {
	for email, u := range s.users {
		if u.ID == userID {
			u.EmailVerifiedAt = "now"
			s.users[email] = u
			return nil
		}
	}
	return identity.ErrUserNotFound
}

func (s *fakeStore) StoreResetToken(_ context.Context, userID, tokenHash string) error {
	s.resetTokens[userID] = tokenHash
	return nil
}

func (s *fakeStore) GetResetToken(_ context.Context, userID string) (string, error) {
	h, ok := s.resetTokens[userID]
	if !ok {
		return "", identity.ErrInvalidToken
	}
	return h, nil
}

func (s *fakeStore) UpdatePassword(_ context.Context, userID, passwordHash string) error {
	for email, u := range s.users {
		if u.ID == userID {
			u.PasswordHash = passwordHash
			s.users[email] = u
			return nil
		}
	}
	return identity.ErrUserNotFound
}

func (s *fakeStore) RevokeSessions(_ context.Context, userID string) error {
	s.sessionsRevoked = true
	return nil
}

func (s *fakeStore) StoreMFAMethod(_ context.Context, userID string, m identity.MFAMethod) error {
	cp := m
	s.mfaMethods[userID] = &cp
	return nil
}

func (s *fakeStore) GetMFAMethod(_ context.Context, userID string) (*identity.MFAMethod, error) {
	m, ok := s.mfaMethods[userID]
	if !ok {
		return nil, identity.ErrUserNotFound
	}
	return m, nil
}

func (s *fakeStore) ConfirmMFAMethod(_ context.Context, userID string) error {
	m, ok := s.mfaMethods[userID]
	if !ok {
		return identity.ErrUserNotFound
	}
	m.Confirmed = true
	return nil
}

func (s *fakeStore) DisableMFAMethod(_ context.Context, userID string) error {
	m, ok := s.mfaMethods[userID]
	if !ok {
		return identity.ErrUserNotFound
	}
	m.Disabled = true
	return nil
}

func (s *fakeStore) GetUserRoles(_ context.Context, userID string) ([]string, error) {
	roles, ok := s.userRoles[userID]
	if !ok {
		return nil, identity.ErrUserNotFound
	}
	return roles, nil
}

func (s *fakeStore) GetRolePermissions(_ context.Context, role string) ([]string, error) {
	perms, ok := s.rolePermissions[role]
	if !ok {
		return nil, nil
	}
	return perms, nil
}

func (s *fakeStore) GrantRole(_ context.Context, userID, role string) error {
	s.userRoles[userID] = append(s.userRoles[userID], role)
	return nil
}

func (s *fakeStore) RevokeRole(_ context.Context, userID, role string) error {
	roles := s.userRoles[userID]
	out := roles[:0]
	for _, r := range roles {
		if r != role {
			out = append(out, r)
		}
	}
	s.userRoles[userID] = out
	return nil
}

func (s *fakeStore) CountActiveAdmins(_ context.Context) (int, error) {
	count := 0
	for _, roles := range s.userRoles {
		if contains(roles, identity.RoleAdmin) {
			count++
		}
	}
	return count, nil
}

func (s *fakeStore) DeactivateUser(_ context.Context, userID string) error {
	for email, u := range s.users {
		if u.ID == userID {
			u.Status = identity.StatusDeactivated
			s.users[email] = u
			return nil
		}
	}
	return identity.ErrUserNotFound
}

func (s *fakeStore) ReactivateUser(_ context.Context, userID string) error {
	for email, u := range s.users {
		if u.ID == userID {
			u.Status = identity.StatusActive
			s.users[email] = u
			return nil
		}
	}
	return identity.ErrUserNotFound
}

func (s *fakeStore) PurgeUser(_ context.Context, userID string) error {
	for email, u := range s.users {
		if u.ID == userID {
			u.Email = ""
			u.PasswordHash = ""
			u.DisplayName = ""
			s.users[email] = u
			return nil
		}
	}
	return identity.ErrUserNotFound
}

func TestRegisterCreatesActiveUserWithHashedPassword(t *testing.T) {
	store := newFakeStore()
	svc := identity.NewAuthService(store)

	email := "  User@Example.COM "
	password := "correct horse battery staple"

	u, err := svc.Register(context.Background(), email, password)
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	if u.Email != "user@example.com" {
		t.Fatalf("expected normalized email, got %q", u.Email)
	}
	if u.Status != identity.StatusActive {
		t.Fatalf("expected ACTIVE status, got %q", u.Status)
	}
	if u.PasswordHash == password {
		t.Fatal("password must be hashed, not stored in plaintext")
	}
	if !identity.VerifyPassword(u.PasswordHash, password) {
		t.Fatal("stored hash must verify against the password")
	}
}

func TestRegisterRejectsDuplicateEmail(t *testing.T) {
	store := newFakeStore()
	svc := identity.NewAuthService(store)

	if _, err := svc.Register(context.Background(), "a@example.com", "password123"); err != nil {
		t.Fatalf("first register: %v", err)
	}
	if _, err := svc.Register(context.Background(), "A@Example.com", "password456"); !errors.Is(err, identity.ErrEmailTaken) {
		t.Fatalf("expected ErrEmailTaken, got %v", err)
	}
}

func TestLoginSucceedsWithCorrectPassword(t *testing.T) {
	store := newFakeStore()
	svc := identity.NewAuthService(store)

	email := "user@example.com"
	password := "correct horse battery staple"
	if _, err := svc.Register(context.Background(), email, password); err != nil {
		t.Fatalf("register: %v", err)
	}

	sess, err := svc.Login(context.Background(), email, password)
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if sess.RefreshTokenHash == "" {
		t.Fatal("expected refresh token hash to be set")
	}
	if sess.RefreshToken == "" {
		t.Fatal("expected raw refresh token to be returned")
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	store := newFakeStore()
	svc := identity.NewAuthService(store)

	if _, err := svc.Register(context.Background(), "user@example.com", "correct password"); err != nil {
		t.Fatalf("register: %v", err)
	}

	if _, err := svc.Login(context.Background(), "user@example.com", "wrong password"); !errors.Is(err, identity.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLoginRejectsSuspendedAccount(t *testing.T) {
	store := newFakeStore()
	svc := identity.NewAuthService(store)

	u, err := svc.Register(context.Background(), "user@example.com", "correct password")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	u.Status = identity.StatusSuspended
	store.users[u.Email] = u

	if _, err := svc.Login(context.Background(), "user@example.com", "correct password"); !errors.Is(err, identity.ErrAccountSuspended) {
		t.Fatalf("expected ErrAccountSuspended, got %v", err)
	}
}
