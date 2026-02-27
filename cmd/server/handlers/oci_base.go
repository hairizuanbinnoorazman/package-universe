package handlers

import "net/http"

// V2Check handles GET /v2/ — OCI API version check.
func (h *OCIHandler) V2Check(w http.ResponseWriter, r *http.Request) {
	setOCIHeaders(w)
	w.WriteHeader(http.StatusOK)
}
