package identity

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argon2Time    = 3
	argon2Memory  = 64 * 1024
	argon2Threads = 4
	argon2KeyLen  = 32
	argon2SaltLen = 16

	maxPasswordBytes        = 1024
	maxEncodedArgon2IDBytes = 512
	minArgon2Time            = 2
	maxArgon2Time            = 6
	minArgon2Memory          = 32 * 1024
	maxArgon2Memory          = 128 * 1024
	minArgon2Threads         = 1
	maxArgon2Threads         = 8
	minArgon2SaltLen         = 16
	maxArgon2SaltLen         = 64
	minArgon2KeyLen          = 16
	maxArgon2KeyLen          = 64
)

var (
	ErrEmptyEmail      = errors.New("email is empty")
	ErrInvalidEmail    = errors.New("email is invalid")
	ErrEmptyPassword   = errors.New("password is empty")
	ErrPasswordTooLong = errors.New("password is too long")
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

func validatePasswordInput(password string) error {
	if password == "" {
		return ErrEmptyPassword
	}
	if len(password) > maxPasswordBytes {
		return ErrPasswordTooLong
	}
	return nil
}

// HashPassword derives an Argon2id PHC-style encoded hash using the current
// repository-owned password policy.
func HashPassword(password string) (string, error) {
	if err := validatePasswordInput(password); err != nil {
		return "", err
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
// Persisted PHC parameters are validated against explicit resource bounds before
// Argon2 is invoked.
func VerifyPassword(encoded, password string) bool {
	matched, _ := VerifyPasswordPolicy(encoded, password)
	return matched
}

// VerifyPasswordPolicy verifies a password and reports whether a valid legacy
// hash should be upgraded to the current write policy after authentication.
func VerifyPasswordPolicy(encoded, password string) (matched bool, needsRehash bool) {
	if err := validatePasswordInput(password); err != nil {
		return false, false
	}
	params, salt, key, err := decodeArgon2id(encoded)
	if err != nil {
		return false, false
	}

	derived := argon2.IDKey([]byte(password), salt, params.time, params.memory, params.threads, uint32(len(key)))
	if subtle.ConstantTimeCompare(derived, key) != 1 {
		return false, false
	}

	return true, strictlyWeakerThanCurrentPolicy(params, len(salt), len(key))
}

type argon2Params struct {
	time    uint32
	memory  uint32
	threads uint8
}

// strictlyWeakerThanCurrentPolicy uses a partial-order comparison: every
// security/resource dimension must be no stronger than the current write policy
// and at least one must be weaker. Mixed or stronger historical profiles are not
// automatically rewritten, which avoids silently downgrading one dimension.
func strictlyWeakerThanCurrentPolicy(p argon2Params, saltLen, keyLen int) bool {
	noStronger := p.time <= argon2Time && p.memory <= argon2Memory && p.threads <= argon2Threads && saltLen <= argon2SaltLen && keyLen <= argon2KeyLen
	weaker := p.time < argon2Time || p.memory < argon2Memory || p.threads < argon2Threads || saltLen < argon2SaltLen || keyLen < argon2KeyLen
	return noStronger && weaker
}

func decodeArgon2id(encoded string) (argon2Params, []byte, []byte, error) {
	if len(encoded) == 0 || len(encoded) > maxEncodedArgon2IDBytes {
		return argon2Params{}, nil, nil, errors.New("invalid argon2id hash length")
	}
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return argon2Params{}, nil, nil, errors.New("invalid argon2id hash")
	}
	if parts[2] != "v="+strconv.Itoa(argon2.Version) {
		return argon2Params{}, nil, nil, errors.New("unsupported argon2 version")
	}

	fields := strings.Split(parts[3], ",")
	if len(fields) != 3 {
		return argon2Params{}, nil, nil, errors.New("invalid argon2id params")
	}
	memory, err := parseUintField(fields[0], "m=", 32)
	if err != nil {
		return argon2Params{}, nil, nil, err
	}
	timeCost, err := parseUintField(fields[1], "t=", 32)
	if err != nil {
		return argon2Params{}, nil, nil, err
	}
	threads, err := parseUintField(fields[2], "p=", 8)
	if err != nil {
		return argon2Params{}, nil, nil, err
	}

	p := argon2Params{memory: uint32(memory), time: uint32(timeCost), threads: uint8(threads)}
	if p.memory < minArgon2Memory || p.memory > maxArgon2Memory ||
		p.time < minArgon2Time || p.time > maxArgon2Time ||
		p.threads < minArgon2Threads || p.threads > maxArgon2Threads {
		return argon2Params{}, nil, nil, errors.New("argon2id params outside accepted policy")
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) < minArgon2SaltLen || len(salt) > maxArgon2SaltLen {
		return argon2Params{}, nil, nil, errors.New("invalid argon2id salt")
	}

	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(key) < minArgon2KeyLen || len(key) > maxArgon2KeyLen {
		return argon2Params{}, nil, nil, errors.New("invalid argon2id key")
	}

	return p, salt, key, nil
}

func parseUintField(field, prefix string, bitSize int) (uint64, error) {
	if !strings.HasPrefix(field, prefix) || len(field) == len(prefix) {
		return 0, errors.New("invalid argon2id parameter")
	}
	v, err := strconv.ParseUint(strings.TrimPrefix(field, prefix), 10, bitSize)
	if err != nil {
		return 0, errors.New("invalid argon2id parameter")
	}
	return v, nil
}
