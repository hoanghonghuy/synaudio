package config

import (
	"errors"
	"os"
	"strings"
)

const (
	AuthAbuseBackendMemory   = "memory"
	AuthAbuseBackendPostgres = "postgres"
)

type AuthAbuseConfig struct {
	Backend AuthAbuseBackend
}

type AuthAbuseBackend string

func LoadAuthAbuse(appEnv string) (AuthAbuseConfig, error) {
	backend := strings.ToLower(strings.TrimSpace(os.Getenv("AUTH_ABUSE_BACKEND")))
	if backend == "" {
		if strings.EqualFold(appEnv, EnvProduction) {
			backend = AuthAbuseBackendPostgres
		} else {
			backend = AuthAbuseBackendMemory
		}
	}
	switch backend {
	case AuthAbuseBackendMemory, AuthAbuseBackendPostgres:
		return AuthAbuseConfig{Backend: AuthAbuseBackend(backend)}, nil
	default:
		return AuthAbuseConfig{}, errors.New("AUTH_ABUSE_BACKEND must be memory or postgres")
	}
}

func LoadTrustedProxyCIDRs() (string, error) {
	raw := strings.TrimSpace(os.Getenv("TRUSTED_PROXY_CIDRS"))
	if raw == "" {
		return "", nil
	}
	return raw, nil
}

// AuthAbuseProductionRequiresSharedState documents the production contract.
func AuthAbuseProductionRequiresSharedState(appEnv string, backend AuthAbuseBackend) error {
	if strings.EqualFold(appEnv, EnvProduction) && backend == AuthAbuseBackendMemory {
		return errors.New("AUTH_ABUSE_BACKEND=memory is not allowed in production")
	}
	return nil
}
