package positions

import (
	"context"
	"time"

	"github.com/google/uuid"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
)

type staleTripLister interface {
	ListStale(ctx context.Context, staleAfter time.Duration) ([]domaintrip.StaleTrip, error)
}

type taskEnqueuer interface {
	Enqueue(ctx context.Context, task domaintrip.StaleTrip) EnqueueResult
}

type ticker interface {
	Ticks() <-chan time.Time
	Stop()
}

type positionRequester interface {
	RequestPosition(ctx context.Context, driverID, tripID uuid.UUID) error
}
