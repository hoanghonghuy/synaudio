package identity

import (
	"context"
)

// MFAMethod represents a user's TOTP MFA method.
type MFAMethod struct {
	Secret    string
	Confirmed bool
	Disabled  bool
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

// ConfirmTOTP validates the code and marks the method confirmed, returning
// one-time recovery codes.
func (s *AuthService) ConfirmTOTP(ctx context.Context, userID, code string, counter uint64) ([]string, error) {
	method, err := s.store.GetMFAMethod(ctx, userID)
	if err != nil {
		return nil, err
	}

	if !ValidateTOTP(method.Secret, code, counter) {
		return nil, ErrInvalidToken
	}

	if err := s.store.ConfirmMFAMethod(ctx, userID); err != nil {
		return nil, err
	}

	codes, _, err := GenerateRecoveryCodes(8)
	if err != nil {
		return nil, err
	}

	return codes, nil
}

// DisableTOTP marks the MFA method disabled.
func (s *AuthService) DisableTOTP(ctx context.Context, userID string) error {
	return s.store.DisableMFAMethod(ctx, userID)
}
