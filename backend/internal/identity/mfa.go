package identity

import (
	"context"
	"errors"
	"time"
)

// MFAMethod represents a user's TOTP MFA method.
type MFAMethod struct {
	Secret    string
	Confirmed bool
	Disabled  bool
}

type mfaSecurityStore interface {
	ReplaceMFARecoveryCodes(ctx context.Context, userID string, codeHashes []string) error
	ConfirmMFAWithRecoveryCodesAndSession(ctx context.Context, userID, sessionID string, codeHashes []string, at time.Time) error
	ConsumeMFARecoveryCode(ctx context.Context, userID, codeHash string) (bool, error)
	MarkSessionMFAAndRecentAuth(ctx context.Context, userID, sessionID string, at time.Time) error
	HasPrivilegedSessionAssurance(ctx context.Context, userID, sessionID string, now time.Time) (bool, error)
	HasRecentAuth(ctx context.Context, userID, sessionID string, cutoff time.Time) (bool, error)
}

// SetupTOTP generates a new TOTP secret and stores it unconfirmed.
func (s *AuthService) SetupTOTP(ctx context.Context, userID string) (string, error) {
	if _, err := s.store.GetUserByID(ctx, userID); err != nil {
		return "", err
	}

	secret, err := GenerateTOTPSecret()
	if err != nil {
		return "", err
	}

	// The secret is stored encrypted at rest by the persistence layer.
	if err := s.store.StoreMFAMethod(ctx, userID, MFAMethod{Secret: secret}); err != nil {
		return "", err
	}

	return secret, nil
}

// ConfirmTOTP validates the code and atomically confirms the MFA method,
// rotates durable recovery-code hashes, and establishes MFA/recent-auth
// assurance on the exact authenticated session. Plaintext recovery codes are
// returned only after that complete persistence outcome commits.
func (s *AuthService) ConfirmTOTP(ctx context.Context, principal Principal, code string, counter uint64) ([]string, error) {
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
	securityStore, ok := s.store.(mfaSecurityStore)
	if !ok {
		return nil, errors.New("privileged security persistence not configured")
	}
	if err := securityStore.ConfirmMFAWithRecoveryCodesAndSession(
		ctx,
		principal.UserID,
		principal.SessionID,
		hashes,
		s.settings.Now().UTC(),
	); err != nil {
		return nil, err
	}
	return codes, nil
}

// MarkSessionMFAAndRecentAuth binds successful MFA proof to the exact logical
// session. This remains available for other MFA challenge flows such as a
// bounded privileged-login/re-auth challenge; TOTP setup confirmation itself
// uses the atomic confirmation method above.
func (s *AuthService) MarkSessionMFAAndRecentAuth(ctx context.Context, principal Principal) error {
	securityStore, ok := s.store.(mfaSecurityStore)
	if !ok {
		return errors.New("privileged security persistence not configured")
	}
	return securityStore.MarkSessionMFAAndRecentAuth(ctx, principal.UserID, principal.SessionID, s.settings.Now().UTC())
}

// ConsumeRecoveryCode validates one durable hashed MFA recovery credential.
// It is intentionally single-use at the persistence layer.
func (s *AuthService) ConsumeRecoveryCode(ctx context.Context, userID, rawCode string) (bool, error) {
	securityStore, ok := s.store.(mfaSecurityStore)
	if !ok {
		return false, errors.New("privileged security persistence not configured")
	}
	return securityStore.ConsumeMFARecoveryCode(ctx, userID, HashToken(rawCode))
}

// DisableTOTP marks the MFA method disabled.
func (s *AuthService) DisableTOTP(ctx context.Context, userID string) error {
	return s.store.DisableMFAMethod(ctx, userID)
}
