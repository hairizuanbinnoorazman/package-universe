package oci

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"testing"
	"time"

	"github.com/hairizuanbinnoorazman/package-universe/storage"
)

func setupTestOCIStorage(t *testing.T) *OCIStorage {
	t.Helper()
	baseDir := t.TempDir()
	store, err := storage.NewLocalStorage(baseDir)
	if err != nil {
		t.Fatalf("failed to create local storage: %v", err)
	}
	sessions := NewSessionManager(30 * time.Minute)
	return NewOCIStorage(store, sessions)
}

func computeSHA256(data []byte) DigestInfo {
	h := sha256.Sum256(data)
	return DigestInfo{
		Algorithm: "sha256",
		Hex:       fmt.Sprintf("%x", h[:]),
	}
}

func TestOCIStorage_MonolithicUpload(t *testing.T) {
	ctx := context.Background()
	s := setupTestOCIStorage(t)

	blobData := []byte("hello world blob content")
	expectedDigest := computeSHA256(blobData)

	// Initiate upload
	uuid, err := s.InitiateUpload(ctx, "myrepo")
	if err != nil {
		t.Fatalf("InitiateUpload failed: %v", err)
	}

	// Write all data
	_, err = s.WriteUploadChunk(ctx, uuid, bytes.NewReader(blobData))
	if err != nil {
		t.Fatalf("WriteUploadChunk failed: %v", err)
	}

	// Complete upload
	digest, err := s.CompleteUpload(ctx, uuid, expectedDigest)
	if err != nil {
		t.Fatalf("CompleteUpload failed: %v", err)
	}
	if digest.String() != expectedDigest.String() {
		t.Errorf("digest = %q, want %q", digest.String(), expectedDigest.String())
	}

	// Verify blob exists
	exists, err := s.BlobExists(ctx, expectedDigest)
	if err != nil {
		t.Fatalf("BlobExists failed: %v", err)
	}
	if !exists {
		t.Error("blob should exist after upload")
	}

	// Download and verify
	rc, err := s.GetBlob(ctx, expectedDigest)
	if err != nil {
		t.Fatalf("GetBlob failed: %v", err)
	}
	defer rc.Close()

	var buf bytes.Buffer
	buf.ReadFrom(rc)
	if !bytes.Equal(buf.Bytes(), blobData) {
		t.Errorf("downloaded data mismatch")
	}
}

func TestOCIStorage_ChunkedUpload(t *testing.T) {
	ctx := context.Background()
	s := setupTestOCIStorage(t)

	chunk1 := []byte("first chunk of data ")
	chunk2 := []byte("second chunk of data")
	fullData := append(chunk1, chunk2...)
	expectedDigest := computeSHA256(fullData)

	// Initiate upload
	uuid, err := s.InitiateUpload(ctx, "myrepo")
	if err != nil {
		t.Fatalf("InitiateUpload failed: %v", err)
	}

	// Write chunk 1
	_, err = s.WriteUploadChunk(ctx, uuid, bytes.NewReader(chunk1))
	if err != nil {
		t.Fatalf("WriteUploadChunk chunk1 failed: %v", err)
	}

	// Write chunk 2
	_, err = s.WriteUploadChunk(ctx, uuid, bytes.NewReader(chunk2))
	if err != nil {
		t.Fatalf("WriteUploadChunk chunk2 failed: %v", err)
	}

	// Complete upload
	digest, err := s.CompleteUpload(ctx, uuid, expectedDigest)
	if err != nil {
		t.Fatalf("CompleteUpload failed: %v", err)
	}
	if digest.String() != expectedDigest.String() {
		t.Errorf("digest = %q, want %q", digest.String(), expectedDigest.String())
	}

	// Verify blob exists and content
	info, err := s.GetBlobInfo(ctx, expectedDigest)
	if err != nil {
		t.Fatalf("GetBlobInfo failed: %v", err)
	}
	if info.Size != int64(len(fullData)) {
		t.Errorf("size = %d, want %d", info.Size, len(fullData))
	}
}

func TestOCIStorage_DigestMismatch(t *testing.T) {
	ctx := context.Background()
	s := setupTestOCIStorage(t)

	blobData := []byte("some data")
	wrongDigest := DigestInfo{Algorithm: "sha256", Hex: "0000000000000000000000000000000000000000000000000000000000000000"}

	uuid, _ := s.InitiateUpload(ctx, "myrepo")
	s.WriteUploadChunk(ctx, uuid, bytes.NewReader(blobData))

	_, err := s.CompleteUpload(ctx, uuid, wrongDigest)
	if err == nil {
		t.Fatal("expected error for digest mismatch")
	}
}

func TestOCIStorage_CancelUpload(t *testing.T) {
	ctx := context.Background()
	s := setupTestOCIStorage(t)

	uuid, _ := s.InitiateUpload(ctx, "myrepo")
	s.WriteUploadChunk(ctx, uuid, bytes.NewReader([]byte("data")))

	err := s.CancelUpload(ctx, uuid)
	if err != nil {
		t.Fatalf("CancelUpload failed: %v", err)
	}

	// Should not be accessible anymore
	_, err = s.WriteUploadChunk(ctx, uuid, bytes.NewReader([]byte("more")))
	if err != ErrUploadNotFound {
		t.Errorf("expected ErrUploadNotFound after cancel, got %v", err)
	}
}

func TestOCIStorage_ManifestPushPull(t *testing.T) {
	ctx := context.Background()
	s := setupTestOCIStorage(t)

	manifestData := []byte(`{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json"}`)
	contentType := "application/vnd.oci.image.manifest.v1+json"

	// Push manifest with tag
	digest, err := s.PutManifest(ctx, "myrepo/myimage", "latest", contentType, manifestData)
	if err != nil {
		t.Fatalf("PutManifest failed: %v", err)
	}

	// Pull by tag
	data, gotDigest, gotCT, err := s.GetManifest(ctx, "myrepo/myimage", "latest")
	if err != nil {
		t.Fatalf("GetManifest by tag failed: %v", err)
	}
	if !bytes.Equal(data, manifestData) {
		t.Error("manifest data mismatch")
	}
	if gotDigest.String() != digest.String() {
		t.Errorf("digest = %q, want %q", gotDigest.String(), digest.String())
	}
	if gotCT != contentType {
		t.Errorf("content type = %q, want %q", gotCT, contentType)
	}

	// Pull by digest
	data2, gotDigest2, _, err := s.GetManifest(ctx, "myrepo/myimage", digest.String())
	if err != nil {
		t.Fatalf("GetManifest by digest failed: %v", err)
	}
	if !bytes.Equal(data2, manifestData) {
		t.Error("manifest data mismatch when pulled by digest")
	}
	if gotDigest2.String() != digest.String() {
		t.Errorf("digest = %q, want %q", gotDigest2.String(), digest.String())
	}

	// Head manifest
	headDigest, headCT, headSize, err := s.ManifestExists(ctx, "myrepo/myimage", "latest")
	if err != nil {
		t.Fatalf("ManifestExists failed: %v", err)
	}
	if headDigest.String() != digest.String() {
		t.Errorf("head digest = %q, want %q", headDigest.String(), digest.String())
	}
	if headCT != contentType {
		t.Errorf("head content type = %q, want %q", headCT, contentType)
	}
	if headSize != int64(len(manifestData)) {
		t.Errorf("head size = %d, want %d", headSize, len(manifestData))
	}
}

func TestOCIStorage_ManifestNotFound(t *testing.T) {
	ctx := context.Background()
	s := setupTestOCIStorage(t)

	_, _, _, err := s.GetManifest(ctx, "nonexistent", "latest")
	if err != ErrManifestNotFound {
		t.Errorf("expected ErrManifestNotFound, got %v", err)
	}
}

func TestOCIStorage_ListTags(t *testing.T) {
	ctx := context.Background()
	s := setupTestOCIStorage(t)

	manifest1 := []byte(`{"schemaVersion":2,"tag":"v1"}`)
	manifest2 := []byte(`{"schemaVersion":2,"tag":"v2"}`)
	ct := "application/vnd.oci.image.manifest.v1+json"

	s.PutManifest(ctx, "myrepo", "v1.0", ct, manifest1)
	s.PutManifest(ctx, "myrepo", "v2.0", ct, manifest2)

	tags, err := s.ListTags(ctx, "myrepo")
	if err != nil {
		t.Fatalf("ListTags failed: %v", err)
	}
	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d: %v", len(tags), tags)
	}

	// Check both tags are present
	tagSet := make(map[string]bool)
	for _, tag := range tags {
		tagSet[tag] = true
	}
	if !tagSet["v1.0"] {
		t.Error("missing tag v1.0")
	}
	if !tagSet["v2.0"] {
		t.Error("missing tag v2.0")
	}
}

func TestOCIStorage_ListTagsEmpty(t *testing.T) {
	ctx := context.Background()
	s := setupTestOCIStorage(t)

	tags, err := s.ListTags(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("ListTags failed: %v", err)
	}
	if len(tags) != 0 {
		t.Errorf("expected 0 tags, got %d", len(tags))
	}
}

func TestOCIStorage_BlobNotFound(t *testing.T) {
	ctx := context.Background()
	s := setupTestOCIStorage(t)

	d := DigestInfo{Algorithm: "sha256", Hex: "0000000000000000000000000000000000000000000000000000000000000000"}

	exists, err := s.BlobExists(ctx, d)
	if err != nil {
		t.Fatalf("BlobExists failed: %v", err)
	}
	if exists {
		t.Error("blob should not exist")
	}

	_, err = s.GetBlob(ctx, d)
	if err != ErrBlobNotFound {
		t.Errorf("expected ErrBlobNotFound, got %v", err)
	}

	_, err = s.GetBlobInfo(ctx, d)
	if err != ErrBlobNotFound {
		t.Errorf("expected ErrBlobNotFound, got %v", err)
	}
}
