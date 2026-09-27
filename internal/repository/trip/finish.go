package trip

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
	"github.com/lovelykeycap/TripGo/internal/infra/postgres"
)

func (r *Repository) Finish(ctx context.Context, id uuid.UUID, finishedAt time.Time) (domaintrip.Trip, error) {
	queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	query, args, err := sq.Update("trips").
		Set("status", domaintrip.StatusCompleted).
		Set("finished_at", finishedAt).
		Set("updated_at", finishedAt).
		Where(sq.Eq{"id": id, "status": domaintrip.StatusActive}).
		Suffix("RETURNING id, user_id, driver_id, " +
			"start_latitude, start_longitude, end_latitude, end_longitude, " +
			"price, status, started_at, finished_at, created_at, updated_at").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return domaintrip.Trip{}, fmt.Errorf("build finish trip query: %w", err)
	}

	var result domaintrip.Trip
	executor := postgres.ExecutorFromContext(queryCtx, r.pool)
	err = executor.QueryRow(queryCtx, query, args...).Scan(
		&result.ID,
		&result.UserID,
		&result.DriverID,
		&result.StartPoint.Latitude,
		&result.StartPoint.Longitude,
		&result.EndPoint.Latitude,
		&result.EndPoint.Longitude,
		&result.Price,
		&result.Status,
		&result.StartedAt,
		&result.FinishedAt,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		if _, err := r.Get(queryCtx, id); err != nil {
			return domaintrip.Trip{}, fmt.Errorf("check trip after finish: %w", err)
		}
		return domaintrip.Trip{}, domaintrip.ErrTripCompleted
	}
	if err != nil {
		return domaintrip.Trip{}, fmt.Errorf("finish trip: %w", err)
	}
	return result, nil
}
