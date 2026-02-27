package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTagsList(t *testing.T) {
	_, router := setupTestOCIHandler(t)

	ct := "application/vnd.oci.image.manifest.v1+json"

	// Push two tags
	manifest1 := []byte(`{"schemaVersion":2,"variant":"a"}`)
	req := httptest.NewRequest("PUT", "/v2/myrepo/manifests/v1.0", bytes.NewReader(manifest1))
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("PUT v1.0: status = %d, body = %s", w.Code, w.Body.String())
	}

	manifest2 := []byte(`{"schemaVersion":2,"variant":"b"}`)
	req = httptest.NewRequest("PUT", "/v2/myrepo/manifests/v2.0", bytes.NewReader(manifest2))
	req.Header.Set("Content-Type", ct)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("PUT v2.0: status = %d, body = %s", w.Code, w.Body.String())
	}

	// List tags
	req = httptest.NewRequest("GET", "/v2/myrepo/tags/list", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET: status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp tagsListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Name != "myrepo" {
		t.Errorf("name = %q, want %q", resp.Name, "myrepo")
	}
	if len(resp.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d: %v", len(resp.Tags), resp.Tags)
	}

	tagSet := make(map[string]bool)
	for _, tag := range resp.Tags {
		tagSet[tag] = true
	}
	if !tagSet["v1.0"] {
		t.Error("missing tag v1.0")
	}
	if !tagSet["v2.0"] {
		t.Error("missing tag v2.0")
	}
}

func TestTagsListEmpty(t *testing.T) {
	_, router := setupTestOCIHandler(t)

	req := httptest.NewRequest("GET", "/v2/nonexistent/tags/list", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp tagsListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Tags) != 0 {
		t.Errorf("expected 0 tags, got %d", len(resp.Tags))
	}
}
