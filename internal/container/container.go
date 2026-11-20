package container

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lovelykeycap/TripGo/internal/config"
	"github.com/lovelykeycap/TripGo/internal/gateway/push"
	httptransport "github.com/lovelykeycap/TripGo/internal/http"
	"github.com/lovelykeycap/TripGo/internal/http/health"
	"github.com/lovelykeycap/TripGo/internal/http/middleware"
	"github.com/lovelykeycap/TripGo/internal/http/ready"
	"github.com/lovelykeycap/TripGo/internal/http/response"
	"github.com/lovelykeycap/TripGo/internal/http/trip/create"
	"github.com/lovelykeycap/TripGo/internal/http/trip/finish"
	"github.com/lovelykeycap/TripGo/internal/http/trip/get"
	positioncreate "github.com/lovelykeycap/TripGo/internal/http/trip/positions/create"
	positionlist "github.com/lovelykeycap/TripGo/internal/http/trip/positions/list"
	"github.com/lovelykeycap/TripGo/internal/infra/postgres"
	"github.com/lovelykeycap/TripGo/internal/infra/postgres/telemetry"
	"github.com/lovelykeycap/TripGo/internal/infra/telemetry/tripmetrics"
	triprepository "github.com/lovelykeycap/TripGo/internal/repository/trip"
	"github.com/lovelykeycap/TripGo/internal/repository/triphistory"
	"github.com/lovelykeycap/TripGo/internal/repository/tripposition"
	tripcreate "github.com/lovelykeycap/TripGo/internal/usecase/trip/create"
	tripfinish "github.com/lovelykeycap/TripGo/internal/usecase/trip/finish"
	tripget "github.com/lovelykeycap/TripGo/internal/usecase/trip/get"
	createposition "github.com/lovelykeycap/TripGo/internal/usecase/trip/positions/create"
	listposition "github.com/lovelykeycap/TripGo/internal/usecase/trip/positions/list"
	"github.com/lovelykeycap/TripGo/internal/worker/positions"
)

type Container struct {
	server          *http.Server
	pool            *pgxpool.Pool
	logger          *slog.Logger
	shutdownTimeout time.Duration
	telemetry       *telemetry.Providers
	positions       positionRuntime
}

func New(rootCtx context.Context, cfg config.Config, logger *slog.Logger) (*Container, error) {
	providers, err := telemetry.New(rootCtx, cfg.ShutdownTimeout, logger)
	if err != nil {
		return nil, fmt.Errorf("initialize telemetry: %w", err)
	}

	traceProvider := providers.TracerProvider
	pushTracer := traceProvider.Tracer("github.com/lovelykeycap/TripGo/internal/gateway/push")
	pushGateway, err := push.New(cfg.PushHTTPURL, cfg.PushRequestTimeout, pushTracer)
	if err != nil {
		return nil, errors.Join(
			fmt.Errorf("initialize push gateway: %w", err),
			shutdownTelemetry(rootCtx, providers, cfg.ShutdownTimeout),
		)
	}

	positionQueue := positions.NewQueue(int(cfg.PositionQueueSize))
	const positionScope = "github.com/lovelykeycap/TripGo/internal/worker/positions"
	positionMetrics, err := positions.NewMetrics(providers.MeterProvider.Meter(positionScope), positionQueue)
	if err != nil {
		return nil, errors.Join(
			fmt.Errorf("initialize position metrics: %w", err),
			shutdownTelemetry(rootCtx, providers, cfg.ShutdownTimeout),
		)
	}
	positionTracer := traceProvider.Tracer(positionScope)

	httpMeter := providers.MeterProvider.Meter("github.com/lovelykeycap/TripGo/internal/http/middleware")
	metricsMiddleware, err := middleware.Metrics(httpMeter)
	if err != nil {
		return nil, errors.Join(
			fmt.Errorf("initialize HTTP metrics: %w", err),
			shutdownTelemetry(rootCtx, providers, cfg.ShutdownTimeout),
		)
	}

	pool, err := postgres.NewPool(rootCtx, cfg, traceProvider, providers.MeterProvider)
	if err != nil {
		if err := shutdownTelemetry(rootCtx, providers, cfg.ShutdownTimeout); err != nil {
			logger.Error("shutdown telemetry", "error", err)
		}
		return nil, fmt.Errorf("initialize PostgreSQL: %w", err)
	}
	logger.Info("PostgreSQL connection established")

	readiness := postgres.NewReadinessChecker(pool, cfg.DatabaseQueryTimeout)

	tripRepo := triprepository.New(pool, cfg.DatabaseQueryTimeout)
	historyRepo := triphistory.New(pool, cfg.DatabaseQueryTimeout)
	positionRepo := tripposition.New(pool, cfg.DatabaseQueryTimeout)
	txManager := postgres.NewTxManager(pool, cfg.DatabaseQueryTimeout)

	tripMeter := providers.MeterProvider.Meter("github.com/lovelykeycap/TripGo/internal/infra/telemetry/tripmetrics")
	tripMetrics, err := tripmetrics.New(tripMeter, tripRepo, logger)
	if err != nil {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(rootCtx), cfg.ShutdownTimeout)
		defer cancel()
		telemetryErr := providers.Shutdown(cleanupCtx)
		poolErr := postgres.ClosePool(cleanupCtx, pool)
		return nil, errors.Join(fmt.Errorf("initialize trip metrics: %w", err), telemetryErr, poolErr)
	}

	createTracer := traceProvider.Tracer("github.com/lovelykeycap/TripGo/internal/usecase/trip/create")
	createTrip := tripcreate.New(tripRepo, historyRepo, txManager, createTracer, logger, tripMetrics)
	getTracer := traceProvider.Tracer("github.com/lovelykeycap/TripGo/internal/usecase/trip/get")
	getTrip := tripget.New(tripRepo, getTracer)
	finishTracer := traceProvider.Tracer("github.com/lovelykeycap/TripGo/internal/usecase/trip/finish")
	finishTrip := tripfinish.New(tripRepo, historyRepo, txManager, finishTracer, logger, tripMetrics)
	createPositionTracer := traceProvider.Tracer("github.com/lovelykeycap/TripGo/internal/usecase/trip/positions/create")
	createPosition := createposition.New(tripRepo, positionRepo, txManager, createPositionTracer, tripMetrics, positionQueue, time.Now)
	listPositionsTracer := traceProvider.Tracer("github.com/lovelykeycap/TripGo/internal/usecase/trip/positions/list")
	listPositions := listposition.New(tripRepo, positionRepo, listPositionsTracer)

	responses := response.New(logger)

	createHandler := create.New(createTrip, responses)
	getHandler := get.New(getTrip, responses)
	finishHandler := finish.New(finishTrip, responses)
	healthHandler := health.New(responses)
	readyHandler := ready.New(readiness, responses, logger)

	apiServer := httptransport.NewServer(httptransport.Handlers{
		Create:         createHandler,
		Get:            getHandler,
		Finish:         finishHandler,
		Health:         healthHandler,
		Ready:          readyHandler,
		CreatePosition: positioncreate.New(createPosition, responses),
		ListPositions:  positionlist.New(listPositions, responses),
	})
	middlewares := []func(http.Handler) http.Handler{
		middleware.Recover(logger, responses),
		middleware.Tracing(traceProvider, providers.MeterProvider),
		middleware.Logging(logger),
		metricsMiddleware,
	}
	router := httptransport.NewRouter(apiServer, responses, middlewares)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadTimeout:       cfg.HTTPReadTimeout,
		ReadHeaderTimeout: cfg.HTTPReadHeaderTimeout,
		WriteTimeout:      cfg.HTTPWriteTimeout,
		IdleTimeout:       cfg.HTTPIdleTimeout,
	}

	container := &Container{
		server:          server,
		pool:            pool,
		logger:          logger,
		shutdownTimeout: cfg.ShutdownTimeout,
		telemetry:       providers,
		positions: positionRuntime{
			scanner:  positions.NewScanner(tripRepo, positionQueue, cfg.PositionStaleAfter, logger, positionTracer, positionMetrics),
			workers:  positions.NewPool(positionQueue, pushGateway, int(cfg.PositionWorkers), cfg.PushRequestTimeout, logger, positionTracer, positionMetrics),
			interval: cfg.PositionCheckInterval,
			gateway:  pushGateway,
		},
	}

	return container, nil
}

func (a *Container) Run(rootCtx context.Context) error {
	signalCtx, stop := signal.NotifyContext(rootCtx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	stopPositions := a.positions.start(rootCtx)

	a.logger.Info("starting HTTP server", "addr", a.server.Addr)

	go func() {
		serverErr <- a.server.ListenAndServe()
	}()

	var serveErr error

	select {
	case serveErr = <-serverErr:
	case <-signalCtx.Done():
	}
	stop()

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(rootCtx), a.shutdownTimeout)
	defer cancel()

	shutdownErr := a.shutdown(shutdownCtx, stopPositions)
	if serveErr == nil {
		serveErr = <-serverErr
	}

	if errors.Is(serveErr, http.ErrServerClosed) {
		serveErr = nil
	} else if serveErr != nil {
		serveErr = fmt.Errorf("serve HTTP: %w", serveErr)
	}

	return errors.Join(serveErr, shutdownErr)
}

func (a *Container) shutdown(ctx context.Context, stopPositions func(context.Context) error) error {
	a.logger.Info("shutting down HTTP server")

	var shutdownErr error
	if err := a.server.Shutdown(ctx); err != nil {
		shutdownErr = fmt.Errorf("shutdown HTTP server: %w", err)
		a.logger.Warn("forcing HTTP connections to close")

		if err := a.server.Close(); err != nil {
			shutdownErr = errors.Join(shutdownErr, fmt.Errorf("close HTTP server: %w", err))
		} else {
			a.logger.Info("HTTP connections closed")
		}
	} else {
		a.logger.Info("HTTP server stopped")
	}

	positionsErr := stopPositions(ctx)

	// The final gauge collection still needs the PostgreSQL pool.
	telemetryErr := a.telemetry.Shutdown(ctx)
	poolErr := postgres.ClosePool(ctx, a.pool)
	if poolErr == nil {
		a.logger.Info("PostgreSQL pool closed")
	}

	return errors.Join(shutdownErr, positionsErr, telemetryErr, poolErr)
}
