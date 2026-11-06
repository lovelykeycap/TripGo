package tripmetrics

import (
	"context"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel/metric"
)

type Metrics struct {
	created   metric.Int64Counter
	completed metric.Int64Counter
}

func New(meter metric.Meter, trips activeTripCounter, logger *slog.Logger) (*Metrics, error) {
	created, err := meter.Int64Counter("tripgo.trips.created",
		metric.WithDescription("Number of successfully created trips"),
		metric.WithUnit("{trip}"),
	)
	if err != nil {
		return nil, fmt.Errorf("create trips created counter: %w", err)
	}
	completed, err := meter.Int64Counter("tripgo.trips.completed",
		metric.WithDescription("Number of successfully completed trips"),
		metric.WithUnit("{trip}"),
	)
	if err != nil {
		return nil, fmt.Errorf("create trips completed counter: %w", err)
	}
	_, err = meter.Int64ObservableGauge("tripgo.trips.active",
		metric.WithDescription("Number of currently active trips"),
		metric.WithUnit("{trip}"),
		metric.WithInt64Callback(func(ctx context.Context, observer metric.Int64Observer) error {
			count, err := trips.CountActive(ctx)
			if err != nil {
				// Skip this observation so other metrics can still be exported.
				logger.WarnContext(ctx, "failed to collect active trips", "error", err)
				return nil
			}
			observer.Observe(count)
			return nil
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("create active trips gauge: %w", err)
	}
	return &Metrics{created: created, completed: completed}, nil
}

func (m *Metrics) RecordCreated(ctx context.Context) {
	m.created.Add(ctx, 1)
}

func (m *Metrics) RecordCompleted(ctx context.Context) {
	m.completed.Add(ctx, 1)
}
