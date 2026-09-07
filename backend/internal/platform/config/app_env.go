package config

import (
	"fmt"
	"strings"
)

func normalizeAppEnv(raw string) (string, error) {
	env := strings.ToLower(strings.TrimSpace(raw))
	switch env {
	case EnvDevelopment, EnvProduction:
		return env, nil
	default:
		return "", fmt.Errorf("APP_ENV must be one of %q or %q", EnvDevelopment, EnvProduction)
	}
}
