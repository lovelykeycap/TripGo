package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
	api "github.com/lovelykeycap/TripGo/internal/generated"
)

func (h *Handler) FinishTrip(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	result, err := h.tripService.Finish(r.Context(), tripID)
	if err != nil {
		switch {
		case errors.Is(err, domaintrip.ErrTripNotFound):
			h.writeProblem(w, r, problemDetails{
				Status: http.StatusNotFound,
				Code:   "trip_not_found",
				Title:  "Trip not found",
				Detail: "Trip does not exist",
			})
		case errors.Is(err, domaintrip.ErrTripCompleted):
			h.writeProblem(w, r, problemDetails{
				Status: http.StatusConflict,
				Code:   "trip_completed",
				Title:  "Trip completed",
				Detail: "Operation is not allowed for a completed trip",
			})
		default:
			h.logger.Error("failed to finish trip", "error", err)
			h.writeProblem(w, r, problemDetails{
				Status: http.StatusInternalServerError,
				Code:   "internal_error",
				Title:  "Internal error",
				Detail: "Unexpected server error",
			})
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(toAPITrip(result)); err != nil {
		h.logger.Error("failed to write trip response", "error", err)
	}
}
