package handlers

import (
	"net/http"

	"github.com/gorilla/mux"
)

// tagsListResponse is the OCI tags list response.
type tagsListResponse struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

// TagsList handles GET /v2/{name}/tags/list — list repository tags.
func (h *OCIHandler) TagsList(w http.ResponseWriter, r *http.Request) {
	setOCIHeaders(w)
	ctx := r.Context()

	vars := mux.Vars(r)
	name := vars["name"]

	tags, err := h.Storage.ListTags(ctx, name)
	if err != nil {
		h.Logger.Error(ctx, "failed to list tags", map[string]interface{}{"error": err.Error()})
		respondOCIError(w, http.StatusInternalServerError, OCIErrorNameUnknown, "failed to list tags")
		return
	}

	if tags == nil {
		tags = []string{}
	}

	respondJSON(w, http.StatusOK, tagsListResponse{
		Name: name,
		Tags: tags,
	})
}
