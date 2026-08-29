package identity

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
)

const (
	StatusActive      = "ACTIVE"
	StatusSuspended   = "SUSPENDED"
	StatusDeactivated = "DEACTIVATED"
)

var (
	ErrEmailTaken         = errors.New("email already taken")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccountSuspended   = errors.New("account suspended")
	ErrInvalidToken       = errors.New("invalid token")
	ErrLastAdmin          = errors.New("cannot remove last active admin")
	ErrForbidden          = errors.New("forbidden")
	ErrUnauthenticated    = errors.New("authentication required")
)

const (
	RoleGuest = "GUEST"
	RoleUser  = "USER"
	RoleAdmin = "ADMIN"
)

const (
	PermStoryCreate = "STORY_CREATE"

	PermAdminRoleGrant  = "ADMIN_ROLE_GRANT"
	PermAdminRoleRevoke = "ADMIN_ROLE_REVOKE"
)

type User struct {
	ID              string
	Email           string
	PasswordHash    string
	DisplayName     string
	Status          string
	EmailVerifiedAt string
}

type Session struct {
	ID               string
	UserID           string
	RefreshToken     string
	RefreshTokenHash string
	ExpiresAt        time.Time
}

// Store is the persistence boundary for the auth service.
type Store interface {
	CreateUser(ctx context.Context, u User) (User, error)
	GetUserByEmail(ctx context.Context, email string) (User, error)
	GetUserByID(ctx context.Context, id string) (User, error)
	CreateSession(ctx context.Context, s Session) error
	GetSessionByRefreshTokenHash(ctx context.Context, refreshTokenHash string) (Session, error)
	RevokeSession(ctx context.Context, refreshTokenHash string) error

	StoreVerificationToken(ctx context.Context, userID, tokenHash string) error
	GetVerificationToken(ctx context.Context, userID string) (string, error)
	MarkEmailVerified(ctx context.Context, userID string) error

	StoreResetToken(ctx context.Context, userID, tokenHash string) error
	GetResetToken(ctx context.Context, userID string) (string, error)
	UpdatePassword(ctx context.Context, userID, passwordHash string) error
	RevokeSessions(ctx context.Context, userID string) error

	StoreMFAMethod(ctx context.Context, userID string, m MFAMethod) error
	GetMFAMethod(ctx context.Context, userID string) (*MFAMethod, error)
	ConfirmMFAMethod(ctx context.Context, userID string) error
	DisableMFAMethod(ctx context.Context, userID string) error

	GetUserRoles(ctx context.Context, userID string) ([]string, error)
	GetRolePermissions(ctx context.Context, role string) ([]string, error)
	GrantRole(ctx context.Context, userID, role string) error
	RevokeRole(ctx context.Context, userID, role string) error
	CountActiveAdmins(ctx context.Context) (int, error)

	DeactivateUser(ctx context.Context, userID string) error
	ReactivateUser(ctx context.Context, userID string) error
	PurgeUser(ctx context.Context, userID string) error
}

type AuthService struct {
	store Store
}

func NewAuthService(store Store) *AuthService {
	return &AuthService{store: store}
}

// Register creates a normal ACTIVE user with a hashed password.
func (s *AuthService) Register(ctx context.Context, email, password string) (User, error) {
	normalized, err := NormalizeEmailChecked(email)
	if err != nil {
		return User{}, err
	}

	hash, err := HashPassword(password)
	if err != nil {
		return User{}, err
	}

	u := User{
		ID:           uuid.NewString(),
		Email:        normalized,
		PasswordHash: hash,
		Status:       StatusActive,
	}

	return s.store.CreateUser(ctx, u)
}

// Login verifies credentials and creates a new refresh session.
func (s *AuthService) Login(ctx context.Context, email, password string) (Session, error) {
	normalized, err := NormalizeEmailChecked(email)
	if err != nil {
		return Session{}, ErrInvalidCredentials
	}

	u, err := s.store.GetUserByEmail(ctx, normalized)
	if err != nil {
		return Session{}, ErrInvalidCredentials
	}

	if u.Status == StatusSuspended {
		return Session{}, ErrAccountSuspended
	}
	if u.Status != StatusActive {
		return Session{}, ErrInvalidCredentials
	}

	if !VerifyPassword(u.PasswordHash, password) {
		return Session{}, ErrInvalidCredentials
	}

	raw, err := NewRefreshToken()
	if err != nil {
		return Session{}, err
	}

	sess := Session{
		UserID:           u.ID,
		RefreshToken:     raw,
		RefreshTokenHash: HashToken(raw),
		ExpiresAt:        time.Now().Add(30 * 24 * time.Hour),
	}

	if err := s.store.CreateSession(ctx, sess); err != nil {
		return Session{}, err
	}

	return sess, nil
}

// RefreshSession rotates a valid refresh token and creates a new session.
func (s *AuthService) RefreshSession(ctx context.Context, token string) (Session, error) {
	if token == "" {
		return Session{}, ErrUnauthenticated
	}

	current, err := s.store.GetSessionByRefreshTokenHash(ctx, HashToken(token))
	if err != nil {
		return Session{}, ErrUnauthenticated
	}
	if err := s.store.RevokeSession(ctx, current.RefreshTokenHash); err != nil {
		return Session{}, err
	}

	raw, err := NewRefreshToken()
	if err != nil {
		return Session{}, err
	}
	next := Session{
		UserID:           current.UserID,
		RefreshToken:     raw,
		RefreshTokenHash: HashToken(raw),
		ExpiresAt:        time.Now().Add(30 * 24 * time.Hour),
	}
	if err := s.store.CreateSession(ctx, next); err != nil {
		return Session{}, err
	}
	return next, nil
}

// ResolveUserID returns the active user associated with the refresh cookie.
// Callers must not accept a user ID from request headers or bodies instead.
func (s *AuthService) ResolveUserID(ctx context.Context, r *http.Request) (string, error) {
	cookie, err := r.Cookie(RefreshCookieName)
	if err != nil || cookie.Value == "" {
		return "", ErrUnauthenticated
	}

	session, err := s.store.GetSessionByRefreshTokenHash(ctx, HashToken(cookie.Value))
	if err != nil {
		return "", ErrUnauthenticated
	}
	user, err := s.store.GetUserByID(ctx, session.UserID)
	if err != nil || user.Status != StatusActive {
		return "", ErrUnauthenticated
	}
	return session.UserID, nil
}
