package identity_test

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/identity"
	"golang.org/x/crypto/argon2"
)

func TestHashPasswordRejectsOversizedPassword(t *testing.T) {
	if _, err := identity.HashPassword(strings.Repeat("a", 1025)); err == nil {
		t.Fatal("expected oversized password to be rejected")
	}
}

func TestVerifyPasswordRejectsUnsafePHCParameters(t *testing.T) {
	password := "correct horse battery staple"
	salt := []byte("0123456789abcdef")
	key := []byte("0123456789abcdef0123456789abcdef")
	unsafe := []string{
		fmt.Sprintf("$argon2id$v=%d$m=%d,t=3,p=4$%s$%s", argon2.Version, 1024*1024, base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key)),
		fmt.Sprintf("$argon2id$v=%d$m=65536,t=99,p=4$%s$%s", argon2.Version, base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key)),
		fmt.Sprintf("$argon2id$v=%d$m=65536,t=3,p=64$%s$%s", argon2.Version, base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key)),
		fmt.Sprintf("$argon2id$v=16$m=65536,t=3,p=4$%s$%s", base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key)),
	}
	for _, encoded := range unsafe {
		if identity.VerifyPassword(encoded, password) {
			t.Fatalf("unsafe PHC parameters unexpectedly verified: %s", encoded)
		}
	}
}

func TestVerifyPasswordPolicyMarksAcceptedLegacyHashForRehash(t *testing.T) {
	password := "legacy password"
	salt := []byte("0123456789abcdef")
	key := argon2.IDKey([]byte(password), salt, 2, 32*1024, 2, 32)
	encoded := fmt.Sprintf("$argon2id$v=%d$m=32768,t=2,p=2$%s$%s", argon2.Version, base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key))

	matched, needsRehash := identity.VerifyPasswordPolicy(encoded, password)
	if !matched {
		t.Fatal("expected accepted legacy hash to verify")
	}
	if !needsRehash {
		t.Fatal("expected weaker accepted legacy hash to require rehash")
	}
}

func TestVerifyPasswordPolicyDoesNotDowngradeStrongerAcceptedHash(t *testing.T) {
	password := "strong historical password"
	salt := []byte("0123456789abcdef")
	key := argon2.IDKey([]byte(password), salt, 4, 96*1024, 4, 32)
	encoded := fmt.Sprintf("$argon2id$v=%d$m=98304,t=4,p=4$%s$%s", argon2.Version, base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key))

	matched, needsRehash := identity.VerifyPasswordPolicy(encoded, password)
	if !matched {
		t.Fatal("expected stronger accepted hash to verify")
	}
	if needsRehash {
		t.Fatal("stronger historical hash must not be recomputed downward")
	}
}

func TestVerifyPasswordPolicyDoesNotRewriteMixedHistoricalProfile(t *testing.T) {
	password := "mixed historical password"
	salt := []byte("0123456789abcdef")
	key := argon2.IDKey([]byte(password), salt, 2, 96*1024, 4, 32)
	encoded := fmt.Sprintf("$argon2id$v=%d$m=98304,t=2,p=4$%s$%s", argon2.Version, base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key))

	matched, needsRehash := identity.VerifyPasswordPolicy(encoded, password)
	if !matched {
		t.Fatal("expected mixed accepted profile to verify")
	}
	if needsRehash {
		t.Fatal("mixed historical profile must not be rewritten by downgrading its stronger dimension")
	}
}

func TestVerifyPasswordRejectsOversizedPlaintextWithoutArgon2(t *testing.T) {
	hash, err := identity.HashPassword("valid password")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if identity.VerifyPassword(hash, strings.Repeat("x", 1025)) {
		t.Fatal("oversized password must fail verification")
	}
}
