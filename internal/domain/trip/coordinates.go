package trip

type Coordinates struct {
	Latitude  float64
	Longitude float64
}

func validateCoordinates(field string, point Coordinates) error {
	if !(point.Latitude >= -90 && point.Latitude <= 90) {
		return &ValidationError{Field: field + ".latitude", Reason: "must be between -90 and 90"}
	}
	if !(point.Longitude >= -180 && point.Longitude <= 180) {
		return &ValidationError{Field: field + ".longitude", Reason: "must be between -180 and 180"}
	}
	return nil
}
