package storage

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"
)

// BlobStorage defines the interface for storing and retrieving binary data.
type BlobStorage interface {
	// Upload stores data from the reader at the specified path.
	Upload(ctx context.Context, path string, reader io.Reader) error

	// Download retrieves data from the specified path.
	Download(ctx context.Context, path string) (io.ReadCloser, error)

	// Delete removes the data at the specified path.
	Delete(ctx context.Context, path string) error

	// Exists checks if data exists at the specified path.
	Exists(ctx context.Context, path string) (bool, error)

	// GetURL returns a URL for accessing the data at the specified path.
	// For local storage, this returns a file:// URL or relative path.
	GetURL(ctx context.Context, path string) (string, error)
}

// NewBlobStorage creates a BlobStorage implementation based on configuration.
func NewBlobStorage(storageType string, config map[string]interface{}) (BlobStorage, error) {
	switch strings.ToLower(storageType) {
	case "local":
		baseDir, ok := config["base_dir"].(string)
		if !ok || baseDir == "" {
			return nil, fmt.Errorf("base_dir is required for local storage")
		}
		return NewLocalStorage(baseDir)

	case "s3":
		bucket, ok := config["bucket"].(string)
		if !ok || bucket == "" {
			return nil, fmt.Errorf("bucket is required for S3 storage")
		}
		region, ok := config["region"].(string)
		if !ok || region == "" {
			return nil, fmt.Errorf("region is required for S3 storage")
		}

		s3Storage, err := NewS3Storage(bucket, region)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize S3 storage: %w", err)
		}

		if expiry, ok := config["presign_expiry"].(time.Duration); ok {
			s3Storage.presignExpiration = expiry
		}

		return s3Storage, nil

	default:
		return nil, fmt.Errorf("unsupported storage type: %s", storageType)
	}
}
