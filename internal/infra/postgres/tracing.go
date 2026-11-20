package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
)

// queryTracer adds the query attributes that otelpgx does not provide itself.
type queryTracer struct {
	*otelpgx.Tracer
	operationDuration metric.Float64Histogram
}

type queryMetricKey struct{}

type queryMetric struct {
	startedAt  time.Time
	attributes []attribute.KeyValue
}

func newQueryTracer(provider trace.TracerProvider, meterProvider metric.MeterProvider) (*queryTracer, error) {
	meter := meterProvider.Meter("github.com/lovelykeycap/TripGo/internal/infra/postgres")
	duration, err := meter.Float64Histogram("db.client.operation.duration",
		metric.WithDescription("Duration of database client operations"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5, 10),
	)
	if err != nil {
		return nil, fmt.Errorf("create database operation duration histogram: %w", err)
	}

	return &queryTracer{
		Tracer: otelpgx.NewTracer(
			otelpgx.WithTracerProvider(provider),
			// The adapter records query duration with per-query attributes once.
			otelpgx.WithMeterProvider(noop.NewMeterProvider()),
		),
		operationDuration: duration,
	}, nil
}

func (t *queryTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	startedAt := time.Now()
	attrs := []attribute.KeyValue{
		attribute.String("db.system.name", "postgresql"),
		attribute.String("db.operation.name", otelpgx.SQLOperationName(ctx, data.SQL)),
	}
	if collection := queryCollection(data.SQL); collection != "" {
		attrs = append(attrs, attribute.String("db.collection.name", collection))
	}

	ctx = t.Tracer.TraceQueryStart(ctx, conn, data)
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetAttributes(attrs...)
	}
	return context.WithValue(ctx, queryMetricKey{}, queryMetric{
		startedAt:  startedAt,
		attributes: attrs,
	})
}

func (t *queryTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	var errorAttr attribute.KeyValue
	if data.Err != nil && !errors.Is(data.Err, pgx.ErrNoRows) {
		errorAttr = attribute.String("error.type", queryErrorType(data.Err))
		trace.SpanFromContext(ctx).SetAttributes(errorAttr)
	}
	if query, ok := ctx.Value(queryMetricKey{}).(queryMetric); ok {
		attrs := query.attributes
		if errorAttr.Key != "" {
			attrs = append(attrs, errorAttr)
		}
		t.operationDuration.Record(ctx, time.Since(query.startedAt).Seconds(), metric.WithAttributes(attrs...))
	}
	t.Tracer.TraceQueryEnd(ctx, conn, data)
}

func queryCollection(sql string) string {
	fields := strings.Fields(sql)
	if len(fields) < 2 {
		return ""
	}

	switch strings.ToUpper(fields[0]) {
	case "UPDATE":
		return strings.Trim(fields[1], `"`)
	case "INSERT":
		if len(fields) > 2 && strings.EqualFold(fields[1], "INTO") {
			return strings.Trim(fields[2], `"`)
		}
	case "SELECT", "DELETE":
		for i := 1; i < len(fields)-1; i++ {
			if strings.EqualFold(fields[i], "FROM") {
				return strings.Trim(fields[i+1], `"`)
			}
		}
	}
	return ""
}

func queryErrorType(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "context.deadline_exceeded"
	}
	if errors.Is(err, context.Canceled) {
		return "context.canceled"
	}
	return fmt.Sprintf("%T", err)
}
