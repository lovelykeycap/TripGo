package health

import (
	"net/http"

	api "github.com/lovelykeycap/TripGo/internal/generated"
	"github.com/lovelykeycap/TripGo/internal/http/response"
)

type Handler struct {
	response *response.Writer
}

func New(response *response.Writer) *Handler {
	return &Handler{response: response}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	h.response.WriteResponse(w, r, http.StatusOK, api.HealthResponse{Status: api.Ok})
}
