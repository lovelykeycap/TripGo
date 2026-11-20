package convert

import (
	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
	api "github.com/lovelykeycap/TripGo/internal/generated"
)

func Position(position domaintrip.Position) api.TripPosition {
	return api.TripPosition{
		Id:     position.ID,
		TripId: position.TripID,
		Coordinates: api.Coordinates{
			Latitude:  position.Coordinates.Latitude,
			Longitude: position.Coordinates.Longitude,
		},
		RecordedAt: position.RecordedAt,
	}
}
