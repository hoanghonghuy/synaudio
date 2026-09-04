package identity

import (
	"context"
	"errors"
	"time"
)

type MFAMethod struct {
	Secret    string
	Confirmed bool
	Disabled  bool
}

type mfaSecurityStore interface {
	ReplaceMFARecoveryCodes(ctx context.Context, userID string, codeHashes []string) error
	ConfirmMFAWithRecoveryCodes(ctx context.Context, userID string, codeHashes []string) error
	ConsumeMFARecoveryCode(ctx context.Context, userID, codeHash string) (bool, error)
	MarkSessionMFAAndRecentAuth(ctx context.Context, userID, sessionID string, at time.Time) error
	HasPrivilegedSessionAssurance(ctx context.Context, userID, sessionID string, now time.Time) (bool, error)
	HasRecentAuth(ctx context.Context, userID, sessionID string, cutoff time.Time) (bool, error)
}

type mfaSessionConfirmationStore interface {
	ConfirmMFAWithRecoveryCodesAndSession(ctx context.Context, userID, sessionID string, codeHashes []string, at time.Time) error
}

// mfaDisableGuardStore owns MFA removal for accounts that may carry ADMIN.
// Implementations must serialize the Last Active Admin decision with disabling
// the MFA method so concurrent privileged transitions cannot leave the final
// ACTIVE administrator without mandatory MFA.
type mfaDisableGuardStore interface {
	DisableMFAMethodSafely(ctx context.Context, userID string) error
}

func (s *AuthService) SetupTOTP(ctx context.Context, userID string) (string, error) {
	if _, err := s.store.GetUserByID(ctx, userID); err != nil {
		return "", err
	}
	secret, err := GenerateTOTPSecret()
	if err != nil {
		return "", err
	}
	if err := s.store.StoreMFAMethod(ctx, userID, MFAMethod{Secret: secret}); err != nil {
		return "", err
	}
	return secret, nil
}

func (s *AuthService) ConfirmTOTP(ctx context.Context, userID, code string, counter uint64) ([]string, error) {
	method, err := s.store.GetMFAMethod(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !ValidateTOTP(method.Secret, code, counter) {
		return nil, ErrInvalidToken
	}
	codes, hashes, err := GenerateRecoveryCodes(8)
	if err != nil {
		return nil, err
	}
	securityStore, ok := s.store.(mfaSecurityStore)
	if !ok {
		return nil, errors.New("privileged security persistence not configured")
	}
	if err := securityStore.ConfirmMFAWithRecoveryCodes(ctx, userID, hashes); err != nil {
		return nil, err
	}
	return codes, nil
}

// ConfirmTOTPForSession is the HTTP security path: confirmation, hashed
// recovery-code rotation, and exact-session MFA/recent-auth assurance commit as
// one transaction before plaintext recovery codes can be returned.
func (s *AuthService) ConfirmTOTPForSession(ctx context.Context, principal Principal, code string, counter uint64) ([]string, error) {
	if principal.UserID == "" || principal.SessionID == "" {
		return nil, ErrUnauthenticated
	}
	method, err := s.store.GetMFAMethod(ctx, principal.UserID)
	if err != nil {
		return nil, err
	}
	if !ValidateTOTP(method.Secret, code, counter) {
		return nil, ErrInvalidToken
	}
	codes, hashes, err := GenerateRecoveryCodes(8)
	if err != nil {
		return nil, err
	}
	store, ok := s.store.(mfaSessionConfirmationStore)
	if !ok {
		return nil, errors.New("atomic session MFA persistence not configured")
	}
	if err := store.ConfirmMFAWithRecoveryCodesAndSession(ctx, principal.UserID, principal.SessionID, hashes, s.settings.Now().UTC()); err != nil {
		return nil, err
	}
	return codes, nil
}

func (s *AuthService) MarkSessionMFAAndRecentAuth(ctx context.Context, principal Principal) error {
	securityStore, ok := s.store.(mfaSecurityStore)
	if !ok {
		return errors.New("privileged security persistence not configured")
	}
	return securityStore.MarkSessionMFAAndRecentAuth(ctx, principal.UserID, principal.SessionID, s.settings.Now().UTC())
}

func (s *AuthService) ConsumeRecoveryCode(ctx context.Context, userID, rawCode string) (bool, error) {
	securityStore, ok := s.store.(mfaSecurityStore)
	if !ok {
		return false, errors.New("privileged security persistence not configured")
	}
	return securityStore.ConsumeMFARecoveryCode(ctx, userID, HashToken(rawCode))
}

func (s *AuthService) DisableTOTP(ctx context.Context, userID string) error {
	guardStore, ok := s.store.(mfaDisableGuardStore)
	if !ok {
		return errors.New("atomic MFA removal guard persistence not configured")
	}
	return guardStore.DisableMFAMethodSafely(ctx, userID)
}
