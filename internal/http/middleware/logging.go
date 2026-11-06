package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/felixge/httpsnoop"
)

func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			metrics := httpsnoop.CaptureMetricsFn(w, func(writer http.ResponseWriter) {
				serveWithRecovery(next, writer, r)
			})
			level := slog.LevelInfo
			if metrics.Code >= http.StatusInternalServerError {
				level = slog.LevelError
			}
			logger.Log(r.Context(), level, "request handled",
				"method", r.Method,
				"route", r.Pattern,
				"status", metrics.Code,
				"duration_ms", float64(metrics.Duration)/float64(time.Millisecond),
			)
		})
	}
}
