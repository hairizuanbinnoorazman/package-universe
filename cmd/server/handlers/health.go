package handlers

import (
	"net/http"
)

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status string `json:"status"`
}

// HealthHandler handles health check requests.
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, HealthResponse{Status: "healthy"})
}
