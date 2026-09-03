package pgstore

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/synaudio/synaudio/backend/internal/identity"
	"github.com/synaudio/synaudio/backend/internal/platform/db"
)

// IdentityStore implements identity.Store backed by PostgreSQL via sqlc.
type IdentityStore struct {
	q *db.Queries
}

func NewIdentityStore(q *db.Queries) *IdentityStore {
	return &IdentityStore{q: q}
}

func (s *IdentityStore) CreateUser(ctx context.Context, u identity.User) (identity.User, error) {
	row, err := s.q.CreateUser(ctx, db.CreateUserParams{
		ID:           toUUID(u.ID),
		Email:        u.Email,
		PasswordHash: toText(u.PasswordHash),
		DisplayName:  toText(u.DisplayName),
		Status:       u.Status,
	})
	if err != nil {
		return identity.User{}, err
	}
	return toIdentityUser(row), nil
}

func (s *IdentityStore) GetUserByEmail(ctx context.Context, email string) (identity.User, error) {
	row, err := s.q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return identity.User{}, identity.ErrUserNotFound
		}
		return identity.User{}, err
	}
	return toIdentityUser(row), nil
}

func (s *IdentityStore) GetUserByID(ctx context.Context, id string) (identity.User, error) {
	row, err := s.q.GetUserByID(ctx, toUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return identity.User{}, identity.ErrUserNotFound
		}
		return identity.User{}, err
	}
	return toIdentityUser(row), nil
}

func (s *IdentityStore) CreateSession(ctx context.Context, sess identity.Session) error {
	id := sess.ID
	if id == "" {
		id = uuid.NewString()
	}
	createdAt := sess.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	_, err := s.q.CreateAuthSession(ctx, db.CreateAuthSessionParams{
		ID:               toUUID(id),
		UserID:           toUUID(sess.UserID),
		RefreshTokenHash: sess.RefreshTokenHash,
		CreatedAt:        pgtypeTimestamptz(createdAt),
		ExpiresAt:        pgtypeTimestamptz(sess.ExpiresAt),
		UserAgentSummary: toText(sess.UserAgentSummary),
		SafeIpMetadata:   toText(sess.SafeIPMetadata),
	})
	return err
}

func (s *IdentityStore) GetSessionByRefreshTokenHash(ctx context.Context, tokenHash string) (identity.Session, error) {
	row, err := s.q.GetSessionByRefreshTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return identity.Session{}, identity.ErrUnauthenticated
		}
		return identity.Session{}, err
	}
	return identity.Session{
		ID:               fromUUID(row.ID),
		UserID:           fromUUID(row.UserID),
		RefreshTokenHash: row.RefreshTokenHash,
		ExpiresAt:        row.ExpiresAt.Time,
	}, nil
}

func (s *IdentityStore) RevokeSession(ctx context.Context, tokenHash string) error {
	return s.q.RevokeSession(ctx, tokenHash)
}

func (s *IdentityStore) StoreVerificationToken(ctx context.Context, userID, tokenHash string) error {
	return s.q.StoreVerificationToken(ctx, db.StoreVerificationTokenParams{
		UserID:    toUUID(userID),
		TokenHash: tokenHash,
	})
}

func (s *IdentityStore) GetVerificationToken(ctx context.Context, userID string) (string, error) {
	h, err := s.q.GetVerificationToken(ctx, toUUID(userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", identity.ErrInvalidToken
		}
		return "", err
	}
	return h, nil
}

func (s *IdentityStore) MarkEmailVerified(ctx context.Context, userID string) error {
	return s.q.MarkEmailVerified(ctx, toUUID(userID))
}

func (s *IdentityStore) StoreResetToken(ctx context.Context, userID, tokenHash string) error {
	return s.q.StoreResetToken(ctx, db.StoreResetTokenParams{
		UserID:    toUUID(userID),
		TokenHash: tokenHash,
	})
}

func (s *IdentityStore) GetResetToken(ctx context.Context, userID string) (string, error) {
	h, err := s.q.GetResetToken(ctx, toUUID(userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", identity.ErrInvalidToken
		}
		return "", err
	}
	return h, nil
}

func (s *IdentityStore) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	return s.q.UpdatePassword(ctx, db.UpdatePasswordParams{
		ID:           toUUID(userID),
		PasswordHash: toText(passwordHash),
	})
}

func (s *IdentityStore) RevokeSessions(ctx context.Context, userID string) error {
	return s.q.RevokeSessions(ctx, toUUID(userID))
}

func (s *IdentityStore) StoreMFAMethod(ctx context.Context, userID string, m identity.MFAMethod) error {
	return s.q.StoreMFAMethod(ctx, db.StoreMFAMethodParams{
		UserID:          toUUID(userID),
		EncryptedSecret: m.Secret,
	})
}

func (s *IdentityStore) GetMFAMethod(ctx context.Context, userID string) (*identity.MFAMethod, error) {
	row, err := s.q.GetMFAMethod(ctx, toUUID(userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, identity.ErrUserNotFound
		}
		return nil, err
	}
	return &identity.MFAMethod{
		Secret:    row.EncryptedSecret,
		Confirmed: row.ConfirmedAt.Valid,
		Disabled:  row.DisabledAt.Valid,
	}, nil
}

func (s *IdentityStore) ConfirmMFAMethod(ctx context.Context, userID string) error {
	return s.q.ConfirmMFAMethod(ctx, toUUID(userID))
}

func (s *IdentityStore) DisableMFAMethod(ctx context.Context, userID string) error {
	return s.q.DisableMFAMethod(ctx, toUUID(userID))
}

func (s *IdentityStore) GetUserRoles(ctx context.Context, userID string) ([]string, error) {
	return s.q.GetUserRoles(ctx, toUUID(userID))
}

func (s *IdentityStore) GetRolePermissions(ctx context.Context, role string) ([]string, error) {
	return s.q.GetRolePermissions(ctx, role)
}

func (s *IdentityStore) GrantRole(ctx context.Context, userID, role string) error {
	return s.q.GrantRole(ctx, db.GrantRoleParams{
		UserID: toUUID(userID),
		Code:   role,
	})
}

func (s *IdentityStore) RevokeRole(ctx context.Context, userID, role string) error {
	return s.q.RevokeRole(ctx, db.RevokeRoleParams{
		UserID: toUUID(userID),
		Code:   role,
	})
}

func (s *IdentityStore) CountActiveAdmins(ctx context.Context) (int, error) {
	n, err := s.q.CountActiveAdmins(ctx)
	return int(n), err
}

func (s *IdentityStore) DeactivateUser(ctx context.Context, userID string) error {
	return s.q.DeactivateUser(ctx, toUUID(userID))
}

func (s *IdentityStore) ReactivateUser(ctx context.Context, userID string) error {
	return s.q.ReactivateUser(ctx, toUUID(userID))
}

func (s *IdentityStore) PurgeUser(ctx context.Context, userID string) error {
	return s.q.PurgeUser(ctx, toUUID(userID))
}

func toIdentityUser(row db.User) identity.User {
	return identity.User{
		ID:              fromUUID(row.ID),
		Email:           row.Email,
		PasswordHash:    fromText(row.PasswordHash),
		DisplayName:     fromText(row.DisplayName),
		Status:          row.Status,
		EmailVerifiedAt: fromTimestamptz(row.EmailVerifiedAt),
	}
}
