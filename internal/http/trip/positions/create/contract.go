package create

//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=create_test

import (
	"context"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
	positioncreate "github.com/lovelykeycap/TripGo/internal/usecase/trip/positions/create"
)

type positionCreator interface {
	Create(ctx context.Context, input positioncreate.Input) (domaintrip.Position, error)
}
