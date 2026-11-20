package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/lovelykeycap/TripGo/internal/config"
	"github.com/lovelykeycap/TripGo/internal/container"
	"github.com/lovelykeycap/TripGo/internal/infra/logging"
)

func main() {
	var level slog.LevelVar

	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: &level,
	})
	logger := slog.New(logging.NewTraceHandler(jsonHandler)).With("service", "trip-service")

	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	level.Set(cfg.LogLevel)

	rootCtx := context.Background()

	app, err := container.New(rootCtx, cfg, logger)
	if err != nil {
		logger.Error("failed to initialize application", "error", err)
		os.Exit(1)
	}

	if err := app.Run(rootCtx); err != nil {
		logger.Error("application failed", "error", err)
		os.Exit(1)
	}
}
