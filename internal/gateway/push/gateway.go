package push

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"
)

type Gateway struct {
	endpoint string
	client   *http.Client
	tracer   trace.Tracer
}

type StatusError struct {
	StatusCode int
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("push service returned HTTP %d", e.StatusCode)
}

type positionRequest struct {
	RequestID   string            `json:"request_id"`
	RecipientID uuid.UUID         `json:"recipient_id"`
	Kind        string            `json:"kind"`
	Data        map[string]string `json:"data"`
}

func New(baseURL string, timeout time.Duration, tracer trace.Tracer) (*Gateway, error) {
	endpoint, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse push service URL: %w", err)
	}
	if (endpoint.Scheme != "http" && endpoint.Scheme != "https") || endpoint.Hostname() == "" {
		return nil, errors.New("push service URL must be an absolute HTTP or HTTPS URL")
	}

	return &Gateway{
		endpoint: endpoint.JoinPath("api", "v1", "push").String(),
		tracer:   tracer,
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				Proxy:             http.ProxyFromEnvironment,
				ForceAttemptHTTP2: true,
				IdleConnTimeout:   90 * time.Second,
			},
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}, nil
}

func (g *Gateway) Close() {
	g.client.CloseIdleConnections()
}

func (g *Gateway) RequestPosition(ctx context.Context, driverID, tripID uuid.UUID) error {
	body, err := json.Marshal(positionRequest{
		RequestID:   uuid.NewString(),
		RecipientID: driverID,
		Kind:        "REQUEST_POSITION",
		Data:        map[string]string{"trip_id": tripID.String()},
	})
	if err != nil {
		return fmt.Errorf("encode push request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, g.endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build push request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	ctx, span := g.tracer.Start(ctx, "POST /api/v1/push",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(semconv.HTTPRequestMethodPost, semconv.ServerAddress(request.URL.Hostname())),
	)
	defer span.End()
	propagation.TraceContext{}.Inject(ctx, propagation.HeaderCarrier(request.Header))

	response, err := g.client.Do(request.WithContext(ctx))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("send push request: %w", err)
	}
	defer response.Body.Close()
	span.SetAttributes(semconv.HTTPResponseStatusCode(response.StatusCode))

	if response.StatusCode != http.StatusAccepted {
		err := &StatusError{StatusCode: response.StatusCode}
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	return nil
}
