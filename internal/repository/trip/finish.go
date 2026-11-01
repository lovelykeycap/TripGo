package trip

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
		Suffix("RETURNING " + strings.Join(tripColumns(), ", ")).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return domaintrip.Trip{}, fmt.Errorf("build finish trip query: %w", err)
	}

	executor := postgres.ExecutorFromContext(queryCtx, r.pool)
	result, err := queryRowToTrip(queryCtx, executor, query, args...)
	if errors.Is(err, pgx.ErrNoRows) {
		if _, err := r.Get(queryCtx, id); err != nil {
			return domaintrip.Trip{}, fmt.Errorf("failed to check trip existence: %w", err)
		}
		return domaintrip.Trip{}, domaintrip.ErrTripCompleted
	}
	if err != nil {
		return domaintrip.Trip{}, fmt.Errorf("update trip: %w", err)
	}

	tripTimeToUtc(&result)


	return result, nil
}
