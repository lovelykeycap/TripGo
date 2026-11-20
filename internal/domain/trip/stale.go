package trip

import "github.com/google/uuid"

type StaleTrip struct {
	TripID   uuid.UUID
	DriverID uuid.UUID
}
