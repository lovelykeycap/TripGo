package trip

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
)

func (s *Service) Finish(ctx context.Context, id uuid.UUID) (domaintrip.Trip, error) {
	finishedAt := time.Now().UTC()
	var result domaintrip.Trip

	err := s.txManager.Do(ctx, func(txCtx context.Context) error {
		var err error
		result, err = s.tripRepo.Finish(txCtx, id, finishedAt)
		if err != nil {
			return err
		}

		fromStatus := domaintrip.StatusActive
		change := domaintrip.StatusChange{
			TripID:     result.ID,
			FromStatus: &fromStatus,
			ToStatus:   result.Status,
			ChangedAt:  finishedAt,
		}
		return s.historyRepo.Add(txCtx, change)
	})
	if err != nil {
		return domaintrip.Trip{}, fmt.Errorf("finish trip: %w", err)
	}
	return result, nil
}
