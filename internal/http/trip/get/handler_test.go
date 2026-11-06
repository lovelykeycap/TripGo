package get_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
	api "github.com/lovelykeycap/TripGo/internal/generated"
	"github.com/lovelykeycap/TripGo/internal/http/response"
	"github.com/lovelykeycap/TripGo/internal/http/trip/get"
)

func TestHandle(t *testing.T) {
	at := time.Date(2026, time.November, 6, 12, 0, 0, 0, time.UTC)
	stored, err := domaintrip.New(uuid.New(), uuid.New(), domaintrip.Coordinates{Latitude: 55.75, Longitude: 37.62}, domaintrip.Coordinates{Latitude: 55.8, Longitude: 37.7}, 500, at)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		err    error
		status int
		code   string
		detail string
	}{
		{name: "success", status: http.StatusOK},
		{name: "not found", err: fmt.Errorf("get trip: %w", domaintrip.ErrTripNotFound), status: http.StatusNotFound, code: "trip_not_found", detail: "Trip does not exist"},
		{name: "repository error", err: errors.New("select trip: database password=secret"), status: http.StatusInternalServerError, code: "internal_error", detail: "Unexpected server error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usecase := NewMocktripGetter(gomock.NewController(t))
			usecase.EXPECT().Get(gomock.Any(), stored.ID).Return(stored, tt.err)
			writer := response.New(slog.New(slog.NewJSONHandler(io.Discard, nil)))
			handler := get.New(usecase, writer)
			request := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+stored.ID.String(), nil)
			recorder := httptest.NewRecorder()

			handler.Handle(recorder, request, stored.ID)

			if recorder.Code != tt.status {
				t.Fatalf("status = %d, want %d; body: %s", recorder.Code, tt.status, recorder.Body.String())
			}
			if tt.err == nil {
				if recorder.Header().Get("Content-Type") != "application/json" {
					t.Errorf("unexpected Content-Type: %q", recorder.Header().Get("Content-Type"))
				}
				var got api.Trip
				if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
					t.Fatal(err)
				}
				want := api.Trip{
					Id: stored.ID, UserId: stored.UserID, DriverId: stored.DriverID,
					StartPoint: api.Coordinates{Latitude: 55.75, Longitude: 37.62},
					EndPoint:   api.Coordinates{Latitude: 55.8, Longitude: 37.7},
					Price:      500, Status: api.Active, StartedAt: at,
				}
				if !reflect.DeepEqual(got, want) {
					t.Errorf("response = %+v, want %+v", got, want)
				}
				return
			}
			if recorder.Header().Get("Content-Type") != "application/problem+json" {
				t.Errorf("unexpected Content-Type: %q", recorder.Header().Get("Content-Type"))
			}
			var problem api.Problem
			if err := json.Unmarshal(recorder.Body.Bytes(), &problem); err != nil {
				t.Fatal(err)
			}
			if problem.Code != tt.code || int(problem.Status) != tt.status || problem.Detail == nil || *problem.Detail != tt.detail {
				t.Errorf("unexpected Problem: %+v", problem)
			}
			if problem.Instance == nil || *problem.Instance != request.URL.Path {
				t.Errorf("unexpected Problem instance: %v", problem.Instance)
			}
			if tt.status == http.StatusInternalServerError && strings.Contains(recorder.Body.String(), "secret") {
				t.Error("internal error details leaked into response")
			}
		})
	}
}
