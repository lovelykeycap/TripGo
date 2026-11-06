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

func New(userID, driverID uuid.UUID, startPoint, endPoint Coordinates, price int64, at time.Time) (Trip, error) {
	if userID == uuid.Nil {
		return Trip{}, &ValidationError{Field: "user_id", Reason: "must be a nonzero UUID"}
	}
	if driverID == uuid.Nil {
		return Trip{}, &ValidationError{Field: "driver_id", Reason: "must be a nonzero UUID"}
	}
	if price < 0 {
		return Trip{}, &ValidationError{Field: "price", Reason: "must not be negative"}
	}
	if err := validateCoordinates("start_point", startPoint); err != nil {
		return Trip{}, err
	}
	if err := validateCoordinates("end_point", endPoint); err != nil {
		return Trip{}, err
	}

	return Trip{
		ID:         uuid.New(),
		UserID:     userID,
		DriverID:   driverID,
		StartPoint: startPoint,
		EndPoint:   endPoint,
		Price:      price,
		Status:     StatusActive,
		StartedAt:  at,
		CreatedAt:  at,
		UpdatedAt:  at,
	}, nil
}

func (t *Trip) Finish(at time.Time) error {
	if t.Status != StatusActive {
		return ErrTripCompleted
	}
	t.Status = StatusCompleted
	t.FinishedAt = &at
	t.UpdatedAt = at
	return nil
}
