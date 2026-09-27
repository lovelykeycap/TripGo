package main

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
	"github.com/joho/godotenv"
	"github.com/lovelykeycap/TripGo/internal/config"
	"github.com/lovelykeycap/TripGo/internal/handlers/httpapi"
	"github.com/lovelykeycap/TripGo/internal/infra/postgres"
	triprepository "github.com/lovelykeycap/TripGo/internal/repository/trip"
	"github.com/lovelykeycap/TripGo/internal/repository/triphistory"
	tripservice "github.com/lovelykeycap/TripGo/internal/service/trip"
)

type App struct {
	server          *http.Server
	pool            *pgxpool.Pool
	logger          *slog.Logger
	shutdownTimeout time.Duration
}

func main() {
	if err := godotenv.Load(".env"); err != nil &&
		!errors.Is(err, os.ErrNotExist) {
		slog.Error("failed to load .env file", "error", err)
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	})).With("service", "trip-service")

	slog.SetDefault(logger)

	rootCtx := context.Background()
	pool, err := postgres.NewPool(rootCtx, cfg)
	if err != nil {
		logger.Error("failed to initialize PostgreSQL", "error", err)
		os.Exit(1)
	}
	logger.Info("PostgreSQL connection established")

	readiness := postgres.NewReadinessChecker(pool, cfg.DatabaseQueryTimeout)
	tripRepo := triprepository.New(pool, cfg.DatabaseQueryTimeout)
	historyRepo := triphistory.New(pool, cfg.DatabaseQueryTimeout)
	txManager := postgres.NewTxManager(pool, cfg.DatabaseQueryTimeout)
	tripService := tripservice.New(tripRepo, historyRepo, txManager)
	handlers := httpapi.NewHandler(logger, readiness, tripService)

	router := httpapi.NewRouter(handlers)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadTimeout:       cfg.HTTPReadTimeout,
		ReadHeaderTimeout: cfg.HTTPReadHeaderTimeout,
		WriteTimeout:      cfg.HTTPWriteTimeout,
		IdleTimeout:       cfg.HTTPIdleTimeout,
	}

	app := &App{
		server:          server,
		pool:            pool,
		logger:          logger,
		shutdownTimeout: cfg.ShutdownTimeout,
	}

	if err := app.Run(rootCtx); err != nil {
		logger.Error("application failed", "error", err)
		os.Exit(1)
	}
}

func (a *App) Run(rootCtx context.Context) error {
	signalCtx, stop := signal.NotifyContext(rootCtx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)

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

	shutdownErr := a.shutdown(shutdownCtx)
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

func (a *App) shutdown(ctx context.Context) error {
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

	poolErr := postgres.ClosePool(ctx, a.pool)
	if poolErr == nil {
		a.logger.Info("PostgreSQL pool closed")
	}
	return errors.Join(shutdownErr, poolErr)
}
