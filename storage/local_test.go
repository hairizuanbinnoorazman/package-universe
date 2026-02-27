package storage

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewLocalStorage(t *testing.T) {
	tests := []struct {
		name      string
		baseDir   string
		wantError bool
	}{
		{
			name:      "valid base directory",
			baseDir:   t.TempDir(),
			wantError: false,
		},
		{
			name:      "creates non-existent directory",
			baseDir:   filepath.Join(t.TempDir(), "new-dir"),
			wantError: false,
		},
		{
			name:      "empty base directory",
			baseDir:   "",
			wantError: true,
		},
		{
			name:      "dot as base directory",
			baseDir:   ".",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, err := NewLocalStorage(tt.baseDir)
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

func TestLocalStorage_Upload(t *testing.T) {
	ctx := context.Background()
	baseDir := t.TempDir()
	storage, err := NewLocalStorage(baseDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	tests := []struct {
		name      string
		path      string
		content   string
		wantError bool
	}{
		{
			name:      "upload simple file",
			path:      "test.txt",
			content:   "hello world",
			wantError: false,
		},
		{
			name:      "upload file in subdirectory",
			path:      "subdir/test.txt",
			content:   "nested content",
			wantError: false,
		},
		{
			name:      "upload file with multiple nested directories",
			path:      "a/b/c/test.txt",
			content:   "deeply nested",
			wantError: false,
		},
		{
			name:      "empty path",
			path:      "",
			content:   "content",
			wantError: true,
		},
		{
			name:      "path traversal attempt",
			path:      "../outside.txt",
			content:   "malicious",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.content)
			err := storage.Upload(ctx, tt.path, reader)

			if tt.wantError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify file was created
			fullPath := filepath.Join(baseDir, tt.path)
			content, err := os.ReadFile(fullPath)
			if err != nil {
				t.Fatalf("failed to read uploaded file: %v", err)
			}

			if string(content) != tt.content {
				t.Errorf("content mismatch: got %q, want %q", string(content), tt.content)
			}
		})
	}
}

func TestLocalStorage_Download(t *testing.T) {
	ctx := context.Background()
	baseDir := t.TempDir()
	storage, err := NewLocalStorage(baseDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// Upload a test file
	testContent := "download test content"
	testPath := "test-download.txt"
	err = storage.Upload(ctx, testPath, strings.NewReader(testContent))
	if err != nil {
		t.Fatalf("failed to upload test file: %v", err)
	}

	t.Run("download existing file", func(t *testing.T) {
		reader, err := storage.Download(ctx, testPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer reader.Close()

		content, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("failed to read downloaded content: %v", err)
		}

		if string(content) != testContent {
			t.Errorf("content mismatch: got %q, want %q", string(content), testContent)
		}
	})

	t.Run("download non-existent file", func(t *testing.T) {
		_, err := storage.Download(ctx, "non-existent.txt")
		if err != ErrFileNotFound {
			t.Errorf("expected ErrFileNotFound but got: %v", err)
		}
	})

	t.Run("empty path", func(t *testing.T) {
		_, err := storage.Download(ctx, "")
		if err == nil {
			t.Error("expected error but got none")
		}
	})

	t.Run("path traversal attempt", func(t *testing.T) {
		_, err := storage.Download(ctx, "../outside.txt")
		if err == nil {
			t.Error("expected error but got none")
		}
	})
}

func TestLocalStorage_Delete(t *testing.T) {
	ctx := context.Background()
	baseDir := t.TempDir()
	storage, err := NewLocalStorage(baseDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// Upload a test file
	testPath := "test-delete.txt"
	err = storage.Upload(ctx, testPath, strings.NewReader("content"))
	if err != nil {
		t.Fatalf("failed to upload test file: %v", err)
	}

	t.Run("delete existing file", func(t *testing.T) {
		err := storage.Delete(ctx, testPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify file was deleted
		exists, err := storage.Exists(ctx, testPath)
		if err != nil {
			t.Fatalf("failed to check existence: %v", err)
		}
		if exists {
			t.Error("file should not exist after deletion")
		}
	})

	t.Run("delete non-existent file", func(t *testing.T) {
		err := storage.Delete(ctx, "non-existent.txt")
		if err != ErrFileNotFound {
			t.Errorf("expected ErrFileNotFound but got: %v", err)
		}
	})

	t.Run("empty path", func(t *testing.T) {
		err := storage.Delete(ctx, "")
		if err == nil {
			t.Error("expected error but got none")
		}
	})
}

func TestLocalStorage_Exists(t *testing.T) {
	ctx := context.Background()
	baseDir := t.TempDir()
	storage, err := NewLocalStorage(baseDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// Upload a test file
	testPath := "test-exists.txt"
	err = storage.Upload(ctx, testPath, strings.NewReader("content"))
	if err != nil {
		t.Fatalf("failed to upload test file: %v", err)
	}

	t.Run("file exists", func(t *testing.T) {
		exists, err := storage.Exists(ctx, testPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !exists {
			t.Error("file should exist")
		}
	})

	t.Run("file does not exist", func(t *testing.T) {
		exists, err := storage.Exists(ctx, "non-existent.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exists {
			t.Error("file should not exist")
		}
	})

	t.Run("empty path", func(t *testing.T) {
		_, err := storage.Exists(ctx, "")
		if err == nil {
			t.Error("expected error but got none")
		}
	})
}

func TestLocalStorage_GetURL(t *testing.T) {
	ctx := context.Background()
	baseDir := t.TempDir()
	storage, err := NewLocalStorage(baseDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// Upload a test file
	testPath := "test-url.txt"
	err = storage.Upload(ctx, testPath, strings.NewReader("content"))
	if err != nil {
		t.Fatalf("failed to upload test file: %v", err)
	}

	t.Run("get URL for existing file", func(t *testing.T) {
		url, err := storage.GetURL(ctx, testPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if url == "" {
			t.Error("URL should not be empty")
		}
		// Verify the URL contains the file path
		if !strings.Contains(url, testPath) {
			t.Errorf("URL should contain path %q, got %q", testPath, url)
		}
	})

	t.Run("get URL for non-existent file", func(t *testing.T) {
		_, err := storage.GetURL(ctx, "non-existent.txt")
		if err != ErrFileNotFound {
			t.Errorf("expected ErrFileNotFound but got: %v", err)
		}
	})

	t.Run("empty path", func(t *testing.T) {
		_, err := storage.GetURL(ctx, "")
		if err == nil {
			t.Error("expected error but got none")
		}
	})
}

func TestLocalStorage_List(t *testing.T) {
	ctx := context.Background()
	baseDir := t.TempDir()
	storage, err := NewLocalStorage(baseDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// Upload files in a directory structure
	storage.Upload(ctx, "mydir/file1.txt", strings.NewReader("content1"))
	storage.Upload(ctx, "mydir/file2.txt", strings.NewReader("content2"))
	storage.Upload(ctx, "mydir/subdir/file3.txt", strings.NewReader("content3"))

	t.Run("list existing directory", func(t *testing.T) {
		names, err := storage.List(ctx, "mydir")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(names) != 3 {
			t.Errorf("expected 3 entries, got %d: %v", len(names), names)
		}
	})

	t.Run("list non-existent directory", func(t *testing.T) {
		names, err := storage.List(ctx, "nonexistent")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(names) != 0 {
			t.Errorf("expected 0 entries, got %d", len(names))
		}
	})

	t.Run("empty prefix", func(t *testing.T) {
		_, err := storage.List(ctx, "")
		if err == nil {
			t.Error("expected error but got none")
		}
	})
}

func TestLocalStorage_UploadLargeFile(t *testing.T) {
	ctx := context.Background()
	baseDir := t.TempDir()
	storage, err := NewLocalStorage(baseDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// Create a 1MB buffer
	size := 1024 * 1024
	data := bytes.Repeat([]byte("x"), size)
	reader := bytes.NewReader(data)

	testPath := "large-file.bin"
	err = storage.Upload(ctx, testPath, reader)
	if err != nil {
		t.Fatalf("failed to upload large file: %v", err)
	}

	// Verify file size
	fullPath := filepath.Join(baseDir, testPath)
	info, err := os.Stat(fullPath)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}

	if info.Size() != int64(size) {
		t.Errorf("file size mismatch: got %d, want %d", info.Size(), size)
	}
}

func TestLocalStorage_PathTraversalPrevention(t *testing.T) {
	ctx := context.Background()
	baseDir := t.TempDir()
	storage, err := NewLocalStorage(baseDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	maliciousPaths := []string{
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32",
		"../../outside.txt",
		"subdir/../../outside.txt",
	}

	for _, path := range maliciousPaths {
		t.Run("block_"+path, func(t *testing.T) {
			err := storage.Upload(ctx, path, strings.NewReader("malicious"))
			if err == nil {
				t.Errorf("should have blocked path traversal for: %s", path)
			}
		})
	}
}
