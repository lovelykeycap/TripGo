package create_test

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
	"github.com/lovelykeycap/TripGo/internal/http/trip/create"
	tripcreate "github.com/lovelykeycap/TripGo/internal/usecase/trip/create"
)

func TestHandle(t *testing.T) {
	const invalidRequestCode = "invalid_request"
	const validBody = `{"user_id":"11111111-1111-4111-8111-111111111111","driver_id":"22222222-2222-4222-8222-222222222222","start_point":{"latitude":55.75,"longitude":37.62},"end_point":{"latitude":55.8,"longitude":37.7},"price":500}`
	input := tripcreate.Input{
		UserID: uuid.MustParse("11111111-1111-4111-8111-111111111111"), DriverID: uuid.MustParse("22222222-2222-4222-8222-222222222222"),
		StartPoint: domaintrip.Coordinates{Latitude: 55.75, Longitude: 37.62},
		EndPoint:   domaintrip.Coordinates{Latitude: 55.8, Longitude: 37.7}, Price: 500,
	}
	at := time.Date(2026, time.November, 6, 12, 0, 0, 0, time.UTC)
	created, err := domaintrip.New(input.UserID, input.DriverID, input.StartPoint, input.EndPoint, input.Price, at)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name      string
		body      string
		decodeErr bool
		err       error
		status    int
		code      string
		detail    string
	}{
		{name: "success", body: validBody, status: http.StatusCreated},
		{name: "driver busy", body: validBody, err: fmt.Errorf("create trip: %w", domaintrip.ErrDriverBusy), status: http.StatusConflict, code: "driver_busy", detail: "Driver already has an active trip"},
		{name: "repository error", body: validBody, err: errors.New("insert trip: database password=secret"), status: http.StatusInternalServerError, code: "internal_error", detail: "Unexpected server error"},
		{name: "malformed JSON", body: `{`, decodeErr: true, status: http.StatusBadRequest, code: invalidRequestCode, detail: "decode request body: unexpected EOF"},
		{name: "missing field", body: strings.Replace(validBody, `,"price":500`, "", 1), decodeErr: true, status: http.StatusBadRequest, code: invalidRequestCode, detail: "price: is required and must not be null"},
		{name: "unknown field", body: strings.Replace(validBody, `"price":500`, `"price":500,"extra":true`, 1), decodeErr: true, status: http.StatusBadRequest, code: invalidRequestCode, detail: "extra: unknown field"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usecase := NewMocktripCreator(gomock.NewController(t))
			if !tt.decodeErr {
				usecase.EXPECT().Create(gomock.Any(), input).Return(created, tt.err)
			}
			writer := response.New(slog.New(slog.NewJSONHandler(io.Discard, nil)))
			handler := create.New(usecase, writer)
			request := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(tt.body))
			request.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()

			handler.Handle(recorder, request, api.CreateTripParams{})

			if recorder.Code != tt.status {
				t.Fatalf("status = %d, want %d; body: %s", recorder.Code, tt.status, recorder.Body.String())
			}
			if tt.status == http.StatusCreated {
				if recorder.Header().Get("Content-Type") != "application/json" {
					t.Errorf("unexpected Content-Type: %q", recorder.Header().Get("Content-Type"))
				}
				if location := recorder.Header().Get("Location"); location != "/api/v1/trips/"+created.ID.String() {
					t.Errorf("unexpected Location: %q", location)
				}
				var got api.Trip
				if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
					t.Fatal(err)
				}
				want := api.Trip{
					Id: created.ID, UserId: input.UserID, DriverId: input.DriverID,
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
