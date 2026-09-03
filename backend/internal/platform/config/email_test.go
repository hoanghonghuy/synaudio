package config

import (
	"strings"
	"testing"
)

func clearEmailEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"EMAIL_MODE", "EMAIL_PAYLOAD_SECRET", "SMTP_HOST", "SMTP_PORT",
		"SMTP_USERNAME", "SMTP_PASSWORD", "SMTP_FROM", "SMTP_TIMEOUT",
	} {
		t.Setenv(key, "")
	}
}

func TestProductionEmailCannotSilentlyUseDisabledMode(t *testing.T) {
	clearEmailEnv(t)
	if _, err := LoadEmail(EnvProduction, "https://app.example.com"); err == nil || !strings.Contains(err.Error(), "EMAIL_MODE=smtp") {
		t.Fatalf("production disabled-mode error = %v, want EMAIL_MODE=smtp requirement", err)
	}
}

func TestSMTPEmailRequiresTrustedRuntimeConfiguration(t *testing.T) {
	clearEmailEnv(t)
	t.Setenv("EMAIL_MODE", "smtp")
	t.Setenv("EMAIL_PAYLOAD_SECRET", "test-email-payload-secret-that-is-long-enough")
	t.Setenv("SMTP_HOST", "smtp.example.com")
	t.Setenv("SMTP_FROM", "no-reply@example.com")

	cfg, err := LoadEmail("development", "https://trusted.example.com/")
	if err != nil {
		t.Fatalf("load smtp config: %v", err)
	}
	if cfg.AppPublicURL != "https://trusted.example.com" {
		t.Fatalf("trusted public URL = %q", cfg.AppPublicURL)
	}
	if cfg.Mode != EmailModeSMTP || cfg.SMTPPort != "587" {
		t.Fatalf("unexpected smtp config: mode=%q port=%q", cfg.Mode, cfg.SMTPPort)
	}
}
