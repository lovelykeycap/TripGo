package create

import (
	"net/http"

	api "github.com/lovelykeycap/TripGo/internal/generated"
	"github.com/lovelykeycap/TripGo/internal/http/response"
	"github.com/lovelykeycap/TripGo/internal/http/trip/convert"
)

type Handler struct {
	tripCreator tripCreator
	response    *response.Writer
}

func New(tripCreator tripCreator, response *response.Writer) *Handler {
	return &Handler{tripCreator: tripCreator, response: response}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request, _ api.CreateTripParams) {
	request, err := decodeCreateTripRequest(r)
	if err != nil {
		h.response.HandleRequestError(w, r, err)
		return
	}

	result, err := h.tripCreator.Create(r.Context(), toInput(request))
	if err != nil {
		h.response.HandleError(w, r, err)
		return
	}

	w.Header().Set("Location", "/api/v1/trips/"+result.ID.String())
	h.response.WriteResponse(w, r, http.StatusCreated, convert.Trip(result))
}
