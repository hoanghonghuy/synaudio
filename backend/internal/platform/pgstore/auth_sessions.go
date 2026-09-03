package pgstore

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/synaudio/synaudio/backend/internal/identity"
	"github.com/synaudio/synaudio/backend/internal/platform/db"
)

func (s *IdentityStore) RotateSession(ctx context.Context, oldHash, newHash string, now, idleCutoff time.Time) (identity.Session, error) {
	row, err := s.q.RotateAuthSession(ctx, db.RotateAuthSessionParams{
		RefreshTokenHash: oldHash,
		NewRefreshHash:   newHash,
		Now:              pgtypeTimestamptz(now),
		IdleCutoff:       pgtypeTimestamptz(idleCutoff),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return identity.Session{}, identity.ErrUnauthenticated
		}
		return identity.Session{}, err
	}
	return toIdentitySession(row), nil
}

func (s *IdentityStore) TouchSession(ctx context.Context, sessionID string, now, idleCutoff time.Time) (identity.Session, error) {
	row, err := s.q.TouchAuthSession(ctx, db.TouchAuthSessionParams{
		ID:         toUUID(sessionID),
		Now:        pgtypeTimestamptz(now),
		IdleCutoff: pgtypeTimestamptz(idleCutoff),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return identity.Session{}, identity.ErrUnauthenticated
		}
		return identity.Session{}, err
	}
	return toIdentitySession(row), nil
}

func (s *IdentityStore) RevokeSessionByID(ctx context.Context, sessionID, userID string) error {
	rows, err := s.q.RevokeAuthSessionByIDForUser(ctx, db.RevokeAuthSessionByIDForUserParams{
		ID:     toUUID(sessionID),
		UserID: toUUID(userID),
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return identity.ErrSessionNotFound
	}
	return nil
}

func (s *IdentityStore) ListActiveSessions(ctx context.Context, userID string, now, idleCutoff time.Time) ([]identity.Session, error) {
	rows, err := s.q.ListActiveAuthSessions(ctx, db.ListActiveAuthSessionsParams{
		UserID:     toUUID(userID),
		Now:        pgtypeTimestamptz(now),
		IdleCutoff: pgtypeTimestamptz(idleCutoff),
	})
	if err != nil {
		return nil, err
	}
	out := make([]identity.Session, 0, len(rows))
	for _, row := range rows {
		out = append(out, toIdentitySession(row))
	}
	return out, nil
}

func toIdentitySession(row db.UserSession) identity.Session {
	return identity.Session{
		ID:               fromUUID(row.ID),
		UserID:           fromUUID(row.UserID),
		RefreshTokenHash: row.RefreshTokenHash,
		CreatedAt:        authSessionTime(row.CreatedAt),
		LastUsedAt:       authSessionTime(row.LastUsedAt),
		ExpiresAt:        authSessionTime(row.ExpiresAt),
		RevokedAt:        authSessionTime(row.RevokedAt),
		UserAgentSummary: fromText(row.UserAgentSummary),
		SafeIPMetadata:   fromText(row.SafeIpMetadata),
	}
}

func authSessionTime(value pgtype.Timestamptz) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time
}
