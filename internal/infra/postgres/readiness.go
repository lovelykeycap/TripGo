package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ReadinessChecker struct {
	pool    *pgxpool.Pool
	timeout time.Duration
}

func NewReadinessChecker(pool *pgxpool.Pool, timeout time.Duration) *ReadinessChecker {
	return &ReadinessChecker{
		pool:    pool,
		timeout: timeout,
	}
}

func (c *ReadinessChecker) Check(ctx context.Context) error {
	checkCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	if err := c.pool.Ping(checkCtx); err != nil {
		return fmt.Errorf("check PostgreSQL readiness: %w", err)
	}
	return nil
}
