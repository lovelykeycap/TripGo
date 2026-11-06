package triphistory

import (
	"context"
	"time"

	"github.com/lovelykeycap/TripGo/internal/infra/postgres"
)

type Repository struct {
	pool         executor
	queryTimeout time.Duration
}

func New(pool executor, queryTimeout time.Duration) *Repository {
	return &Repository{
		pool:         pool,
		queryTimeout: queryTimeout,
	}
}

func (r *Repository) executor(ctx context.Context) executor {
	if tx, ok := postgres.TransactionFromContext(ctx); ok {
		return tx
	}
	return r.pool
}
