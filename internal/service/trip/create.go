package trip

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
)

func (s *Service) Create(ctx context.Context, input CreateInput) (domaintrip.Trip, error) {
	now := time.Now().UTC()
	newTrip := domaintrip.Trip{
		ID:         uuid.New(),
		UserID:     input.UserID,
		DriverID:   input.DriverID,
		StartPoint: input.StartPoint,
		EndPoint:   input.EndPoint,
		Price:      input.Price,
		Status:     domaintrip.StatusActive,
		StartedAt:  now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	change := domaintrip.StatusChange{
		TripID:     newTrip.ID,
		FromStatus: nil,
		ToStatus:   newTrip.Status,
		ChangedAt:  now,
	}

	err := s.txManager.Do(ctx, func(txCtx context.Context) error {
		if err := s.tripRepo.Create(txCtx, newTrip); err != nil {
			return err
		}
		return s.historyRepo.Add(txCtx, change)
	})
	if err != nil {
		return domaintrip.Trip{}, fmt.Errorf("create trip: %w", err)
	}
	return newTrip, nil
}
