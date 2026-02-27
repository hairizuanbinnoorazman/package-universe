package handlers

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/hairizuanbinnoorazman/package-universe/oci"
)

// HeadManifest handles HEAD /v2/{name}/manifests/{reference} — check manifest existence.
func (h *OCIHandler) HeadManifest(w http.ResponseWriter, r *http.Request) {
	setOCIHeaders(w)
	ctx := r.Context()

	vars := mux.Vars(r)
	name := vars["name"]
	reference := vars["reference"]

	digest, contentType, size, err := h.Storage.ManifestExists(ctx, name, reference)
	if err != nil {
		if errors.Is(err, oci.ErrManifestNotFound) {
			respondOCIError(w, http.StatusNotFound, OCIErrorManifestUnknown, "manifest not found")
			return
		}
		h.Logger.Error(ctx, "failed to check manifest", map[string]interface{}{"error": err.Error()})
		respondOCIError(w, http.StatusInternalServerError, OCIErrorManifestUnknown, "internal error")
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	w.Header().Set("Docker-Content-Digest", digest.String())
	w.WriteHeader(http.StatusOK)
}

// GetManifest handles GET /v2/{name}/manifests/{reference} — download manifest.
func (h *OCIHandler) GetManifest(w http.ResponseWriter, r *http.Request) {
	setOCIHeaders(w)
	ctx := r.Context()

	vars := mux.Vars(r)
	name := vars["name"]
	reference := vars["reference"]

	data, digest, contentType, err := h.Storage.GetManifest(ctx, name, reference)
	if err != nil {
		if errors.Is(err, oci.ErrManifestNotFound) {
			respondOCIError(w, http.StatusNotFound, OCIErrorManifestUnknown, "manifest not found")
			return
		}
		h.Logger.Error(ctx, "failed to get manifest", map[string]interface{}{"error": err.Error()})
		respondOCIError(w, http.StatusInternalServerError, OCIErrorManifestUnknown, "internal error")
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Header().Set("Docker-Content-Digest", digest.String())
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// PutManifest handles PUT /v2/{name}/manifests/{reference} — upload manifest.
func (h *OCIHandler) PutManifest(w http.ResponseWriter, r *http.Request) {
	setOCIHeaders(w)
	ctx := r.Context()

	vars := mux.Vars(r)
	name := vars["name"]
	reference := vars["reference"]

	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/vnd.oci.image.manifest.v1+json"
	}

	data, err := io.ReadAll(r.Body)
	if err != nil {
		h.Logger.Error(ctx, "failed to read manifest body", map[string]interface{}{"error": err.Error()})
		respondOCIError(w, http.StatusBadRequest, OCIErrorManifestInvalid, "failed to read manifest")
		return
	}

	digest, err := h.Storage.PutManifest(ctx, name, reference, contentType, data)
	if err != nil {
		h.Logger.Error(ctx, "failed to put manifest", map[string]interface{}{"error": err.Error()})
		respondOCIError(w, http.StatusInternalServerError, OCIErrorManifestInvalid, "failed to store manifest")
		return
	}

	w.Header().Set("Location", "/v2/"+name+"/manifests/"+digest.String())
	w.Header().Set("Docker-Content-Digest", digest.String())
	w.WriteHeader(http.StatusCreated)
}
