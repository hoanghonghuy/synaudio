package identity

import (
	"context"
	"errors"
	"net/http"
	"strings"
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
	ErrSessionNotFound    = errors.New("session not found")
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

// Session is one logical refresh session (normally one signed-in browser/device).
// Refresh token rotation updates RefreshTokenHash in place, preserving the
// session identity used for revocation and access-token binding.
type Session struct {
	ID               string
	UserID           string
	RefreshToken     string
	RefreshTokenHash string
	CreatedAt        time.Time
	LastUsedAt       time.Time
	ExpiresAt        time.Time
	RevokedAt        time.Time
	UserAgentSummary string
	SafeIPMetadata   string
}

type SessionMetadata struct {
	UserAgentSummary string
	SafeIPMetadata   string
}

type Principal struct {
	UserID    string
	SessionID string
}

type AccessToken struct {
	Token     string
	ExpiresAt time.Time
}

// Store is the persistence boundary for the identity service.
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

// SessionStore contains the atomic operations required by the V1 access/refresh
// architecture. Rotation is compare-and-swap on the old refresh-token hash so
// concurrent reuse cannot create multiple valid descendants.
type SessionStore interface {
	RotateSession(ctx context.Context, oldHash, newHash string, now, idleCutoff time.Time) (Session, error)
	TouchSession(ctx context.Context, sessionID string, now, idleCutoff time.Time) (Session, error)
	RevokeSessionByID(ctx context.Context, sessionID, userID string) error
	ListActiveSessions(ctx context.Context, userID string, now, idleCutoff time.Time) ([]Session, error)
}

type AuthSettings struct {
	AccessTokenSecret     string
	AccessTokenTTL        time.Duration
	RefreshSessionTTL     time.Duration
	RefreshSessionIdleTTL time.Duration
	Now                    func() time.Time
}

type AuthOption func(*AuthService)

func WithAuthSettings(settings AuthSettings) AuthOption {
	return func(s *AuthService) {
		if settings.AccessTokenSecret != "" {
			s.settings.AccessTokenSecret = settings.AccessTokenSecret
		}
		if settings.AccessTokenTTL > 0 {
			s.settings.AccessTokenTTL = settings.AccessTokenTTL
		}
		if settings.RefreshSessionTTL > 0 {
			s.settings.RefreshSessionTTL = settings.RefreshSessionTTL
		}
		if settings.RefreshSessionIdleTTL > 0 {
			s.settings.RefreshSessionIdleTTL = settings.RefreshSessionIdleTTL
		}
		if settings.Now != nil {
			s.settings.Now = settings.Now
		}
	}
}

type AuthService struct {
	store        Store
	settings     AuthSettings
	accessTokens *AccessTokenManager
}

func NewAuthService(store Store, opts ...AuthOption) *AuthService {
	settings := AuthSettings{
		AccessTokenSecret:     "test-and-development-access-token-secret-change-me",
		AccessTokenTTL:        15 * time.Minute,
		RefreshSessionTTL:     30 * 24 * time.Hour,
		RefreshSessionIdleTTL: 7 * 24 * time.Hour,
		Now:                    time.Now,
	}
	s := &AuthService{store: store, settings: settings}
	for _, opt := range opts {
		opt(s)
	}
	manager, err := NewAccessTokenManager(s.settings.AccessTokenSecret, s.settings.AccessTokenTTL, s.settings.Now)
	if err == nil {
		s.accessTokens = manager
	}
	return s
}

func (s *AuthService) sessionStore() (SessionStore, error) {
	store, ok := s.store.(SessionStore)
	if !ok {
		return nil, errors.New("session lifecycle persistence not configured")
	}
	return store, nil
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

// Login verifies credentials and creates a new logical refresh session.
func (s *AuthService) Login(ctx context.Context, email, password string) (Session, error) {
	return s.LoginWithMetadata(ctx, email, password, SessionMetadata{})
}

func (s *AuthService) LoginWithMetadata(ctx context.Context, email, password string, metadata SessionMetadata) (Session, error) {
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

	now := s.settings.Now().UTC()
	sess := Session{
		ID:               uuid.NewString(),
		UserID:           u.ID,
		RefreshToken:     raw,
		RefreshTokenHash: HashToken(raw),
		CreatedAt:        now,
		LastUsedAt:       now,
		ExpiresAt:        now.Add(s.settings.RefreshSessionTTL),
		UserAgentSummary: strings.TrimSpace(metadata.UserAgentSummary),
		SafeIPMetadata:   strings.TrimSpace(metadata.SafeIPMetadata),
	}

	if err := s.store.CreateSession(ctx, sess); err != nil {
		return Session{}, err
	}

	return sess, nil
}

// RefreshSession atomically rotates a valid opaque refresh credential in place.
// Once one request swaps the old hash, concurrent/replayed uses of that old token
// can no longer match the persisted session.
func (s *AuthService) RefreshSession(ctx context.Context, token string) (Session, error) {
	if strings.TrimSpace(token) == "" {
		return Session{}, ErrUnauthenticated
	}
	store, err := s.sessionStore()
	if err != nil {
		return Session{}, err
	}

	raw, err := NewRefreshToken()
	if err != nil {
		return Session{}, err
	}
	now := s.settings.Now().UTC()
	sess, err := store.RotateSession(ctx, HashToken(token), HashToken(raw), now, now.Add(-s.settings.RefreshSessionIdleTTL))
	if err != nil {
		if errors.Is(err, ErrUnauthenticated) || errors.Is(err, ErrSessionNotFound) {
			return Session{}, ErrUnauthenticated
		}
		return Session{}, err
	}

	user, err := s.store.GetUserByID(ctx, sess.UserID)
	if err != nil || user.Status != StatusActive {
		_ = store.RevokeSessionByID(ctx, sess.ID, sess.UserID)
		return Session{}, ErrUnauthenticated
	}

	sess.RefreshToken = raw
	return sess, nil
}

func (s *AuthService) IssueAccessToken(sess Session) (AccessToken, error) {
	if s.accessTokens == nil {
		return AccessToken{}, errors.New("access token manager not configured")
	}
	token, expiresAt, err := s.accessTokens.Issue(sess.UserID, sess.ID)
	if err != nil {
		return AccessToken{}, err
	}
	return AccessToken{Token: token, ExpiresAt: expiresAt}, nil
}

// AuthenticateRequest validates the Bearer JWT, validates/touches its bound
// persisted session, and re-checks account state. The refresh cookie is never
// used as the normal authorization credential for protected API traffic.
func (s *AuthService) AuthenticateRequest(ctx context.Context, r *http.Request) (Principal, User, error) {
	token, err := accessTokenFromRequest(r)
	if err != nil {
		return Principal{}, User{}, ErrUnauthenticated
	}
	return s.AuthenticateAccessToken(ctx, token)
}

func (s *AuthService) AuthenticateAccessToken(ctx context.Context, token string) (Principal, User, error) {
	if s.accessTokens == nil {
		return Principal{}, User{}, ErrUnauthenticated
	}
	claims, err := s.accessTokens.Parse(token)
	if err != nil {
		return Principal{}, User{}, ErrUnauthenticated
	}
	store, err := s.sessionStore()
	if err != nil {
		return Principal{}, User{}, ErrUnauthenticated
	}

	now := s.settings.Now().UTC()
	sess, err := store.TouchSession(ctx, claims.SessionID, now, now.Add(-s.settings.RefreshSessionIdleTTL))
	if err != nil || sess.UserID != claims.Subject {
		return Principal{}, User{}, ErrUnauthenticated
	}
	user, err := s.store.GetUserByID(ctx, claims.Subject)
	if err != nil || user.Status != StatusActive {
		_ = store.RevokeSessionByID(ctx, sess.ID, sess.UserID)
		return Principal{}, User{}, ErrUnauthenticated
	}
	return Principal{UserID: claims.Subject, SessionID: claims.SessionID}, user, nil
}

// ResolveUserID is the compatibility boundary used by domain handlers. It now
// resolves exclusively from a valid Bearer access token, never a refresh cookie.
func (s *AuthService) ResolveUserID(ctx context.Context, r *http.Request) (string, error) {
	principal, _, err := s.AuthenticateRequest(ctx, r)
	if err != nil {
		return "", ErrUnauthenticated
	}
	return principal.UserID, nil
}

func (s *AuthService) ListSessions(ctx context.Context, principal Principal) ([]Session, error) {
	store, err := s.sessionStore()
	if err != nil {
		return nil, err
	}
	now := s.settings.Now().UTC()
	return store.ListActiveSessions(ctx, principal.UserID, now, now.Add(-s.settings.RefreshSessionIdleTTL))
}

func (s *AuthService) RevokeSessionByID(ctx context.Context, principal Principal, sessionID string) error {
	if strings.TrimSpace(sessionID) == "" {
		return ErrSessionNotFound
	}
	store, err := s.sessionStore()
	if err != nil {
		return err
	}
	return store.RevokeSessionByID(ctx, sessionID, principal.UserID)
}

func (s *AuthService) RevokeAllSessions(ctx context.Context, userID string) error {
	if strings.TrimSpace(userID) == "" {
		return ErrUnauthenticated
	}
	return s.store.RevokeSessions(ctx, userID)
}

func accessTokenFromRequest(r *http.Request) (string, error) {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if header == "" {
		return "", ErrUnauthenticated
	}
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		return "", ErrUnauthenticated
	}
	return parts[1], nil
}
