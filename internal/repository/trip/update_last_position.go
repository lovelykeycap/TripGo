package trip

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
)

func (r *Repository) UpdateLastPosition(ctx context.Context, id uuid.UUID, recordedAt, updatedAt time.Time) error {
	queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	query, args, err := sq.Update("trips").
		Set(lastPositionColumn, sq.Expr("GREATEST(last_position_at, ?)", recordedAt)).
		Set("updated_at", updatedAt).
		Where(sq.Eq{"id": id, statusColumn: domaintrip.StatusActive}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update last position query: %w", err)
	}

	result, err := r.executor(queryCtx).Exec(queryCtx, query, args...)
	if err != nil {
		return fmt.Errorf("update last position: %w", err)
	}
	if result.RowsAffected() == 0 {
		if err := r.CheckExists(queryCtx, id); err != nil {
			return fmt.Errorf("check trip after position update: %w", err)
		}
		return domaintrip.ErrTripCompleted
	}
	return nil
}
