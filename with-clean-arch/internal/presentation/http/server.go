package http

import (
	"errors"
	"golang-things/with-worker-pool/internal/presentation/http/delivery"
	"golang-things/with-worker-pool/internal/presentation/http/delivery/health"
	"net/http"
)

type ServerConfig struct {
	Port string
}

type Server struct {
	Server *http.Server
	*delivery.RouteRegistry
}

func NewServer(
	port string,
) *Server {
	healthMainHandler := health.NewMainHandler()
	healthRouter := health.NewRouter(healthMainHandler)

	routeRegistry := delivery.NewRouteRegistry(
		healthRouter,
		// add other routers here
	)

	return &Server{
		Server: &http.Server{
			Addr: ":" + port,
		},
		RouteRegistry: routeRegistry,
	}
}

func (s *Server) StartServer() error {
	mux := http.NewServeMux()
	s.RouteRegistry.RegisterAll(mux)

	s.Server.Handler = mux

	if err := s.Server.ListenAndServe(); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			// Server closed
			return nil
		}
		return err
	}

	return nil
}
