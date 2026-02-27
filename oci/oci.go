package oci

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/hairizuanbinnoorazman/package-universe/storage"
)

// ManifestInfo holds metadata about a stored manifest.
type ManifestInfo struct {
	Digest      DigestInfo
	ContentType string
	Size        int64
}

// BlobInfo holds metadata about a stored blob.
type BlobInfo struct {
	Digest DigestInfo
	Size   int64
}

// OCIStorage provides OCI-specific storage operations on top of BlobStorage.
type OCIStorage struct {
	store    storage.BlobStorage
	sessions *SessionManager
}

// NewOCIStorage creates a new OCIStorage wrapping the given BlobStorage.
func NewOCIStorage(store storage.BlobStorage, sessions *SessionManager) *OCIStorage {
	return &OCIStorage{
		store:    store,
		sessions: sessions,
	}
}

// BlobExists checks if a blob with the given digest exists.
func (s *OCIStorage) BlobExists(ctx context.Context, digest DigestInfo) (bool, error) {
	return s.store.Exists(ctx, BlobDataPath(digest))
}

// GetBlob retrieves a blob by digest.
func (s *OCIStorage) GetBlob(ctx context.Context, digest DigestInfo) (io.ReadCloser, error) {
	rc, err := s.store.Download(ctx, BlobDataPath(digest))
	if err != nil {
		if err == storage.ErrFileNotFound {
			return nil, ErrBlobNotFound
		}
		return nil, fmt.Errorf("failed to get blob: %w", err)
	}
	return rc, nil
}

// GetBlobInfo returns size information for a blob.
func (s *OCIStorage) GetBlobInfo(ctx context.Context, digest DigestInfo) (*BlobInfo, error) {
	exists, err := s.BlobExists(ctx, digest)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrBlobNotFound
	}

	// Read to determine size
	rc, err := s.store.Download(ctx, BlobDataPath(digest))
	if err != nil {
		return nil, fmt.Errorf("failed to get blob info: %w", err)
	}
	defer rc.Close()

	size, err := io.Copy(io.Discard, rc)
	if err != nil {
		return nil, fmt.Errorf("failed to read blob size: %w", err)
	}

	return &BlobInfo{
		Digest: digest,
		Size:   size,
	}, nil
}

// InitiateUpload starts a new blob upload session and returns the UUID.
func (s *OCIStorage) InitiateUpload(ctx context.Context, repository string) (string, error) {
	uuid, err := s.sessions.Create(repository)
	if err != nil {
		return "", fmt.Errorf("failed to create upload session: %w", err)
	}

	// Create an empty upload data file
	err = s.store.Upload(ctx, UploadDataPath(uuid), strings.NewReader(""))
	if err != nil {
		s.sessions.Delete(uuid)
		return "", fmt.Errorf("failed to initialize upload: %w", err)
	}

	return uuid, nil
}

// WriteUploadChunk appends data to an in-progress upload.
func (s *OCIStorage) WriteUploadChunk(ctx context.Context, uuid string, data io.Reader) (int64, error) {
	session, err := s.sessions.Get(uuid)
	if err != nil {
		return 0, err
	}

	uploadPath := UploadDataPath(uuid)

	// Read existing data if any
	var existingData []byte
	if session.BytesWritten > 0 {
		rc, err := s.store.Download(ctx, uploadPath)
		if err != nil && err != storage.ErrFileNotFound {
			return 0, fmt.Errorf("failed to read existing upload: %w", err)
		}
		if rc != nil {
			existingData, err = io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return 0, fmt.Errorf("failed to read existing upload data: %w", err)
			}
		}
	}

	// Read new chunk
	newData, err := io.ReadAll(data)
	if err != nil {
		return 0, fmt.Errorf("failed to read upload chunk: %w", err)
	}

	// Combine and write back
	combined := append(existingData, newData...)
	err = s.store.Upload(ctx, uploadPath, bytes.NewReader(combined))
	if err != nil {
		return 0, fmt.Errorf("failed to write upload chunk: %w", err)
	}

	totalSize := int64(len(combined))
	s.sessions.UpdateBytes(uuid, totalSize)

	return totalSize, nil
}

// CompleteUpload finalizes an upload, verifying the digest and moving to content-addressable storage.
func (s *OCIStorage) CompleteUpload(ctx context.Context, uuid string, expectedDigest DigestInfo) (DigestInfo, error) {
	_, err := s.sessions.Get(uuid)
	if err != nil {
		return DigestInfo{}, err
	}

	uploadPath := UploadDataPath(uuid)

	// Download the upload data
	rc, err := s.store.Download(ctx, uploadPath)
	if err != nil {
		return DigestInfo{}, fmt.Errorf("failed to read upload: %w", err)
	}

	// Read through verifying reader
	vr := NewVerifyingReader(rc)
	data, err := io.ReadAll(vr)
	rc.Close()
	if err != nil {
		return DigestInfo{}, fmt.Errorf("failed to read upload data: %w", err)
	}

	// Verify digest
	if err := vr.Verify(expectedDigest); err != nil {
		return DigestInfo{}, err
	}

	// Store at content-addressable path
	blobPath := BlobDataPath(expectedDigest)
	err = s.store.Upload(ctx, blobPath, bytes.NewReader(data))
	if err != nil {
		return DigestInfo{}, fmt.Errorf("failed to store blob: %w", err)
	}

	// Clean up upload
	s.store.Delete(ctx, uploadPath)
	s.sessions.Delete(uuid)

	return expectedDigest, nil
}

// CancelUpload removes an in-progress upload.
func (s *OCIStorage) CancelUpload(ctx context.Context, uuid string) error {
	_, err := s.sessions.Get(uuid)
	if err != nil {
		return err
	}

	s.store.Delete(ctx, UploadDataPath(uuid))
	s.sessions.Delete(uuid)
	return nil
}

// PutManifest stores a manifest by digest, and if reference is a tag, creates a tag link.
func (s *OCIStorage) PutManifest(ctx context.Context, name, reference string, contentType string, data []byte) (DigestInfo, error) {
	// Compute digest
	vr := NewVerifyingReader(bytes.NewReader(data))
	_, err := io.Copy(io.Discard, vr)
	if err != nil {
		return DigestInfo{}, fmt.Errorf("failed to compute manifest digest: %w", err)
	}
	digest := vr.Digest()

	// Store the manifest blob
	blobPath := BlobDataPath(digest)
	err = s.store.Upload(ctx, blobPath, bytes.NewReader(data))
	if err != nil {
		return DigestInfo{}, fmt.Errorf("failed to store manifest: %w", err)
	}

	// Store content-type metadata as a small file alongside the manifest
	metaPath := ManifestRevisionLinkPath(name, digest)
	metaContent := digest.String() + "\n" + contentType
	err = s.store.Upload(ctx, metaPath, strings.NewReader(metaContent))
	if err != nil {
		return DigestInfo{}, fmt.Errorf("failed to store manifest revision link: %w", err)
	}

	// If reference looks like a tag (not a digest), create tag link
	if !isDigestReference(reference) {
		tagPath := ManifestTagCurrentLinkPath(name, reference)
		err = s.store.Upload(ctx, tagPath, strings.NewReader(metaContent))
		if err != nil {
			return DigestInfo{}, fmt.Errorf("failed to store tag link: %w", err)
		}
	}

	return digest, nil
}

// GetManifest retrieves a manifest by tag or digest reference.
func (s *OCIStorage) GetManifest(ctx context.Context, name, reference string) ([]byte, DigestInfo, string, error) {
	var digest DigestInfo
	var contentType string

	if isDigestReference(reference) {
		d, err := ParseDigest(reference)
		if err != nil {
			return nil, DigestInfo{}, "", err
		}
		digest = d

		// Read content type from revision link
		ct, err := s.readManifestMeta(ctx, name, digest)
		if err != nil {
			return nil, DigestInfo{}, "", err
		}
		contentType = ct
	} else {
		// Look up tag
		tagPath := ManifestTagCurrentLinkPath(name, reference)
		rc, err := s.store.Download(ctx, tagPath)
		if err != nil {
			if err == storage.ErrFileNotFound {
				return nil, DigestInfo{}, "", ErrManifestNotFound
			}
			return nil, DigestInfo{}, "", fmt.Errorf("failed to read tag: %w", err)
		}
		linkData, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, DigestInfo{}, "", fmt.Errorf("failed to read tag link: %w", err)
		}

		parts := strings.SplitN(string(linkData), "\n", 2)
		d, err := ParseDigest(strings.TrimSpace(parts[0]))
		if err != nil {
			return nil, DigestInfo{}, "", fmt.Errorf("invalid digest in tag link: %w", err)
		}
		digest = d
		if len(parts) > 1 {
			contentType = strings.TrimSpace(parts[1])
		}
	}

	// Read manifest data from blob storage
	rc, err := s.store.Download(ctx, BlobDataPath(digest))
	if err != nil {
		if err == storage.ErrFileNotFound {
			return nil, DigestInfo{}, "", ErrManifestNotFound
		}
		return nil, DigestInfo{}, "", fmt.Errorf("failed to read manifest: %w", err)
	}
	data, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		return nil, DigestInfo{}, "", fmt.Errorf("failed to read manifest data: %w", err)
	}

	if contentType == "" {
		contentType = "application/vnd.oci.image.manifest.v1+json"
	}

	return data, digest, contentType, nil
}

// ManifestExists checks if a manifest exists by tag or digest reference.
func (s *OCIStorage) ManifestExists(ctx context.Context, name, reference string) (DigestInfo, string, int64, error) {
	data, digest, contentType, err := s.GetManifest(ctx, name, reference)
	if err != nil {
		return DigestInfo{}, "", 0, err
	}
	return digest, contentType, int64(len(data)), nil
}

// ListTags returns all tags for a repository.
func (s *OCIStorage) ListTags(ctx context.Context, name string) ([]string, error) {
	tagsDir := ManifestTagsDir(name)
	entries, err := s.store.List(ctx, tagsDir)
	if err != nil {
		return []string{}, nil
	}
	return entries, nil
}

// readManifestMeta reads the content type from a manifest revision link.
func (s *OCIStorage) readManifestMeta(ctx context.Context, name string, digest DigestInfo) (string, error) {
	linkPath := ManifestRevisionLinkPath(name, digest)
	rc, err := s.store.Download(ctx, linkPath)
	if err != nil {
		if err == storage.ErrFileNotFound {
			return "", ErrManifestNotFound
		}
		return "", fmt.Errorf("failed to read manifest meta: %w", err)
	}
	data, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		return "", fmt.Errorf("failed to read manifest meta data: %w", err)
	}

	parts := strings.SplitN(string(data), "\n", 2)
	if len(parts) > 1 {
		return strings.TrimSpace(parts[1]), nil
	}
	return "", nil
}

// isDigestReference checks if a reference looks like a digest (contains ":").
func isDigestReference(ref string) bool {
	return strings.Contains(ref, ":")
}
