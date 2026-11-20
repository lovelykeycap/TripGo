package positions_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/goleak"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
	"github.com/lovelykeycap/TripGo/internal/worker/positions"
)

const testTimeout = 5 * time.Second

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

type manualTicker struct {
	ticks   chan time.Time
	stopped chan struct{}
}

func (t manualTicker) Ticks() <-chan time.Time { return t.ticks }
func (t manualTicker) Stop()                   { close(t.stopped) }

type listerFunc func(context.Context, time.Duration) ([]domaintrip.StaleTrip, error)

func (f listerFunc) ListStale(ctx context.Context, staleAfter time.Duration) ([]domaintrip.StaleTrip, error) {
	return f(ctx, staleAfter)
}

type requesterFunc func(context.Context, uuid.UUID, uuid.UUID) error

func (f requesterFunc) RequestPosition(ctx context.Context, driverID, tripID uuid.UUID) error {
	return f(ctx, driverID, tripID)
}

type queueRecorder struct {
	queue   *positions.Queue
	results chan positions.EnqueueResult
}

func (r queueRecorder) Enqueue(ctx context.Context, trip domaintrip.StaleTrip) positions.EnqueueResult {
	result := r.queue.Enqueue(ctx, trip)
	r.results <- result
	return result
}

func receive[T any](t *testing.T, values <-chan T) T {
	t.Helper()
	select {
	case value := <-values:
		return value
	case <-time.After(testTimeout):
		t.Fatal("timed out waiting for background component")
		var zero T
		return zero
	}
}

func start(t *testing.T, run func(context.Context)) (context.CancelFunc, <-chan struct{}) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		defer close(done)
		run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		receive(t, done)
	})
	return cancel, done
}

func testMetrics(t *testing.T, queue *positions.Queue) *positions.Metrics {
	t.Helper()
	metrics, err := positions.NewMetrics(noop.NewMeterProvider().Meter("test"), queue)
	if err != nil {
		t.Fatal(err)
	}
	return metrics
}

func testPool(t *testing.T, queue *positions.Queue, requester requesterFunc, workers int) *positions.Pool {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tracer := tracenoop.NewTracerProvider().Tracer("test")
	return positions.NewPool(queue, requester, workers, testTimeout, logger, tracer, testMetrics(t, queue))
}

func enqueue(t *testing.T, queue *positions.Queue, trip domaintrip.StaleTrip) {
	t.Helper()
	if result := queue.Enqueue(t.Context(), trip); result != positions.Queued {
		t.Fatalf("Enqueue(%s) = %s, want queued", trip.TripID, result)
	}
}

func TestScannerDeduplicatesAndDropsOnFullQueue(t *testing.T) {
	first := domaintrip.StaleTrip{TripID: uuid.New()}
	second := domaintrip.StaleTrip{TripID: uuid.New()}
	queue := positions.NewQueue(1)
	recorder := queueRecorder{queue: queue, results: make(chan positions.EnqueueResult, 4)}
	ticker := manualTicker{ticks: make(chan time.Time, 2), stopped: make(chan struct{})}
	repo := listerFunc(func(context.Context, time.Duration) ([]domaintrip.StaleTrip, error) {
		return []domaintrip.StaleTrip{first, second}, nil
	})
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tracer := tracenoop.NewTracerProvider().Tracer("test")
	scanner := positions.NewScanner(repo, recorder, time.Minute, logger, tracer, testMetrics(t, queue))
	cancel, done := start(t, func(ctx context.Context) { scanner.Run(ctx, ticker) })

	for _, round := range []positions.EnqueueResult{positions.Queued, positions.Duplicate} {
		ticker.ticks <- time.Time{}
		if got := receive(t, recorder.results); got != round {
			t.Errorf("first trip result = %s, want %s", got, round)
		}
		if got := receive(t, recorder.results); got != positions.Dropped {
			t.Errorf("second trip result = %s, want dropped", got)
		}
	}
	cancel()
	receive(t, done)
	receive(t, ticker.stopped)
}

func TestPoolSkipsCancelledTaskAndContinuesAfterPushError(t *testing.T) {
	queue := positions.NewQueue(3)
	cancelled := domaintrip.StaleTrip{TripID: uuid.New(), DriverID: uuid.New()}
	requeued := domaintrip.StaleTrip{TripID: cancelled.TripID, DriverID: uuid.New()}
	next := domaintrip.StaleTrip{TripID: uuid.New(), DriverID: uuid.New()}
	enqueue(t, queue, cancelled)
	queue.Cancel(cancelled.TripID)
	enqueue(t, queue, requeued)
	enqueue(t, queue, next)

	var drivers []uuid.UUID
	pool := testPool(t, queue, func(_ context.Context, driver, _ uuid.UUID) error {
		drivers = append(drivers, driver)
		if driver == requeued.DriverID {
			return errors.New("push service unavailable")
		}
		return nil
	}, 1)
	pool.CloseQueue()
	_, done := start(t, pool.Run)
	receive(t, done)
	if want := []uuid.UUID{requeued.DriverID, next.DriverID}; !slices.Equal(drivers, want) {
		t.Errorf("drivers called = %v, want %v", drivers, want)
	}
}

func TestPoolLimitsConcurrentRequestsAndDrainsQueue(t *testing.T) {
	synctest.Test(t, checkPoolConcurrency)
}

func checkPoolConcurrency(t *testing.T) {
	t.Helper()
	const workers, total = 3, 8
	queue := positions.NewQueue(total)
	want := make(map[uuid.UUID]bool, total)
	for range total {
		id := uuid.New()
		want[id] = true
		enqueue(t, queue, domaintrip.StaleTrip{TripID: id})
	}

	entered := make(chan uuid.UUID, total)
	release := make(chan struct{}, total)
	pool := testPool(t, queue, func(ctx context.Context, _, tripID uuid.UUID) error {
		entered <- tripID
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}, workers)
	pool.CloseQueue()
	_, done := start(t, pool.Run)
	for range workers {
		if id := receive(t, entered); !want[id] {
			t.Errorf("unexpected or duplicate trip %s", id)
		} else {
			delete(want, id)
		}
	}
	synctest.Wait()
	if len(entered) != 0 {
		t.Errorf("more than %d concurrent requests started", workers)
	}
	for range total {
		release <- struct{}{}
	}
	receive(t, done)
	for len(entered) > 0 {
		if id := <-entered; !want[id] {
			t.Errorf("unexpected or duplicate trip %s", id)
		} else {
			delete(want, id)
		}
	}
	if len(want) != 0 {
		t.Errorf("trips not processed: %v", want)
	}
}

func TestPoolCancelsInFlightRequests(t *testing.T) {
	queue := positions.NewQueue(1)
	enqueue(t, queue, domaintrip.StaleTrip{TripID: uuid.New()})
	entered := make(chan struct{})
	stopped := make(chan error, 1)
	pool := testPool(t, queue, func(ctx context.Context, _, _ uuid.UUID) error {
		close(entered)
		<-ctx.Done()
		stopped <- ctx.Err()
		return ctx.Err()
	}, 1)
	cancel, done := start(t, pool.Run)
	receive(t, entered)
	cancel()
	receive(t, done)
	if err := receive(t, stopped); !errors.Is(err, context.Canceled) {
		t.Errorf("request error = %v, want context cancellation", err)
	}
}
