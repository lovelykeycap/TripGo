package tripposition

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
)

func (r *Repository) Create(ctx context.Context, position domaintrip.Position) (domaintrip.Position, error) {
	queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	query, args, err := sq.Insert("trip_positions").
		Columns("trip_id", "latitude", "longitude", "recorded_at", "created_at").
		Values(position.TripID, position.Coordinates.Latitude, position.Coordinates.Longitude, position.RecordedAt, position.CreatedAt).
		Suffix("RETURNING id, recorded_at, created_at").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return domaintrip.Position{}, fmt.Errorf("build insert trip position query: %w", err)
	}

	result := position
	if err := r.executor(queryCtx).QueryRow(queryCtx, query, args...).Scan(&result.ID, &result.RecordedAt, &result.CreatedAt); err != nil {
		return domaintrip.Position{}, fmt.Errorf("insert trip position: %w", err)
	}
	result.RecordedAt = result.RecordedAt.UTC()
	result.CreatedAt = result.CreatedAt.UTC()
	return result, nil
}
