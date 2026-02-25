package storage

import (
	"context"
	"strings"
	"testing"
)

func TestNewS3Storage(t *testing.T) {
	tests := []struct {
		name      string
		bucket    string
		region    string
		wantError bool
	}{
		{
			name:      "valid bucket and region",
			bucket:    "test-bucket",
			region:    "us-east-1",
			wantError: false,
		},
		{
			name:      "empty bucket",
			bucket:    "",
			region:    "us-east-1",
			wantError: true,
		},
		{
			name:      "empty region",
			bucket:    "test-bucket",
			region:    "",
			wantError: true,
		},
		{
			name:      "both empty",
			bucket:    "",
			region:    "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, err := NewS3Storage(tt.bucket, tt.region)
			if tt.wantError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if storage == nil {
				t.Fatal("expected storage but got nil")
			}
			if storage.bucket != tt.bucket {
				t.Errorf("bucket mismatch: got %q, want %q", storage.bucket, tt.bucket)
			}
		})
	}
}

func TestValidatePath(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		wantError bool
	}{
		{
			name:      "valid simple path",
			path:      "test.txt",
			wantError: false,
		},
		{
			name:      "valid nested path",
			path:      "subdir/test.txt",
			wantError: false,
		},
		{
			name:      "valid deeply nested path",
			path:      "a/b/c/test.txt",
			wantError: false,
		},
		{
			name:      "empty path",
			path:      "",
			wantError: true,
		},
		{
			name:      "path traversal with ..",
			path:      "../outside.txt",
			wantError: true,
		},
		{
			name:      "path traversal in middle (cleaned to valid)",
			path:      "subdir/../outside.txt",
			wantError: false, // filepath.Clean normalizes this to "outside.txt" which is valid
		},
		{
			name:      "absolute path",
			path:      "/etc/passwd",
			wantError: true,
		},
		{
			name:      "path starting with dot (cleaned to valid)",
			path:      "./test.txt",
			wantError: false, // filepath.Clean normalizes this to "test.txt" which is valid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePath(tt.path)
			if tt.wantError {
				if err == nil {
					t.Errorf("expected error for path %q but got none", tt.path)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for path %q: %v", tt.path, err)
				}
			}
		})
	}
}

func TestS3Storage_PathValidation(t *testing.T) {
	// Create storage instance
	storage, err := NewS3Storage("test-bucket", "us-east-1")
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	ctx := context.Background()

	maliciousPaths := []string{
		"",
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32",
		"../../outside.txt",
		"subdir/../../outside.txt",
		"/absolute/path.txt",
		"./relative.txt",
	}

	t.Run("upload rejects malicious paths", func(t *testing.T) {
		for _, path := range maliciousPaths {
			err := storage.Upload(ctx, path, strings.NewReader("test"))
			if err == nil {
				t.Errorf("should have blocked path: %s", path)
			}
		}
	})

	t.Run("download rejects malicious paths", func(t *testing.T) {
		for _, path := range maliciousPaths {
			_, err := storage.Download(ctx, path)
			if err == nil {
				t.Errorf("should have blocked path: %s", path)
			}
		}
	})

	t.Run("delete rejects malicious paths", func(t *testing.T) {
		for _, path := range maliciousPaths {
			err := storage.Delete(ctx, path)
			if err == nil {
				t.Errorf("should have blocked path: %s", path)
			}
		}
	})

	t.Run("exists rejects malicious paths", func(t *testing.T) {
		for _, path := range maliciousPaths {
			_, err := storage.Exists(ctx, path)
			if err == nil {
				t.Errorf("should have blocked path: %s", path)
			}
		}
	})

	t.Run("getURL rejects malicious paths", func(t *testing.T) {
		for _, path := range maliciousPaths {
			_, err := storage.GetURL(ctx, path)
			if err == nil {
				t.Errorf("should have blocked path: %s", path)
			}
		}
	})
}

func TestS3Storage_PresignExpiration(t *testing.T) {
	storage, err := NewS3Storage("test-bucket", "us-east-1")
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// Check default presign expiration
	defaultExpiry := storage.presignExpiration
	if defaultExpiry != 15*60*1000000000 { // 15 minutes in nanoseconds
		t.Errorf("default presign expiration should be 15 minutes, got %v", defaultExpiry)
	}
}

// TestNewBlobStorage tests the factory function
func TestNewBlobStorage(t *testing.T) {
	tests := []struct {
		name        string
		storageType string
		config      map[string]interface{}
		wantError   bool
	}{
		{
			name:        "local storage",
			storageType: "local",
			config: map[string]interface{}{
				"base_dir": t.TempDir(),
			},
			wantError: false,
		},
		{
			name:        "local storage uppercase",
			storageType: "LOCAL",
			config: map[string]interface{}{
				"base_dir": t.TempDir(),
			},
			wantError: false,
		},
		{
			name:        "local storage missing base_dir",
			storageType: "local",
			config:      map[string]interface{}{},
			wantError:   true,
		},
		{
			name:        "s3 storage",
			storageType: "s3",
			config: map[string]interface{}{
				"bucket": "test-bucket",
				"region": "us-east-1",
			},
			wantError: false,
		},
		{
			name:        "s3 storage uppercase",
			storageType: "S3",
			config: map[string]interface{}{
				"bucket": "test-bucket",
				"region": "us-east-1",
			},
			wantError: false,
		},
		{
			name:        "s3 storage missing bucket",
			storageType: "s3",
			config: map[string]interface{}{
				"region": "us-east-1",
			},
			wantError: true,
		},
		{
			name:        "s3 storage missing region",
			storageType: "s3",
			config: map[string]interface{}{
				"bucket": "test-bucket",
			},
			wantError: true,
		},
		{
			name:        "unsupported storage type",
			storageType: "gcs",
			config:      map[string]interface{}{},
			wantError:   true,
		},
		{
			name:        "empty storage type",
			storageType: "",
			config:      map[string]interface{}{},
			wantError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, err := NewBlobStorage(tt.storageType, tt.config)
			if tt.wantError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if storage == nil {
				t.Fatal("expected storage but got nil")
			}
		})
	}
}

func TestIsS3NotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantTrue bool
	}{
		{
			name:     "nil error",
			err:      nil,
			wantTrue: false,
		},
		{
			name:     "generic error",
			err:      context.Canceled,
			wantTrue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isS3NotFoundError(tt.err)
			if result != tt.wantTrue {
				t.Errorf("isS3NotFoundError(%v) = %v, want %v", tt.err, result, tt.wantTrue)
			}
		})
	}
}
