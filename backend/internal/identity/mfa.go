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

// ConfirmTOTP validates the code, confirms the method, and replaces recovery
// credentials with hashes when the persistence adapter supports the V1 security
// contract. Plaintext recovery codes are returned only once to the caller.
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
	if securityStore, ok := s.store.(mfaSecurityStore); ok {
		if err := securityStore.ReplaceMFARecoveryCodes(ctx, userID, hashes); err != nil {
			return nil, err
		}
	}
	if err := s.store.ConfirmMFAMethod(ctx, userID); err != nil {
		return nil, err
	}

	return codes, nil
}

// MarkSessionMFAAndRecentAuth binds successful MFA proof to the exact logical
// session. This prevents a password-only or stale parallel session from gaining
// privileged capability merely because the user has MFA configured globally.
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
