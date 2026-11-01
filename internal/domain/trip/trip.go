package trip

import (
	"time"

	"github.com/google/uuid"
)

type Trip struct {
	ID       uuid.UUID
	UserID   uuid.UUID
	DriverID uuid.UUID

	StartPoint Coordinates
	EndPoint   Coordinates
	Price      int64

	Status     Status
	StartedAt  time.Time
	FinishedAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}
