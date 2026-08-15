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

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected mock AI/TTS in production to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "mock") {
		t.Fatalf("expected mock provider rejection, got %v", err)
	}
}
