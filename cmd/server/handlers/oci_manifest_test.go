package handlers

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestManifestPushPull(t *testing.T) {
	_, router := setupTestOCIHandler(t)

	manifestData := []byte(`{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json","config":{},"layers":[]}`)
	contentType := "application/vnd.oci.image.manifest.v1+json"
	expectedDigest := fmt.Sprintf("sha256:%x", sha256.Sum256(manifestData))

	// PUT manifest
	req := httptest.NewRequest("PUT", "/v2/myrepo/myimage/manifests/latest", bytes.NewReader(manifestData))
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("PUT: status = %d, want %d, body = %s", w.Code, http.StatusCreated, w.Body.String())
	}
	if w.Header().Get("Docker-Content-Digest") != expectedDigest {
		t.Errorf("digest = %q, want %q", w.Header().Get("Docker-Content-Digest"), expectedDigest)
	}

	// GET manifest by tag
	req = httptest.NewRequest("GET", "/v2/myrepo/myimage/manifests/latest", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET by tag: status = %d, want %d", w.Code, http.StatusOK)
	}
	if !bytes.Equal(w.Body.Bytes(), manifestData) {
		t.Error("manifest data mismatch")
	}
	if w.Header().Get("Content-Type") != contentType {
		t.Errorf("content type = %q, want %q", w.Header().Get("Content-Type"), contentType)
	}
	if w.Header().Get("Docker-Content-Digest") != expectedDigest {
		t.Errorf("digest = %q, want %q", w.Header().Get("Docker-Content-Digest"), expectedDigest)
	}

	// GET manifest by digest
	req = httptest.NewRequest("GET", fmt.Sprintf("/v2/myrepo/myimage/manifests/%s", expectedDigest), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET by digest: status = %d, want %d", w.Code, http.StatusOK)
	}
	if !bytes.Equal(w.Body.Bytes(), manifestData) {
		t.Error("manifest data mismatch when pulled by digest")
	}

	// HEAD manifest
	req = httptest.NewRequest("HEAD", "/v2/myrepo/myimage/manifests/latest", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HEAD: status = %d, want %d", w.Code, http.StatusOK)
	}
	if w.Header().Get("Docker-Content-Digest") != expectedDigest {
		t.Errorf("HEAD digest = %q, want %q", w.Header().Get("Docker-Content-Digest"), expectedDigest)
	}
	if w.Header().Get("Content-Type") != contentType {
		t.Errorf("HEAD content type = %q, want %q", w.Header().Get("Content-Type"), contentType)
	}
}

func TestManifestNotFound(t *testing.T) {
	_, router := setupTestOCIHandler(t)

	req := httptest.NewRequest("GET", "/v2/nonexistent/manifests/latest", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("GET: status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHeadManifestNotFound(t *testing.T) {
	_, router := setupTestOCIHandler(t)

	req := httptest.NewRequest("HEAD", "/v2/nonexistent/manifests/latest", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("HEAD: status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestManifestMultipleTags(t *testing.T) {
	_, router := setupTestOCIHandler(t)

	manifest1 := []byte(`{"schemaVersion":2,"tag":"v1"}`)
	manifest2 := []byte(`{"schemaVersion":2,"tag":"v2"}`)
	ct := "application/vnd.oci.image.manifest.v1+json"

	// Push v1
	req := httptest.NewRequest("PUT", "/v2/myrepo/manifests/v1.0", bytes.NewReader(manifest1))
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("PUT v1: status = %d", w.Code)
	}

	// Push v2
	req = httptest.NewRequest("PUT", "/v2/myrepo/manifests/v2.0", bytes.NewReader(manifest2))
	req.Header.Set("Content-Type", ct)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("PUT v2: status = %d", w.Code)
	}

	// Get v1
	req = httptest.NewRequest("GET", "/v2/myrepo/manifests/v1.0", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if !bytes.Equal(w.Body.Bytes(), manifest1) {
		t.Error("v1 manifest data mismatch")
	}

	// Get v2
	req = httptest.NewRequest("GET", "/v2/myrepo/manifests/v2.0", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if !bytes.Equal(w.Body.Bytes(), manifest2) {
		t.Error("v2 manifest data mismatch")
	}
}
