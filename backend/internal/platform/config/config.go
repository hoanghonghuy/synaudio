package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const (
	EnvDevelopment = "development"
	EnvProduction  = "production"
)

type Config struct {
	AppEnv             string
	HTTPAddr           string
	DatabaseURL        string
	StorageProvider    string
	StorageEndpoint    string
	StorageBucket      string
	StorageAccessKey   string
	StorageSecretKey   string
	AIMode             string
	TTSMode            string
	AppPublicURL       string
	APIPublicURL       string
	CORSAllowedOrigins []string

	AllowRemoteDatabaseInDev bool
	AllowRemoteStorageInDev  bool
}

func Load() (Config, error) {
	cfg := Config{
		AppEnv:                   strings.ToLower(strings.TrimSpace(getenv("APP_ENV", EnvDevelopment))),
		HTTPAddr:                 getenv("HTTP_ADDR", ":8080"),
		DatabaseURL:              strings.TrimSpace(os.Getenv("DATABASE_URL")),
		StorageProvider:          strings.ToLower(strings.TrimSpace(getenv("STORAGE_PROVIDER", "minio"))),
		StorageEndpoint:          strings.TrimSpace(os.Getenv("STORAGE_ENDPOINT")),
		StorageBucket:            strings.TrimSpace(os.Getenv("STORAGE_BUCKET")),
		StorageAccessKey:         strings.TrimSpace(os.Getenv("STORAGE_ACCESS_KEY")),
		StorageSecretKey:         strings.TrimSpace(os.Getenv("STORAGE_SECRET_KEY")),
		AIMode:                   strings.ToLower(strings.TrimSpace(getenv("AI_MODE", "mock"))),
		TTSMode:                  strings.ToLower(strings.TrimSpace(getenv("TTS_MODE", "mock"))),
		AppPublicURL:             strings.TrimSpace(os.Getenv("APP_PUBLIC_URL")),
		APIPublicURL:             strings.TrimSpace(os.Getenv("API_PUBLIC_URL")),
		CORSAllowedOrigins:       splitCSV(os.Getenv("CORS_ALLOWED_ORIGINS")),
		AllowRemoteDatabaseInDev: getenvBool("ALLOW_REMOTE_DATABASE_IN_DEV", false),
		AllowRemoteStorageInDev:  getenvBool("ALLOW_REMOTE_STORAGE_IN_DEV", false),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.StorageEndpoint == "" || cfg.StorageBucket == "" {
		return Config{}, fmt.Errorf("STORAGE_ENDPOINT and STORAGE_BUCKET are required")
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) validate() error {
	if c.AppEnv == EnvDevelopment {
		if isRemoteDatabaseURL(c.DatabaseURL) && !c.AllowRemoteDatabaseInDev {
			return fmt.Errorf("remote DATABASE_URL is blocked in development; set ALLOW_REMOTE_DATABASE_IN_DEV=true to override")
		}
		if isRemoteStorage(c) && !c.AllowRemoteStorageInDev {
			return fmt.Errorf("remote STORAGE endpoint/provider is blocked in development; set ALLOW_REMOTE_STORAGE_IN_DEV=true to override")
		}
	}

	if c.AppEnv == EnvProduction {
		if c.AIMode == "mock" || c.TTSMode == "mock" {
			return fmt.Errorf("mock AI_MODE/TTS_MODE are not allowed in production")
		}
		if c.StorageProvider == "minio" {
			return fmt.Errorf("local-only storage provider minio is not allowed in production")
		}
		if c.AllowRemoteDatabaseInDev || c.AllowRemoteStorageInDev {
			return fmt.Errorf("development safety overrides are not allowed in production")
		}
	}

	return nil
}

func isRemoteDatabaseURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return true
	}
	host := u.Hostname()
	if host == "" {
		return true
	}
	return !isLocalHost(host)
}

func isRemoteStorage(c Config) bool {
	if c.StorageProvider == "r2" {
		return true
	}
	u, err := url.Parse(c.StorageEndpoint)
	if err != nil {
		return true
	}
	host := u.Hostname()
	if host == "" {
		return true
	}
	return !isLocalHost(host)
}

func isLocalHost(host string) bool {
	h := strings.ToLower(host)
	switch h {
	case "localhost", "127.0.0.1", "::1", "postgres", "minio", "db", "host.docker.internal":
		return true
	}

	ip := net.ParseIP(h)
	if ip != nil {
		return ip.IsLoopback() || ip.IsPrivate()
	}

	return false
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func getenvBool(key string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return v
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
