package positions

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"

	domaintrip "github.com/lovelykeycap/TripGo/internal/domain/trip"
)

type EnqueueResult string

const (
	Queued    EnqueueResult = "queued"
	Duplicate EnqueueResult = "duplicate"
	Dropped   EnqueueResult = "dropped"
)

type Queue struct {
	tasks   chan *queuedTask
	mu      sync.Mutex
	pending map[uuid.UUID]*queuedTask
	closed  bool
}

type queuedTask struct {
	trip    domaintrip.StaleTrip
	source  trace.SpanContext
	started bool
}

func NewQueue(capacity int) *Queue {
	return &Queue{
		tasks:   make(chan *queuedTask, capacity),
		pending: make(map[uuid.UUID]*queuedTask, capacity),
	}
}

func (q *Queue) Enqueue(ctx context.Context, task domaintrip.StaleTrip) EnqueueResult {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return Dropped
	}
	if _, exists := q.pending[task.TripID]; exists {
		return Duplicate
	}
	queued := &queuedTask{trip: task, source: trace.SpanContextFromContext(ctx)}
	select {
	case q.tasks <- queued:
		q.pending[task.TripID] = queued
		return Queued
	default:
		return Dropped
	}
}

func (q *Queue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()

	if !q.closed {
		q.closed = true
		close(q.tasks)
	}
}

func (q *Queue) Cancel(tripID uuid.UUID) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if task := q.pending[tripID]; task != nil && !task.started {
		delete(q.pending, tripID)
	}
}

func (q *Queue) start(task *queuedTask) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.pending[task.trip.TripID] != task {
		return false
	}
	task.started = true
	return true
}

func (q *Queue) finish(tripID uuid.UUID) {
	q.mu.Lock()
	defer q.mu.Unlock()
	delete(q.pending, tripID)
}
