package positions

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Pool struct {
	queue          *Queue
	requester      positionRequester
	workers        int
	requestTimeout time.Duration
	logger         *slog.Logger
	tracer         trace.Tracer
	metrics        *Metrics
}

func NewPool(queue *Queue, requester positionRequester, workers int, requestTimeout time.Duration, logger *slog.Logger, tracer trace.Tracer, metrics *Metrics) *Pool {
	return &Pool{
		queue: queue, requester: requester, workers: workers, requestTimeout: requestTimeout,
		logger: logger, tracer: tracer, metrics: metrics,
	}
}

func (p *Pool) CloseQueue() {
	p.queue.Close()
}

func (p *Pool) Run(ctx context.Context) {
	var workers sync.WaitGroup
	for range p.workers {
		workers.Go(func() {
			p.work(ctx)
		})
	}
	workers.Wait()
}

func (p *Pool) work(ctx context.Context) {
	for ctx.Err() == nil {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-p.queue.tasks:
			if !ok {
				return
			}
			if !p.queue.start(task) {
				p.metrics.recordCancelled(ctx)
				continue
			}
			p.request(ctx, task)
		}
	}
}

func (p *Pool) request(ctx context.Context, task *queuedTask) {
	defer p.queue.finish(task.trip.TripID)
	if ctx.Err() != nil {
		return
	}
	ctx, span := p.tracer.Start(ctx, "process position request",
		trace.WithNewRoot(),
		trace.WithLinks(trace.Link{SpanContext: task.source}),
	)
	defer span.End()

	requestCtx, cancel := context.WithTimeout(ctx, p.requestTimeout)
	defer cancel()

	p.metrics.active.Add(1)
	defer p.metrics.active.Add(-1)
	started := time.Now()
	err := p.requester.RequestPosition(requestCtx, task.trip.DriverID, task.trip.TripID)
	p.metrics.recordRequest(ctx, time.Since(started), err)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		p.logger.WarnContext(ctx, "failed to request position", "trip_id", task.trip.TripID, "error", err)
	}
}
