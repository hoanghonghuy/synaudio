package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
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
	GeminiAPIKey       string
	GeminiTextModel    string
	GeminiTTSModel     string
	GeminiTTSVoice     string
	AppPublicURL       string
	APIPublicURL       string
	CORSAllowedOrigins []string

	AccessTokenSecret     string
	AccessTokenTTL        time.Duration
	RefreshSessionTTL     time.Duration
	RefreshSessionIdleTTL time.Duration
	RecentAuthWindow      time.Duration

	AllowRemoteDatabaseInDev bool
	AllowRemoteStorageInDev  bool
}

func Load() (Config, error) {
	appEnv := strings.ToLower(strings.TrimSpace(getenv("APP_ENV", EnvDevelopment)))
	accessTokenSecret := strings.TrimSpace(os.Getenv("ACCESS_TOKEN_SECRET"))
	if accessTokenSecret == "" && appEnv == EnvDevelopment {
		// Development remains zero-setup while production is required to supply an
		// explicit secret. The value is intentionally environment-local, not a
		// production default.
		accessTokenSecret = "development-only-access-token-secret-change-me"
	}

	accessTokenTTL, err := getenvDuration("ACCESS_TOKEN_TTL", 15*time.Minute)
	if err != nil {
		return Config{}, err
	}
	refreshSessionTTL, err := getenvDuration("REFRESH_SESSION_TTL", 30*24*time.Hour)
	if err != nil {
		return Config{}, err
	}
	refreshSessionIdleTTL, err := getenvDuration("REFRESH_SESSION_IDLE_TTL", 7*24*time.Hour)
	if err != nil {
		return Config{}, err
	}
	recentAuthWindow, err := getenvDuration("RECENT_AUTH_WINDOW", 10*time.Minute)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		AppEnv:                   appEnv,
		HTTPAddr:                 getenv("HTTP_ADDR", ":8080"),
		DatabaseURL:              strings.TrimSpace(os.Getenv("DATABASE_URL")),
		StorageProvider:          strings.ToLower(strings.TrimSpace(getenv("STORAGE_PROVIDER", "minio"))),
		StorageEndpoint:          strings.TrimSpace(os.Getenv("STORAGE_ENDPOINT")),
		StorageBucket:            strings.TrimSpace(os.Getenv("STORAGE_BUCKET")),
		StorageAccessKey:         strings.TrimSpace(os.Getenv("STORAGE_ACCESS_KEY")),
		StorageSecretKey:         strings.TrimSpace(os.Getenv("STORAGE_SECRET_KEY")),
		AIMode:                   strings.ToLower(strings.TrimSpace(getenv("AI_MODE", "mock"))),
		TTSMode:                  strings.ToLower(strings.TrimSpace(getenv("TTS_MODE", "mock"))),
		GeminiAPIKey:             strings.TrimSpace(os.Getenv("GEMINI_API_KEY")),
		GeminiTextModel:          strings.TrimSpace(getenv("GEMINI_TEXT_MODEL", "gemini-3.7-flash")),
		GeminiTTSModel:           strings.TrimSpace(getenv("GEMINI_TTS_MODEL", "gemini-3.1-flash-tts-preview")),
		GeminiTTSVoice:           strings.TrimSpace(getenv("GEMINI_TTS_VOICE", "Kore")),
		AppPublicURL:             strings.TrimSpace(os.Getenv("APP_PUBLIC_URL")),
		APIPublicURL:             strings.TrimSpace(os.Getenv("API_PUBLIC_URL")),
		CORSAllowedOrigins:       splitCSV(os.Getenv("CORS_ALLOWED_ORIGINS")),
		AccessTokenSecret:        accessTokenSecret,
		AccessTokenTTL:           accessTokenTTL,
		RefreshSessionTTL:        refreshSessionTTL,
		RefreshSessionIdleTTL:    refreshSessionIdleTTL,
		RecentAuthWindow:         recentAuthWindow,
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
	if len(c.AccessTokenSecret) < 32 {
		return fmt.Errorf("ACCESS_TOKEN_SECRET must be at least 32 bytes")
	}
	if c.AccessTokenTTL <= 0 {
		return fmt.Errorf("ACCESS_TOKEN_TTL must be positive")
	}
	if c.RefreshSessionTTL <= 0 {
		return fmt.Errorf("REFRESH_SESSION_TTL must be positive")
	}
	if c.RefreshSessionIdleTTL <= 0 || c.RefreshSessionIdleTTL > c.RefreshSessionTTL {
		return fmt.Errorf("REFRESH_SESSION_IDLE_TTL must be positive and no greater than REFRESH_SESSION_TTL")
	}
	if c.RecentAuthWindow <= 0 {
		return fmt.Errorf("RECENT_AUTH_WINDOW must be positive")
	}

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

func getenvDuration(key string, fallback time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid duration: %w", key, err)
	}
	return value, nil
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
