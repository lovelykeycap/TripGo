package httpapi

import (
	"log/slog"

	api "github.com/lovelykeycap/TripGo/internal/generated"
	tripservice "github.com/lovelykeycap/TripGo/internal/service/trip"
)

type Handler struct {
	api.Unimplemented
	logger      *slog.Logger
	readiness   readinessChecker
	tripService tripservice.TripService
}

func NewHandler(logger *slog.Logger,
	readiness readinessChecker,
	tripService tripservice.TripService) *Handler {
	return &Handler{
		logger:      logger,
		readiness:   readiness,
		tripService: tripService,
	}
}
