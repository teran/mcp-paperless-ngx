package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/teran/mcp-paperless-ngx/handlers"
)

// testHTTPClient is a shared HTTP client for tests that never follows redirects.
var testHTTPClient = &http.Client{ //nolint:gochecknoglobals
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

// ---------------------------------------------------------------------------
// parseRateLimit tests
// ---------------------------------------------------------------------------

func TestParseRateLimit(t *testing.T) {
	t.Parallel()

	t.Run("valid value", func(t *testing.T) {
		t.Parallel()
		if got := parseRateLimit("50", 100); got != 50 {
			t.Errorf("parseRateLimit(%q, 100) = %d, want 50", "50", got)
		}
	})

	t.Run("empty string returns default", func(t *testing.T) {
		t.Parallel()
		if got := parseRateLimit("", 100); got != 100 {
			t.Errorf("parseRateLimit(%q, 100) = %d, want 100", "", got)
		}
	})

	t.Run("invalid string returns default", func(t *testing.T) {
		t.Parallel()
		if got := parseRateLimit("abc", 100); got != 100 {
			t.Errorf("parseRateLimit(%q, 100) = %d, want 100", "abc", got)
		}
	})

	t.Run("zero value returns default", func(t *testing.T) {
		t.Parallel()
		if got := parseRateLimit("0", 100); got != 100 {
			t.Errorf("parseRateLimit(%q, 100) = %d, want 100", "0", got)
		}
	})

	t.Run("negative value returns default", func(t *testing.T) {
		t.Parallel()
		if got := parseRateLimit("-5", 100); got != 100 {
			t.Errorf("parseRateLimit(%q, 100) = %d, want 100", "-5", got)
		}
	})
}

// ---------------------------------------------------------------------------
// parseDurationSeconds tests
// ---------------------------------------------------------------------------

func TestParseDurationSeconds(t *testing.T) {
	t.Parallel()

	t.Run("valid value", func(t *testing.T) {
		t.Parallel()
		if got := parseDurationSeconds("30", 300); got != 30*time.Second {
			t.Errorf("parseDurationSeconds(%q, 300) = %v, want %v", "30", got, 30*time.Second)
		}
	})

	t.Run("empty string returns default", func(t *testing.T) {
		t.Parallel()
		if got := parseDurationSeconds("", 300); got != 300*time.Second {
			t.Errorf("parseDurationSeconds(%q, 300) = %v, want %v", "", got, 300*time.Second)
		}
	})

	t.Run("invalid string returns default", func(t *testing.T) {
		t.Parallel()
		if got := parseDurationSeconds("abc", 300); got != 300*time.Second {
			t.Errorf("parseDurationSeconds(%q, 300) = %v, want %v", "abc", got, 300*time.Second)
		}
	})

	t.Run("zero value returns zero", func(t *testing.T) {
		t.Parallel()
		if got := parseDurationSeconds("0", 300); got != 0 {
			t.Errorf("parseDurationSeconds(%q, 300) = %v, want 0", "0", got)
		}
	})

	t.Run("negative value returns default", func(t *testing.T) {
		t.Parallel()
		if got := parseDurationSeconds("-1", 300); got != 300*time.Second {
			t.Errorf("parseDurationSeconds(%q, 300) = %v, want %v", "-1", got, 300*time.Second)
		}
	})
}

// ---------------------------------------------------------------------------
// injectClientMiddleware tests
// ---------------------------------------------------------------------------

func TestInjectClientMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("with valid token in context creates services and passes through", func(t *testing.T) {
		middleware := injectClientMiddleware("http://paperless:8000", testHTTPClient)

		var handlerCalled bool
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true

			// Verify that services are accessible from context
			if svc := handlers.DocServiceFromContext(r.Context()); svc == nil {
				t.Error("handlers.DocServiceFromContext returned nil")
			}
			if svc := handlers.CorrServiceFromContext(r.Context()); svc == nil {
				t.Error("handlers.CorrServiceFromContext returned nil")
			}
			if svc := handlers.DocTypeServiceFromContext(r.Context()); svc == nil {
				t.Error("handlers.DocTypeServiceFromContext returned nil")
			}
			if svc := handlers.TagServiceFromContext(r.Context()); svc == nil {
				t.Error("handlers.TagServiceFromContext returned nil")
			}

			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequestWithContext(
			handlers.WithClient(context.Background(), "valid-token"),
			http.MethodGet, "/", nil,
		)
		rr := httptest.NewRecorder()
		middleware(next).ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
		if !handlerCalled {
			t.Error("next handler was not called")
		}
	})

	t.Run("missing token in context returns 401", func(t *testing.T) {
		middleware := injectClientMiddleware("http://paperless:8000", testHTTPClient)

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("next handler should not be called when token is missing")
		})

		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()
		middleware(next).ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", rr.Code)
		}
	})
}
