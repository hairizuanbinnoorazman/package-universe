package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/hairizuanbinnoorazman/package-universe/logger"
	"github.com/hairizuanbinnoorazman/package-universe/oci"
)

// OCI error codes per the distribution spec.
const (
	OCIErrorBlobUnknown         = "BLOB_UNKNOWN"
	OCIErrorBlobUploadInvalid   = "BLOB_UPLOAD_INVALID"
	OCIErrorBlobUploadUnknown   = "BLOB_UPLOAD_UNKNOWN"
	OCIErrorDigestInvalid       = "DIGEST_INVALID"
	OCIErrorManifestBlobUnknown = "MANIFEST_BLOB_UNKNOWN"
	OCIErrorManifestInvalid     = "MANIFEST_INVALID"
	OCIErrorManifestUnknown     = "MANIFEST_UNKNOWN"
	OCIErrorNameInvalid         = "NAME_INVALID"
	OCIErrorNameUnknown         = "NAME_UNKNOWN"
	OCIErrorSizeInvalid         = "SIZE_INVALID"
	OCIErrorUnauthorized        = "UNAUTHORIZED"
	OCIErrorUnsupported         = "UNSUPPORTED"
)

// OCIHandler holds dependencies for OCI registry handlers.
type OCIHandler struct {
	Storage *oci.OCIStorage
	Logger  logger.Logger
}

// ociError represents a single OCI error in the response.
type ociError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

// ociErrorResponse is the OCI error response envelope.
type ociErrorResponse struct {
	Errors []ociError `json:"errors"`
}

// respondOCIError writes an OCI-spec error response.
func respondOCIError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Docker-Distribution-API-Version", "registry/2.0")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ociErrorResponse{
		Errors: []ociError{{Code: code, Message: message}},
	})
}

// setOCIHeaders sets the standard OCI response headers.
func setOCIHeaders(w http.ResponseWriter) {
	w.Header().Set("Docker-Distribution-API-Version", "registry/2.0")
}
