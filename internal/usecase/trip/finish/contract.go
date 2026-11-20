package finish

//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=finish_test

import (
	"context"

	"github.com/google/uuid"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
)

type tripRepository interface {
	Get(ctx context.Context, id uuid.UUID) (domaintrip.Trip, error)
	Finish(ctx context.Context, finishedTrip domaintrip.Trip) error
}

type tripHistoryRepository interface {
	Add(ctx context.Context, change domaintrip.StatusChange) error
}

type txManager interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

type tripMetrics interface {
	RecordCompleted(ctx context.Context)
}
