package finish_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
	"github.com/lovelykeycap/TripGo/internal/usecase/trip/finish"
)

func TestFinish(t *testing.T) {
	repositoryErr := errors.New("database unavailable")
	historyErr := errors.New("history insert failed")
	commitErr := errors.New("commit failed")
	tests := []struct {
		name       string
		completed  bool
		getErr     error
		finishErr  error
		historyErr error
		commitErr  error
		wantErr    error
	}{
		{name: "success"},
		{name: "already completed", completed: true, wantErr: domaintrip.ErrTripCompleted},
		{name: "trip not found", getErr: domaintrip.ErrTripNotFound, wantErr: domaintrip.ErrTripNotFound},
		{name: "get error", getErr: repositoryErr, wantErr: repositoryErr},
		{name: "concurrent finish", finishErr: domaintrip.ErrTripCompleted, wantErr: domaintrip.ErrTripCompleted},
		{name: "update error", finishErr: repositoryErr, wantErr: repositoryErr},
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
			at := time.Date(2026, time.November, 6, 12, 0, 0, 0, time.UTC)
			original, err := domaintrip.New(uuid.New(), uuid.New(), domaintrip.Coordinates{}, domaintrip.Coordinates{}, 500, at)
			if err != nil {
				t.Fatal(err)
			}
			if tt.completed {
				if err := original.Finish(at.Add(time.Hour)); err != nil {
					t.Fatal(err)
				}
			}
			repo.EXPECT().Get(gomock.Any(), original.ID).Return(original, tt.getErr)
			var saved domaintrip.Trip
			if tt.getErr == nil && !tt.completed {
				txCtx, cancel := context.WithCancel(t.Context())
				defer cancel()
				tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, fn func(context.Context) error) error {
					if err := fn(txCtx); err != nil {
						return err
					}
					return tt.commitErr
				})
				update := repo.EXPECT().Finish(txCtx, gomock.Any()).DoAndReturn(func(_ context.Context, value domaintrip.Trip) error {
					saved = value
					if value.Status != domaintrip.StatusCompleted || value.FinishedAt == nil {
						t.Fatalf("repository received unfinished trip: %+v", value)
					}
					want := original
					want.Status = domaintrip.StatusCompleted
					want.FinishedAt = value.FinishedAt
					want.UpdatedAt = *value.FinishedAt
					if !reflect.DeepEqual(value, want) {
						t.Errorf("saved trip = %+v, want %+v", value, want)
					}
					return tt.finishErr
				})
				if tt.finishErr == nil {
					history.EXPECT().Add(txCtx, gomock.Any()).After(update).DoAndReturn(func(_ context.Context, change domaintrip.StatusChange) error {
						fromStatus := domaintrip.StatusActive
						want := domaintrip.StatusChange{
							TripID: original.ID, FromStatus: &fromStatus,
							ToStatus: domaintrip.StatusCompleted, ChangedAt: *saved.FinishedAt,
						}
						if !reflect.DeepEqual(change, want) {
							t.Errorf("history change = %+v, want %+v", change, want)
						}
						return tt.historyErr
					})
				}
				if tt.wantErr == nil {
					metrics.EXPECT().RecordCompleted(gomock.Any())
				}
			}
			usecase := finish.New(repo, history, tx, noop.NewTracerProvider().Tracer("test"), slog.New(slog.NewJSONHandler(io.Discard, nil)), metrics)

			got, err := usecase.Finish(t.Context(), original.ID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Finish() error = %v, want %v", err, tt.wantErr)
			}
			if err != nil {
				if got != (domaintrip.Trip{}) {
					t.Errorf("Finish() returned trip on failure: %+v", got)
				}
				return
			}
			if !reflect.DeepEqual(got, saved) {
				t.Errorf("Finish() = %+v, want saved trip %+v", got, saved)
			}
		})
	}
}
