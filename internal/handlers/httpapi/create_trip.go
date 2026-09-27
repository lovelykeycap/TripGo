package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
	api "github.com/lovelykeycap/TripGo/internal/generated"
	tripservice "github.com/lovelykeycap/TripGo/internal/service/trip"
)

func (h *Handler) CreateTrip(w http.ResponseWriter, r *http.Request, _ api.CreateTripParams) {
	request, err := decodeCreateTripRequest(r)
	if err != nil {
		h.HandleRequestError(w, r, err)
		return
	}

	input := tripservice.CreateInput{
		UserID:   request.UserId,
		DriverID: request.DriverId,
		StartPoint: domaintrip.Coordinates{
			Latitude:  request.StartPoint.Latitude,
			Longitude: request.StartPoint.Longitude,
		},
		EndPoint: domaintrip.Coordinates{
			Latitude:  request.EndPoint.Latitude,
			Longitude: request.EndPoint.Longitude,
		},
		Price: request.Price,
	}
	result, err := h.tripService.Create(r.Context(), input)
	if err != nil {
		if errors.Is(err, domaintrip.ErrDriverBusy) {
			h.writeProblem(w, r, problemDetails{
				Status: http.StatusConflict,
				Code:   "driver_busy",
				Title:  "Driver busy",
				Detail: "Driver already has an active trip",
			})
			return
		}
		h.logger.Error("failed to create trip", "error", err)
		h.writeProblem(w, r, problemDetails{
			Status: http.StatusInternalServerError,
			Code:   "internal_error",
			Title:  "Internal error",
			Detail: "Unexpected server error",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", "/api/v1/trips/"+result.ID.String())
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(toAPITrip(result)); err != nil {
		h.logger.Error("failed to write trip response", "error", err)
	}
}
