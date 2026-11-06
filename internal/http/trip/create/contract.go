package create

//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=create_test

import (
	"context"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
	tripcreate "github.com/lovelykeycap/TripGo/internal/usecase/trip/create"
)

type tripCreator interface {
	Create(ctx context.Context, input tripcreate.Input) (domaintrip.Trip, error)
}
