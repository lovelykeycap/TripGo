package create

//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=create_test

import (
	"context"
	"time"

	"github.com/google/uuid"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
)

type tripRepository interface {
	UpdateLastPosition(ctx context.Context, id uuid.UUID, recordedAt, updatedAt time.Time) error
}

type positionRepository interface {
	Create(ctx context.Context, position domaintrip.Position) (domaintrip.Position, error)
}

type txManager interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

type positionMetrics interface {
	RecordSaved(ctx context.Context)
}

type positionRequestCanceler interface {
	Cancel(tripID uuid.UUID)
}
