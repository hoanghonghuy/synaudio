package config_test

import (
	"strings"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/platform/config"
)

func TestLoadRejectsRemoteDatabaseInDevelopmentByDefault(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("DATABASE_URL", "postgres://user:pass@ep-prod-123.neon.tech/synaudio?sslmode=require")
	t.Setenv("STORAGE_PROVIDER", "minio")
	t.Setenv("STORAGE_ENDPOINT", "http://localhost:9000")
	t.Setenv("STORAGE_BUCKET", "synaudio")
	t.Setenv("STORAGE_ACCESS_KEY", "minio")
	t.Setenv("STORAGE_SECRET_KEY", "minio123")
	t.Setenv("ALLOW_REMOTE_DATABASE_IN_DEV", "false")
	t.Setenv("ALLOW_REMOTE_STORAGE_IN_DEV", "false")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected remote database in development to fail")
	}
	if !strings.Contains(err.Error(), "ALLOW_REMOTE_DATABASE_IN_DEV") {
		t.Fatalf("expected remote database guard message, got %v", err)
	}
}

func TestLoadAllowsLocalDatabaseInDevelopment(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("DATABASE_URL", "postgres://synaudio:synaudio@localhost:5432/synaudio?sslmode=disable")
	t.Setenv("STORAGE_PROVIDER", "minio")
	t.Setenv("STORAGE_ENDPOINT", "http://localhost:9000")
	t.Setenv("STORAGE_BUCKET", "synaudio")
	t.Setenv("STORAGE_ACCESS_KEY", "minio")
	t.Setenv("STORAGE_SECRET_KEY", "minio123")
	t.Setenv("ALLOW_REMOTE_DATABASE_IN_DEV", "false")
	t.Setenv("ALLOW_REMOTE_STORAGE_IN_DEV", "false")
	t.Setenv("AI_MODE", "mock")
	t.Setenv("TTS_MODE", "mock")
	t.Setenv("HTTP_ADDR", ":8080")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("expected local development config to load, got %v", err)
	}
	if cfg.AppEnv != "development" {
		t.Fatalf("expected development env, got %q", cfg.AppEnv)
	}
	if cfg.DatabaseURL == "" {
		t.Fatal("expected database URL to be set")
	}
	if cfg.AccessTokenTTL <= 0 || cfg.RefreshSessionIdleTTL <= 0 {
		t.Fatal("expected positive authentication lifetimes")
	}
}

func TestLoadRejectsRemoteStorageInDevelopmentByDefault(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("DATABASE_URL", "postgres://synaudio:synaudio@localhost:5432/synaudio?sslmode=disable")
	t.Setenv("STORAGE_PROVIDER", "r2")
	t.Setenv("STORAGE_ENDPOINT", "https://accountid.r2.cloudflarestorage.com")
	t.Setenv("STORAGE_BUCKET", "synaudio-prod")
	t.Setenv("STORAGE_ACCESS_KEY", "key")
	t.Setenv("STORAGE_SECRET_KEY", "secret")
	t.Setenv("ALLOW_REMOTE_DATABASE_IN_DEV", "false")
	t.Setenv("ALLOW_REMOTE_STORAGE_IN_DEV", "false")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected remote storage in development to fail")
	}
	if !strings.Contains(err.Error(), "ALLOW_REMOTE_STORAGE_IN_DEV") {
		t.Fatalf("expected remote storage guard message, got %v", err)
	}
}

func TestLoadRejectsMockProvidersInProduction(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("DATABASE_URL", "postgres://user:pass@ep-prod-123.neon.tech/synaudio?sslmode=require")
	t.Setenv("STORAGE_PROVIDER", "r2")
	t.Setenv("STORAGE_ENDPOINT", "https://accountid.r2.cloudflarestorage.com")
	t.Setenv("STORAGE_BUCKET", "synaudio-prod")
	t.Setenv("STORAGE_ACCESS_KEY", "key")
	t.Setenv("STORAGE_SECRET_KEY", "secret")
	t.Setenv("AI_MODE", "mock")
	t.Setenv("TTS_MODE", "mock")
	t.Setenv("APP_PUBLIC_URL", "https://app.example.com")
	t.Setenv("API_PUBLIC_URL", "https://api.example.com")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com")
	t.Setenv("ACCESS_TOKEN_SECRET", "production-test-access-token-secret-at-least-32-bytes")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected mock AI/TTS in production to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "mock") {
		t.Fatalf("expected mock provider rejection, got %v", err)
	}
}

func TestLoadProductionDoesNotRequireLegacyAccessTokenSecret(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("DATABASE_URL", "postgres://user:pass@ep-prod-123.neon.tech/synaudio?sslmode=require")
	t.Setenv("STORAGE_PROVIDER", "r2")
	t.Setenv("STORAGE_ENDPOINT", "https://accountid.r2.cloudflarestorage.com")
	t.Setenv("STORAGE_BUCKET", "synaudio-prod")
	t.Setenv("STORAGE_ACCESS_KEY", "key")
	t.Setenv("STORAGE_SECRET_KEY", "secret")
	t.Setenv("AI_MODE", "gemini")
	t.Setenv("TTS_MODE", "gemini")
	t.Setenv("ACCESS_TOKEN_SECRET", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("production base config must not depend on legacy ACCESS_TOKEN_SECRET: %v", err)
	}
	if cfg.AccessTokenSecret != "" {
		t.Fatalf("expected empty production compatibility secret, got %q", cfg.AccessTokenSecret)
	}
}

func TestLoadRejectsShortAccessTokenSecret(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("DATABASE_URL", "postgres://synaudio:synaudio@localhost:5432/synaudio?sslmode=disable")
	t.Setenv("STORAGE_PROVIDER", "minio")
	t.Setenv("STORAGE_ENDPOINT", "http://localhost:9000")
	t.Setenv("STORAGE_BUCKET", "synaudio")
	t.Setenv("ACCESS_TOKEN_SECRET", "too-short")

	_, err := config.Load()
	if err == nil || !strings.Contains(err.Error(), "ACCESS_TOKEN_SECRET") {
		t.Fatalf("expected short access token secret rejection, got %v", err)
	}
}

func TestAdminMFARequiredConfig(t *testing.T) {
	// Dev default: true
	t.Setenv("APP_ENV", "development")
	t.Setenv("DATABASE_URL", "postgres://synaudio:synaudio@localhost:5432/synaudio?sslmode=disable")
	t.Setenv("STORAGE_PROVIDER", "minio")
	t.Setenv("STORAGE_ENDPOINT", "http://localhost:9000")
	t.Setenv("STORAGE_BUCKET", "synaudio")
	t.Setenv("ADMIN_MFA_REQUIRED", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected dev load error: %v", err)
	}
	if !cfg.AdminMFARequired {
		t.Fatal("expected AdminMFARequired to default to true in development")
	}

	// Dev explicit false
	t.Setenv("ADMIN_MFA_REQUIRED", "false")
	cfg, err = config.Load()
	if err != nil {
		t.Fatalf("unexpected dev load error with ADMIN_MFA_REQUIRED=false: %v", err)
	}
	if cfg.AdminMFARequired {
		t.Fatal("expected AdminMFARequired to be false when configured false in development")
	}

	// Production explicit false -> must be rejected
	t.Setenv("APP_ENV", "production")
	t.Setenv("DATABASE_URL", "postgres://user:pass@ep-prod-123.neon.tech/synaudio?sslmode=require")
	t.Setenv("STORAGE_PROVIDER", "r2")
	t.Setenv("STORAGE_ENDPOINT", "https://accountid.r2.cloudflarestorage.com")
	t.Setenv("STORAGE_BUCKET", "synaudio-prod")
	t.Setenv("STORAGE_ACCESS_KEY", "key")
	t.Setenv("STORAGE_SECRET_KEY", "secret")
	t.Setenv("AI_MODE", "gemini")
	t.Setenv("TTS_MODE", "gemini")
	t.Setenv("ADMIN_MFA_REQUIRED", "false")

	_, err = config.Load()
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "mfa") {
		t.Fatalf("expected production to reject ADMIN_MFA_REQUIRED=false, got err: %v", err)
	}
}

func TestLoadOpenAIConfig(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("DATABASE_URL", "postgres://synaudio:synaudio@localhost:5432/synaudio?sslmode=disable")
	t.Setenv("STORAGE_PROVIDER", "minio")
	t.Setenv("STORAGE_ENDPOINT", "http://localhost:9000")
	t.Setenv("STORAGE_BUCKET", "synaudio")
	t.Setenv("STORAGE_ACCESS_KEY", "minio")
	t.Setenv("STORAGE_SECRET_KEY", "minio123")
	t.Setenv("ALLOW_REMOTE_DATABASE_IN_DEV", "false")
	t.Setenv("ALLOW_REMOTE_STORAGE_IN_DEV", "false")
	t.Setenv("AI_MODE", "openai")
	t.Setenv("OPENAI_BASE_URL", "https://yuhh-9router.duckdns.org/v1")
	t.Setenv("OPENAI_API_KEY", "sk-d21f2272f14871ca-gizpjq-e919334a")
	t.Setenv("OPENAI_MODEL", "code")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("expected config to load, got %v", err)
	}
	if cfg.AIMode != "openai" {
		t.Fatalf("expected AIMode=openai, got %q", cfg.AIMode)
	}
	if cfg.OpenAIBaseURL != "https://yuhh-9router.duckdns.org/v1" {
		t.Fatalf("expected OpenAIBaseURL, got %q", cfg.OpenAIBaseURL)
	}
	if cfg.OpenAIAPIKey != "sk-d21f2272f14871ca-gizpjq-e919334a" {
		t.Fatalf("expected OpenAIAPIKey, got %q", cfg.OpenAIAPIKey)
	}
	if cfg.OpenAIModel != "code" {
		t.Fatalf("expected OpenAIModel=code, got %q", cfg.OpenAIModel)
	}
}


