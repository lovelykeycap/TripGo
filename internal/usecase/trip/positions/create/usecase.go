package create

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/trace"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
	"github.com/lovelykeycap/TripGo/internal/usecase/trip/tracing"
)

type Usecase struct {
	tripRepo     tripRepository
	positionRepo positionRepository
	txManager    txManager
	tracer       trace.Tracer
	metrics      positionMetrics
	canceler     positionRequestCanceler
	now          func() time.Time
}

func New(
	tripRepo tripRepository,
	positionRepo positionRepository,
	txManager txManager,
	tracer trace.Tracer,
	metrics positionMetrics,
	canceler positionRequestCanceler,
	now func() time.Time,
) *Usecase {
	return &Usecase{
		tripRepo:     tripRepo,
		positionRepo: positionRepo,
		txManager:    txManager,
		tracer:       tracer,
		metrics:      metrics,
		canceler:     canceler,
		now:          now,
	}
}

func (s *Usecase) Create(ctx context.Context, input Input) (domaintrip.Position, error) {
	ctx, span := s.tracer.Start(ctx, "create trip position")
	defer span.End()

	position, err := domaintrip.NewPosition(input.TripID, input.Coordinates, input.RecordedAt, s.now())
	if err != nil {
		tracing.RecordError(span, err)
		return domaintrip.Position{}, fmt.Errorf("create trip position: %w", err)
	}
	position.RecordedAt = position.RecordedAt.UTC().Truncate(time.Microsecond)
	position.CreatedAt = position.CreatedAt.UTC().Truncate(time.Microsecond)

	var saved domaintrip.Position
	err = s.txManager.Do(ctx, func(txCtx context.Context) error {
		if err := s.tripRepo.UpdateLastPosition(txCtx, position.TripID, position.RecordedAt, position.CreatedAt); err != nil {
			return err
		}
		var err error
		saved, err = s.positionRepo.Create(txCtx, position)
		return err
	})
	if err != nil {
		tracing.RecordError(span, err)
		return domaintrip.Position{}, fmt.Errorf("create trip position: %w", err)
	}

	s.canceler.Cancel(position.TripID)
	s.metrics.RecordSaved(ctx)
	return saved, nil
}
