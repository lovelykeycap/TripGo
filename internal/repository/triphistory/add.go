package triphistory

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
	"github.com/lovelykeycap/TripGo/internal/infra/postgres"
)

func (r *Repository) Add(ctx context.Context, change domaintrip.StatusChange) error {
	queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	query, args, err := sq.Insert("trip_status_history").
		Columns("trip_id", "from_status", "to_status", "reason", "changed_at").
		Values(change.TripID, change.FromStatus, change.ToStatus, change.Reason, change.ChangedAt).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build add trip history query: %w", err)
	}

	executor := postgres.ExecutorFromContext(queryCtx, r.pool)
	if _, err := executor.Exec(queryCtx, query, args...); err != nil {
		return fmt.Errorf("add trip history: %w", err)
	}
	return nil
}
