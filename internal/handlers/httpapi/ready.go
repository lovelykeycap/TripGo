package httpapi

import (
	"encoding/json"
	"net/http"

	api "github.com/lovelykeycap/TripGo/internal/generated"
)

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	statusCode := http.StatusOK
	response := api.HealthResponse{Status: api.Ok}

	if err := h.readiness.Check(r.Context()); err != nil {
		statusCode = http.StatusServiceUnavailable
		response.Status = api.Unavailable
		h.logger.Warn("readiness check failed", "error", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("failed to write readiness response", "error", err)
	}
}
