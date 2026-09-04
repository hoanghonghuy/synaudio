package storage

import (
	"strings"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/platform/config"
)

func TestParseStorageEndpointPreservesHTTPS(t *testing.T) {
	endpoint, secure, host, err := parseStorageEndpoint("https://account.r2.cloudflarestorage.com")
	if err != nil {
		t.Fatalf("parse https endpoint: %v", err)
	}
	if endpoint != "account.r2.cloudflarestorage.com" || !secure || host != "account.r2.cloudflarestorage.com" {
		t.Fatalf("unexpected parsed endpoint: endpoint=%q secure=%v host=%q", endpoint, secure, host)
	}
}

func TestParseStorageEndpointAllowsLocalHTTP(t *testing.T) {
	endpoint, secure, host, err := parseStorageEndpoint("http://minio:9000")
	if err != nil {
		t.Fatalf("parse local http endpoint: %v", err)
	}
	if endpoint != "minio:9000" || secure || host != "minio" {
		t.Fatalf("unexpected parsed endpoint: endpoint=%q secure=%v host=%q", endpoint, secure, host)
	}
}

func TestParseStorageEndpointRejectsAmbiguousValues(t *testing.T) {
	for _, raw := range []string{
		"account.r2.cloudflarestorage.com",
		"ftp://storage.example.com",
		"https://storage.example.com/bucket",
		"https://user:pass@storage.example.com",
		"https://storage.example.com?x=1",
	} {
		if _, _, _, err := parseStorageEndpoint(raw); err == nil {
			t.Fatalf("expected %q to be rejected", raw)
		}
	}
}

func TestNewMinIORejectsRemoteHTTPInProduction(t *testing.T) {
	_, err := NewMinIO(config.Config{
		AppEnv:           config.EnvProduction,
		StorageEndpoint:  "http://storage.example.com",
		StorageBucket:    "bucket",
		StorageAccessKey: "key",
		StorageSecretKey: "secret",
	})
	if err == nil || !strings.Contains(err.Error(), "insecure remote STORAGE_ENDPOINT") {
		t.Fatalf("expected insecure production endpoint rejection, got %v", err)
	}
}

func TestNewMinIOKeepsHTTPSForRemoteEndpointWithoutNetworkDiscovery(t *testing.T) {
	store, err := NewMinIO(config.Config{
		AppEnv:           config.EnvProduction,
		StorageEndpoint:  "https://account.r2.cloudflarestorage.com",
		StorageBucket:    "bucket",
		StorageAccessKey: "key",
		StorageSecretKey: "secret",
	})
	if err != nil {
		t.Fatalf("new storage client: %v", err)
	}

	endpoint := store.client.EndpointURL()
	if endpoint.Scheme != "https" {
		t.Fatalf("expected https client endpoint, got %q", endpoint.String())
	}
	if endpoint.Host != "account.r2.cloudflarestorage.com" {
		t.Fatalf("unexpected client endpoint host: %q", endpoint.Host)
	}
}
