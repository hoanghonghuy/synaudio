package identity_test

import (
	"context"
	"time"

	"github.com/synaudio/synaudio/backend/internal/identity"
)

func (s *fakeStore) RotateSession(_ context.Context, oldHash, newHash string, now, idleCutoff time.Time) (identity.Session, error) {
	if s.sessionsRevoked {
		return identity.Session{}, identity.ErrUnauthenticated
	}
	for id, sess := range s.sessions {
		lastUsed := sess.LastUsedAt
		if lastUsed.IsZero() {
			lastUsed = sess.CreatedAt
		}
		if sess.RefreshTokenHash != oldHash || !sess.RevokedAt.IsZero() || !sess.ExpiresAt.After(now) || lastUsed.Before(idleCutoff) {
			continue
		}
		sess.RefreshTokenHash = newHash
		sess.LastUsedAt = now
		s.sessions[id] = sess
		return sess, nil
	}
	return identity.Session{}, identity.ErrUnauthenticated
}

func (s *fakeStore) TouchSession(_ context.Context, sessionID string, now, idleCutoff time.Time) (identity.Session, error) {
	if s.sessionsRevoked {
		return identity.Session{}, identity.ErrUnauthenticated
	}
	sess, ok := s.sessions[sessionID]
	if !ok || !sess.RevokedAt.IsZero() || !sess.ExpiresAt.After(now) {
		return identity.Session{}, identity.ErrUnauthenticated
	}
	lastUsed := sess.LastUsedAt
	if lastUsed.IsZero() {
		lastUsed = sess.CreatedAt
	}
	if lastUsed.Before(idleCutoff) {
		return identity.Session{}, identity.ErrUnauthenticated
	}
	sess.LastUsedAt = now
	s.sessions[sessionID] = sess
	return sess, nil
}

func (s *fakeStore) RevokeSessionByID(_ context.Context, sessionID, userID string) error {
	sess, ok := s.sessions[sessionID]
	if !ok || sess.UserID != userID || !sess.RevokedAt.IsZero() {
		return identity.ErrSessionNotFound
	}
	sess.RevokedAt = time.Now().UTC()
	s.sessions[sessionID] = sess
	return nil
}

func (s *fakeStore) ListActiveSessions(_ context.Context, userID string, now, idleCutoff time.Time) ([]identity.Session, error) {
	if s.sessionsRevoked {
		return []identity.Session{}, nil
	}
	out := []identity.Session{}
	for _, sess := range s.sessions {
		lastUsed := sess.LastUsedAt
		if lastUsed.IsZero() {
			lastUsed = sess.CreatedAt
		}
		if sess.UserID == userID && sess.RevokedAt.IsZero() && sess.ExpiresAt.After(now) && !lastUsed.Before(idleCutoff) {
			out = append(out, sess)
		}
	}
	return out, nil
}
