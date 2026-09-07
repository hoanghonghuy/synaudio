package config

import (
	"strings"
	"testing"
)

func TestLoadWorkerIDRequiresExplicitProductionIdentity(t *testing.T) {
	t.Setenv("WORKER_ID", "")
	if _, err := LoadWorkerID(EnvProduction); err == nil {
		t.Fatal("production must require explicit WORKER_ID")
	}
}

func TestLoadWorkerIDAcceptsBoundedExplicitIdentity(t *testing.T) {
	t.Setenv("WORKER_ID", "synaudio-worker-7")
	got, err := LoadWorkerID(EnvProduction)
	if err != nil {
		t.Fatal(err)
	}
	if got != "synaudio-worker-7" {
		t.Fatalf("unexpected worker id %q", got)
	}
}

func TestLoadWorkerIDDevelopmentFallbackIsPerCallAndValid(t *testing.T) {
	t.Setenv("WORKER_ID", "")
	first, err := LoadWorkerID(EnvDevelopment)
	if err != nil {
		t.Fatal(err)
	}
	second, err := LoadWorkerID(EnvDevelopment)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatalf("development fallback must not be a shared constant: %q", first)
	}
	for _, id := range []string{first, second} {
		if !strings.HasPrefix(id, "dev-") {
			t.Fatalf("unexpected development id %q", id)
		}
		if err := validateWorkerID(id); err != nil {
			t.Fatalf("generated id must satisfy validation: %v", err)
		}
	}
}

func TestLoadWorkerIDRejectsInvalidOrUnboundedIdentity(t *testing.T) {
	for _, id := range []string{
		"worker with spaces",
		"worker\nforged",
		strings.Repeat("x", maxWorkerIDLength+1),
	} {
		t.Run(id, func(t *testing.T) {
			t.Setenv("WORKER_ID", id)
			if _, err := LoadWorkerID(EnvProduction); err == nil {
				t.Fatalf("expected invalid WORKER_ID %q to fail", id)
			}
		})
	}
}
