package create

import (
	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
	api "github.com/lovelykeycap/TripGo/internal/generated"
	tripcreate "github.com/lovelykeycap/TripGo/internal/usecase/trip/create"
)

func toInput(request api.CreateTripJSONRequestBody) tripcreate.Input {
	return tripcreate.Input{
		UserID:   request.UserId,
		DriverID: request.DriverId,
		StartPoint: domaintrip.Coordinates{
			Latitude:  request.StartPoint.Latitude,
			Longitude: request.StartPoint.Longitude,
		},
		EndPoint: domaintrip.Coordinates{
			Latitude:  request.EndPoint.Latitude,
			Longitude: request.EndPoint.Longitude,
		},
		Price: request.Price,
	}
}
