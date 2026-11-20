package container

import (
	"context"
	"fmt"
	"time"

	"github.com/lovelykeycap/TripGo/internal/gateway/push"
	"github.com/lovelykeycap/TripGo/internal/worker/positions"
)

type positionRuntime struct {
	scanner  *positions.Scanner
	workers  *positions.Pool
	interval time.Duration
	gateway  *push.Gateway
}

func (p positionRuntime) start(rootCtx context.Context) func(context.Context) error {
	backgroundCtx := context.WithoutCancel(rootCtx)
	scannerCtx, cancelScanner := context.WithCancel(backgroundCtx)
	workersCtx, cancelWorkers := context.WithCancel(backgroundCtx)
	scannerDone := make(chan struct{})
	workersDone := make(chan struct{})

	ticker := positions.NewTicker(p.interval)
	go func() {
		defer close(scannerDone)
		p.scanner.Run(scannerCtx, ticker)
	}()
	go func() {
		defer close(workersDone)
		p.workers.Run(workersCtx)
	}()

	return func(ctx context.Context) error {
		defer cancelWorkers()
		defer p.gateway.Close()

		cancelScanner()
		<-scannerDone
		p.workers.CloseQueue()

		select {
		case <-workersDone:
			return nil
		case <-ctx.Done():
			cancelWorkers()
			<-workersDone
			return fmt.Errorf("drain position requests: %w", ctx.Err())
		}
	}
}
