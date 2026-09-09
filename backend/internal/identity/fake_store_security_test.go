package identity_test

import (
	"context"
	"time"

	"github.com/synaudio/synaudio/backend/internal/identity"
)

// The shared fakeStore predates privileged-security persistence. Keep the
// production service fail-closed while teaching the test double the newer
// persistence contract used by ConfirmTOTP. Security-specific tests use
// privilegedSecurityFakeStore, which overrides these methods with stateful
// assertions.
func (s *fakeStore) ReplaceMFARecoveryCodes(_ context.Context, _ string, hashes []string) error {
	s.recoveryHashes = map[string]bool{}
	for _, hash := range hashes {
		s.recoveryHashes[hash] = true
	}
	return nil
}

func (s *fakeStore) ConfirmMFAWithRecoveryCodes(ctx context.Context, userID string, hashes []string) error {
	if err := s.ConfirmMFAMethod(ctx, userID); err != nil {
		return err
	}
	return s.ReplaceMFARecoveryCodes(ctx, userID, hashes)
}

func (s *fakeStore) ConfirmMFAWithRecoveryCodesAndSession(ctx context.Context, userID, sessionID string, hashes []string, at time.Time) error {
	if err := s.ConfirmMFAWithRecoveryCodes(ctx, userID, hashes); err != nil {
		return err
	}
	return s.MarkSessionMFAAndRecentAuth(ctx, userID, sessionID, at)
}

func (s *fakeStore) ConsumeMFARecoveryCode(_ context.Context, _ string, hash string) (bool, error) {
	if s.recoveryHashes == nil || !s.recoveryHashes[hash] {
		return false, nil
	}
	delete(s.recoveryHashes, hash)
	return true, nil
}

func (s *fakeStore) MarkSessionMFAAndRecentAuth(_ context.Context, userID, sessionID string, at time.Time) error {
	sess, ok := s.sessions[sessionID]
	if !ok || sess.UserID != userID {
		return identity.ErrUnauthenticated
	}
	if s.assuredSessions == nil {
		s.assuredSessions = map[string]time.Time{}
	}
	if s.recentSessions == nil {
		s.recentSessions = map[string]time.Time{}
	}
	s.assuredSessions[sessionID] = at
	s.recentSessions[sessionID] = at
	return nil
}

func (s *fakeStore) HasPrivilegedSessionAssurance(_ context.Context, userID, sessionID string, _ time.Time) (bool, error) {
	sess, ok := s.sessions[sessionID]
	if !ok || sess.UserID != userID {
		return false, nil
	}
	_, ok = s.assuredSessions[sessionID]
	return ok, nil
}

func (s *fakeStore) HasRecentAuth(_ context.Context, userID, sessionID string, cutoff time.Time) (bool, error) {
	sess, ok := s.sessions[sessionID]
	if !ok || sess.UserID != userID {
		return false, nil
	}
	at, ok := s.recentSessions[sessionID]
	return ok && !at.Before(cutoff), nil
}

func (s *fakeStore) DisableMFAMethodSafely(ctx context.Context, userID string) error {
	return s.DisableMFAMethod(ctx, userID)
}

func (s *fakeStore) RevokeAdminRoleSafely(ctx context.Context, targetID string) error {
	count, err := s.CountActiveAdmins(ctx)
	if err != nil {
		return err
	}
	if count <= 1 && contains(s.userRoles[targetID], identity.RoleAdmin) {
		return identity.ErrLastAdmin
	}
	return s.RevokeRole(ctx, targetID, identity.RoleAdmin)
}
