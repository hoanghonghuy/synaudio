package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadAccessTokenKeyringDevelopmentFallback(t *testing.T) {
	t.Setenv("ACCESS_TOKEN_ACTIVE_KID", "")
	t.Setenv("ACCESS_TOKEN_KEYS", "")
	t.Setenv("ACCESS_TOKEN_MAX_TTL", "")

	secret := strings.Repeat("d", 40)
	cfg, err := LoadAccessTokenKeyring(EnvDevelopment, secret, 15*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ActiveKeyID != developmentAccessTokenKeyID || cfg.Keys[cfg.ActiveKeyID] != secret {
		t.Fatalf("unexpected development keyring: %#v", cfg)
	}
	if cfg.MaxTTL != 15*time.Minute {
		t.Fatalf("unexpected max ttl: %s", cfg.MaxTTL)
	}
}

func TestLoadAccessTokenKeyringProductionRequiresExplicitKeyring(t *testing.T) {
	t.Setenv("ACCESS_TOKEN_ACTIVE_KID", "")
	t.Setenv("ACCESS_TOKEN_KEYS", "")
	if _, err := LoadAccessTokenKeyring(EnvProduction, strings.Repeat("x", 40), 15*time.Minute); err == nil {
		t.Fatal("production must not fall back to the development secret")
	}
}

func TestLoadAccessTokenKeyringAcceptsRotationOverlap(t *testing.T) {
	t.Setenv("ACCESS_TOKEN_ACTIVE_KID", "new")
	t.Setenv("ACCESS_TOKEN_KEYS", "old="+strings.Repeat("o", 40)+",new="+strings.Repeat("n", 40))
	t.Setenv("ACCESS_TOKEN_MAX_TTL", "20m")

	cfg, err := LoadAccessTokenKeyring(EnvProduction, "", 15*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ActiveKeyID != "new" || len(cfg.Keys) != 2 || cfg.MaxTTL != 20*time.Minute {
		t.Fatalf("unexpected keyring: %#v", cfg)
	}
}

func TestLoadAccessTokenKeyringRejectsMalformedAmbiguousAndWeakConfig(t *testing.T) {
	cases := []struct {
		name   string
		active string
		keys   string
	}{
		{name: "missing active", active: "missing", keys: "known=" + strings.Repeat("k", 40)},
		{name: "duplicate id", active: "key", keys: "key=" + strings.Repeat("a", 40) + ",key=" + strings.Repeat("b", 40)},
		{name: "weak secret", active: "key", keys: "key=short"},
		{name: "malformed entry", active: "key", keys: "key"},
		{name: "invalid id", active: "bad.id", keys: "bad.id=" + strings.Repeat("a", 40)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("ACCESS_TOKEN_ACTIVE_KID", tc.active)
			t.Setenv("ACCESS_TOKEN_KEYS", tc.keys)
			if _, err := LoadAccessTokenKeyring(EnvProduction, "", 15*time.Minute); err == nil {
				t.Fatal("expected invalid keyring to fail")
			}
		})
	}
}

func TestLoadAccessTokenKeyringRejectsTTLAboveMaximum(t *testing.T) {
	t.Setenv("ACCESS_TOKEN_ACTIVE_KID", "key")
	t.Setenv("ACCESS_TOKEN_KEYS", "key="+strings.Repeat("a", 40))
	t.Setenv("ACCESS_TOKEN_MAX_TTL", "10m")
	if _, err := LoadAccessTokenKeyring(EnvProduction, "", 15*time.Minute); err == nil {
		t.Fatal("ACCESS_TOKEN_TTL above maximum accepted lifetime must fail")
	}
}
