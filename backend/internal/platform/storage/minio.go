package storage

import (
	"bytes"
	"context"
	"fmt"

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
	client, err := minio.New(cfg.StorageEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.StorageAccessKey, cfg.StorageSecretKey, ""),
		Secure: false,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	return &MinIO{client: client, bucket: cfg.StorageBucket}, nil
}

// Put stores data at the given object key.
func (m *MinIO) Put(ctx context.Context, key string, data []byte) error {
	_, err := m.client.PutObject(ctx, m.bucket, key, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("put object %q: %w", key, err)
	}
	return nil
}
