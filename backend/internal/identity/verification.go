package identity

import (
	"context"
)

// RequestEmailVerification creates a one-time hashed verification token.
func (s *AuthService) RequestEmailVerification(ctx context.Context, userID string) (string, error) {
	if _, err := s.store.GetUserByID(ctx, userID); err != nil {
		return "", err
	}

	raw, err := NewRefreshToken()
	if err != nil {
		return "", err
	}

	if err := s.store.StoreVerificationToken(ctx, userID, HashToken(raw)); err != nil {
		return "", err
	}

	return raw, nil
}

// VerifyEmail validates the token and marks the account email as verified.
func (s *AuthService) VerifyEmail(ctx context.Context, userID, token string) error {
	stored, err := s.store.GetVerificationToken(ctx, userID)
	if err != nil {
		return ErrInvalidToken
	}
	if !VerifyTokenHash(stored, token) {
		return ErrInvalidToken
	}
	return s.store.MarkEmailVerified(ctx, userID)
}

// RequestPasswordReset creates a one-time hashed reset token.
func (s *AuthService) RequestPasswordReset(ctx context.Context, userID string) (string, error) {
	if _, err := s.store.GetUserByID(ctx, userID); err != nil {
		return "", err
	}

	raw, err := NewRefreshToken()
	if err != nil {
		return "", err
	}

	if err := s.store.StoreResetToken(ctx, userID, HashToken(raw)); err != nil {
		return "", err
	}

	return raw, nil
}

// ResetPassword validates the token, updates the password hash, and revokes
// all existing sessions by default.
func (s *AuthService) ResetPassword(ctx context.Context, userID, token, newPassword string) error {
	stored, err := s.store.GetResetToken(ctx, userID)
	if err != nil {
		return ErrInvalidToken
	}
	if !VerifyTokenHash(stored, token) {
		return ErrInvalidToken
	}

	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}

	if err := s.store.UpdatePassword(ctx, userID, hash); err != nil {
		return err
	}

	if err := s.store.RevokeSessions(ctx, userID); err != nil {
		return err
	}

	return nil
}
