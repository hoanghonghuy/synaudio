package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	accessTokenKeyringLimit = 4
	developmentAccessTokenKeyID = "development"
)

// AccessTokenKeyringConfig is the production-facing access-token signing contract.
// Keys contains the active signer plus any explicitly retained verification keys.
type AccessTokenKeyringConfig struct {
	ActiveKeyID string
	Keys        map[string]string
	MaxTTL      time.Duration
}

// LoadAccessTokenKeyring loads the bounded access-token keyring independently of
// the legacy single-secret field. Production never falls back to the development
// key; development may use the already-validated local legacy secret for zero-setup
// ergonomics while still exercising kid-based tokens.
func LoadAccessTokenKeyring(appEnv, developmentSecret string, accessTTL time.Duration) (AccessTokenKeyringConfig, error) {
	maxTTL, err := getenvDuration("ACCESS_TOKEN_MAX_TTL", accessTTL)
	if err != nil {
		return AccessTokenKeyringConfig{}, err
	}
	if maxTTL <= 0 || accessTTL <= 0 || accessTTL > maxTTL {
		return AccessTokenKeyringConfig{}, fmt.Errorf("ACCESS_TOKEN_MAX_TTL must be positive and no less than ACCESS_TOKEN_TTL")
	}

	activeKeyID := strings.TrimSpace(os.Getenv("ACCESS_TOKEN_ACTIVE_KID"))
	rawKeys := strings.TrimSpace(os.Getenv("ACCESS_TOKEN_KEYS"))
	if activeKeyID == "" && rawKeys == "" && appEnv == EnvDevelopment {
		if len(developmentSecret) < 32 {
			return AccessTokenKeyringConfig{}, fmt.Errorf("development access-token secret must be at least 32 bytes")
		}
		return AccessTokenKeyringConfig{
			ActiveKeyID: developmentAccessTokenKeyID,
			Keys: map[string]string{developmentAccessTokenKeyID: developmentSecret},
			MaxTTL: maxTTL,
		}, nil
	}
	if activeKeyID == "" || rawKeys == "" {
		return AccessTokenKeyringConfig{}, fmt.Errorf("ACCESS_TOKEN_ACTIVE_KID and ACCESS_TOKEN_KEYS are required together")
	}

	keys, err := parseAccessTokenKeys(rawKeys)
	if err != nil {
		return AccessTokenKeyringConfig{}, err
	}
	if err := validateAccessTokenKeyID(activeKeyID); err != nil {
		return AccessTokenKeyringConfig{}, fmt.Errorf("ACCESS_TOKEN_ACTIVE_KID: %w", err)
	}
	if _, ok := keys[activeKeyID]; !ok {
		return AccessTokenKeyringConfig{}, fmt.Errorf("ACCESS_TOKEN_ACTIVE_KID must identify a configured key")
	}

	return AccessTokenKeyringConfig{ActiveKeyID: activeKeyID, Keys: keys, MaxTTL: maxTTL}, nil
}

func parseAccessTokenKeys(raw string) (map[string]string, error) {
	parts := strings.Split(raw, ",")
	if len(parts) == 0 || len(parts) > accessTokenKeyringLimit {
		return nil, fmt.Errorf("ACCESS_TOKEN_KEYS must contain 1 to %d keys", accessTokenKeyringLimit)
	}
	keys := make(map[string]string, len(parts))
	for _, part := range parts {
		id, secret, ok := strings.Cut(strings.TrimSpace(part), "=")
		id = strings.TrimSpace(id)
		if !ok || id == "" || secret == "" {
			return nil, fmt.Errorf("ACCESS_TOKEN_KEYS entries must use kid=secret")
		}
		if err := validateAccessTokenKeyID(id); err != nil {
			return nil, fmt.Errorf("ACCESS_TOKEN_KEYS: %w", err)
		}
		if len(secret) < 32 {
			return nil, fmt.Errorf("ACCESS_TOKEN_KEYS contains a key shorter than 32 bytes")
		}
		if _, exists := keys[id]; exists {
			return nil, fmt.Errorf("ACCESS_TOKEN_KEYS contains a duplicate key id")
		}
		keys[id] = secret
	}
	return keys, nil
}

func validateAccessTokenKeyID(id string) error {
	if id == "" || len(id) > 64 || strings.ContainsAny(id, ".,= ") {
		return fmt.Errorf("access-token key id is invalid")
	}
	return nil
}
