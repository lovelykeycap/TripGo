package create

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/trace"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
	"github.com/lovelykeycap/TripGo/internal/usecase/trip/tracing"
)

type Usecase struct {
	tripRepo    tripRepository
	historyRepo tripHistoryRepository
	txManager   txManager
	tracer      trace.Tracer
	logger      *slog.Logger
	metrics     tripMetrics
}

func New(
	tripRepo tripRepository,
	historyRepo tripHistoryRepository,
	txManager txManager,
	tracer trace.Tracer,
	logger *slog.Logger,
	metrics tripMetrics,
) *Usecase {
	return &Usecase{
		tripRepo:    tripRepo,
		historyRepo: historyRepo,
		txManager:   txManager,
		tracer:      tracer,
		logger:      logger,
		metrics:     metrics,
	}
}

func (s *Usecase) Create(ctx context.Context, input Input) (domaintrip.Trip, error) {
	ctx, span := s.tracer.Start(ctx, "create trip")
	defer span.End()

	now := time.Now().Truncate(time.Microsecond).UTC()
	newTrip, err := domaintrip.New(
		input.UserID, input.DriverID, input.StartPoint, input.EndPoint, input.Price, now,
	)
	if err != nil {
		tracing.RecordError(span, err)
		return domaintrip.Trip{}, fmt.Errorf("create trip: %w", err)
	}
	change := domaintrip.StatusChange{
		TripID:     newTrip.ID,
		FromStatus: nil,
		ToStatus:   newTrip.Status,
		ChangedAt:  now,
	}

	err = s.txManager.Do(ctx, func(txCtx context.Context) error {
		if err := s.tripRepo.Create(txCtx, newTrip); err != nil {
			return err
		}
		return s.historyRepo.Add(txCtx, change)
	})
	if err != nil {
		tracing.RecordError(span, err)
		return domaintrip.Trip{}, fmt.Errorf("create trip: %w", err)
	}
	s.metrics.RecordCreated(ctx)
	s.logger.InfoContext(ctx, "trip created", "trip_id", newTrip.ID)
	return newTrip, nil
}
