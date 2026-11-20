//go:build integration

package trip_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/trace/noop"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
	"github.com/lovelykeycap/TripGo/internal/infra/postgres"
	triprepo "github.com/lovelykeycap/TripGo/internal/repository/trip"
	"github.com/lovelykeycap/TripGo/internal/repository/tripposition"
	"github.com/lovelykeycap/TripGo/internal/usecase/trip/positions/create"
)

func databaseTime(t *testing.T, pool *pgxpool.Pool) time.Time {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), queryTimeout)
	defer cancel()
	var now time.Time
	if err := pool.QueryRow(ctx, "SELECT CURRENT_TIMESTAMP").Scan(&now); err != nil {
		t.Fatal(err)
	}
	return now.UTC()
}

func TestIntegrationListStale(t *testing.T) {
	pool := newPool(t)
	repo := triprepo.New(pool, queryTimeout)
	now := databaseTime(t, pool)
	older := now.Add(-10 * time.Minute)
	fresh := now.Add(-10 * time.Second)
	tests := []struct {
		started   time.Time
		position  time.Time
		completed bool
	}{
		{started: older},
		{started: now.Add(-time.Hour), position: now.Add(-3 * time.Minute)},
		{started: older, position: fresh},
		{started: fresh},
		{started: older, completed: true},
	}
	fixtures := make([]domaintrip.Trip, 0, len(tests))
	for _, tt := range tests {
		trip := newTrip(t, pool)
		trip.StartedAt, trip.CreatedAt, trip.UpdatedAt = tt.started, tt.started, tt.started
		if err := repo.Create(t.Context(), trip); err != nil {
			t.Fatal(err)
		}
		if !tt.position.IsZero() {
			if err := repo.UpdateLastPosition(t.Context(), trip.ID, tt.position, now); err != nil {
				t.Fatal(err)
			}
		}
		if tt.completed {
			if err := trip.Finish(now); err != nil {
				t.Fatal(err)
			}
			if err := repo.Finish(t.Context(), trip); err != nil {
				t.Fatal(err)
			}
		}
		fixtures = append(fixtures, trip)
	}

	stale, err := repo.ListStale(t.Context(), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	var got []domaintrip.StaleTrip
	for _, entry := range stale {
		if slices.ContainsFunc(fixtures, func(trip domaintrip.Trip) bool { return trip.ID == entry.TripID }) {
			got = append(got, entry)
		}
	}
	want := []domaintrip.StaleTrip{
		{TripID: fixtures[0].ID, DriverID: fixtures[0].DriverID},
		{TripID: fixtures[1].ID, DriverID: fixtures[1].DriverID},
	}
	if !slices.Equal(got, want) {
		t.Errorf("stale trips = %+v, want %+v in order", got, want)
	}
}

type positionRepositoryFunc func(context.Context, domaintrip.Position) (domaintrip.Position, error)

func (f positionRepositoryFunc) Create(ctx context.Context, position domaintrip.Position) (domaintrip.Position, error) {
	return f(ctx, position)
}

type positionEffects struct {
	saves   int
	removed []uuid.UUID
}

func (e *positionEffects) RecordSaved(context.Context) { e.saves++ }
func (e *positionEffects) Cancel(id uuid.UUID)         { e.removed = append(e.removed, id) }

func TestIntegrationPositionCreate(t *testing.T) {
	pool := newPool(t)
	trips := triprepo.New(pool, queryTimeout)
	positions := tripposition.New(pool, queryTimeout)
	tx := postgres.NewTxManager(pool, queryTimeout)
	tracer := noop.NewTracerProvider().Tracer("test")
	clock := databaseTime(t, pool)
	now := func() time.Time { return clock }
	effects := &positionEffects{}
	uc := create.New(trips, positions, tx, tracer, effects, effects, now)
	trip := newTrip(t, pool)
	if err := trips.Create(t.Context(), trip); err != nil {
		t.Fatal(err)
	}

	input := create.Input{TripID: trip.ID, Coordinates: trip.StartPoint, RecordedAt: clock.Add(-time.Second)}
	recent, err := uc.Create(t.Context(), input)
	if err != nil {
		t.Fatal(err)
	}
	input.RecordedAt = clock.Add(-time.Minute)
	late, err := uc.Create(t.Context(), input)
	if err != nil {
		t.Fatal(err)
	}
	assertPositionState(t, trips, positions, trip.ID, clock, recent.RecordedAt, []domaintrip.Position{late, recent})

	writeAborted := errors.New("abort after writing position")
	failing := positionRepositoryFunc(func(ctx context.Context, pos domaintrip.Position) (domaintrip.Position, error) {
		written, err := positions.Create(ctx, pos)
		if err != nil {
			return domaintrip.Position{}, err
		}
		return written, writeAborted
	})
	failingUsecase := create.New(trips, failing, tx, tracer, effects, effects, now)
	committedAt := clock
	clock = clock.Add(time.Second)
	input.RecordedAt = clock
	if _, err := failingUsecase.Create(t.Context(), input); !errors.Is(err, writeAborted) {
		t.Fatalf("Create() error = %v, want %v", err, writeAborted)
	}
	assertPositionState(t, trips, positions, trip.ID, committedAt, recent.RecordedAt, []domaintrip.Position{late, recent})
	if effects.saves != 2 || !slices.Equal(effects.removed, []uuid.UUID{trip.ID, trip.ID}) {
		t.Errorf("effects after commits and rollback: saves = %d, cancellations = %v", effects.saves, effects.removed)
	}
}

func assertPositionState(
	t *testing.T,
	trips *triprepo.Repository,
	positions *tripposition.Repository,
	id uuid.UUID,
	updated, last time.Time,
	want []domaintrip.Position,
) {
	t.Helper()
	trip, err := trips.Get(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	if trip.LastPositionAt == nil || !trip.LastPositionAt.Equal(last) || !trip.UpdatedAt.Equal(updated) {
		t.Errorf("last position = %v, updated = %v; want %s and %s", trip.LastPositionAt, trip.UpdatedAt, last, updated)
	}
	got, err := positions.List(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, want) {
		t.Errorf("route = %+v, want %+v", got, want)
	}
}
