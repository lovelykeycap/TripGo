package finish_test

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
	"github.com/lovelykeycap/TripGo/internal/http/trip/finish"
)

func TestHandle(t *testing.T) {
	at := time.Date(2026, time.November, 6, 12, 0, 0, 0, time.UTC)
	finishedAt := at.Add(time.Hour)
	completed, err := domaintrip.New(uuid.New(), uuid.New(), domaintrip.Coordinates{}, domaintrip.Coordinates{}, 500, at)
	if err != nil {
		t.Fatal(err)
	}
	if err := completed.Finish(finishedAt); err != nil {
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
		{name: "already completed", err: fmt.Errorf("finish trip: %w", domaintrip.ErrTripCompleted), status: http.StatusConflict, code: "trip_completed", detail: "Operation is not allowed for a completed trip"},
		{name: "not found", err: domaintrip.ErrTripNotFound, status: http.StatusNotFound, code: "trip_not_found", detail: "Trip does not exist"},
		{name: "repository error", err: errors.New("update trip: database password=secret"), status: http.StatusInternalServerError, code: "internal_error", detail: "Unexpected server error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usecase := NewMocktripFinisher(gomock.NewController(t))
			usecase.EXPECT().Finish(gomock.Any(), completed.ID).Return(completed, tt.err)
			writer := response.New(slog.New(slog.NewJSONHandler(io.Discard, nil)))
			handler := finish.New(usecase, writer)
			request := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+completed.ID.String()+"/finish", nil)
			recorder := httptest.NewRecorder()

			handler.Handle(recorder, request, completed.ID)

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
					Id: completed.ID, UserId: completed.UserID, DriverId: completed.DriverID,
					Price: 500, Status: api.Completed, StartedAt: at, FinishedAt: &finishedAt,
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
