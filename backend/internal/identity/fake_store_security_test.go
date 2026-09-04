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
func (s *fakeStore) ReplaceMFARecoveryCodes(_ context.Context, _ string, _ []string) error {
	return nil
}

func (s *fakeStore) ConfirmMFAWithRecoveryCodes(ctx context.Context, userID string, hashes []string) error {
	if err := s.ConfirmMFAMethod(ctx, userID); err != nil {
		return err
	}
	return s.ReplaceMFARecoveryCodes(ctx, userID, hashes)
}

func (s *fakeStore) ConsumeMFARecoveryCode(_ context.Context, _ string, _ string) (bool, error) {
	return false, nil
}

func (s *fakeStore) MarkSessionMFAAndRecentAuth(_ context.Context, userID, sessionID string, _ time.Time) error {
	sess, ok := s.sessions[sessionID]
	if !ok || sess.UserID != userID {
		return identity.ErrUnauthenticated
	}
	return nil
}

func (s *fakeStore) HasPrivilegedSessionAssurance(_ context.Context, _ string, _ string, _ time.Time) (bool, error) {
	return false, nil
}

func (s *fakeStore) HasRecentAuth(_ context.Context, _ string, _ string, _ time.Time) (bool, error) {
	return false, nil
}
