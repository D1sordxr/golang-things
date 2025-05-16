package main

import (
	"context"
	"golang.org/x/time/rate"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	limiter := NewRateLimiterMiddleware(10, time.Minute, 5*time.Minute)
	go limiter.RunJanitor(ctx)

	mux := http.NewServeMux()
	mux.Handle("/ping", limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("pong"))
	})))

	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type RateLimiterMiddleware struct {
	mu      sync.Mutex
	clients map[string]*ipLimiter
	rate    rate.Limit
	burst   int
	ttl     time.Duration
}

func NewRateLimiterMiddleware(reqPerMinute int, per time.Duration, ttl time.Duration) *RateLimiterMiddleware {
	limit := rate.Every(per / time.Duration(reqPerMinute))
	return &RateLimiterMiddleware{
		clients: make(map[string]*ipLimiter),
		rate:    limit,
		burst:   reqPerMinute,
		ttl:     ttl,
	}
}

func (m *RateLimiterMiddleware) getLimiter(ip string) *rate.Limiter {
	m.mu.Lock()
	defer m.mu.Unlock()

	lim, exists := m.clients[ip]
	if !exists {
		lim = &ipLimiter{
			limiter:  rate.NewLimiter(m.rate, m.burst),
			lastSeen: time.Now(),
		}
		m.clients[ip] = lim
	}

	lim.lastSeen = time.Now()
	return lim.limiter
}

func (m *RateLimiterMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := realIP(r)
		limiter := m.getLimiter(ip)

		if !limiter.Allow() {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (m *RateLimiterMiddleware) RunJanitor(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			m.mu.Lock()
			for ip, lim := range m.clients {
				if now.Sub(lim.lastSeen) > m.ttl {
					delete(m.clients, ip)
				}
			}
			m.mu.Unlock()
		}
	}
}

func realIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := net.ParseIP(xff)
		if ips != nil {
			return ips.String()
		}
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
