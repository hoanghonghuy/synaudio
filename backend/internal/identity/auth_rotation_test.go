package identity

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNewRotatingAuthServiceUsesConfiguredActiveSigner(t *testing.T) {
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	service, err := NewRotatingAuthService(nil, AuthSettings{
		AccessTokenTTL: 15 * time.Minute,
		Now:            func() time.Time { return now },
	}, "new", map[string]string{
		"old": strings.Repeat("o", 40),
		"new": strings.Repeat("n", 40),
	}, 20*time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	issued, err := service.IssueAccessToken(Session{ID: "session-1", UserID: "user-1"})
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(issued.Token, ".")
	if len(parts) != 3 {
		t.Fatalf("unexpected JWT shape: %d segments", len(parts))
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		t.Fatal(err)
	}
	var header struct {
		KeyID string `json:"kid"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		t.Fatal(err)
	}
	if header.KeyID != "new" {
		t.Fatalf("expected active signer kid new, got %q", header.KeyID)
	}
	if _, err := service.accessTokens.Parse(issued.Token); err != nil {
		t.Fatalf("configured verifier must accept token from active signer: %v", err)
	}
}

func TestNewRotatingAuthServiceAcceptsLegacyConstructorTokensDuringBootstrap(t *testing.T) {
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	secret := strings.Repeat("l", 40)
	legacy, err := NewAccessTokenManager(secret, 15*time.Minute, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	legacyToken, _, err := legacy.Issue("user-legacy", "session-legacy")
	if err != nil {
		t.Fatal(err)
	}

	service, err := NewRotatingAuthService(nil, AuthSettings{
		AccessTokenTTL: 15 * time.Minute,
		Now:            func() time.Time { return now },
	}, "legacy", map[string]string{
		"legacy": secret,
	}, 15*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := service.accessTokens.Parse(legacyToken)
	if err != nil {
		t.Fatalf("bootstrap keyring must accept tokens issued by the legacy constructor: %v", err)
	}
	if claims.Subject != "user-legacy" || claims.SessionID != "session-legacy" {
		t.Fatalf("unexpected legacy claims: %#v", claims)
	}
}

func TestNewRotatingAuthServiceRejectsInvalidKeyring(t *testing.T) {
	_, err := NewRotatingAuthService(nil, AuthSettings{AccessTokenTTL: 15 * time.Minute}, "missing", map[string]string{
		"known": strings.Repeat("k", 40),
	}, 15*time.Minute)
	if err == nil {
		t.Fatal("expected invalid active key to fail composition")
	}
}
