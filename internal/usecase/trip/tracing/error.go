package tracing

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
)

func RecordError(span trace.Span, err error) {
	if err == nil || !span.IsRecording() {
		return
	}

	errorType := fmt.Sprintf("%T", err)
	var validationErr *domaintrip.ValidationError
	switch {
	case errors.Is(err, domaintrip.ErrTripNotFound):
		errorType = "trip_not_found"
	case errors.Is(err, domaintrip.ErrTripCompleted):
		errorType = "trip_completed"
	case errors.Is(err, domaintrip.ErrDriverBusy):
		errorType = "driver_busy"
	case errors.As(err, &validationErr):
		errorType = "invalid_request"
	case errors.Is(err, context.DeadlineExceeded):
		errorType = "context.deadline_exceeded"
	case errors.Is(err, context.Canceled):
		errorType = "context.canceled"
	}

	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	span.SetAttributes(attribute.String("error.type", errorType))
}
