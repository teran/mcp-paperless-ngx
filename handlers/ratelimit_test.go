package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/time/rate"
)

func TestRateLimitMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("allows requests within global limit", func(t *testing.T) {
		t.Parallel()

		cfg := RateLimiterConfig{
			GlobalLimit:    rate.Limit(100),
			PerClientLimit: rate.Limit(10),
			Burst:          10,
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
			PerClientLimit: rate.Limit(1),
			Burst:          1,
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
			PerClientLimit: rate.Limit(1),
			Burst:          2,
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

	t.Run("respects X-Forwarded-For header", func(t *testing.T) {
		t.Parallel()

		cfg := RateLimiterConfig{
			GlobalLimit:    rate.Limit(100),
			PerClientLimit: rate.Limit(1),
			Burst:          1,
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
			PerClientLimit: rate.Limit(10),
			Burst:          1,
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
