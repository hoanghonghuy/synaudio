package identity_test

import (
	"testing"

	"github.com/synaudio/synaudio/backend/internal/identity"
)

func TestGenerateTOTPSecretIsBase32(t *testing.T) {
	secret, err := identity.GenerateTOTPSecret()
	if err != nil {
		t.Fatalf("generate secret: %v", err)
	}
	if len(secret) < 16 {
		t.Fatalf("expected secret length >= 16, got %d", len(secret))
	}
	// Base32 alphabet only (A-Z, 2-7), no padding.
	for _, c := range secret {
		if !((c >= 'A' && c <= 'Z') || (c >= '2' && c <= '7')) {
			t.Fatalf("secret contains invalid base32 char %q", c)
		}
	}
}

func TestTOTPCodeIsSixDigits(t *testing.T) {
	secret, _ := identity.GenerateTOTPSecret()
	code, err := identity.TOTPCode(secret, 0)
	if err != nil {
		t.Fatalf("totp code: %v", err)
	}
	if len(code) != 6 {
		t.Fatalf("expected 6-digit code, got %q", code)
	}
	for _, c := range code {
		if c < '0' || c > '9' {
			t.Fatalf("code contains non-digit %q", c)
		}
	}
}

func TestTOTPValidateAcceptsCurrentCode(t *testing.T) {
	secret, _ := identity.GenerateTOTPSecret()
	now := identity.TOTPTimeStep(0)
	code, _ := identity.TOTPCode(secret, now)

	if !identity.ValidateTOTP(secret, code, now) {
		t.Fatal("expected current code to validate")
	}
}

func TestTOTPValidateRejectsWrongCode(t *testing.T) {
	secret, _ := identity.GenerateTOTPSecret()
	now := identity.TOTPTimeStep(0)

	if identity.ValidateTOTP(secret, "000000", now) {
		t.Fatal("expected wrong code to be rejected")
	}
}

func TestGenerateRecoveryCodesAreHashedAndUnique(t *testing.T) {
	codes, hashes, err := identity.GenerateRecoveryCodes(8)
	if err != nil {
		t.Fatalf("generate recovery codes: %v", err)
	}
	if len(codes) != 8 || len(hashes) != 8 {
		t.Fatalf("expected 8 codes and hashes, got %d/%d", len(codes), len(hashes))
	}

	seen := map[string]bool{}
	for i, c := range codes {
		if seen[c] {
			t.Fatal("recovery codes must be unique")
		}
		seen[c] = true
		if hashes[i] == c {
			t.Fatal("recovery code hash must not equal raw code")
		}
		if !identity.VerifyTokenHash(hashes[i], c) {
			t.Fatal("recovery code hash must verify against raw code")
		}
	}
}
