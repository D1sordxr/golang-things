package middlewares

import (
	"golang-things/with-worker-pool/pkg"
	"net/http"
	"net/http/httptest"
	"time"
)

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
	log pkg.Log
}

func NewLoggingMiddleware(log pkg.Log) *LoggingMiddleware {
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
