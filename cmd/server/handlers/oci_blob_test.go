package handlers

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBlobUploadChunked(t *testing.T) {
	_, router := setupTestOCIHandler(t)

	// Step 1: Initiate upload
	req := httptest.NewRequest("POST", "/v2/myrepo/blobs/uploads/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("initiate: status = %d, want %d", w.Code, http.StatusAccepted)
	}
	location := w.Header().Get("Location")
	if location == "" {
		t.Fatal("initiate: missing Location header")
	}
	uuid := w.Header().Get("Docker-Upload-UUID")
	if uuid == "" {
		t.Fatal("initiate: missing Docker-Upload-UUID header")
	}

	// Step 2: PATCH chunk
	blobData := []byte("hello world blob data")
	req = httptest.NewRequest("PATCH", location, bytes.NewReader(blobData))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("patch: status = %d, want %d", w.Code, http.StatusAccepted)
	}

	// Step 3: PUT to complete
	digest := fmt.Sprintf("sha256:%x", sha256.Sum256(blobData))
	completeURL := fmt.Sprintf("%s?digest=%s", location, digest)
	req = httptest.NewRequest("PUT", completeURL, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		body, _ := io.ReadAll(w.Body)
		t.Fatalf("complete: status = %d, want %d, body = %s", w.Code, http.StatusCreated, string(body))
	}
	if w.Header().Get("Docker-Content-Digest") != digest {
		t.Errorf("digest header = %q, want %q", w.Header().Get("Docker-Content-Digest"), digest)
	}

	// Verify blob exists via HEAD
	req = httptest.NewRequest("HEAD", fmt.Sprintf("/v2/myrepo/blobs/%s", digest), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("HEAD: status = %d, want %d", w.Code, http.StatusOK)
	}

	// Verify blob content via GET
	req = httptest.NewRequest("GET", fmt.Sprintf("/v2/myrepo/blobs/%s", digest), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET: status = %d, want %d", w.Code, http.StatusOK)
	}
	if !bytes.Equal(w.Body.Bytes(), blobData) {
		t.Errorf("blob content mismatch")
	}
}

func TestBlobUploadMonolithic(t *testing.T) {
	_, router := setupTestOCIHandler(t)

	blobData := []byte("monolithic blob data")
	digest := fmt.Sprintf("sha256:%x", sha256.Sum256(blobData))

	req := httptest.NewRequest("POST", fmt.Sprintf("/v2/myrepo/blobs/uploads/?digest=%s", digest), bytes.NewReader(blobData))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		body, _ := io.ReadAll(w.Body)
		t.Fatalf("status = %d, want %d, body = %s", w.Code, http.StatusCreated, string(body))
	}
	if w.Header().Get("Docker-Content-Digest") != digest {
		t.Errorf("digest = %q, want %q", w.Header().Get("Docker-Content-Digest"), digest)
	}
}

func TestBlobHeadNotFound(t *testing.T) {
	_, router := setupTestOCIHandler(t)

	req := httptest.NewRequest("HEAD", "/v2/myrepo/blobs/sha256:0000000000000000000000000000000000000000000000000000000000000000", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestBlobGetNotFound(t *testing.T) {
	_, router := setupTestOCIHandler(t)

	req := httptest.NewRequest("GET", "/v2/myrepo/blobs/sha256:0000000000000000000000000000000000000000000000000000000000000000", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestBlobInvalidDigest(t *testing.T) {
	_, router := setupTestOCIHandler(t)

	req := httptest.NewRequest("HEAD", "/v2/myrepo/blobs/invalid-digest", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var errResp ociErrorResponse
	json.NewDecoder(w.Body).Decode(&errResp)
	if len(errResp.Errors) == 0 || errResp.Errors[0].Code != OCIErrorDigestInvalid {
		t.Errorf("expected DIGEST_INVALID error code")
	}
}

func TestCancelUpload(t *testing.T) {
	_, router := setupTestOCIHandler(t)

	// Initiate
	req := httptest.NewRequest("POST", "/v2/myrepo/blobs/uploads/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	location := w.Header().Get("Location")

	// Cancel
	req = httptest.NewRequest("DELETE", location, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("cancel: status = %d, want %d", w.Code, http.StatusNoContent)
	}

	// Try to PATCH after cancel — should fail
	req = httptest.NewRequest("PATCH", location, strings.NewReader("data"))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("patch after cancel: status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestCancelUploadNotFound(t *testing.T) {
	_, router := setupTestOCIHandler(t)

	req := httptest.NewRequest("DELETE", "/v2/myrepo/blobs/uploads/nonexistent-uuid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestCompleteBlobUploadMissingDigest(t *testing.T) {
	_, router := setupTestOCIHandler(t)

	// Initiate
	req := httptest.NewRequest("POST", "/v2/myrepo/blobs/uploads/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	location := w.Header().Get("Location")

	// PUT without digest query param
	req = httptest.NewRequest("PUT", location, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestBlobDigestMismatch(t *testing.T) {
	_, router := setupTestOCIHandler(t)

	blobData := []byte("some data")
	wrongDigest := "sha256:0000000000000000000000000000000000000000000000000000000000000000"

	// Monolithic upload with wrong digest
	req := httptest.NewRequest("POST", fmt.Sprintf("/v2/myrepo/blobs/uploads/?digest=%s", wrongDigest), bytes.NewReader(blobData))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
