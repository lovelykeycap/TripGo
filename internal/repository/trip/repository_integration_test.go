//go:build integration

package trip_test

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
	"github.com/lovelykeycap/TripGo/internal/infra/postgres"
	triprepo "github.com/lovelykeycap/TripGo/internal/repository/trip"
	"github.com/lovelykeycap/TripGo/internal/repository/triphistory"
)

const queryTimeout = 5 * time.Second

func newPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("DATABASE_URL is required; run make test-integration after preparing the database")
	}
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.ConnectTimeout = queryTimeout
	config.MaxConns = 4
	ctx, cancel := context.WithTimeout(t.Context(), queryTimeout)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("PostgreSQL is unavailable: %v", err)
	}
	return pool
}

func newTrip(t *testing.T, pool *pgxpool.Pool) domaintrip.Trip {
	t.Helper()
	at := time.Date(2026, time.November, 6, 12, 0, 0, 123456000, time.UTC)
	value, err := domaintrip.New(
		uuid.New(), uuid.New(),
		domaintrip.Coordinates{Latitude: 55.75, Longitude: 37.62},
		domaintrip.Coordinates{Latitude: 55.8, Longitude: 37.7}, 500, at,
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
		defer cancel()
		if _, err := pool.Exec(ctx, "DELETE FROM trips WHERE id = $1", value.ID); err != nil {
			t.Errorf("delete test trip: %v", err)
		}
	})
	return value
}

func TestIntegrationTripCreateAndGet(t *testing.T) {
	pool := newPool(t)
	repo := triprepo.New(pool, queryTimeout)
	value := newTrip(t, pool)
	if err := repo.Create(t.Context(), value); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Get(t.Context(), value.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, value) {
		t.Errorf("Get() = %+v, want %+v", got, value)
	}

	other := newTrip(t, pool)
	other.DriverID = value.DriverID
	if err := repo.Create(t.Context(), other); !errors.Is(err, domaintrip.ErrDriverBusy) {
		t.Errorf("Create() error = %v, want ErrDriverBusy", err)
	}
	if _, err := repo.Get(t.Context(), uuid.New()); !errors.Is(err, domaintrip.ErrTripNotFound) {
		t.Errorf("Get() error = %v, want ErrTripNotFound", err)
	}
}

func TestIntegrationTripFinish(t *testing.T) {
	pool := newPool(t)
	repo := triprepo.New(pool, queryTimeout)
	value := newTrip(t, pool)
	if err := repo.Create(t.Context(), value); err != nil {
		t.Fatal(err)
	}
	if err := value.Finish(value.StartedAt.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := repo.Finish(t.Context(), value); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Get(t.Context(), value.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, value) {
		t.Errorf("finished trip = %+v, want %+v", got, value)
	}
	if err := repo.Finish(t.Context(), value); !errors.Is(err, domaintrip.ErrTripCompleted) {
		t.Errorf("repeated Finish() error = %v, want ErrTripCompleted", err)
	}
	value.ID = uuid.New()
	if err := repo.Finish(t.Context(), value); !errors.Is(err, domaintrip.ErrTripNotFound) {
		t.Errorf("missing trip Finish() error = %v, want ErrTripNotFound", err)
	}
}

func TestIntegrationTransaction(t *testing.T) {
	callbackErr := errors.New("operation failed after both inserts")
	tests := []struct {
		name           string
		invalidHistory bool
		afterInsertErr error
	}{
		{name: "commit both inserts"},
		{name: "rollback on history error", invalidHistory: true},
		{name: "rollback both inserts", afterInsertErr: callbackErr},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := newPool(t)
			repo := triprepo.New(pool, queryTimeout)
			history := triphistory.New(pool, queryTimeout)
			manager := postgres.NewTxManager(pool, queryTimeout)
			value := newTrip(t, pool)
			change := domaintrip.StatusChange{TripID: value.ID, ToStatus: domaintrip.StatusActive, ChangedAt: value.CreatedAt}
			if tt.invalidHistory {
				change.ToStatus = "invalid"
			}
			err := manager.Do(t.Context(), func(ctx context.Context) error {
				if err := repo.Create(ctx, value); err != nil {
					return err
				}
				got, err := repo.Get(ctx, value.ID)
				if err != nil {
					return err
				}
				if !reflect.DeepEqual(got, value) {
					t.Errorf("Get() inside transaction = %+v, want %+v", got, value)
				}
				if err := history.Add(ctx, change); err != nil {
					return err
				}
				return tt.afterInsertErr
			})
			if tt.invalidHistory {
				var pgErr *pgconn.PgError
				if !errors.As(err, &pgErr) || pgErr.Code != "23514" {
					t.Fatalf("Do() error = %v, want check constraint violation", err)
				}
			} else if !errors.Is(err, tt.afterInsertErr) {
				t.Fatalf("Do() error = %v, want %v", err, tt.afterInsertErr)
			}

			var tripCount, historyCount int
			ctx, cancel := context.WithTimeout(t.Context(), queryTimeout)
			defer cancel()
			if err := pool.QueryRow(ctx, "SELECT (SELECT COUNT(*) FROM trips WHERE id = $1), (SELECT COUNT(*) FROM trip_status_history WHERE trip_id = $1)", value.ID).Scan(&tripCount, &historyCount); err != nil {
				t.Fatal(err)
			}
			if tt.invalidHistory || tt.afterInsertErr != nil {
				if tripCount != 0 || historyCount != 0 {
					t.Errorf("rollback left rows: trips=%d, history=%d", tripCount, historyCount)
				}
				return
			}
			if tripCount != 1 || historyCount != 1 {
				t.Fatalf("commit row counts: trips=%d, history=%d; want 1, 1", tripCount, historyCount)
			}
			var got domaintrip.StatusChange
			if err := pool.QueryRow(ctx, "SELECT trip_id, from_status, to_status, reason, changed_at FROM trip_status_history WHERE trip_id = $1", value.ID).Scan(&got.TripID, &got.FromStatus, &got.ToStatus, &got.Reason, &got.ChangedAt); err != nil {
				t.Fatal(err)
			}
			got.ChangedAt = got.ChangedAt.UTC()
			if !reflect.DeepEqual(got, change) {
				t.Errorf("stored history = %+v, want %+v", got, change)
			}
		})
	}
}

func TestIntegrationConcurrentFinish(t *testing.T) {
	pool := newPool(t)
	repo := triprepo.New(pool, queryTimeout)
	history := triphistory.New(pool, queryTimeout)
	manager := postgres.NewTxManager(pool, queryTimeout)
	value := newTrip(t, pool)
	if err := repo.Create(t.Context(), value); err != nil {
		t.Fatal(err)
	}
	if err := value.Finish(value.StartedAt.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	fromStatus := domaintrip.StatusActive
	change := domaintrip.StatusChange{TripID: value.ID, FromStatus: &fromStatus, ToStatus: value.Status, ChangedAt: *value.FinishedAt}
	start := make(chan struct{})
	results := make(chan error, 2)
	for range 2 {
		go func() {
			<-start
			results <- manager.Do(t.Context(), func(ctx context.Context) error {
				if err := repo.Finish(ctx, value); err != nil {
					return err
				}
				return history.Add(ctx, change)
			})
		}()
	}
	close(start)
	var success, completed int
	for range 2 {
		switch err := <-results; {
		case err == nil:
			success++
		case errors.Is(err, domaintrip.ErrTripCompleted):
			completed++
		default:
			t.Errorf("unexpected concurrent Finish() error: %v", err)
		}
	}
	if success != 1 || completed != 1 {
		t.Errorf("concurrent results: success=%d, completed=%d; want 1, 1", success, completed)
	}
	ctx, cancel := context.WithTimeout(t.Context(), queryTimeout)
	defer cancel()
	var count int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM trip_status_history WHERE trip_id = $1", value.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("history row count = %d, want 1", count)
	}
	got, err := repo.Get(t.Context(), value.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, value) {
		t.Errorf("finished trip = %+v, want %+v", got, value)
	}
}
