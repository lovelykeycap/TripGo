package domainerror_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
	"github.com/lovelykeycap/TripGo/internal/http/domainerror"
)

func TestProblem(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		status  int
		code    string
		title   string
		detail  string
		typeURL string
	}{
		{
			name: "not found", err: domaintrip.ErrTripNotFound,
			status: http.StatusNotFound, code: "trip_not_found", title: "Trip not found",
			detail: "Trip does not exist", typeURL: "https://tripgo.example/problems/trip-not-found",
		},
		{
			name: "completed", err: domaintrip.ErrTripCompleted,
			status: http.StatusConflict, code: "trip_completed", title: "Trip completed",
			detail: "Operation is not allowed for a completed trip", typeURL: "https://tripgo.example/problems/trip-completed",
		},
		{
			name: "driver busy", err: domaintrip.ErrDriverBusy,
			status: http.StatusConflict, code: "driver_busy", title: "Driver busy",
			detail: "Driver already has an active trip", typeURL: "https://tripgo.example/problems/driver-busy",
		},
		{
			name: "validation", err: &domaintrip.ValidationError{Field: "price", Reason: "must not be negative"},
			status: http.StatusBadRequest, code: "invalid_request", title: "Invalid request",
			detail: "price: must not be negative", typeURL: "https://tripgo.example/problems/invalid-request",
		},
		{
			name: "unexpected error", err: errors.New("database password=secret"),
			status: http.StatusInternalServerError, code: "internal_error", title: "Internal error",
			detail: "Unexpected server error", typeURL: "https://tripgo.example/problems/internal-error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, problem := domainerror.Problem(fmt.Errorf("operation: %w", tt.err))
			if status != tt.status || int(problem.Status) != tt.status {
				t.Errorf("statuses = %d, %d, want %d", status, problem.Status, tt.status)
			}
			if problem.Code != tt.code || problem.Title != tt.title || problem.Type != tt.typeURL {
				t.Errorf("unexpected Problem: %+v", problem)
			}
			if problem.Detail == nil || *problem.Detail != tt.detail {
				t.Errorf("Problem detail = %v, want %q", problem.Detail, tt.detail)
			}
		})
	}
}
