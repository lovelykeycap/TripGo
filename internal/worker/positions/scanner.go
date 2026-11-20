package positions

import (
	"context"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Scanner struct {
	trips      staleTripLister
	queue      taskEnqueuer
	staleAfter time.Duration
	logger     *slog.Logger
	tracer     trace.Tracer
	metrics    *Metrics
}

func NewScanner(trips staleTripLister, queue taskEnqueuer, staleAfter time.Duration, logger *slog.Logger, tracer trace.Tracer, metrics *Metrics) *Scanner {
	return &Scanner{trips: trips, queue: queue, staleAfter: staleAfter, logger: logger, tracer: tracer, metrics: metrics}
}

func (s *Scanner) Run(ctx context.Context, ticker ticker) {
	defer ticker.Stop()
	ticks := ticker.Ticks()
	for ctx.Err() == nil {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-ticks:
			if !ok {
				return
			}
			s.scan(ctx)
		}
	}
}

func (s *Scanner) scan(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}
	ctx, span := s.tracer.Start(ctx, "scan stale trips", trace.WithNewRoot())
	defer span.End()

	trips, err := s.trips.ListStale(ctx, s.staleAfter)
	if err != nil {
		if ctx.Err() == nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			s.logger.WarnContext(ctx, "failed to list stale trips", "error", err)
		}
		return
	}
	for _, trip := range trips {
		if ctx.Err() != nil {
			return
		}
		result := s.queue.Enqueue(ctx, trip)
		s.metrics.recordEnqueue(ctx, result)
		if result == Dropped {
			s.logger.WarnContext(ctx, "position request queue is full", "trip_id", trip.TripID)
		}
	}
}
