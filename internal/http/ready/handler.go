package ready

import (
	"log/slog"
	"net/http"

	api "github.com/lovelykeycap/TripGo/internal/generated"
	"github.com/lovelykeycap/TripGo/internal/http/response"
)

type Handler struct {
	readiness readinessChecker
	response  *response.Writer
	logger    *slog.Logger
}

func New(readiness readinessChecker, response *response.Writer, logger *slog.Logger) *Handler {
	return &Handler{readiness: readiness, response: response, logger: logger}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	status := http.StatusOK
	body := api.HealthResponse{Status: api.Ok}
	if err := h.readiness.Check(r.Context()); err != nil {
		status = http.StatusServiceUnavailable
		body.Status = api.Unavailable
		h.logger.WarnContext(r.Context(), "readiness check failed", "error", err)
	}
	h.response.WriteResponse(w, r, status, body)
}
