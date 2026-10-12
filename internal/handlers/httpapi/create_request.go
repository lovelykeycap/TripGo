package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"slices"

	"github.com/google/uuid"

	api "github.com/lovelykeycap/TripGo/internal/generated"
)

func decodeCreateTripRequest(r *http.Request) (api.CreateTripJSONRequestBody, error) {
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		return api.CreateTripJSONRequestBody{}, errors.New("Content-Type must be application/json")
	}

	decoder := json.NewDecoder(r.Body)
	var body json.RawMessage
	if err := decoder.Decode(&body); err != nil {
		return api.CreateTripJSONRequestBody{}, fmt.Errorf("decode request body: %w", err)
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return api.CreateTripJSONRequestBody{}, errors.New("request body must contain a single JSON object")
	}

	fields, err := validateRequiredObject(body, "user_id", "driver_id", "start_point", "end_point", "price")
	if err != nil {
		return api.CreateTripJSONRequestBody{}, err
	}
	for _, name := range []string{"start_point", "end_point"} {
		if _, err := validateRequiredObject(fields[name], "latitude", "longitude"); err != nil {
			return api.CreateTripJSONRequestBody{}, fmt.Errorf("validate %s: %w", name, err)
		}
	}

	var request api.CreateTripJSONRequestBody
	if err := json.Unmarshal(body, &request); err != nil {
		return api.CreateTripJSONRequestBody{}, fmt.Errorf("decode trip data: %w", err)
	}
	if request.UserId == uuid.Nil || request.DriverId == uuid.Nil {
		return api.CreateTripJSONRequestBody{}, errors.New("user_id and driver_id must be nonzero UUIDs")
	}
	if request.Price < 0 {
		return api.CreateTripJSONRequestBody{}, errors.New("price must not be negative")
	}
	if !validCoordinates(request.StartPoint) || !validCoordinates(request.EndPoint) {
		return api.CreateTripJSONRequestBody{}, errors.New("coordinates are outside the allowed ranges")
	}
	return request, nil
}

func validateRequiredObject(body json.RawMessage, requiredFields ...string) (map[string]json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return nil, fmt.Errorf("decode JSON object: %w", err)
	}
	for _, name := range requiredFields {
		value, ok := fields[name]
		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return nil, fmt.Errorf("field %q is required and must not be null", name)
		}
	}
	for name := range fields {
		if !slices.Contains(requiredFields, name) {
			return nil, fmt.Errorf("unknown field %q", name)
		}
	}
	return fields, nil
}

func validCoordinates(point api.Coordinates) bool {
	return point.Latitude >= -90 && point.Latitude <= 90 &&
		point.Longitude >= -180 && point.Longitude <= 180
}
