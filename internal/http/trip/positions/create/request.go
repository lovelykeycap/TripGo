package create

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"time"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
	api "github.com/lovelykeycap/TripGo/internal/generated"
)

func decodePositionRequest(r *http.Request) (api.CreateTripPositionJSONRequestBody, error) {
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		return api.CreateTripPositionJSONRequestBody{}, errors.New("Content-Type must be application/json")
	}

	var body map[string]json.RawMessage
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&body); err != nil {
		return api.CreateTripPositionJSONRequestBody{}, fmt.Errorf("decode position data: %w", err)
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return api.CreateTripPositionJSONRequestBody{}, errors.New("request body must contain a single JSON object")
	}
	const requiredReason = "is required and must not be null"
	var coordinates map[string]json.RawMessage
	var recordedAt *time.Time
	if err := decodeFields(body, map[string]any{"coordinates": &coordinates, "recorded_at": &recordedAt}); err != nil {
		return api.CreateTripPositionJSONRequestBody{}, err
	}
	if coordinates == nil {
		return api.CreateTripPositionJSONRequestBody{}, &domaintrip.ValidationError{Field: "coordinates", Reason: requiredReason}
	}
	var latitude, longitude *float64
	if err := decodeFields(coordinates, map[string]any{"latitude": &latitude, "longitude": &longitude}); err != nil {
		return api.CreateTripPositionJSONRequestBody{}, err
	}
	switch {
	case latitude == nil:
		return api.CreateTripPositionJSONRequestBody{}, &domaintrip.ValidationError{Field: "coordinates.latitude", Reason: requiredReason}
	case longitude == nil:
		return api.CreateTripPositionJSONRequestBody{}, &domaintrip.ValidationError{Field: "coordinates.longitude", Reason: requiredReason}
	case recordedAt == nil:
		return api.CreateTripPositionJSONRequestBody{}, &domaintrip.ValidationError{Field: "recorded_at", Reason: requiredReason}
	}

	return api.CreateTripPositionJSONRequestBody{
		Coordinates: api.Coordinates{Latitude: *latitude, Longitude: *longitude},
		RecordedAt:  *recordedAt,
	}, nil
}

func decodeFields(fields map[string]json.RawMessage, destinations map[string]any) error {
	for key, value := range fields {
		destination, ok := destinations[key]
		if !ok {
			return fmt.Errorf("decode position data: unknown field %q", key)
		}
		if err := json.Unmarshal(value, destination); err != nil {
			return fmt.Errorf("decode position field %q: %w", key, err)
		}
	}
	return nil
}
