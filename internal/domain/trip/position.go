package trip

import (
	"time"

	"github.com/google/uuid"
)

type Position struct {
	ID          int64
	TripID      uuid.UUID
	Coordinates Coordinates
	RecordedAt  time.Time
	CreatedAt   time.Time
}

func NewPosition(tripID uuid.UUID, coordinates Coordinates, recordedAt, at time.Time) (Position, error) {
	if tripID == uuid.Nil {
		return Position{}, &ValidationError{Field: "trip_id", Reason: nonzeroUUIDReason}
	}
	if err := validateCoordinates("coordinates", coordinates); err != nil {
		return Position{}, err
	}
	if recordedAt.After(at.Add(time.Minute)) {
		return Position{}, &ValidationError{Field: "recorded_at", Reason: "must not be more than one minute in the future"}
	}

	return Position{
		TripID:      tripID,
		Coordinates: coordinates,
		RecordedAt:  recordedAt,
		CreatedAt:   at,
	}, nil
}
