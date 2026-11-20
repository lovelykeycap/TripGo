package tripmetrics

import "context"

type activeTripCounter interface {
	CountActive(ctx context.Context) (int64, error)
}
