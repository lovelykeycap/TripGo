package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type transactionKey struct{}

func TransactionFromContext(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(transactionKey{}).(pgx.Tx)
	return tx, ok
}
