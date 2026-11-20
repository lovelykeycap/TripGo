package trip_test

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/lovelykeycap/TripGo/internal/domain/trip"
)

func TestNew(t *testing.T) {
	userID := uuid.New()
	driverID := uuid.New()
	at := time.Date(2026, time.November, 6, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name       string
		startPoint trip.Coordinates
		endPoint   trip.Coordinates
		price      int64
	}{
		{
			name:       "regular trip",
			startPoint: trip.Coordinates{Latitude: 55.75, Longitude: 37.62},
			endPoint:   trip.Coordinates{Latitude: 55.8, Longitude: 37.7},
			price:      500,
		},
		{
			name:       "coordinate boundaries and zero price",
			startPoint: trip.Coordinates{Latitude: -90, Longitude: -180},
			endPoint:   trip.Coordinates{Latitude: 90, Longitude: 180},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := trip.New(userID, driverID, tt.startPoint, tt.endPoint, tt.price, at)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			if got.ID == uuid.Nil {
				t.Fatal("New() returned a zero trip ID")
			}

			want := trip.Trip{
				ID:         got.ID,
				UserID:     userID,
				DriverID:   driverID,
				StartPoint: tt.startPoint,
				EndPoint:   tt.endPoint,
				Price:      tt.price,
				Status:     trip.StatusActive,
				StartedAt:  at,
				CreatedAt:  at,
				UpdatedAt:  at,
			}
			if got != want {
				t.Errorf("New() = %+v, want %+v", got, want)
			}
		})
	}
}

func TestNewValidation(t *testing.T) {
	tests := []struct {
		name      string
		change    func(*trip.Trip)
		wantField string
	}{
		{
			name:      "zero user ID",
			change:    func(input *trip.Trip) { input.UserID = uuid.Nil },
			wantField: "user_id",
		},
		{
			name:      "zero driver ID",
			change:    func(input *trip.Trip) { input.DriverID = uuid.Nil },
			wantField: "driver_id",
		},
		{
			name:      "negative price",
			change:    func(input *trip.Trip) { input.Price = -1 },
			wantField: "price",
		},
		{
			name:      "start latitude below minimum",
			change:    func(input *trip.Trip) { input.StartPoint.Latitude = -91 },
			wantField: "start_point.latitude",
		},
		{
			name:      "start longitude above maximum",
			change:    func(input *trip.Trip) { input.StartPoint.Longitude = 181 },
			wantField: "start_point.longitude",
		},
		{
			name:      "end latitude above maximum",
			change:    func(input *trip.Trip) { input.EndPoint.Latitude = 91 },
			wantField: "end_point.latitude",
		},
		{
			name:      "end longitude below minimum",
			change:    func(input *trip.Trip) { input.EndPoint.Longitude = -181 },
			wantField: "end_point.longitude",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := trip.Trip{UserID: uuid.New(), DriverID: uuid.New(), Price: 500}
			tt.change(&input)
			_, err := trip.New(input.UserID, input.DriverID, input.StartPoint, input.EndPoint, input.Price, time.Time{})

			var validationErr *trip.ValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("New() error = %v, want *trip.ValidationError", err)
			}
			if validationErr.Field != tt.wantField {
				t.Errorf("validation field = %q, want %q", validationErr.Field, tt.wantField)
			}
			if validationErr.Reason == "" {
				t.Error("validation reason is empty")
			}
		})
	}
}

func TestTripFinish(t *testing.T) {
	startedAt := time.Date(2026, time.November, 6, 12, 0, 0, 0, time.UTC)
	finishedAt := startedAt.Add(time.Hour)
	at := finishedAt.Add(time.Minute)
	tests := []struct {
		name       string
		status     trip.Status
		finishedAt *time.Time
		wantErr    error
	}{
		{name: "active trip", status: trip.StatusActive},
		{
			name:       "already completed trip",
			status:     trip.StatusCompleted,
			finishedAt: &finishedAt,
			wantErr:    trip.ErrTripCompleted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trip.Trip{
				ID:         uuid.New(),
				UserID:     uuid.New(),
				DriverID:   uuid.New(),
				StartPoint: trip.Coordinates{Latitude: 55.75, Longitude: 37.62},
				EndPoint:   trip.Coordinates{Latitude: 55.8, Longitude: 37.7},
				Price:      500,
				Status:     tt.status,
				StartedAt:  startedAt,
				FinishedAt: tt.finishedAt,
				CreatedAt:  startedAt,
				UpdatedAt:  startedAt,
			}
			want := got
			if tt.wantErr == nil {
				want.Status = trip.StatusCompleted
				want.FinishedAt = &at
				want.UpdatedAt = at
			}

			if err := got.Finish(at); !errors.Is(err, tt.wantErr) {
				t.Fatalf("Finish() error = %v, want %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("trip after Finish() = %+v, want %+v", got, want)
			}
		})
	}
}
