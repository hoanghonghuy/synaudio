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

func TestNewMinIORejectsHTTPInProduction(t *testing.T) {
	for _, endpoint := range []string{
		"http://storage.example.com",
		"http://10.0.0.25:9000",
		"http://172.16.4.5:9000",
		"http://192.168.1.20:9000",
		"http://127.0.0.1:9000",
		"http://minio:9000",
	} {
		_, err := NewMinIO(config.Config{
			AppEnv:           config.EnvProduction,
			StorageEndpoint:  endpoint,
			StorageBucket:    "bucket",
			StorageAccessKey: "key",
			StorageSecretKey: "secret",
		})
		if err == nil || !strings.Contains(err.Error(), "insecure STORAGE_ENDPOINT") {
			t.Fatalf("expected production HTTP endpoint %q rejection, got %v", endpoint, err)
		}
	}
}

func TestNewMinIOAllowsExplicitLocalHTTPInDevelopment(t *testing.T) {
	store, err := NewMinIO(config.Config{
		AppEnv:           config.EnvDevelopment,
		StorageEndpoint:  "http://minio:9000",
		StorageBucket:    "bucket",
		StorageAccessKey: "key",
		StorageSecretKey: "secret",
	})
	if err != nil {
		t.Fatalf("new development storage client: %v", err)
	}
	if endpoint := store.client.EndpointURL(); endpoint.Scheme != "http" || endpoint.Host != "minio:9000" {
		t.Fatalf("unexpected development endpoint: %q", endpoint.String())
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
