package handlers

import (
	"log"
	"net"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/time/rate"
)

// RateLimiterConfig holds configuration for the rate limiting middleware.
// GlobalLimit is the maximum number of requests per second across all clients.
// GlobalBurst is the initial burst capacity for the global limiter.
// PerClientLimit is the maximum number of requests per second per client IP.
// PerClientBurst is the initial burst capacity for each per-client limiter.
type RateLimiterConfig struct {
	GlobalLimit    rate.Limit
	GlobalBurst    int
	PerClientLimit rate.Limit
	PerClientBurst int
}

// rateLimiter implements token-bucket rate limiting with a global limiter
// and per-client limiters tracked by IP address.
type rateLimiter struct {
	config  RateLimiterConfig
	global  *rate.Limiter
	clients map[string]*rate.Limiter
	mu      sync.Mutex
}

// NewRateLimiter creates a new rate limiter with the given configuration.
func NewRateLimiter(config RateLimiterConfig) *rateLimiter {
	burst := config.GlobalBurst
	if burst <= 0 {
		burst = int(config.GlobalLimit) * 2 // default: 2x the rate
	}
	return &rateLimiter{
		config:  config,
		global:  rate.NewLimiter(config.GlobalLimit, burst),
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
		perBurst := rl.config.PerClientBurst
		if perBurst <= 0 {
			perBurst = int(rl.config.PerClientLimit) * 2 // default: 2x the rate
		}
		limiter = rate.NewLimiter(rl.config.PerClientLimit, perBurst)
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
				log.Printf("WARN rate_limit exceeded client_ip=%s method=%s", clientIP, r.Method) //nolint:gosec // clientIP and method are safe values
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// extractClientIP extracts the client IP address from the request, checking
// proxy headers in order of specificity. When behind a reverse proxy (nginx,
// HAProxy, etc.), the proxy should set X-Client-IP or X-Forwarded-For headers.
//
// Header precedence (highest first):
// 1. X-Client-IP — explicitly set by reverse proxy configuration
// 2. X-Forwarded-For — first (leftmost) IP in the chain from the proxy
// 3. RemoteAddr — direct connection fallback (with port stripped)
func extractClientIP(r *http.Request) string {
	if clientIP := r.Header.Get("X-Client-Ip"); clientIP != "" {
		host, _, err := net.SplitHostPort(clientIP)
		if err == nil {
			return host
		}
		return clientIP
	}

	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		// X-Forwarded-For format: "client, proxy1, proxy2"
		// Take the first (leftmost) IP as the real client.
		if idx := strings.IndexByte(fwd, ','); idx >= 0 {
			fwd = strings.TrimSpace(fwd[:idx])
		}
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
