package identity_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/synaudio/synaudio/backend/internal/identity"
)

func (s *privilegedSecurityFakeStore) ConfirmMFAWithRecoveryCodesAndSession(ctx context.Context, userID, sessionID string, hashes []string, at time.Time) error {
	if s.confirmAtomicErr != nil {
		return s.confirmAtomicErr
	}
	sess, ok := s.sessions[sessionID]
	if !ok || sess.UserID != userID {
		return identity.ErrUnauthenticated
	}
	if err := s.fakeStore.ConfirmMFAMethod(ctx, userID); err != nil {
		return err
	}
	if err := s.ReplaceMFARecoveryCodes(ctx, userID, hashes); err != nil {
		return err
	}
	s.assuredSessions[sessionID] = at
	s.recentSessions[sessionID] = at
	return nil
}

func TestConfirmTOTPForSessionDoesNotExposePartialConfirmation(t *testing.T) {
	store := newPrivilegedSecurityFakeStore()
	svc := identity.NewAuthService(store)
	u, err := svc.Register(context.Background(), "atomic-admin@example.com", "correct password")
	if err != nil {
		t.Fatal(err)
	}
	sess, err := svc.Login(context.Background(), u.Email, "correct password")
	if err != nil {
		t.Fatal(err)
	}
	secret, err := svc.SetupTOTP(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	store.recoveryHashes["existing-hash"] = true
	store.confirmAtomicErr = errors.New("session assurance write failed")
	counter := identity.TOTPTimeStep(0)
	code, err := identity.TOTPCode(secret, counter)
	if err != nil {
		t.Fatal(err)
	}

	codes, err := svc.ConfirmTOTPForSession(context.Background(), identity.Principal{UserID: u.ID, SessionID: sess.ID}, code, counter)
	if err == nil {
		t.Fatal("atomic session confirmation failure must be returned")
	}
	if len(codes) != 0 {
		t.Fatal("plaintext recovery codes must not be exposed before atomic commit")
	}
	if len(store.recoveryHashes) != 1 || !store.recoveryHashes["existing-hash"] {
		t.Fatal("failed session confirmation must preserve prior recovery credentials")
	}
	method, err := store.GetMFAMethod(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if method.Confirmed {
		t.Fatal("failed session confirmation must not confirm MFA")
	}
	if _, ok := store.assuredSessions[sess.ID]; ok {
		t.Fatal("failed session confirmation must not mark MFA assurance")
	}
	if _, ok := store.recentSessions[sess.ID]; ok {
		t.Fatal("failed session confirmation must not mark recent-auth assurance")
	}
}
