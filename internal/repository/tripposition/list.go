package tripposition

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
)

func (r *Repository) List(ctx context.Context, tripID uuid.UUID) ([]domaintrip.Position, error) {
	queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	query, args, err := sq.Select("id", "trip_id", "latitude", "longitude", "recorded_at", "created_at").
		From("trip_positions").
		Where(sq.Eq{"trip_id": tripID}).
		OrderBy("recorded_at ASC", "id ASC").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list trip positions query: %w", err)
	}

	rows, err := r.executor(queryCtx).Query(queryCtx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("select trip positions: %w", err)
	}
	defer rows.Close()

	positions := make([]domaintrip.Position, 0)
	for rows.Next() {
		var position domaintrip.Position
		if err := rows.Scan(
			&position.ID,
			&position.TripID,
			&position.Coordinates.Latitude,
			&position.Coordinates.Longitude,
			&position.RecordedAt,
			&position.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan trip position: %w", err)
		}
		position.RecordedAt = position.RecordedAt.UTC()
		position.CreatedAt = position.CreatedAt.UTC()
		positions = append(positions, position)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read trip positions: %w", err)
	}
	return positions, nil
}
