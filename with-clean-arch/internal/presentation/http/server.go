package http

import (
	"golang-things/with-worker-pool/internal/presentation/http/delivery"
	"golang-things/with-worker-pool/internal/presentation/http/delivery/health"
	"net/http"
)

type ServerConfig struct {
	Port string
}

type Server struct {
	Port string
	Mux  *http.ServeMux
	*delivery.RouteRegistry
}

func NewServer() *Server {
	healthMainHandler := health.NewMainHandler()
	healthRouter := health.NewRouter(healthMainHandler)

	routeRegistry := delivery.NewRouteRegistry(
		healthRouter,
		// add other routers here
	)

	mux := http.NewServeMux()
	return &Server{
		Mux:           mux,
		RouteRegistry: routeRegistry,
	}
}

func (s *Server) StartServer() error {
	s.RouteRegistry.RegisterAll(s.Mux)

	if err := http.ListenAndServe(":"+s.Port, s.Mux); err != nil {
		return err
	}

	return nil
}
