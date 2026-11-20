package response

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	api "github.com/lovelykeycap/TripGo/internal/generated"
	"github.com/lovelykeycap/TripGo/internal/http/domainerror"
)

type Details struct {
	Status int32
	Code   string
	Title  string
	Detail string
}

type Writer struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *Writer {
	return &Writer{logger: logger}
}

func (h *Writer) HandleRequestError(w http.ResponseWriter, r *http.Request, err error) {
	h.WriteProblem(w, r, Details{
		Status: http.StatusBadRequest,
		Code:   "invalid_request",
		Title:  "Invalid request",
		Detail: err.Error(),
	})
}

func (h *Writer) HandleError(w http.ResponseWriter, r *http.Request, err error) {
	status, problem := domainerror.Problem(err)
	if status >= http.StatusInternalServerError {
		h.logger.ErrorContext(r.Context(), "request failed", "error", err)
	}
	h.writeProblemResponse(w, r, status, problem)
}

func (h *Writer) WriteProblem(w http.ResponseWriter, r *http.Request, details Details) {
	problem := api.Problem{
		Type:   "https://tripgo.example/problems/" + strings.ReplaceAll(details.Code, "_", "-"),
		Title:  details.Title,
		Status: details.Status,
		Code:   details.Code,
		Detail: &details.Detail,
	}
	h.writeProblemResponse(w, r, int(details.Status), problem)
}

func (h *Writer) writeProblemResponse(w http.ResponseWriter, r *http.Request, status int, problem api.Problem) {
	instance := r.URL.Path
	problem.Instance = &instance
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(problem); err != nil {
		h.logger.ErrorContext(r.Context(), "failed to write problem response", "error", err)
	}
}

func (h *Writer) WriteResponse(w http.ResponseWriter, r *http.Request, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		h.logger.ErrorContext(r.Context(), "failed to write JSON response", "error", err)
	}
}
