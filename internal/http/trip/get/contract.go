package get

//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=get_test

import (
	"context"

	"github.com/google/uuid"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
)

type tripGetter interface {
	Get(ctx context.Context, id uuid.UUID) (domaintrip.Trip, error)
}
