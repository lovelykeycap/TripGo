package trip

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
	"github.com/lovelykeycap/TripGo/internal/infra/postgres"
)

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (domaintrip.Trip, error) {
	queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	query, args, err := sq.Select(
		"id", "user_id", "driver_id",
		"start_latitude", "start_longitude", "end_latitude", "end_longitude",
		"price", "status", "started_at", "finished_at", "created_at", "updated_at",
	).
		From("trips").
		Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return domaintrip.Trip{}, fmt.Errorf("build get trip query: %w", err)
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
		return domaintrip.Trip{}, domaintrip.ErrTripNotFound
	}
	if err != nil {
		return domaintrip.Trip{}, fmt.Errorf("get trip: %w", err)
	}

	return result, nil
}
