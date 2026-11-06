package middleware

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func Metrics(meter metric.Meter) (func(http.Handler) http.Handler, error) {
	activeRequests, err := meter.Int64UpDownCounter("http.server.active_requests",
		metric.WithDescription("Number of active HTTP server requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, fmt.Errorf("create active HTTP requests counter: %w", err)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attrs := []attribute.KeyValue{
				attribute.String("http.request.method", metricMethod(r.Method)),
			}
			if route := requestRoute(r); route != "" {
				attrs = append(attrs, attribute.String("http.route", route))
			}
			options := metric.WithAttributes(attrs...)
			activeRequests.Add(r.Context(), 1, options)
			defer activeRequests.Add(r.Context(), -1, options)

			serveWithRecovery(next, w, r)
		})
	}, nil
}

func requestRoute(r *http.Request) string {
	if r.Pattern != "" {
		return r.Pattern
	}
	routeCtx := chi.RouteContext(r.Context())
	if routeCtx == nil || routeCtx.Routes == nil {
		return ""
	}

	path := routeCtx.RoutePath
	if path == "" {
		path = r.URL.RawPath
	}
	if path == "" {
		path = r.URL.Path
	}
	if path == "" {
		path = "/"
	}
	method := routeCtx.RouteMethod
	if method == "" {
		method = r.Method
	}

	// Find mutates its context, so keep the request's routing context intact.
	return routeCtx.Routes.Find(chi.NewRouteContext(), method, path)
}

func metricMethod(method string) string {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
		http.MethodDelete, http.MethodConnect, http.MethodOptions, http.MethodTrace,
		http.MethodPatch, "QUERY":
		return method
	default:
		return "_OTHER"
	}
}
