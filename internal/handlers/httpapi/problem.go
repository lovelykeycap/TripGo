package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	api "github.com/lovelykeycap/TripGo/internal/generated"
)

type problemDetails struct {
	Status int
	Code   string
	Title  string
	Detail string
}

func (h *Handler) HandleRequestError(w http.ResponseWriter, r *http.Request, _ error) {
	h.writeProblem(w, r, problemDetails{
		Status: http.StatusBadRequest,
		Code:   "invalid_request",
		Title:  "Invalid request",
		Detail: "Request validation failed",
	})
}

func (h *Handler) writeProblem(w http.ResponseWriter, r *http.Request, details problemDetails) {
	instance := r.URL.Path
	problem := api.Problem{
		Type:     "https://tripgo.example/problems/" + strings.ReplaceAll(details.Code, "_", "-"),
		Title:    details.Title,
		Status:   int32(details.Status),
		Code:     details.Code,
		Detail:   &details.Detail,
		Instance: &instance,
	}

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(details.Status)
	if err := json.NewEncoder(w).Encode(problem); err != nil {
		h.logger.Error("failed to write problem response", "error", err)
	}
}
