package identity

import (
	"context"
)

// recoveryCredentialStore is the production-capable persistence extension for
// one-active, single-use recovery credentials. Keeping it separate from Store
// avoids forcing unrelated test doubles to implement PostgreSQL atomicity while
// allowing the production store to enforce the stronger contract.
type recoveryCredentialStore interface {
	ReplaceVerificationToken(ctx context.Context, userID, tokenHash string) error
	ReplaceResetToken(ctx context.Context, userID, tokenHash string) error
	ConsumeVerificationToken(ctx context.Context, userID, tokenHash string) error
	ResetPasswordAtomic(ctx context.Context, userID, tokenHash, passwordHash string) error
}

// RequestEmailVerification creates a one-time hashed verification token.
func (s *AuthService) RequestEmailVerification(ctx context.Context, userID string) (string, error) {
	if _, err := s.store.GetUserByID(ctx, userID); err != nil {
		return "", err
	}

	raw, err := NewRefreshToken()
	if err != nil {
		return "", err
	}
	hash := HashToken(raw)

	if recoveryStore, ok := s.store.(recoveryCredentialStore); ok {
		if err := recoveryStore.ReplaceVerificationToken(ctx, userID, hash); err != nil {
			return "", err
		}
	} else if err := s.store.StoreVerificationToken(ctx, userID, hash); err != nil {
		return "", err
	}

	return raw, nil
}

// VerifyEmail validates the latest token and atomically consumes it with the
// verified-email transition when the persistence adapter supports the V1
// recovery contract.
func (s *AuthService) VerifyEmail(ctx context.Context, userID, token string) error {
	stored, err := s.store.GetVerificationToken(ctx, userID)
	if err != nil {
		return ErrInvalidToken
	}
	if !VerifyTokenHash(stored, token) {
		return ErrInvalidToken
	}
	if recoveryStore, ok := s.store.(recoveryCredentialStore); ok {
		return recoveryStore.ConsumeVerificationToken(ctx, userID, stored)
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
	hash := HashToken(raw)

	if recoveryStore, ok := s.store.(recoveryCredentialStore); ok {
		if err := recoveryStore.ReplaceResetToken(ctx, userID, hash); err != nil {
			return "", err
		}
	} else if err := s.store.StoreResetToken(ctx, userID, hash); err != nil {
		return "", err
	}

	return raw, nil
}

// ResetPassword validates the latest token. Production persistence consumes the
// credential, updates the password and revokes sessions atomically so replay or
// a mid-transition database failure cannot leave ambiguous recovery state.
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

	if recoveryStore, ok := s.store.(recoveryCredentialStore); ok {
		return recoveryStore.ResetPasswordAtomic(ctx, userID, stored, hash)
	}

	if err := s.store.UpdatePassword(ctx, userID, hash); err != nil {
		return err
	}
	if err := s.store.RevokeSessions(ctx, userID); err != nil {
		return err
	}
	return nil
}
