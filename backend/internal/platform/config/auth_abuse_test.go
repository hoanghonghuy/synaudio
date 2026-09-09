package config_test

import (
	"strings"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/platform/config"
)

func TestLoadAuthAbuseDefaultsToMemoryInDevelopment(t *testing.T) {
	t.Setenv("AUTH_ABUSE_BACKEND", "")
	cfg, err := config.LoadAuthAbuse(config.EnvDevelopment)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Backend != config.AuthAbuseBackendMemory {
		t.Fatalf("expected memory backend, got %q", cfg.Backend)
	}
}

func TestLoadAuthAbuseDefaultsToPostgresInProduction(t *testing.T) {
	t.Setenv("AUTH_ABUSE_BACKEND", "")
	cfg, err := config.LoadAuthAbuse(config.EnvProduction)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Backend != config.AuthAbuseBackendPostgres {
		t.Fatalf("expected postgres backend, got %q", cfg.Backend)
	}
}

func TestAuthAbuseProductionRejectsMemoryBackend(t *testing.T) {
	err := config.AuthAbuseProductionRequiresSharedState(config.EnvProduction, config.AuthAbuseBackendMemory)
	if err == nil {
		t.Fatal("expected production memory backend to be rejected")
	}
}

func TestValidateTrustedProxyCIDRsRejectsInvalidValue(t *testing.T) {
	err := config.ValidateTrustedProxyCIDRs("not-a-cidr")
	if err == nil || !strings.Contains(err.Error(), "TRUSTED_PROXY_CIDRS") {
		t.Fatalf("expected invalid CIDR error, got %v", err)
	}
}
