package trip

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
)

func (r *Repository) CountActive(ctx context.Context) (int64, error) {
	queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	query, args, err := sq.Select("COUNT(*)").
		From("trips").
		Where(sq.Eq{statusColumn: domaintrip.StatusActive}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build active trip count query: %w", err)
	}

	var count int64
	if err := r.executor(queryCtx).QueryRow(queryCtx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("select active trip count: %w", err)
	}
	return count, nil
}
