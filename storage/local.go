package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var (
	// ErrFileNotFound is returned when a requested file does not exist.
	ErrFileNotFound = errors.New("file not found")

	// ErrInvalidPath is returned when a path is invalid or contains path traversal.
	ErrInvalidPath = errors.New("invalid path")
)

// LocalStorage implements BlobStorage using the local filesystem.
type LocalStorage struct {
	baseDir string
}

// NewLocalStorage creates a new local filesystem storage.
// The baseDir will be created if it doesn't exist.
func NewLocalStorage(baseDir string) (*LocalStorage, error) {
	// Clean and validate base directory
	baseDir = filepath.Clean(baseDir)
	if baseDir == "" || baseDir == "." {
		return nil, fmt.Errorf("%w: base directory cannot be empty", ErrInvalidPath)
	}

	// Create base directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	return &LocalStorage{
		baseDir: baseDir,
	}, nil
}

// Upload stores data from the reader at the specified path.
func (s *LocalStorage) Upload(ctx context.Context, path string, reader io.Reader) error {
	fullPath, err := s.validateAndJoinPath(path)
	if err != nil {
		return err
	}

	// Create parent directory if it doesn't exist
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create the file
	file, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Copy data from reader to file
	if _, err := io.Copy(file, reader); err != nil {
		// Clean up partial file on error
		os.Remove(fullPath)
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Download retrieves data from the specified path.
func (s *LocalStorage) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	fullPath, err := s.validateAndJoinPath(path)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(fullPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrFileNotFound
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, nil
}

// Delete removes the data at the specified path.
func (s *LocalStorage) Delete(ctx context.Context, path string) error {
	fullPath, err := s.validateAndJoinPath(path)
	if err != nil {
		return err
	}

	if err := os.Remove(fullPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrFileNotFound
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// Exists checks if data exists at the specified path.
func (s *LocalStorage) Exists(ctx context.Context, path string) (bool, error) {
	fullPath, err := s.validateAndJoinPath(path)
	if err != nil {
		return false, err
	}

	_, err = os.Stat(fullPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check file existence: %w", err)
	}

	return true, nil
}

// GetURL returns a relative path for accessing the file.
func (s *LocalStorage) GetURL(ctx context.Context, path string) (string, error) {
	fullPath, err := s.validateAndJoinPath(path)
	if err != nil {
		return "", err
	}

	// Check if file exists
	exists, err := s.Exists(ctx, path)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", ErrFileNotFound
	}

	return fullPath, nil
}

// validateAndJoinPath validates the path and joins it with the base directory.
// It prevents path traversal attacks by ensuring the final path is within baseDir.
func (s *LocalStorage) validateAndJoinPath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("%w: path cannot be empty", ErrInvalidPath)
	}

	// Clean the path to remove any ".." or "." elements
	cleanPath := filepath.Clean(path)

	// Join with base directory
	fullPath := filepath.Join(s.baseDir, cleanPath)

	// Ensure the final path is still within baseDir (prevent path traversal)
	relPath, err := filepath.Rel(s.baseDir, fullPath)
	if err != nil || len(relPath) > 0 && relPath[0] == '.' {
		return "", fmt.Errorf("%w: path traversal detected", ErrInvalidPath)
	}

	return fullPath, nil
}
