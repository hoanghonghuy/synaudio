package identity_test

import (
	"strings"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/identity"
)

func TestNormalizeEmailTrimsAndLowercases(t *testing.T) {
	got := identity.NormalizeEmail("  User.Name@Example.COM ")
	want := "user.name@example.com"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestNormalizeEmailRejectsEmpty(t *testing.T) {
	if _, err := identity.NormalizeEmailChecked("   "); err == nil {
		t.Fatal("expected error for empty email")
	}
}

func TestNormalizeEmailRejectsMissingAt(t *testing.T) {
	if _, err := identity.NormalizeEmailChecked("not-an-email"); err == nil {
		t.Fatal("expected error for email without @")
	}
}

func TestHashPasswordProducesArgon2idAndVerifies(t *testing.T) {
	password := "correct horse battery staple"

	hash, err := identity.HashPassword(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Fatalf("expected argon2id hash prefix, got %q", hash)
	}

	if !identity.VerifyPassword(hash, password) {
		t.Fatal("expected password to verify against its hash")
	}

	if identity.VerifyPassword(hash, "wrong password") {
		t.Fatal("expected wrong password to fail verification")
	}
}

func TestHashPasswordRejectsEmpty(t *testing.T) {
	if _, err := identity.HashPassword(""); err == nil {
		t.Fatal("expected error for empty password")
	}
}
