package domainerror

import (
	"errors"
	"net/http"
	"strings"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
	api "github.com/lovelykeycap/TripGo/internal/generated"
)

func Problem(err error) (int, api.Problem) {
	status := http.StatusInternalServerError
	code := "internal_error"
	title := "Internal error"
	detail := "Unexpected server error"
	var validationErr *domaintrip.ValidationError

	switch {
	case errors.Is(err, domaintrip.ErrTripNotFound):
		status = http.StatusNotFound
		code = "trip_not_found"
		title = "Trip not found"
		detail = "Trip does not exist"
	case errors.Is(err, domaintrip.ErrTripCompleted):
		status = http.StatusConflict
		code = "trip_completed"
		title = "Trip completed"
		detail = "Operation is not allowed for a completed trip"
	case errors.Is(err, domaintrip.ErrDriverBusy):
		status = http.StatusConflict
		code = "driver_busy"
		title = "Driver busy"
		detail = "Driver already has an active trip"
	case errors.As(err, &validationErr):
		status = http.StatusBadRequest
		code = "invalid_request"
		title = "Invalid request"
		detail = validationErr.Error()
	}

	return status, api.Problem{
		Type:   "https://tripgo.example/problems/" + strings.ReplaceAll(code, "_", "-"),
		Title:  title,
		Status: int32(status),
		Code:   code,
		Detail: &detail,
	}
}
