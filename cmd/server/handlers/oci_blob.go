package handlers

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/hairizuanbinnoorazman/package-universe/oci"
)

// HeadBlob handles HEAD /v2/{name}/blobs/{digest} — check blob existence.
func (h *OCIHandler) HeadBlob(w http.ResponseWriter, r *http.Request) {
	setOCIHeaders(w)
	ctx := r.Context()

	vars := mux.Vars(r)
	digestStr := vars["digest"]

	digest, err := oci.ParseDigest(digestStr)
	if err != nil {
		respondOCIError(w, http.StatusBadRequest, OCIErrorDigestInvalid, "invalid digest format")
		return
	}

	info, err := h.Storage.GetBlobInfo(ctx, digest)
	if err != nil {
		if errors.Is(err, oci.ErrBlobNotFound) {
			respondOCIError(w, http.StatusNotFound, OCIErrorBlobUnknown, "blob not found")
			return
		}
		h.Logger.Error(ctx, "failed to get blob info", map[string]interface{}{"error": err.Error()})
		respondOCIError(w, http.StatusInternalServerError, OCIErrorBlobUnknown, "internal error")
		return
	}

	w.Header().Set("Content-Length", strconv.FormatInt(info.Size, 10))
	w.Header().Set("Docker-Content-Digest", info.Digest.String())
	w.WriteHeader(http.StatusOK)
}

// GetBlob handles GET /v2/{name}/blobs/{digest} — download blob.
func (h *OCIHandler) GetBlob(w http.ResponseWriter, r *http.Request) {
	setOCIHeaders(w)
	ctx := r.Context()

	vars := mux.Vars(r)
	digestStr := vars["digest"]

	digest, err := oci.ParseDigest(digestStr)
	if err != nil {
		respondOCIError(w, http.StatusBadRequest, OCIErrorDigestInvalid, "invalid digest format")
		return
	}

	rc, err := h.Storage.GetBlob(ctx, digest)
	if err != nil {
		if errors.Is(err, oci.ErrBlobNotFound) {
			respondOCIError(w, http.StatusNotFound, OCIErrorBlobUnknown, "blob not found")
			return
		}
		h.Logger.Error(ctx, "failed to get blob", map[string]interface{}{"error": err.Error()})
		respondOCIError(w, http.StatusInternalServerError, OCIErrorBlobUnknown, "internal error")
		return
	}
	defer rc.Close()

	w.Header().Set("Docker-Content-Digest", digest.String())
	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)
	io.Copy(w, rc)
}

// InitiateBlobUpload handles POST /v2/{name}/blobs/uploads/ — start an upload.
func (h *OCIHandler) InitiateBlobUpload(w http.ResponseWriter, r *http.Request) {
	setOCIHeaders(w)
	ctx := r.Context()

	vars := mux.Vars(r)
	name := vars["name"]

	// Check for monolithic upload (digest in query param with body)
	digestParam := r.URL.Query().Get("digest")
	if digestParam != "" {
		h.handleMonolithicUpload(w, r, name, digestParam)
		return
	}

	uuid, err := h.Storage.InitiateUpload(ctx, name)
	if err != nil {
		h.Logger.Error(ctx, "failed to initiate upload", map[string]interface{}{"error": err.Error()})
		respondOCIError(w, http.StatusInternalServerError, OCIErrorBlobUploadInvalid, "failed to initiate upload")
		return
	}

	w.Header().Set("Location", fmt.Sprintf("/v2/%s/blobs/uploads/%s", name, uuid))
	w.Header().Set("Docker-Upload-UUID", uuid)
	w.Header().Set("Range", "0-0")
	w.WriteHeader(http.StatusAccepted)
}

// handleMonolithicUpload handles a single-request blob upload (POST with digest query param).
func (h *OCIHandler) handleMonolithicUpload(w http.ResponseWriter, r *http.Request, name, digestStr string) {
	ctx := r.Context()

	expectedDigest, err := oci.ParseDigest(digestStr)
	if err != nil {
		respondOCIError(w, http.StatusBadRequest, OCIErrorDigestInvalid, "invalid digest format")
		return
	}

	uuid, err := h.Storage.InitiateUpload(ctx, name)
	if err != nil {
		h.Logger.Error(ctx, "failed to initiate monolithic upload", map[string]interface{}{"error": err.Error()})
		respondOCIError(w, http.StatusInternalServerError, OCIErrorBlobUploadInvalid, "failed to initiate upload")
		return
	}

	_, err = h.Storage.WriteUploadChunk(ctx, uuid, r.Body)
	if err != nil {
		h.Logger.Error(ctx, "failed to write monolithic upload", map[string]interface{}{"error": err.Error()})
		respondOCIError(w, http.StatusInternalServerError, OCIErrorBlobUploadInvalid, "failed to write data")
		return
	}

	digest, err := h.Storage.CompleteUpload(ctx, uuid, expectedDigest)
	if err != nil {
		if errors.Is(err, oci.ErrDigestMismatch) {
			respondOCIError(w, http.StatusBadRequest, OCIErrorDigestInvalid, "digest mismatch")
			return
		}
		h.Logger.Error(ctx, "failed to complete monolithic upload", map[string]interface{}{"error": err.Error()})
		respondOCIError(w, http.StatusInternalServerError, OCIErrorBlobUploadInvalid, "failed to complete upload")
		return
	}

	w.Header().Set("Location", fmt.Sprintf("/v2/%s/blobs/%s", name, digest.String()))
	w.Header().Set("Docker-Content-Digest", digest.String())
	w.WriteHeader(http.StatusCreated)
}

// PatchBlobUpload handles PATCH /v2/{name}/blobs/uploads/{uuid} — chunked upload data.
func (h *OCIHandler) PatchBlobUpload(w http.ResponseWriter, r *http.Request) {
	setOCIHeaders(w)
	ctx := r.Context()

	vars := mux.Vars(r)
	name := vars["name"]
	uuid := vars["uuid"]

	totalSize, err := h.Storage.WriteUploadChunk(ctx, uuid, r.Body)
	if err != nil {
		if errors.Is(err, oci.ErrUploadNotFound) {
			respondOCIError(w, http.StatusNotFound, OCIErrorBlobUploadUnknown, "upload not found")
			return
		}
		h.Logger.Error(ctx, "failed to write upload chunk", map[string]interface{}{"error": err.Error()})
		respondOCIError(w, http.StatusInternalServerError, OCIErrorBlobUploadInvalid, "failed to write chunk")
		return
	}

	w.Header().Set("Location", fmt.Sprintf("/v2/%s/blobs/uploads/%s", name, uuid))
	w.Header().Set("Docker-Upload-UUID", uuid)
	w.Header().Set("Range", fmt.Sprintf("0-%d", totalSize-1))
	w.WriteHeader(http.StatusAccepted)
}

// CompleteBlobUpload handles PUT /v2/{name}/blobs/uploads/{uuid}?digest= — finish upload.
func (h *OCIHandler) CompleteBlobUpload(w http.ResponseWriter, r *http.Request) {
	setOCIHeaders(w)
	ctx := r.Context()

	vars := mux.Vars(r)
	name := vars["name"]
	uuid := vars["uuid"]

	digestStr := r.URL.Query().Get("digest")
	if digestStr == "" {
		respondOCIError(w, http.StatusBadRequest, OCIErrorDigestInvalid, "digest query parameter required")
		return
	}

	expectedDigest, err := oci.ParseDigest(digestStr)
	if err != nil {
		respondOCIError(w, http.StatusBadRequest, OCIErrorDigestInvalid, "invalid digest format")
		return
	}

	// If there's a body, write it as the final chunk
	if r.ContentLength > 0 || r.ContentLength == -1 {
		_, err := h.Storage.WriteUploadChunk(ctx, uuid, r.Body)
		if err != nil && err != oci.ErrUploadNotFound {
			h.Logger.Error(ctx, "failed to write final chunk", map[string]interface{}{"error": err.Error()})
			respondOCIError(w, http.StatusInternalServerError, OCIErrorBlobUploadInvalid, "failed to write final chunk")
			return
		}
		if errors.Is(err, oci.ErrUploadNotFound) {
			respondOCIError(w, http.StatusNotFound, OCIErrorBlobUploadUnknown, "upload not found")
			return
		}
	}

	digest, err := h.Storage.CompleteUpload(ctx, uuid, expectedDigest)
	if err != nil {
		if errors.Is(err, oci.ErrUploadNotFound) {
			respondOCIError(w, http.StatusNotFound, OCIErrorBlobUploadUnknown, "upload not found")
			return
		}
		if errors.Is(err, oci.ErrDigestMismatch) {
			respondOCIError(w, http.StatusBadRequest, OCIErrorDigestInvalid, "digest mismatch")
			return
		}
		h.Logger.Error(ctx, "failed to complete upload", map[string]interface{}{"error": err.Error()})
		respondOCIError(w, http.StatusInternalServerError, OCIErrorBlobUploadInvalid, "failed to complete upload")
		return
	}

	w.Header().Set("Location", fmt.Sprintf("/v2/%s/blobs/%s", name, digest.String()))
	w.Header().Set("Docker-Content-Digest", digest.String())
	w.WriteHeader(http.StatusCreated)
}

// CancelBlobUpload handles DELETE /v2/{name}/blobs/uploads/{uuid} — cancel upload.
func (h *OCIHandler) CancelBlobUpload(w http.ResponseWriter, r *http.Request) {
	setOCIHeaders(w)
	ctx := r.Context()

	vars := mux.Vars(r)
	uuid := vars["uuid"]

	err := h.Storage.CancelUpload(ctx, uuid)
	if err != nil {
		if errors.Is(err, oci.ErrUploadNotFound) {
			respondOCIError(w, http.StatusNotFound, OCIErrorBlobUploadUnknown, "upload not found")
			return
		}
		h.Logger.Error(ctx, "failed to cancel upload", map[string]interface{}{"error": err.Error()})
		respondOCIError(w, http.StatusInternalServerError, OCIErrorBlobUploadInvalid, "failed to cancel upload")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
