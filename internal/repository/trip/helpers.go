package trip

import (
	"context"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
	"github.com/lovelykeycap/TripGo/internal/infra/postgres"
)

func tripColumns() []string {
    return []string{
        "id", "user_id", "driver_id",
        "start_latitude", "start_longitude", "end_latitude", "end_longitude",
        "price", "status", "started_at", "finished_at", "created_at", "updated_at",
    }
}

func tripValues(trip domaintrip.Trip) []any {
    return []any{
        trip.ID, trip.UserID, trip.DriverID,
        trip.StartPoint.Latitude, trip.StartPoint.Longitude,
        trip.EndPoint.Latitude, trip.EndPoint.Longitude,
        trip.Price, trip.Status, trip.StartedAt, trip.FinishedAt,
        trip.CreatedAt, trip.UpdatedAt,
    }
}

func tripTimeToUtc(trip *domaintrip.Trip) {
	trip.CreatedAt = trip.CreatedAt.UTC()
	trip.StartedAt = trip.StartedAt.UTC()

	if trip.FinishedAt != nil {
		finishedAt := trip.FinishedAt.UTC()
		trip.FinishedAt = &finishedAt
	}
	
	trip.UpdatedAt = trip.UpdatedAt.UTC()
}

func queryRowToTrip(queryCtx context.Context, 
	executor postgres.Executor, 
	query string, args... any) (domaintrip.Trip, error) {
		var result domaintrip.Trip
		err := executor.QueryRow(queryCtx, query, args...).Scan(
		&result.ID,
		&result.UserID,
		&result.DriverID,
		&result.StartPoint.Latitude,
		&result.StartPoint.Longitude,
		&result.EndPoint.Latitude,
		&result.EndPoint.Longitude,
		&result.Price,
		&result.Status,
		&result.StartedAt,
		&result.FinishedAt,
		&result.CreatedAt,
		&result.UpdatedAt,
	)

	return result, err
}