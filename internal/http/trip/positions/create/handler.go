package create

import (
	"net/http"

	api "github.com/lovelykeycap/TripGo/internal/generated"
	"github.com/lovelykeycap/TripGo/internal/http/response"
	"github.com/lovelykeycap/TripGo/internal/http/trip/convert"
	positioncreate "github.com/lovelykeycap/TripGo/internal/usecase/trip/positions/create"
)

type Handler struct {
	positionCreator positionCreator
	response        *response.Writer
}

func New(positionCreator positionCreator, response *response.Writer) *Handler {
	return &Handler{positionCreator: positionCreator, response: response}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	request, err := decodePositionRequest(r)
	if err != nil {
		h.response.HandleRequestError(w, r, err)
		return
	}
	result, err := h.positionCreator.Create(r.Context(), positioncreate.Input{
		TripID:      tripID,
		Coordinates: toCoordinates(request.Coordinates),
		RecordedAt:  request.RecordedAt,
	})
	if err != nil {
		h.response.HandleError(w, r, err)
		return
	}
	h.response.WriteResponse(w, r, http.StatusCreated, convert.Position(result))
}
