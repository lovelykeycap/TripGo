package list

//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=list_test

import (
	"context"

	"github.com/google/uuid"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
)

type tripRepository interface {
	CheckExists(ctx context.Context, id uuid.UUID) error
}

type positionRepository interface {
	List(ctx context.Context, tripID uuid.UUID) ([]domaintrip.Position, error)
}
