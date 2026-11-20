package trip

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgtype"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
)

func (r *Repository) ListStale(ctx context.Context, staleAfter time.Duration) ([]domaintrip.StaleTrip, error) {
	queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	const lastActivity = "COALESCE(last_position_at, started_at)"
	interval := pgtype.Interval{Microseconds: staleAfter.Microseconds(), Valid: true}
	query, args, err := sq.Select("id", "driver_id").
		From("trips").
		Where("status = 'active'").
		Where(lastActivity+" < CURRENT_TIMESTAMP - ?::interval", interval).
		OrderBy(lastActivity, "id").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list stale trips query: %w", err)
	}

	rows, err := r.executor(queryCtx).Query(queryCtx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("select stale trips: %w", err)
	}
	defer rows.Close()

	var trips []domaintrip.StaleTrip
	for rows.Next() {
		var trip domaintrip.StaleTrip
		if err := rows.Scan(&trip.TripID, &trip.DriverID); err != nil {
			return nil, fmt.Errorf("scan stale trip: %w", err)
		}
		trips = append(trips, trip)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read stale trips: %w", err)
	}
	return trips, nil
}
