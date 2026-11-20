package trip

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
)

func (r *Repository) CheckExists(ctx context.Context, id uuid.UUID) error {
	queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	var exists bool
	const query = "SELECT EXISTS (SELECT 1 FROM trips WHERE id = $1)"
	if err := r.executor(queryCtx).QueryRow(queryCtx, query, id).Scan(&exists); err != nil {
		return fmt.Errorf("check trip existence: %w", err)
	}
	if !exists {
		return domaintrip.ErrTripNotFound
	}
	return nil
}
