package delivery

import "net/http"

type RouteConfigurator interface {
	RegisterRoutes(mux *http.ServeMux)
}

type RouteRegistry struct {
	Configurators []RouteConfigurator
}

func NewRouteRegistry(configurators ...RouteConfigurator) *RouteRegistry {
	return &RouteRegistry{
		Configurators: configurators,
	}
}

func (r *RouteRegistry) RegisterAll(mux *http.ServeMux) {
	for _, c := range r.Configurators {
		c.RegisterRoutes(mux)
	}
}
