package finish

//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=finish_test

import (
	"context"

	"github.com/google/uuid"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
)

type tripFinisher interface {
	Finish(ctx context.Context, id uuid.UUID) (domaintrip.Trip, error)
}
