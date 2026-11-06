package create_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
	"github.com/lovelykeycap/TripGo/internal/usecase/trip/create"
)

func TestCreate(t *testing.T) {
	repositoryErr := errors.New("insert failed")
	historyErr := errors.New("history insert failed")
	commitErr := errors.New("commit failed")
	tests := []struct {
		name       string
		invalid    bool
		repoErr    error
		historyErr error
		commitErr  error
		wantErr    error
	}{
		{name: "success"},
		{name: "invalid input", invalid: true},
		{name: "driver busy", repoErr: domaintrip.ErrDriverBusy, wantErr: domaintrip.ErrDriverBusy},
		{name: "repository error", repoErr: repositoryErr, wantErr: repositoryErr},
		{name: "history error", historyErr: historyErr, wantErr: historyErr},
		{name: "commit error", commitErr: commitErr, wantErr: commitErr},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := NewMocktripRepository(ctrl)
			history := NewMocktripHistoryRepository(ctrl)
			tx := NewMocktxManager(ctrl)
			metrics := NewMocktripMetrics(ctrl)
			input := create.Input{
				UserID:     uuid.New(),
				DriverID:   uuid.New(),
				StartPoint: domaintrip.Coordinates{Latitude: 55.75, Longitude: 37.62},
				EndPoint:   domaintrip.Coordinates{Latitude: 55.8, Longitude: 37.7},
				Price:      500,
			}
			var stored domaintrip.Trip
			if tt.invalid {
				input.Price = -1
			} else {
				txCtx, cancel := context.WithCancel(t.Context())
				defer cancel()
				tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, fn func(context.Context) error) error {
					if err := fn(txCtx); err != nil {
						return err
					}
					return tt.commitErr
				})
				insert := repo.EXPECT().Create(txCtx, gomock.Any()).DoAndReturn(func(_ context.Context, value domaintrip.Trip) error {
					stored = value
					return tt.repoErr
				})
				if tt.repoErr == nil {
					history.EXPECT().Add(txCtx, gomock.Any()).After(insert).DoAndReturn(func(_ context.Context, change domaintrip.StatusChange) error {
						want := domaintrip.StatusChange{TripID: stored.ID, ToStatus: domaintrip.StatusActive, ChangedAt: stored.CreatedAt}
						if change != want {
							t.Errorf("history change = %+v, want %+v", change, want)
						}
						return tt.historyErr
					})
				}
				if tt.wantErr == nil {
					metrics.EXPECT().RecordCreated(gomock.Any())
				}
			}
			usecase := create.New(repo, history, tx, noop.NewTracerProvider().Tracer("test"), slog.New(slog.NewJSONHandler(io.Discard, nil)), metrics)

			got, err := usecase.Create(t.Context(), input)
			if tt.invalid {
				var validationErr *domaintrip.ValidationError
				if !errors.As(err, &validationErr) || validationErr.Field != "price" {
					t.Fatalf("Create() error = %v, want price validation error", err)
				}
			} else if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Create() error = %v, want %v", err, tt.wantErr)
			}
			if err != nil {
				if got != (domaintrip.Trip{}) {
					t.Errorf("Create() returned trip on failure: %+v", got)
				}
				return
			}
			if got != stored || got.ID == uuid.Nil {
				t.Errorf("Create() = %+v, want inserted trip %+v with nonzero ID", got, stored)
			}
			if got.UserID != input.UserID || got.DriverID != input.DriverID || got.StartPoint != input.StartPoint || got.EndPoint != input.EndPoint || got.Price != input.Price || got.Status != domaintrip.StatusActive {
				t.Errorf("created trip does not match input: %+v", got)
			}
		})
	}
}
