package config

import (
	"strings"
	"testing"
)

func TestNormalizeAppEnvAcceptsReviewedModes(t *testing.T) {
	cases := map[string]string{
		"development":   EnvDevelopment,
		" DEVELOPMENT ": EnvDevelopment,
		"production":    EnvProduction,
		" Production ":  EnvProduction,
	}
	for raw, want := range cases {
		t.Run(raw, func(t *testing.T) {
			got, err := normalizeAppEnv(raw)
			if err != nil {
				t.Fatal(err)
			}
			if got != want {
				t.Fatalf("normalizeAppEnv(%q) = %q, want %q", raw, got, want)
			}
		})
	}
}

func TestNormalizeAppEnvRejectsUnknownModes(t *testing.T) {
	for _, raw := range []string{"prodution", "prod", "staging", "test"} {
		t.Run(raw, func(t *testing.T) {
			if _, err := normalizeAppEnv(raw); err == nil {
				t.Fatalf("expected unknown APP_ENV %q to fail", raw)
			}
		})
	}
}

func TestLoadRejectsUnknownAppEnvBeforeOtherConfiguration(t *testing.T) {
	t.Setenv("APP_ENV", "prodution")
	cfg, err := Load()
	if err == nil {
		t.Fatalf("expected unknown APP_ENV to fail, got config %#v", cfg)
	}
	if !strings.Contains(err.Error(), "APP_ENV") {
		t.Fatalf("expected APP_ENV classification error, got %v", err)
	}
}
