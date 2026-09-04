package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/synaudio/synaudio/backend/internal/platform/config"
)

// MinIO implements object storage backed by MinIO or any S3-compatible
// endpoint (including Cloudflare R2).
type MinIO struct {
	client *minio.Client
	bucket string
}

func NewMinIO(cfg config.Config) (*MinIO, error) {
	endpoint, secure, _, err := parseStorageEndpoint(cfg.StorageEndpoint)
	if err != nil {
		return nil, err
	}
	// Production object storage must always use TLS. Development remains free to
	// use explicit HTTP endpoints such as local MinIO, but private/RFC1918 hosts
	// are not treated as "local" exceptions in production because they can be
	// remote network services.
	if cfg.AppEnv == config.EnvProduction && !secure {
		return nil, fmt.Errorf("insecure STORAGE_ENDPOINT is not allowed in production")
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.StorageAccessKey, cfg.StorageSecretKey, ""),
		Secure: secure,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	return &MinIO{client: client, bucket: cfg.StorageBucket}, nil
}

// parseStorageEndpoint accepts only an explicit HTTP(S) authority. S3 client
// transport must never be guessed from a stripped or missing scheme because
// that can silently downgrade an intended HTTPS endpoint.
func parseStorageEndpoint(raw string) (endpoint string, secure bool, host string, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false, "", fmt.Errorf("STORAGE_ENDPOINT is required")
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", false, "", fmt.Errorf("invalid STORAGE_ENDPOINT: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", false, "", fmt.Errorf("STORAGE_ENDPOINT must use http or https")
	}
	if u.Host == "" || u.Hostname() == "" {
		return "", false, "", fmt.Errorf("STORAGE_ENDPOINT must include a host")
	}
	if u.User != nil || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" {
		return "", false, "", fmt.Errorf("STORAGE_ENDPOINT must contain only scheme and authority")
	}

	return u.Host, u.Scheme == "https", u.Hostname(), nil
}

// Put stores data at the given object key.
func (m *MinIO) Put(ctx context.Context, key string, data []byte) error {
	_, err := m.client.PutObject(ctx, m.bucket, key, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("put object %q: %w", key, err)
	}
	return nil
}

// Get loads the complete object bytes for an object key.
func (m *MinIO) Get(ctx context.Context, key string) ([]byte, error) {
	obj, err := m.client.GetObject(ctx, m.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get object %q: %w", key, err)
	}
	defer obj.Close()

	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, fmt.Errorf("read object %q: %w", key, err)
	}
	return data, nil
}

// PresignedGetObject returns a presigned URL for downloading an object.
func (m *MinIO) PresignedGetObject(ctx context.Context, key string, expiry time.Duration) (string, error) {
	url, err := m.client.PresignedGetObject(ctx, m.bucket, key, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("presign object %q: %w", key, err)
	}
	return url.String(), nil
}

// Ping verifies the storage backend is reachable.
func (m *MinIO) Ping(ctx context.Context) error {
	_, err := m.client.BucketExists(ctx, m.bucket)
	if err != nil {
		return fmt.Errorf("storage ping: %w", err)
	}
	return nil
}
