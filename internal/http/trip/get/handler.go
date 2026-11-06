package get

import (
	"net/http"

	api "github.com/lovelykeycap/TripGo/internal/generated"
	"github.com/lovelykeycap/TripGo/internal/http/response"
	"github.com/lovelykeycap/TripGo/internal/http/trip/convert"
)

type Handler struct {
	tripGetter tripGetter
	response   *response.Writer
}

func New(tripGetter tripGetter, response *response.Writer) *Handler {
	return &Handler{tripGetter: tripGetter, response: response}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	result, err := h.tripGetter.Get(r.Context(), tripID)
	if err != nil {
		h.response.HandleError(w, r, err)
		return
	}
	h.response.WriteResponse(w, r, http.StatusOK, convert.Trip(result))
}
