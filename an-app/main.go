package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"
)

func main() {
	app := NewApp()
	app.Run()
}

type Log interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

// Presentation layer

type useCase interface {
	Process(ctx context.Context, data []byte) ([]byte, error)
}

type Handler struct {
	uc useCase
}

func NewHandler(
	log Log,
	uc useCase,
) *Handler {
	return &Handler{
		uc: uc,
	}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	data, err := io.ReadAll(r.Body)
	if err != nil {
		w.Write([]byte("Status 404"))
		return
	}

	resp, err := h.uc.Process(ctx, data)
	if err != nil {
		w.Write([]byte("Status 500"))
		return
	}

	w.Write(resp)
}

type RetryMiddleware struct {
	// retries int
}

func (RetryMiddleware) RetryWithBackoff(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			// var lastErr error
			backoff := time.Second * 1

			for i := 0; i < 5; i++ {
				rr := httptest.NewRecorder()
				next.ServeHTTP(rr, r)

				if rr.Code < 500 {
					for k, v := range rr.Header() {
						w.Header()[k] = v
					}
					w.WriteHeader(rr.Code)
					rr.Body.WriteTo(w)
					return
				}

				// lastErr = fmt.Errorf("Attempt %d failed with status code %d", i+1, rr.Code)
				time.Sleep(backoff)
				backoff *= 2
			}

			http.Error(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
		},
	)
}

const (
	tokens = 3
)

type LimiterMiddleware struct {
	tokens chan struct{}
}

func NewLimiterMiddleware() *LimiterMiddleware {
	return &LimiterMiddleware{
		tokens: make(chan struct{}, tokens),
	}
}

func (m *LimiterMiddleware) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			m.tokens <- struct{}{} // could be an uuid and logged for tracing effect
			defer func() { <-m.tokens }()

			next.ServeHTTP(w, r)
		},
	)
}

type LoggingMiddleware struct {
	log Log
}

func NewLoggingMiddleware(log Log) *LoggingMiddleware {
	return &LoggingMiddleware{
		log: log,
	}
}

func (m *LoggingMiddleware) Log(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			now := time.Now()
			m.log.Info("Starting request...", "time", now)
			next.ServeHTTP(w, r)
			m.log.Info("Request finished.", "time-since-start", time.Since(now))
		},
	)
}

type Router struct {
	logMid     *LoggingMiddleware
	limiterMid *LimiterMiddleware
	retryMid   *RetryMiddleware
	hand       *Handler
	// or interfaces if needed
}

func NewRouter(
	logMid *LoggingMiddleware,
	limiterMid *LimiterMiddleware,
	retryMid *RetryMiddleware,
	hand *Handler,
) *Router {
	return &Router{
		logMid:     logMid,
		limiterMid: limiterMid,
		retryMid:   retryMid,
		hand:       hand,
	}
}

func (r *Router) SetupRoutes(mux *http.ServeMux) {
	mux.Handle(
		"/api/process",
		r.logMid.Log(
			r.limiterMid.Limit(
				r.retryMid.RetryWithBackoff(
					http.HandlerFunc(r.hand.Handle),
				),
			),
		),
	)
}

type Server struct {
	Mux    *http.ServeMux
	Router *Router // can be replaced with interface
}

func NewServer(
	r *Router,
) *Server {
	return &Server{
		Mux:    http.NewServeMux(),
		Router: r,
	}
}

func (s *Server) StartServer() {
	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()

		s.Router.SetupRoutes(s.Mux)
		if err := http.ListenAndServe(":9090", s.Mux); err != nil {
			// log err
		}
	}()
	wg.Wait()
}

type mockUC struct{}

func (mockUC) Process(ctx context.Context, data []byte) ([]byte, error) {
	time.Sleep(time.Second * 5)

	return nil, nil
}

type App struct {
	*Server
}

func NewApp() *App {
	log := slog.Default()

	logMid := NewLoggingMiddleware(log)
	limitMid := NewLimiterMiddleware()
	retryMid := new(RetryMiddleware)

	useCase := new(mockUC)

	handler := NewHandler(log, useCase)

	router := NewRouter(
		logMid,
		limitMid,
		retryMid,
		handler,
	)

	server := NewServer(router)

	return &App{
		Server: server,
	}
}

func (a *App) Run() {
	a.Server.StartServer()
}
