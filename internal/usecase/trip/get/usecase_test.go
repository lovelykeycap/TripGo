package get_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
	"github.com/lovelykeycap/TripGo/internal/usecase/trip/get"
)

func TestGet(t *testing.T) {
	repositoryErr := errors.New("database unavailable")
	tests := []struct {
		name string
		err  error
	}{
		{name: "success"},
		{name: "trip not found", err: domaintrip.ErrTripNotFound},
		{name: "repository error", err: repositoryErr},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMocktripRepository(gomock.NewController(t))
			stored, err := domaintrip.New(uuid.New(), uuid.New(), domaintrip.Coordinates{}, domaintrip.Coordinates{}, 500, time.Now().UTC())
			if err != nil {
				t.Fatal(err)
			}
			repo.EXPECT().Get(gomock.Any(), stored.ID).Return(stored, tt.err)
			usecase := get.New(repo, noop.NewTracerProvider().Tracer("test"))

			got, err := usecase.Get(t.Context(), stored.ID)
			if !errors.Is(err, tt.err) {
				t.Fatalf("Get() error = %v, want %v", err, tt.err)
			}
			want := stored
			if tt.err != nil {
				want = domaintrip.Trip{}
			}
			if got != want {
				t.Errorf("Get() = %+v, want %+v", got, want)
			}
		})
	}
}
