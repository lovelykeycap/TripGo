package container

import (
	"context"
	"time"

	"github.com/lovelykeycap/TripGo/internal/infra/postgres/telemetry"
)

func shutdownTelemetry(rootCtx context.Context, providers *telemetry.Providers, timeout time.Duration) error {
	shutdownCtx, cancel := context.WithTimeout(
		context.WithoutCancel(rootCtx),
		timeout,
	)
	defer cancel()

	return providers.Shutdown(shutdownCtx)
}
