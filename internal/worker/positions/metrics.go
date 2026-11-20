package positions

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type Metrics struct {
	staleChecks metric.Int64Counter
	requests    metric.Int64Counter
	duration    metric.Float64Histogram
	active      atomic.Int64
}

func NewMetrics(meter metric.Meter, queue *Queue) (*Metrics, error) {
	m := &Metrics{}
	var err error
	m.staleChecks, err = meter.Int64Counter("tripgo.stale_checks", metric.WithUnit("{trip}"))
	if err != nil {
		return nil, fmt.Errorf("create stale checks counter: %w", err)
	}
	m.requests, err = meter.Int64Counter("tripgo.position_requests", metric.WithUnit("{request}"))
	if err != nil {
		return nil, fmt.Errorf("create position requests counter: %w", err)
	}
	m.duration, err = meter.Float64Histogram("tripgo.position_requests.duration", metric.WithUnit("s"))
	if err != nil {
		return nil, fmt.Errorf("create position request duration histogram: %w", err)
	}
	_, err = meter.Int64ObservableGauge("tripgo.position_requests.queue.size",
		metric.WithUnit("{task}"),
		metric.WithInt64Callback(func(_ context.Context, observer metric.Int64Observer) error {
			observer.Observe(int64(len(queue.tasks)))
			return nil
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("create position queue size gauge: %w", err)
	}
	_, err = meter.Int64ObservableGauge("tripgo.position_requests.workers.active",
		metric.WithUnit("{worker}"),
		metric.WithInt64Callback(func(_ context.Context, observer metric.Int64Observer) error {
			observer.Observe(m.active.Load())
			return nil
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("create active position workers gauge: %w", err)
	}
	return m, nil
}

func (m *Metrics) recordEnqueue(ctx context.Context, result EnqueueResult) {
	m.staleChecks.Add(ctx, 1, metric.WithAttributes(attribute.String("result", string(result))))
}

func (m *Metrics) recordRequest(ctx context.Context, duration time.Duration, err error) {
	result := "success"
	if err != nil {
		result = "error"
	}
	attrs := metric.WithAttributes(attribute.String("result", result))
	m.requests.Add(ctx, 1, attrs)
	m.duration.Record(ctx, duration.Seconds(), attrs)
}

func (m *Metrics) recordCancelled(ctx context.Context) {
	m.requests.Add(ctx, 1, metric.WithAttributes(attribute.String("result", "cancelled")))
}
