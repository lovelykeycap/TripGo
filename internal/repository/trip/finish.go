package trip

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
)

func (r *Repository) Finish(ctx context.Context, finishedTrip domaintrip.Trip) error {
	queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	query, args, err := sq.Update("trips").
		Set(statusColumn, finishedTrip.Status).
		Set("finished_at", finishedTrip.FinishedAt).
		Set("updated_at", finishedTrip.UpdatedAt).
		Where(sq.Eq{"id": finishedTrip.ID, statusColumn: domaintrip.StatusActive}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build finish trip query: %w", err)
	}

	executor := r.executor(queryCtx)
	result, err := executor.Exec(queryCtx, query, args...)
	if err != nil {
		return fmt.Errorf("update trip: %w", err)
	}
	if result.RowsAffected() == 0 {
		if _, err := r.Get(queryCtx, finishedTrip.ID); err != nil {
			return fmt.Errorf("failed to check trip existence: %w", err)
		}
		return domaintrip.ErrTripCompleted
	}
	return nil
}
