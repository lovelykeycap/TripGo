package middleware

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func Tracing(provider trace.TracerProvider, meterProvider metric.MeterProvider) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			serveWithRecovery(next, w, r)

			if r.Pattern != "" {
				trace.SpanFromContext(r.Context()).SetAttributes(
					attribute.String("http.route", r.Pattern),
				)
			}
		})

		return otelhttp.NewHandler(
			handler,
			"HTTP",
			otelhttp.WithTracerProvider(provider),
			otelhttp.WithMeterProvider(meterProvider),
			otelhttp.WithPropagators(propagation.TraceContext{}),
		)
	}
}
