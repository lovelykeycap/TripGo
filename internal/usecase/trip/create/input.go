package create

import (
	"github.com/google/uuid"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
)

type Input struct {
	UserID   uuid.UUID
	DriverID uuid.UUID

	StartPoint domaintrip.Coordinates
	EndPoint   domaintrip.Coordinates
	Price      int64
}
