package httpapi

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	api "github.com/lovelykeycap/TripGo/internal/generated"
)

func NewRouter(h *Handler) http.Handler {
	router := chi.NewRouter()
	router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		h.writeProblem(w, r, problemDetails{
			Status: http.StatusNotFound,
			Code:   "route_not_found",
			Title:  "Route not found",
			Detail: "Requested route does not exist",
		})
	})
	router.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if r.URL.RawPath != "" {
			path = r.URL.RawPath
		}
		var allowed []string
		for _, method := range []string{
			http.MethodGet, http.MethodHead, http.MethodPost,
			http.MethodPut, http.MethodPatch, http.MethodDelete,
			http.MethodConnect, http.MethodOptions, http.MethodTrace,
		} {
			if router.Match(chi.NewRouteContext(), method, path) {
				allowed = append(allowed, method)
			}
		}
		w.Header().Set("Allow", strings.Join(allowed, ", "))
		h.writeProblem(w, r, problemDetails{
			Status: http.StatusMethodNotAllowed,
			Code:   "method_not_allowed",
			Title:  "Method not allowed",
			Detail: "Requested method is not allowed for this route",
		})
	})

	return api.HandlerWithOptions(h, api.ChiServerOptions{
		BaseRouter:       router,
		ErrorHandlerFunc: h.HandleRequestError,
	})
}
