package health_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	api "github.com/lovelykeycap/TripGo/internal/generated"
	"github.com/lovelykeycap/TripGo/internal/http/health"
	"github.com/lovelykeycap/TripGo/internal/http/response"
)

func TestHandle(t *testing.T) {
	writer := response.New(slog.New(slog.NewJSONHandler(io.Discard, nil)))
	handler := health.New(writer)
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	recorder := httptest.NewRecorder()

	handler.Handle(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if recorder.Header().Get("Content-Type") != "application/json" {
		t.Errorf("unexpected Content-Type: %q", recorder.Header().Get("Content-Type"))
	}
	var body api.HealthResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Status != api.Ok {
		t.Errorf("body status = %q, want %q", body.Status, api.Ok)
	}
}
