package health

import "net/http"

type handler interface {
	Handle(w http.ResponseWriter, r *http.Request)
}

type Router struct {
	check handler
}

func NewRouter(h handler) *Router {
	return &Router{
		check: h,
	}
}

func (r *Router) RegisterRoutes(mux *http.ServeMux) {
	mux.Handle("/health", http.HandlerFunc(r.check.Handle))
}
