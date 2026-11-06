package create

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"slices"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
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

	fields, err := validateRequiredObject(body, "", "user_id", "driver_id", "start_point", "end_point", "price")
	if err != nil {
		return api.CreateTripJSONRequestBody{}, err
	}
	for _, name := range []string{"start_point", "end_point"} {
		if _, err := validateRequiredObject(fields[name], name+".", "latitude", "longitude"); err != nil {
			return api.CreateTripJSONRequestBody{}, err
		}
	}

	var request api.CreateTripJSONRequestBody
	if err := json.Unmarshal(body, &request); err != nil {
		return api.CreateTripJSONRequestBody{}, fmt.Errorf("decode trip data: %w", err)
	}
	return request, nil
}

func validateRequiredObject(body json.RawMessage, prefix string, requiredFields ...string) (map[string]json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return nil, fmt.Errorf("decode JSON object: %w", err)
	}
	for _, name := range requiredFields {
		value, ok := fields[name]
		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return nil, &domaintrip.ValidationError{Field: prefix + name, Reason: "is required and must not be null"}
		}
	}
	for name := range fields {
		if !slices.Contains(requiredFields, name) {
			return nil, &domaintrip.ValidationError{Field: prefix + name, Reason: "unknown field"}
		}
	}
	return fields, nil
}
