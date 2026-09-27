package httpapi

import "context"

type readinessChecker interface {
	Check(ctx context.Context) error
}
