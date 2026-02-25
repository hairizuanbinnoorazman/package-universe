package handlers

import (
	"net/http"
)

// ReadyResponse represents the readiness check response.
type ReadyResponse struct {
	Status string `json:"status"`
}

// ReadyHandler handles readiness check requests.
func ReadyHandler(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, ReadyResponse{Status: "ready"})
}
