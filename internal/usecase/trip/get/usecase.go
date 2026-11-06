package get

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
	"github.com/lovelykeycap/TripGo/internal/usecase/trip/tracing"
)

type Usecase struct {
	tripRepo tripRepository
	tracer   trace.Tracer
}

func New(tripRepo tripRepository, tracer trace.Tracer) *Usecase {
	return &Usecase{tripRepo: tripRepo, tracer: tracer}
}

func (s *Usecase) Get(ctx context.Context, id uuid.UUID) (domaintrip.Trip, error) {
	ctx, span := s.tracer.Start(ctx, "get trip")
	defer span.End()

	result, err := s.tripRepo.Get(ctx, id)
	if err != nil {
		tracing.RecordError(span, err)
		return domaintrip.Trip{}, fmt.Errorf("get trip from repository: %w", err)
	}
	return result, nil
}
