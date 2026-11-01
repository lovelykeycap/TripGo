package trip

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
)

func (s *Service) Get(ctx context.Context, id uuid.UUID) (domaintrip.Trip, error) {
	result, err := s.tripRepo.Get(ctx, id)
	if err != nil {
		return domaintrip.Trip{}, fmt.Errorf("get trip from repository: %w", err)
	}
	return result, nil
}
