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

// VerifyEmailByEmail resolves the user by email then verifies the token.
func (s *AuthService) VerifyEmailByEmail(ctx context.Context, email, token string) error {
	u, err := s.store.GetUserByEmail(ctx, NormalizeEmail(email))
	if err != nil {
		return ErrInvalidToken
	}
	return s.VerifyEmail(ctx, u.ID, token)
}

// ResendEmailVerification resolves the user by email and issues a new token.
func (s *AuthService) ResendEmailVerification(ctx context.Context, email string) (string, error) {
	u, err := s.store.GetUserByEmail(ctx, NormalizeEmail(email))
	if err != nil {
		return "", ErrUserNotFound
	}
	return s.RequestEmailVerification(ctx, u.ID)
}

// RequestPasswordResetByEmail resolves the user by email and issues a reset token.
func (s *AuthService) RequestPasswordResetByEmail(ctx context.Context, email string) (string, error) {
	u, err := s.store.GetUserByEmail(ctx, NormalizeEmail(email))
	if err != nil {
		return "", ErrUserNotFound
	}
	return s.RequestPasswordReset(ctx, u.ID)
}

// ResetPasswordByEmail resolves the user by email then resets the password.
func (s *AuthService) ResetPasswordByEmail(ctx context.Context, email, token, newPassword string) error {
	u, err := s.store.GetUserByEmail(ctx, NormalizeEmail(email))
	if err != nil {
		return ErrInvalidToken
	}
	return s.ResetPassword(ctx, u.ID, token, newPassword)
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
