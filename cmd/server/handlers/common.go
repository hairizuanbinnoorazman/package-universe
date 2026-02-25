package handlers

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error string `json:"error"`
}

// respondJSON writes a JSON response with the given status code.
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError writes an error response with the given status code.
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, ErrorResponse{Error: message})
}
