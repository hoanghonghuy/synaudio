package identity

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"time"
)

const (
	totpPeriod = 30
	totpDigits = 6
)

// GenerateTOTPSecret returns a new random base32-encoded TOTP secret.
func GenerateTOTPSecret() (string, error) {
	buf := make([]byte, 20)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate totp secret: %w", err)
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf), nil
}

// TOTPTimeStep converts a unix timestamp to a TOTP counter.
func TOTPTimeStep(unix int64) uint64 {
	if unix == 0 {
		unix = time.Now().Unix()
	}
	return uint64(unix) / totpPeriod
}

// TOTPCode computes the 6-digit TOTP code for the given counter.
func TOTPCode(secret string, counter uint64) (string, error) {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		return "", fmt.Errorf("decode totp secret: %w", err)
	}

	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], counter)

	mac := hmac.New(sha1.New, key)
	mac.Write(msg[:])
	sum := mac.Sum(nil)

	offset := sum[len(sum)-1] & 0x0f
	code := (binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff) % 1_000_000

	return fmt.Sprintf("%06d", code), nil
}

// ValidateTOTP checks the code against the secret for the given counter,
// allowing a small window of adjacent steps.
func ValidateTOTP(secret, code string, counter uint64) bool {
	for delta := int64(-1); delta <= 1; delta++ {
		c := uint64(int64(counter) + delta)
		expected, err := TOTPCode(secret, c)
		if err != nil {
			return false
		}
		if hmac.Equal([]byte(expected), []byte(code)) {
			return true
		}
	}
	return false
}

// GenerateRecoveryCodes returns raw codes and their hashes.
func GenerateRecoveryCodes(count int) ([]string, []string, error) {
	codes := make([]string, 0, count)
	hashes := make([]string, 0, count)
	seen := map[string]bool{}

	for len(codes) < count {
		raw, err := NewRefreshToken()
		if err != nil {
			return nil, nil, err
		}
		// Use a short, human-friendly code derived from the token.
		code := raw[:10]
		if seen[code] {
			continue
		}
		seen[code] = true
		codes = append(codes, code)
		hashes = append(hashes, HashToken(code))
	}

	return codes, hashes, nil
}
