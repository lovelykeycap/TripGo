package httpapi

import (
	"log/slog"

	tripservice "github.com/lovelykeycap/TripGo/internal/service/trip"
)

type Handler struct {
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
