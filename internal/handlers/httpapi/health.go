package httpapi

import (
	"encoding/json"
	"net/http"

	api "github.com/lovelykeycap/TripGo/internal/generated"
)

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	hr := api.HealthResponse{
		Status: api.Ok,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err := json.NewEncoder(w).Encode(hr)

	if err != nil {
		h.logger.Error("failed to write health response", "error", err)
	}
}
