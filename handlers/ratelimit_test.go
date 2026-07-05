package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestNewRateLimiterEviction(t *testing.T) {
	t.Parallel()

	cfg := RateLimiterConfig{
		GlobalLimit:    rate.Limit(100),
		GlobalBurst:    100,
		PerClientLimit: rate.Limit(10),
		PerClientBurst: 10,
	}
	rl := NewRateLimiter(cfg)
	defer rl.Stop()

	// Add some client entries.
	rl.Allow("10.0.0.1")
	rl.Allow("10.0.0.2")

	rl.mu.Lock()
	if len(rl.clients) != 2 {
		t.Errorf("expected 2 clients before eviction, got %d", len(rl.clients))
	}
	// Simulate expired entries by backdating lastSeen.
	rl.clients["10.0.0.1"].lastSeen = time.Now().Add(-clientTTL - time.Minute)
	rl.clients["10.0.0.2"].lastSeen = time.Now().Add(-clientTTL - time.Minute)
	rl.mu.Unlock()

	// Run eviction.
	rl.evictExpired()

	rl.mu.Lock()
	if len(rl.clients) != 0 {
		t.Errorf("expected 0 clients after eviction, got %d", len(rl.clients))
	}
	rl.mu.Unlock()
}

func TestRateLimiter_Stop(t *testing.T) {
	t.Parallel()

	cfg := RateLimiterConfig{
		GlobalLimit:    rate.Limit(100),
		GlobalBurst:    100,
		PerClientLimit: rate.Limit(10),
		PerClientBurst: 10,
	}
	rl := NewRateLimiter(cfg)
	rl.Stop() // should not panic or deadlock

	// After Stop the limiter should still accept requests gracefully
	// (eviction goroutine is stopped but Allow still works).
	if !rl.Allow("10.0.0.1") {
		t.Errorf("expected Allow to return true after Stop")
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("allows requests within global limit", func(t *testing.T) {
		t.Parallel()

		cfg := RateLimiterConfig{
			GlobalLimit:    rate.Limit(100),
			GlobalBurst:    100,
			PerClientLimit: rate.Limit(10),
			PerClientBurst: 10,
		}
		handler := RateLimitMiddleware(cfg)

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		for range 5 {
			req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil)
			req.RemoteAddr = "10.0.0.1:12345"
			rr := httptest.NewRecorder()
			handler(next).ServeHTTP(rr, req)
			if rr.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d", rr.Code)
			}
		}
	})

	t.Run("rate limits different clients independently", func(t *testing.T) {
		t.Parallel()

		cfg := RateLimiterConfig{
			GlobalLimit:    rate.Limit(100),
			GlobalBurst:    100,
			PerClientLimit: rate.Limit(1),
			PerClientBurst: 1,
		}
		handler := RateLimitMiddleware(cfg)

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		// Second request from same IP should be rate limited
		{
			req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil)
			req.RemoteAddr = "10.0.0.1:12345"
			rr := httptest.NewRecorder()
			handler(next).ServeHTTP(rr, req)
			if rr.Code != http.StatusOK {
				t.Errorf("first request: expected 200, got %d", rr.Code)
			}
		}
		{
			req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil)
			req.RemoteAddr = "10.0.0.1:12345"
			rr := httptest.NewRecorder()
			handler(next).ServeHTTP(rr, req)
			if rr.Code != http.StatusTooManyRequests {
				t.Errorf("second request: expected 429, got %d", rr.Code)
			}
		}
	})

	t.Run("different IPs are independently limited", func(t *testing.T) {
		t.Parallel()

		cfg := RateLimiterConfig{
			GlobalLimit:    rate.Limit(100),
			GlobalBurst:    100,
			PerClientLimit: rate.Limit(1),
			PerClientBurst: 1,
		}
		handler := RateLimitMiddleware(cfg)

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		for _, ip := range []string{"10.0.0.1:12345", "10.0.0.2:12345"} {
			req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil)
			req.RemoteAddr = ip
			rr := httptest.NewRecorder()
			handler(next).ServeHTTP(rr, req)
			if rr.Code != http.StatusOK {
				t.Errorf("first request from %s: expected 200, got %d", ip, rr.Code)
			}
		}
	})

	t.Run("respects X-Client-IP header over X-Forwarded-For", func(t *testing.T) {
		t.Parallel()

		cfg := RateLimiterConfig{
			GlobalLimit:    rate.Limit(100),
			GlobalBurst:    100,
			PerClientLimit: rate.Limit(1),
			PerClientBurst: 1,
		}
		handler := RateLimitMiddleware(cfg)

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		{
			req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil)
			req.RemoteAddr = "10.0.0.1:12345"
			req.Header.Set("X-Client-Ip", "203.0.113.1")
			req.Header.Set("X-Forwarded-For", "10.0.0.1")
			rr := httptest.NewRecorder()
			handler(next).ServeHTTP(rr, req)
			if rr.Code != http.StatusOK {
				t.Errorf("first request: expected 200, got %d", rr.Code)
			}
		}
		// Second request with same X-Client-IP should be blocked (per-client limit=1)
		{
			req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil)
			req.RemoteAddr = "10.0.0.2:12345"
			req.Header.Set("X-Client-Ip", "203.0.113.1")
			rr := httptest.NewRecorder()
			handler(next).ServeHTTP(rr, req)
			if rr.Code != http.StatusTooManyRequests {
				t.Errorf("second request with same X-Client-IP: expected 429, got %d", rr.Code)
			}
		}
		// Request with different X-Client-IP should be allowed
		{
			req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil)
			req.RemoteAddr = "10.0.0.3:12345"
			req.Header.Set("X-Client-Ip", "203.0.113.2")
			rr := httptest.NewRecorder()
			handler(next).ServeHTTP(rr, req)
			if rr.Code != http.StatusOK {
				t.Errorf("request from different X-Client-IP: expected 200, got %d", rr.Code)
			}
		}
	})

	t.Run("respects X-Forwarded-For header", func(t *testing.T) {
		t.Parallel()

		cfg := RateLimiterConfig{
			GlobalLimit:    rate.Limit(100),
			GlobalBurst:    100,
			PerClientLimit: rate.Limit(1),
			PerClientBurst: 1,
		}
		handler := RateLimitMiddleware(cfg)

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		{
			req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil)
			req.RemoteAddr = "10.0.0.1:12345"
			req.Header.Set("X-Forwarded-For", "203.0.113.1")
			rr := httptest.NewRecorder()
			handler(next).ServeHTTP(rr, req)
			if rr.Code != http.StatusOK {
				t.Errorf("first request: expected 200, got %d", rr.Code)
			}
		}
		{
			req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil)
			req.RemoteAddr = "10.0.0.1:12345"
			req.Header.Set("X-Forwarded-For", "203.0.113.1")
			rr := httptest.NewRecorder()
			handler(next).ServeHTTP(rr, req)
			if rr.Code != http.StatusTooManyRequests {
				t.Errorf("second request from same X-Forwarded-For: expected 429, got %d", rr.Code)
			}
		}
	})

	t.Run("global limit blocks all clients", func(t *testing.T) {
		t.Parallel()

		cfg := RateLimiterConfig{
			GlobalLimit:    rate.Limit(1),
			GlobalBurst:    1,
			PerClientLimit: rate.Limit(10),
			PerClientBurst: 10,
		}
		handler := RateLimitMiddleware(cfg)

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		// First request from any client is allowed
		{
			req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil)
			req.RemoteAddr = "10.0.0.1:12345"
			rr := httptest.NewRecorder()
			handler(next).ServeHTTP(rr, req)
			if rr.Code != http.StatusOK {
				t.Errorf("first request: expected 200, got %d", rr.Code)
			}
		}

		// Second request from different client is blocked by global rate
		{
			req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil)
			req.RemoteAddr = "10.0.0.2:12345"
			rr := httptest.NewRecorder()
			handler(next).ServeHTTP(rr, req)
			if rr.Code != http.StatusTooManyRequests {
				t.Errorf("second request: expected 429, got %d", rr.Code)
			}
		}
	})
}
