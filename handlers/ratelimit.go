package handlers

import (
	"log"
	"net"
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

// RateLimiterConfig holds configuration for the rate limiting middleware.
// GlobalLimit is the maximum number of requests per second across all clients.
// PerClientLimit is the maximum number of requests per second per client IP.
type RateLimiterConfig struct {
	GlobalLimit    rate.Limit
	PerClientLimit rate.Limit
	Burst          int
}

// rateLimiter implements token-bucket rate limiting with a global limiter
// and per-client limiters tracked by IP address.
type rateLimiter struct {
	config    RateLimiterConfig
	global    *rate.Limiter
	clients   map[string]*rate.Limiter
	mu        sync.Mutex
}

// NewRateLimiter creates a new rate limiter with the given configuration.
func NewRateLimiter(config RateLimiterConfig) *rateLimiter {
	return &rateLimiter{
		config:  config,
		global:  rate.NewLimiter(config.GlobalLimit, config.Burst),
		clients: make(map[string]*rate.Limiter),
	}
}

// Allow checks whether a request from the given client IP should be allowed.
// Returns true if the request is within rate limits.
func (rl *rateLimiter) Allow(clientIP string) bool {
	if !rl.global.Allow() {
		return false
	}

	rl.mu.Lock()
	limiter, exists := rl.clients[clientIP]
	if !exists {
		limiter = rate.NewLimiter(rl.config.PerClientLimit, rl.config.Burst)
		rl.clients[clientIP] = limiter
	}
	rl.mu.Unlock()

	return limiter.Allow()
}

// RateLimitMiddleware returns an HTTP middleware that rate-limits requests
// using a token-bucket algorithm with the given configuration.
// Returns 429 Too Many Requests when the limit is exceeded.
func RateLimitMiddleware(cfg RateLimiterConfig) func(http.Handler) http.Handler {
	rl := NewRateLimiter(cfg)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP := extractClientIP(r)
			if !rl.Allow(clientIP) {
				log.Printf("WARN rate_limit exceeded client_ip=%s method=%s", clientIP, r.Method)
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// extractClientIP extracts the client IP address from the request, preferring
// the X-Forwarded-For header when behind a reverse proxy.
func extractClientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		host, _, err := net.SplitHostPort(fwd)
		if err == nil {
			return host
		}
		return fwd
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
