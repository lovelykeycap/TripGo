package create

//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=create_test

import (
	"context"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
)

type tripRepository interface {
	Create(ctx context.Context, newTrip domaintrip.Trip) error
}

type tripHistoryRepository interface {
	Add(ctx context.Context, change domaintrip.StatusChange) error
}

type txManager interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

type tripMetrics interface {
	RecordCreated(ctx context.Context)
}
