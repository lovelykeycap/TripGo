package ready_test

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/mock/gomock"

	api "github.com/lovelykeycap/TripGo/internal/generated"
	"github.com/lovelykeycap/TripGo/internal/http/ready"
	"github.com/lovelykeycap/TripGo/internal/http/response"
)

func TestHandle(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		status     int
		bodyStatus api.HealthResponseStatus
	}{
		{name: "ready", status: http.StatusOK, bodyStatus: api.Ok},
		{name: "database unavailable", err: errors.New("database unavailable"), status: http.StatusServiceUnavailable, bodyStatus: api.Unavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			readiness := NewMockreadinessChecker(gomock.NewController(t))
			readiness.EXPECT().Check(gomock.Any()).Return(tt.err)
			logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
			handler := ready.New(readiness, response.New(logger), logger)
			request := httptest.NewRequest(http.MethodGet, "/ready", nil)
			recorder := httptest.NewRecorder()

			handler.Handle(recorder, request)

			if recorder.Code != tt.status {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.status)
			}
			if recorder.Header().Get("Content-Type") != "application/json" {
				t.Errorf("unexpected Content-Type: %q", recorder.Header().Get("Content-Type"))
			}
			var body api.HealthResponse
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Status != tt.bodyStatus {
				t.Errorf("body status = %q, want %q", body.Status, tt.bodyStatus)
			}
		})
	}
}
