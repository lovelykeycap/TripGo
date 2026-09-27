package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr              string
	LogLevel              slog.Level
	ShutdownTimeout       time.Duration
	HTTPReadTimeout       time.Duration
	HTTPReadHeaderTimeout time.Duration
	HTTPWriteTimeout      time.Duration
	HTTPIdleTimeout       time.Duration

	DatabaseURL             string
	DatabaseMaxConns        int32
	DatabaseMinConns        int32
	DatabaseMaxConnLifetime time.Duration
	DatabaseConnectTimeout  time.Duration
	DatabaseQueryTimeout    time.Duration
}

func Load() (Config, error) {
	var cfg Config
	var err error

	cfg.HTTPAddr, err = requiredEnv("HTTP_ADDR")
	if err != nil {
		return Config{}, err
	}

	logLevel, err := requiredEnv("LOG_LEVEL")
	if err != nil {
		return Config{}, err
	}
	if err := cfg.LogLevel.UnmarshalText([]byte(logLevel)); err != nil {
		return Config{}, fmt.Errorf("parse LOG_LEVEL: %w", err)
	}

	cfg.ShutdownTimeout, err = durationEnv("SHUTDOWN_TIMEOUT")
	if err != nil {
		return Config{}, err
	}
	cfg.HTTPReadTimeout, err = durationEnv("HTTP_READ_TIMEOUT")
	if err != nil {
		return Config{}, err
	}
	cfg.HTTPReadHeaderTimeout, err = durationEnv("HTTP_READ_HEADER_TIMEOUT")
	if err != nil {
		return Config{}, err
	}
	cfg.HTTPWriteTimeout, err = durationEnv("HTTP_WRITE_TIMEOUT")
	if err != nil {
		return Config{}, err
	}
	cfg.HTTPIdleTimeout, err = durationEnv("HTTP_IDLE_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	cfg.DatabaseURL, err = requiredEnv("DATABASE_URL")
	if err != nil {
		return Config{}, err
	}
	cfg.DatabaseMaxConns, err = int32Env("DATABASE_MAX_CONNS")
	if err != nil {
		return Config{}, err
	}
	cfg.DatabaseMinConns, err = int32Env("DATABASE_MIN_CONNS")
	if err != nil {
		return Config{}, err
	}
	if cfg.DatabaseMaxConns <= 0 {
		return Config{}, fmt.Errorf("DATABASE_MAX_CONNS must be greater than zero")
	}
	if cfg.DatabaseMinConns < 0 {
		return Config{}, fmt.Errorf("DATABASE_MIN_CONNS must not be negative")
	}
	if cfg.DatabaseMinConns > cfg.DatabaseMaxConns {
		return Config{}, fmt.Errorf("DATABASE_MIN_CONNS must not exceed DATABASE_MAX_CONNS")
	}

	cfg.DatabaseMaxConnLifetime, err = durationEnv("DATABASE_MAX_CONN_LIFETIME")
	if err != nil {
		return Config{}, err
	}
	cfg.DatabaseConnectTimeout, err = durationEnv("DATABASE_CONNECT_TIMEOUT")
	if err != nil {
		return Config{}, err
	}
	cfg.DatabaseQueryTimeout, err = durationEnv("DATABASE_QUERY_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func requiredEnv(name string) (string, error) {
	value := os.Getenv(name)
	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%s is required", name)
	}
	return value, nil
}

func durationEnv(name string) (time.Duration, error) {
	value, err := requiredEnv(name)
	if err != nil {
		return 0, err
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	if duration <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", name)
	}
	return duration, nil
}

func int32Env(name string) (int32, error) {
	value, err := requiredEnv(name)
	if err != nil {
		return 0, err
	}
	number, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	return int32(number), nil
}
