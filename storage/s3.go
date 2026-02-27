package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

// S3Storage implements BlobStorage using AWS S3.
type S3Storage struct {
	client            *s3.Client
	presignClient     *s3.PresignClient
	bucket            string
	presignExpiration time.Duration
}

// NewS3Storage creates a new S3 storage client.
// It uses AWS SDK v2's default credential chain (IAM role on EC2).
func NewS3Storage(bucket, region string) (*S3Storage, error) {
	if bucket == "" {
		return nil, fmt.Errorf("S3 bucket name cannot be empty")
	}
	if region == "" {
		return nil, fmt.Errorf("S3 region cannot be empty")
	}

	// Load AWS config using default credential chain (IAM role on EC2)
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(cfg)

	return &S3Storage{
		client:            client,
		presignClient:     s3.NewPresignClient(client),
		bucket:            bucket,
		presignExpiration: 15 * time.Minute,
	}, nil
}

// Upload stores data from the reader at the specified path.
func (s *S3Storage) Upload(ctx context.Context, path string, reader io.Reader) error {
	if err := validatePath(path); err != nil {
		return err
	}

	// Clean the path for S3 key
	cleanPath := filepath.ToSlash(filepath.Clean(path))

	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(cleanPath),
		Body:   reader,
	})
	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	return nil
}

// Download retrieves data from the specified path.
func (s *S3Storage) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	if err := validatePath(path); err != nil {
		return nil, err
	}

	// Clean the path for S3 key
	cleanPath := filepath.ToSlash(filepath.Clean(path))

	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(cleanPath),
	})
	if err != nil {
		if isS3NotFoundError(err) {
			return nil, ErrFileNotFound
		}
		return nil, fmt.Errorf("failed to download from S3: %w", err)
	}

	return result.Body, nil
}

// Delete removes the data at the specified path.
func (s *S3Storage) Delete(ctx context.Context, path string) error {
	if err := validatePath(path); err != nil {
		return err
	}

	// Clean the path for S3 key
	cleanPath := filepath.ToSlash(filepath.Clean(path))

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(cleanPath),
	})
	if err != nil {
		if isS3NotFoundError(err) {
			return ErrFileNotFound
		}
		return fmt.Errorf("failed to delete from S3: %w", err)
	}

	return nil
}

// Exists checks if data exists at the specified path.
func (s *S3Storage) Exists(ctx context.Context, path string) (bool, error) {
	if err := validatePath(path); err != nil {
		return false, err
	}

	// Clean the path for S3 key
	cleanPath := filepath.ToSlash(filepath.Clean(path))

	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(cleanPath),
	})
	if err != nil {
		if isS3NotFoundError(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check S3 object existence: %w", err)
	}

	return true, nil
}

// GetURL returns a presigned URL for accessing the data at the specified path.
func (s *S3Storage) GetURL(ctx context.Context, path string) (string, error) {
	if err := validatePath(path); err != nil {
		return "", err
	}

	// Clean the path for S3 key
	cleanPath := filepath.ToSlash(filepath.Clean(path))

	// Check if file exists
	exists, err := s.Exists(ctx, path)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", ErrFileNotFound
	}

	// Generate presigned URL
	presignResult, err := s.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(cleanPath),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = s.presignExpiration
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return presignResult.URL, nil
}

// List returns the names of objects with the given prefix using S3 ListObjectsV2 with delimiter.
func (s *S3Storage) List(ctx context.Context, prefix string) ([]string, error) {
	if err := validatePath(prefix); err != nil {
		return nil, err
	}

	cleanPrefix := filepath.ToSlash(filepath.Clean(prefix))
	if !strings.HasSuffix(cleanPrefix, "/") {
		cleanPrefix += "/"
	}

	delimiter := "/"
	result, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:    aws.String(s.bucket),
		Prefix:    aws.String(cleanPrefix),
		Delimiter: &delimiter,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list S3 objects: %w", err)
	}

	var names []string
	for _, prefix := range result.CommonPrefixes {
		if prefix.Prefix != nil {
			name := strings.TrimPrefix(*prefix.Prefix, cleanPrefix)
			name = strings.TrimSuffix(name, "/")
			if name != "" {
				names = append(names, name)
			}
		}
	}
	for _, obj := range result.Contents {
		if obj.Key != nil {
			name := strings.TrimPrefix(*obj.Key, cleanPrefix)
			if name != "" {
				names = append(names, name)
			}
		}
	}

	return names, nil
}

// validatePath validates the path to prevent path traversal attacks.
// This maintains security consistency with LocalStorage even though S3 doesn't have filesystem paths.
func validatePath(path string) error {
	if path == "" {
		return fmt.Errorf("%w: path cannot be empty", ErrInvalidPath)
	}

	// Clean the path to remove any ".." or "." elements
	cleanPath := filepath.Clean(path)

	// Check for path traversal attempts
	if len(cleanPath) > 0 && cleanPath[0] == '.' {
		return fmt.Errorf("%w: path traversal detected", ErrInvalidPath)
	}

	// Check for absolute paths (should be relative)
	if filepath.IsAbs(cleanPath) {
		return fmt.Errorf("%w: absolute paths not allowed", ErrInvalidPath)
	}

	return nil
}

// isS3NotFoundError checks if an error is an S3 "not found" error.
func isS3NotFoundError(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		code := apiErr.ErrorCode()
		return code == "NoSuchKey" || code == "NotFound"
	}
	return false
}
