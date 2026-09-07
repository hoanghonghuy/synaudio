package identity

import "time"

// NewRotatingAuthService composes AuthService with the production keyring-aware
// access-token manager while preserving the existing refresh/session settings.
// The returned error is a startup/configuration error and must not be ignored by
// production composition.
func NewRotatingAuthService(store Store, settings AuthSettings, activeKeyID string, keys map[string]string, maxTTL time.Duration) (*AuthService, error) {
	s := NewAuthService(store, WithAuthSettings(settings))
	manager, err := NewRotatingAccessTokenManager(activeKeyID, keys, s.settings.AccessTokenTTL, maxTTL, s.settings.Now)
	if err != nil {
		return nil, err
	}
	s.accessTokens = manager
	return s, nil
}
