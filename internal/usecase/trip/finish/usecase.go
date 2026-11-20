package finish

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
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

func (s *Usecase) Finish(ctx context.Context, id uuid.UUID) (domaintrip.Trip, error) {
	ctx, span := s.tracer.Start(ctx, "finish trip")
	defer span.End()

	result, err := s.tripRepo.Get(ctx, id)
	if err != nil {
		tracing.RecordError(span, err)
		return domaintrip.Trip{}, fmt.Errorf("get trip to finish: %w", err)
	}

	fromStatus := result.Status
	finishedAt := time.Now().UTC().Truncate(time.Microsecond)
	if err := result.Finish(finishedAt); err != nil {
		tracing.RecordError(span, err)
		return domaintrip.Trip{}, fmt.Errorf("finish trip: %w", err)
	}
	change := domaintrip.StatusChange{
		TripID:     result.ID,
		FromStatus: &fromStatus,
		ToStatus:   result.Status,
		ChangedAt:  finishedAt,
	}

	err = s.txManager.Do(ctx, func(txCtx context.Context) error {
		if err := s.tripRepo.Finish(txCtx, result); err != nil {
			return err
		}
		return s.historyRepo.Add(txCtx, change)
	})
	if err != nil {
		tracing.RecordError(span, err)
		return domaintrip.Trip{}, fmt.Errorf("finish trip: %w", err)
	}
	s.metrics.RecordCompleted(ctx)
	s.logger.InfoContext(ctx, "trip completed", "trip_id", result.ID)
	return result, nil
}
