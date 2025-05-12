package health

import "net/http"

type MainHandler struct{}

// NewMainHandler could be replaced with new(MainHandler) if you want to use a constructor
func NewMainHandler() *MainHandler {
	return &MainHandler{}
}

func (h *MainHandler) Handle(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
