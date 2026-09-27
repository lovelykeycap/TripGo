package trip

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgconn"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
	"github.com/lovelykeycap/TripGo/internal/infra/postgres"
)

func (r *Repository) Create(ctx context.Context, newTrip domaintrip.Trip) error {
	queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	query, args, err := sq.Insert("trips").
		Columns(
			"id", "user_id", "driver_id",
			"start_latitude", "start_longitude", "end_latitude", "end_longitude",
			"price", "status", "started_at", "finished_at", "created_at", "updated_at",
		).
		Values(
			newTrip.ID, newTrip.UserID, newTrip.DriverID,
			newTrip.StartPoint.Latitude, newTrip.StartPoint.Longitude,
			newTrip.EndPoint.Latitude, newTrip.EndPoint.Longitude,
			newTrip.Price, newTrip.Status, newTrip.StartedAt, newTrip.FinishedAt,
			newTrip.CreatedAt, newTrip.UpdatedAt,
		).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build create trip query: %w", err)
	}

	executor := postgres.ExecutorFromContext(queryCtx, r.pool)
	_, err = executor.Exec(queryCtx, query, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "trips_active_driver_idx" {
			return domaintrip.ErrDriverBusy
		}
		return fmt.Errorf("create trip: %w", err)
	}
	return nil
}
