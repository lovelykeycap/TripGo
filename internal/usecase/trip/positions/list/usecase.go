package list

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
	"github.com/lovelykeycap/TripGo/internal/usecase/trip/tracing"
)

type Usecase struct {
	tripRepo     tripRepository
	positionRepo positionRepository
	tracer       trace.Tracer
}

func New(tripRepo tripRepository, positionRepo positionRepository, tracer trace.Tracer) *Usecase {
	return &Usecase{tripRepo: tripRepo, positionRepo: positionRepo, tracer: tracer}
}

func (s *Usecase) List(ctx context.Context, tripID uuid.UUID) ([]domaintrip.Position, error) {
	ctx, span := s.tracer.Start(ctx, "list trip positions")
	defer span.End()

	if err := s.tripRepo.CheckExists(ctx, tripID); err != nil {
		tracing.RecordError(span, err)
		return nil, fmt.Errorf("check trip for positions: %w", err)
	}

	positions, err := s.positionRepo.List(ctx, tripID)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, fmt.Errorf("list trip positions: %w", err)
	}
	return positions, nil
}
