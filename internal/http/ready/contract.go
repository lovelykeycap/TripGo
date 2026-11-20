package ready

//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=ready_test

import "context"

type readinessChecker interface {
	Check(ctx context.Context) error
}
