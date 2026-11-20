package http

import (
	"net/http"

	api "github.com/lovelykeycap/TripGo/internal/generated"
	"github.com/lovelykeycap/TripGo/internal/http/health"
	"github.com/lovelykeycap/TripGo/internal/http/ready"
	"github.com/lovelykeycap/TripGo/internal/http/trip/create"
	"github.com/lovelykeycap/TripGo/internal/http/trip/finish"
	"github.com/lovelykeycap/TripGo/internal/http/trip/get"
	positioncreate "github.com/lovelykeycap/TripGo/internal/http/trip/positions/create"
	positionlist "github.com/lovelykeycap/TripGo/internal/http/trip/positions/list"
)

type Handlers struct {
	Create         *create.Handler
	Get            *get.Handler
	Finish         *finish.Handler
	Health         *health.Handler
	Ready          *ready.Handler
	CreatePosition *positioncreate.Handler
	ListPositions  *positionlist.Handler
}

type server struct {
	handlers Handlers
}

func NewServer(handlers Handlers) api.ServerInterface {
	return &server{handlers: handlers}
}

func (s *server) CreateTrip(w http.ResponseWriter, r *http.Request, params api.CreateTripParams) {
	s.handlers.Create.Handle(w, r, params)
}

func (s *server) GetTrip(w http.ResponseWriter, r *http.Request, id api.TripId) {
	s.handlers.Get.Handle(w, r, id)
}

func (s *server) FinishTrip(w http.ResponseWriter, r *http.Request, id api.TripId) {
	s.handlers.Finish.Handle(w, r, id)
}

func (s *server) CreateTripPosition(w http.ResponseWriter, r *http.Request, id api.TripId) {
	s.handlers.CreatePosition.Handle(w, r, id)
}

func (s *server) ListTripPositions(w http.ResponseWriter, r *http.Request, id api.TripId) {
	s.handlers.ListPositions.Handle(w, r, id)
}

func (s *server) Health(w http.ResponseWriter, r *http.Request) {
	s.handlers.Health.Handle(w, r)
}

func (s *server) Ready(w http.ResponseWriter, r *http.Request) {
	s.handlers.Ready.Handle(w, r)
}
