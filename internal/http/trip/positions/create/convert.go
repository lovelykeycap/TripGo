package create

import (
	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
	api "github.com/lovelykeycap/TripGo/internal/generated"
)

func toCoordinates(coordinates api.Coordinates) domaintrip.Coordinates {
	return domaintrip.Coordinates{Latitude: coordinates.Latitude, Longitude: coordinates.Longitude}
}
