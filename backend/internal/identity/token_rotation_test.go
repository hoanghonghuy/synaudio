package identity

import (
	"strings"
	"testing"
	"time"
)

func TestRotatingAccessTokenManagerAcceptsOldKeyDuringOverlap(t *testing.T) {
	now := time.Date(2026, 9, 7, 10, 0, 0, 0, time.UTC)
	oldSecret := strings.Repeat("o", 40)
	newSecret := strings.Repeat("n", 40)
	old, err := NewRotatingAccessTokenManager("old", map[string]string{"old": oldSecret}, 15*time.Minute, 15*time.Minute, func() time.Time { return now })
	if err != nil { t.Fatal(err) }
	token, _, err := old.Issue("user-1", "session-1"); if err != nil { t.Fatal(err) }
	overlap, err := NewRotatingAccessTokenManager("new", map[string]string{"old": oldSecret, "new": newSecret}, 15*time.Minute, 15*time.Minute, func() time.Time { return now })
	if err != nil { t.Fatal(err) }
	if _, err := overlap.Parse(token); err != nil { t.Fatalf("old token should verify during overlap: %v", err) }
	removed, err := NewRotatingAccessTokenManager("new", map[string]string{"new": newSecret}, 15*time.Minute, 15*time.Minute, func() time.Time { return now })
	if err != nil { t.Fatal(err) }
	if _, err := removed.Parse(token); err == nil { t.Fatal("removed old key must fail closed") }
}

func TestRotatingAccessTokenManagerNewTokenWorksAcrossMixedReplicaKeyring(t *testing.T) {
	now := time.Date(2026, 9, 7, 10, 0, 0, 0, time.UTC)
	keys := map[string]string{"old": strings.Repeat("o", 40), "new": strings.Repeat("n", 40)}
	newSigner, err := NewRotatingAccessTokenManager("new", keys, 15*time.Minute, 20*time.Minute, func() time.Time { return now }); if err != nil { t.Fatal(err) }
	oldReplica, err := NewRotatingAccessTokenManager("old", keys, 15*time.Minute, 20*time.Minute, func() time.Time { return now }); if err != nil { t.Fatal(err) }
	token, _, err := newSigner.Issue("user-1", "session-1"); if err != nil { t.Fatal(err) }
	if _, err := oldReplica.Parse(token); err != nil { t.Fatalf("mixed replica must verify new-key token after keyring predeploy: %v", err) }
}

func TestRotatingAccessTokenManagerRejectsInvalidKeyringAndOversizedToken(t *testing.T) {
	secret := strings.Repeat("x", 40)
	if _, err := NewRotatingAccessTokenManager("missing", map[string]string{"known": secret}, time.Minute, time.Minute, time.Now); err == nil { t.Fatal("missing active key must fail") }
	if _, err := NewRotatingAccessTokenManager("known", map[string]string{"known": "short"}, time.Minute, time.Minute, time.Now); err == nil { t.Fatal("weak key must fail") }
	m, err := NewRotatingAccessTokenManager("known", map[string]string{"known": secret}, time.Minute, time.Minute, time.Now); if err != nil { t.Fatal(err) }
	if _, err := m.Parse(strings.Repeat("x", maxAccessTokenBytes+1)); err == nil { t.Fatal("oversized token must fail") }
}

func TestRotatingAccessTokenManagerRejectsTokenBeyondMaximumLifetime(t *testing.T) {
	now := time.Date(2026, 9, 7, 10, 0, 0, 0, time.UTC)
	secret := strings.Repeat("x", 40)
	longIssuer, err := NewRotatingAccessTokenManager("k1", map[string]string{"k1": secret}, 20*time.Minute, 20*time.Minute, func() time.Time { return now }); if err != nil { t.Fatal(err) }
	token, _, err := longIssuer.Issue("user-1", "session-1"); if err != nil { t.Fatal(err) }
	strictVerifier, err := NewRotatingAccessTokenManager("k1", map[string]string{"k1": secret}, 10*time.Minute, 15*time.Minute, func() time.Time { return now }); if err != nil { t.Fatal(err) }
	if _, err := strictVerifier.Parse(token); err == nil { t.Fatal("token exceeding verifier maximum lifetime must fail") }
}
