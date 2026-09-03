package identity

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const accessTokenIssuer = "synaudio"

// NewRefreshToken returns a random opaque refresh token (URL-safe base64).
func NewRefreshToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// HashToken hashes an opaque token for at-rest storage.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// VerifyTokenHash reports whether token matches the stored hash.
func VerifyTokenHash(hash, token string) bool {
	expected, err := base64.RawURLEncoding.DecodeString(hash)
	if err != nil {
		return false
	}
	sum := sha256.Sum256([]byte(token))
	return subtle.ConstantTimeCompare(expected, sum[:]) == 1
}

// AccessClaims are the minimal signed claims required by the V1 authenticated
// boundary. The session ID deliberately remains part of the token so server-side
// revocation can invalidate a still-unexpired access token immediately.
type AccessClaims struct {
	Issuer    string `json:"iss"`
	Subject   string `json:"sub"`
	SessionID string `json:"sid"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

type accessTokenHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
}

// AccessTokenManager implements the small HS256 JWT surface required by the
// application without introducing token persistence or a vendor-specific auth
// dependency.
type AccessTokenManager struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

func NewAccessTokenManager(secret string, ttl time.Duration, now func() time.Time) (*AccessTokenManager, error) {
	if len(secret) < 32 {
		return nil, errors.New("access token secret must be at least 32 bytes")
	}
	if ttl <= 0 {
		return nil, errors.New("access token ttl must be positive")
	}
	if now == nil {
		now = time.Now
	}
	return &AccessTokenManager{secret: []byte(secret), ttl: ttl, now: now}, nil
}

func (m *AccessTokenManager) Issue(userID, sessionID string) (string, time.Time, error) {
	if m == nil || strings.TrimSpace(userID) == "" || strings.TrimSpace(sessionID) == "" {
		return "", time.Time{}, ErrInvalidToken
	}

	now := m.now().UTC()
	expiresAt := now.Add(m.ttl)
	headerJSON, err := json.Marshal(accessTokenHeader{Algorithm: "HS256", Type: "JWT"})
	if err != nil {
		return "", time.Time{}, err
	}
	claimsJSON, err := json.Marshal(AccessClaims{
		Issuer:    accessTokenIssuer,
		Subject:   userID,
		SessionID: sessionID,
		IssuedAt:  now.Unix(),
		ExpiresAt: expiresAt.Unix(),
	})
	if err != nil {
		return "", time.Time{}, err
	}

	header := base64.RawURLEncoding.EncodeToString(headerJSON)
	claims := base64.RawURLEncoding.EncodeToString(claimsJSON)
	unsigned := header + "." + claims
	signature := m.sign(unsigned)
	return unsigned + "." + signature, expiresAt, nil
}

func (m *AccessTokenManager) Parse(token string) (AccessClaims, error) {
	if m == nil {
		return AccessClaims{}, ErrInvalidToken
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return AccessClaims{}, ErrInvalidToken
	}

	unsigned := parts[0] + "." + parts[1]
	expected := m.sign(unsigned)
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return AccessClaims{}, ErrInvalidToken
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return AccessClaims{}, ErrInvalidToken
	}
	var header accessTokenHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil || header.Algorithm != "HS256" || header.Type != "JWT" {
		return AccessClaims{}, ErrInvalidToken
	}

	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return AccessClaims{}, ErrInvalidToken
	}
	var claims AccessClaims
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return AccessClaims{}, ErrInvalidToken
	}

	now := m.now().UTC().Unix()
	if claims.Issuer != accessTokenIssuer || claims.Subject == "" || claims.SessionID == "" || claims.ExpiresAt <= now {
		return AccessClaims{}, ErrInvalidToken
	}
	// Reject tokens issued materially in the future rather than accepting large
	// clock-skew windows that could extend effective token lifetime.
	if claims.IssuedAt <= 0 || claims.IssuedAt > now+60 || claims.ExpiresAt <= claims.IssuedAt {
		return AccessClaims{}, ErrInvalidToken
	}
	return claims, nil
}

func (m *AccessTokenManager) sign(unsigned string) string {
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write([]byte(unsigned))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
