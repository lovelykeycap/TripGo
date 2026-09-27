package trip

import (
	"context"
	"time"

	"github.com/google/uuid"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
)

type TripService interface {
	Create(ctx context.Context, input CreateInput) (domaintrip.Trip, error)
	Get(ctx context.Context, id uuid.UUID) (domaintrip.Trip, error)
	Finish(ctx context.Context, id uuid.UUID) (domaintrip.Trip, error)
}

type tripRepository interface {
	Create(ctx context.Context, newTrip domaintrip.Trip) error
	Get(ctx context.Context, id uuid.UUID) (domaintrip.Trip, error)
	Finish(ctx context.Context, id uuid.UUID, finishedAt time.Time) (domaintrip.Trip, error)
}

type tripHistoryRepository interface {
	Add(ctx context.Context, change domaintrip.StatusChange) error
}

type txManager interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}
