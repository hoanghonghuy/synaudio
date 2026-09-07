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

const (
	accessTokenIssuer = "synaudio"
	maxAccessTokenBytes = 8192
	maxAccessTokenHeaderBytes = 2048
	maxAccessTokenClaimsBytes = 6144
	maxAccessTokenSignatureBytes = 512
)

func NewRefreshToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil { return "", fmt.Errorf("generate refresh token: %w", err) }
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func HashToken(token string) string { sum := sha256.Sum256([]byte(token)); return base64.RawURLEncoding.EncodeToString(sum[:]) }
func VerifyTokenHash(hash, token string) bool {
	expected, err := base64.RawURLEncoding.DecodeString(hash); if err != nil { return false }
	sum := sha256.Sum256([]byte(token)); return subtle.ConstantTimeCompare(expected, sum[:]) == 1
}

type AccessClaims struct { Issuer string `json:"iss"`; Subject string `json:"sub"`; SessionID string `json:"sid"`; IssuedAt int64 `json:"iat"`; ExpiresAt int64 `json:"exp"` }
type accessTokenHeader struct { Algorithm string `json:"alg"`; Type string `json:"typ"`; KeyID string `json:"kid"` }

// AccessTokenManager signs with one active key and verifies only explicitly configured keys.
// Previous keys are supplied by configuration during a bounded rollout overlap.
type AccessTokenManager struct {
	activeKeyID string
	keys map[string][]byte
	ttl time.Duration
	maxTTL time.Duration
	now func() time.Time
}

// NewAccessTokenManager preserves the development/test single-key API. Production
// composition should use NewRotatingAccessTokenManager so tokens carry an operational key id.
func NewAccessTokenManager(secret string, ttl time.Duration, now func() time.Time) (*AccessTokenManager, error) {
	return NewRotatingAccessTokenManager("legacy", map[string]string{"legacy": secret}, ttl, ttl, now)
}

func NewRotatingAccessTokenManager(activeKeyID string, keys map[string]string, ttl, maxTTL time.Duration, now func() time.Time) (*AccessTokenManager, error) {
	activeKeyID = strings.TrimSpace(activeKeyID)
	if activeKeyID == "" { return nil, errors.New("access token active key id is required") }
	if ttl <= 0 || maxTTL <= 0 || ttl > maxTTL { return nil, errors.New("access token ttl must be positive and no greater than maximum ttl") }
	if len(keys) == 0 || len(keys) > 4 { return nil, errors.New("access token keyring must contain 1 to 4 keys") }
	parsed := make(map[string][]byte, len(keys))
	for id, secret := range keys {
		id = strings.TrimSpace(id)
		if id == "" || len(id) > 64 || strings.ContainsAny(id, ".,= ") { return nil, errors.New("access token key id is invalid") }
		if len(secret) < 32 { return nil, fmt.Errorf("access token key %q must be at least 32 bytes", id) }
		if _, exists := parsed[id]; exists { return nil, fmt.Errorf("duplicate access token key id %q", id) }
		parsed[id] = []byte(secret)
	}
	if _, ok := parsed[activeKeyID]; !ok { return nil, errors.New("access token active key is not present in keyring") }
	if now == nil { now = time.Now }
	return &AccessTokenManager{activeKeyID: activeKeyID, keys: parsed, ttl: ttl, maxTTL: maxTTL, now: now}, nil
}

func (m *AccessTokenManager) Issue(userID, sessionID string) (string, time.Time, error) {
	if m == nil || strings.TrimSpace(userID) == "" || strings.TrimSpace(sessionID) == "" { return "", time.Time{}, ErrInvalidToken }
	secret, ok := m.keys[m.activeKeyID]; if !ok { return "", time.Time{}, ErrInvalidToken }
	now := m.now().UTC(); expiresAt := now.Add(m.ttl)
	headerJSON, err := json.Marshal(accessTokenHeader{Algorithm:"HS256", Type:"JWT", KeyID:m.activeKeyID}); if err != nil { return "", time.Time{}, err }
	claimsJSON, err := json.Marshal(AccessClaims{Issuer:accessTokenIssuer, Subject:userID, SessionID:sessionID, IssuedAt:now.Unix(), ExpiresAt:expiresAt.Unix()}); if err != nil { return "", time.Time{}, err }
	header := base64.RawURLEncoding.EncodeToString(headerJSON); claims := base64.RawURLEncoding.EncodeToString(claimsJSON); unsigned := header+"."+claims
	return unsigned+"."+signWithKey(secret, unsigned), expiresAt, nil
}

func (m *AccessTokenManager) Parse(token string) (AccessClaims, error) {
	if m == nil || len(token) == 0 || len(token) > maxAccessTokenBytes { return AccessClaims{}, ErrInvalidToken }
	parts := strings.Split(token, ".")
	if len(parts)!=3 || parts[0]=="" || parts[1]=="" || parts[2]=="" || len(parts[0])>maxAccessTokenHeaderBytes || len(parts[1])>maxAccessTokenClaimsBytes || len(parts[2])>maxAccessTokenSignatureBytes { return AccessClaims{}, ErrInvalidToken }
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0]); if err != nil || len(headerBytes)>maxAccessTokenHeaderBytes { return AccessClaims{}, ErrInvalidToken }
	var header accessTokenHeader
	if err := json.Unmarshal(headerBytes,&header); err != nil || header.Algorithm!="HS256" || header.Type!="JWT" || header.KeyID=="" { return AccessClaims{}, ErrInvalidToken }
	secret, ok := m.keys[header.KeyID]; if !ok { return AccessClaims{}, ErrInvalidToken }
	unsigned := parts[0]+"."+parts[1]; expected := signWithKey(secret,unsigned)
	if !hmac.Equal([]byte(expected),[]byte(parts[2])) { return AccessClaims{}, ErrInvalidToken }
	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1]); if err != nil || len(claimsBytes)>maxAccessTokenClaimsBytes { return AccessClaims{}, ErrInvalidToken }
	var claims AccessClaims; if err:=json.Unmarshal(claimsBytes,&claims); err!=nil { return AccessClaims{}, ErrInvalidToken }
	now:=m.now().UTC().Unix()
	if claims.Issuer!=accessTokenIssuer || claims.Subject=="" || claims.SessionID=="" || claims.ExpiresAt<=now || claims.IssuedAt<=0 || claims.IssuedAt>now+60 || claims.ExpiresAt<=claims.IssuedAt { return AccessClaims{}, ErrInvalidToken }
	maxSeconds := int64(m.maxTTL/time.Second)
	if maxSeconds <= 0 || claims.ExpiresAt-claims.IssuedAt > maxSeconds { return AccessClaims{}, ErrInvalidToken }
	return claims,nil
}

func signWithKey(secret []byte, unsigned string) string { mac:=hmac.New(sha256.New,secret); _,_=mac.Write([]byte(unsigned)); return base64.RawURLEncoding.EncodeToString(mac.Sum(nil)) }
