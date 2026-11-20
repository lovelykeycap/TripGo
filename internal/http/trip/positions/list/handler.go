package list

import (
	"net/http"

	api "github.com/lovelykeycap/TripGo/internal/generated"
	"github.com/lovelykeycap/TripGo/internal/http/response"
	"github.com/lovelykeycap/TripGo/internal/http/trip/convert"
)

type Handler struct {
	positionLister positionLister
	response       *response.Writer
}

func New(positionLister positionLister, response *response.Writer) *Handler {
	return &Handler{positionLister: positionLister, response: response}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	result, err := h.positionLister.List(r.Context(), tripID)
	if err != nil {
		h.response.HandleError(w, r, err)
		return
	}

	positions := make([]api.TripPosition, 0, len(result))
	for _, position := range result {
		positions = append(positions, convert.Position(position))
	}
	h.response.WriteResponse(w, r, http.StatusOK, api.TripPositionsResponse{Positions: positions})
}
