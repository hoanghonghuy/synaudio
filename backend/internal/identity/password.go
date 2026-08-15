package identity

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argon2Time    = 3
	argon2Memory  = 64 * 1024
	argon2Threads = 4
	argon2KeyLen  = 32
	argon2SaltLen = 16
)

var (
	ErrEmptyEmail    = errors.New("email is empty")
	ErrInvalidEmail  = errors.New("email is invalid")
	ErrEmptyPassword = errors.New("password is empty")
)

// NormalizeEmail trims surrounding whitespace and lowercases the local part
// and domain. Email uniqueness is enforced on this normalized form.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// NormalizeEmailChecked normalizes and validates a basic email shape.
func NormalizeEmailChecked(email string) (string, error) {
	normalized := NormalizeEmail(email)
	if normalized == "" {
		return "", ErrEmptyEmail
	}
	at := strings.LastIndex(normalized, "@")
	if at <= 0 || at == len(normalized)-1 {
		return "", ErrInvalidEmail
	}
	return normalized, nil
}

// HashPassword derives an Argon2id PHC-style encoded hash for the password.
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", ErrEmptyPassword
	}

	salt := make([]byte, argon2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}

	key := argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)

	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argon2Memory,
		argon2Time,
		argon2Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// VerifyPassword reports whether password matches the encoded Argon2id hash.
func VerifyPassword(encoded, password string) bool {
	params, salt, key, err := decodeArgon2id(encoded)
	if err != nil {
		return false
	}

	derived := argon2.IDKey([]byte(password), salt, params.time, params.memory, params.threads, uint32(len(key)))

	return subtle.ConstantTimeCompare(derived, key) == 1
}

type argon2Params struct {
	time    uint32
	memory  uint32
	threads uint8
}

func decodeArgon2id(encoded string) (argon2Params, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	// parts: ["", "argon2id", "v=19", "m=...,t=...,p=...", salt, key]
	if len(parts) != 6 || parts[1] != "argon2id" {
		return argon2Params{}, nil, nil, errors.New("invalid argon2id hash")
	}

	var p argon2Params
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.memory, &p.time, &p.threads); err != nil {
		return argon2Params{}, nil, nil, fmt.Errorf("parse params: %w", err)
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return argon2Params{}, nil, nil, fmt.Errorf("decode salt: %w", err)
	}

	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return argon2Params{}, nil, nil, fmt.Errorf("decode key: %w", err)
	}

	return p, salt, key, nil
}
