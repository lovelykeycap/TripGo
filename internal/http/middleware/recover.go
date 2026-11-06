package middleware

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/felixge/httpsnoop"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/lovelykeycap/TripGo/internal/http/response"
)

type recoveryKey struct{}

type panicRecovery struct {
	logger          *slog.Logger
	responses       *response.Writer
	responseStarted bool
}

func Recover(logger *slog.Logger, responses *response.Writer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			recovery := &panicRecovery{logger: logger, responses: responses}
			r = r.WithContext(context.WithValue(r.Context(), recoveryKey{}, recovery))
			serveWithRecovery(next, recovery.wrap(w), r)
		})
	}
}

// Inner boundaries keep recovery inside the logging and tracing lifetimes.
// The outer Recover middleware supplies the shared policy and is a final fallback.
func serveWithRecovery(next http.Handler, w http.ResponseWriter, r *http.Request) {
	defer recoverRequest(w, r)
	next.ServeHTTP(w, r)
}

func recoverRequest(w http.ResponseWriter, r *http.Request) {
	recovered := recover()
	if recovered == nil {
		return
	}

	recovery, ok := r.Context().Value(recoveryKey{}).(*panicRecovery)
	if !ok || recovered == http.ErrAbortHandler {
		panic(recovered)
	}
	recovery.handle(w, r, recovered)
}

func (p *panicRecovery) handle(w http.ResponseWriter, r *http.Request, recovered any) {
	err := fmt.Errorf("panic: %v", recovered)
	span := trace.SpanFromContext(r.Context())
	span.RecordError(err)
	span.SetStatus(codes.Error, "request panicked")
	span.SetAttributes(attribute.String("error.type", "panic"))
	p.logger.ErrorContext(r.Context(), "request panicked",
		"error", err,
		"stack", string(debug.Stack()),
	)

	if p.responseStarted {
		return
	}
	w.Header().Del("Content-Length")
	p.responses.WriteProblem(w, r, response.Details{
		Status: http.StatusInternalServerError,
		Code:   "internal_error",
		Title:  "Internal error",
		Detail: "Unexpected server error",
	})
}

func (p *panicRecovery) wrap(w http.ResponseWriter) http.ResponseWriter {
	return httpsnoop.Wrap(w, httpsnoop.Hooks{
		WriteHeader: func(next httpsnoop.WriteHeaderFunc) httpsnoop.WriteHeaderFunc {
			return func(status int) {
				next(status)
				if status >= http.StatusOK || status == http.StatusSwitchingProtocols {
					p.responseStarted = true
				}
			}
		},
		Write: func(next httpsnoop.WriteFunc) httpsnoop.WriteFunc {
			return func(body []byte) (int, error) {
				p.responseStarted = true
				return next(body)
			}
		},
		Flush: func(next httpsnoop.FlushFunc) httpsnoop.FlushFunc {
			return func() {
				p.responseStarted = true
				next()
			}
		},
		ReadFrom: func(next httpsnoop.ReadFromFunc) httpsnoop.ReadFromFunc {
			return func(reader io.Reader) (int64, error) {
				p.responseStarted = true
				return next(reader)
			}
		},
	})
}
