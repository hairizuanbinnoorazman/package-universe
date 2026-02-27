package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/hairizuanbinnoorazman/package-universe/logger"
	"github.com/hairizuanbinnoorazman/package-universe/oci"
	"github.com/hairizuanbinnoorazman/package-universe/storage"
)

func setupTestOCIHandler(t *testing.T) (*OCIHandler, *mux.Router) {
	t.Helper()
	baseDir := t.TempDir()
	store, err := storage.NewLocalStorage(baseDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	sessions := oci.NewSessionManager(30 * time.Minute)
	ociStorage := oci.NewOCIStorage(store, sessions)

	handler := &OCIHandler{
		Storage: ociStorage,
		Logger:  logger.NewTestLogger(),
	}

	router := mux.NewRouter()

	// Register all OCI routes
	router.HandleFunc("/v2/", handler.V2Check).Methods("GET")
	router.HandleFunc("/v2/{name:.+}/blobs/uploads/", handler.InitiateBlobUpload).Methods("POST")
	router.HandleFunc("/v2/{name:.+}/blobs/uploads/{uuid}", handler.PatchBlobUpload).Methods("PATCH")
	router.HandleFunc("/v2/{name:.+}/blobs/uploads/{uuid}", handler.CompleteBlobUpload).Methods("PUT")
	router.HandleFunc("/v2/{name:.+}/blobs/uploads/{uuid}", handler.CancelBlobUpload).Methods("DELETE")
	router.HandleFunc("/v2/{name:.+}/blobs/{digest}", handler.HeadBlob).Methods("HEAD")
	router.HandleFunc("/v2/{name:.+}/blobs/{digest}", handler.GetBlob).Methods("GET")
	router.HandleFunc("/v2/{name:.+}/manifests/{reference}", handler.HeadManifest).Methods("HEAD")
	router.HandleFunc("/v2/{name:.+}/manifests/{reference}", handler.GetManifest).Methods("GET")
	router.HandleFunc("/v2/{name:.+}/manifests/{reference}", handler.PutManifest).Methods("PUT")
	router.HandleFunc("/v2/{name:.+}/tags/list", handler.TagsList).Methods("GET")

	return handler, router
}

func TestV2Check(t *testing.T) {
	_, router := setupTestOCIHandler(t)

	req := httptest.NewRequest("GET", "/v2/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	apiVersion := w.Header().Get("Docker-Distribution-API-Version")
	if apiVersion != "registry/2.0" {
		t.Errorf("API version = %q, want %q", apiVersion, "registry/2.0")
	}
}
