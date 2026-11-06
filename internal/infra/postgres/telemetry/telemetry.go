package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

type Providers struct {
	TracerProvider *sdktrace.TracerProvider
	MeterProvider  *sdkmetric.MeterProvider
}

func New(ctx context.Context, shutdownTimeout time.Duration, logger *slog.Logger) (*Providers, error) {
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		logger.Warn("OpenTelemetry error", "error", err)
	}))

	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(semconv.ServiceVersion(buildVersion())),
	)
	if err != nil {
		return nil, fmt.Errorf("create telemetry resource: %w", err)
	}

	traceExporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("create trace exporter: %w", err)
	}

	metricExporter, err := otlpmetrichttp.New(ctx)
	if err != nil {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
		defer cancel()
		cleanupErr := traceExporter.Shutdown(cleanupCtx)
		if cleanupErr != nil {
			cleanupErr = fmt.Errorf("shutdown trace exporter: %w", cleanupErr)
		}
		return nil, errors.Join(fmt.Errorf("create metric exporter: %w", err), cleanupErr)
	}

	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(traceExporter),
	)
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
	)

	otel.SetTracerProvider(traceProvider)
	otel.SetMeterProvider(meterProvider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return &Providers{TracerProvider: traceProvider, MeterProvider: meterProvider}, nil
}

func (p *Providers) Shutdown(ctx context.Context) error {
	metricErr := p.MeterProvider.Shutdown(ctx)
	if metricErr != nil {
		metricErr = fmt.Errorf("shutdown metrics: %w", metricErr)
	}
	traceErr := p.TracerProvider.Shutdown(ctx)
	if traceErr != nil {
		traceErr = fmt.Errorf("shutdown tracing: %w", traceErr)
	}
	return errors.Join(traceErr, metricErr)
}
