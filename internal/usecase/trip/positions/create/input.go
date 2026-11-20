package create

import (
	"time"

	"github.com/google/uuid"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
)

type Input struct {
	TripID      uuid.UUID
	Coordinates domaintrip.Coordinates
	RecordedAt  time.Time
}
