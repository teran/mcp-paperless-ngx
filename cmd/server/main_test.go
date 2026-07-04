package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/teran/mcp-paperless-ngx/handlers"
)

// ---------------------------------------------------------------------------
// sanitizeLog tests
// ---------------------------------------------------------------------------

func TestSanitizeLog(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "removes newlines", input: "hello\nworld", want: "helloworld"},
		{name: "removes carriage returns", input: "hello\rworld", want: "helloworld"},
		{name: "truncates long string", input: strings.Repeat("a", 600), want: strings.Repeat("a", 500) + "..."},
		{name: "keeps short string", input: "hello", want: "hello"},
		{name: "empty string", input: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := sanitizeLog(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeLog(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// injectClientMiddleware tests
// ---------------------------------------------------------------------------

func TestInjectClientMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("with valid token in context creates services and passes through", func(t *testing.T) {
		middleware := injectClientMiddleware("http://paperless:8000")

		var handlerCalled bool
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true

			// Verify that services are accessible from context
			if svc := docServiceFromContext(r.Context()); svc == nil {
				t.Error("docServiceFromContext returned nil")
			}
			if svc := corrServiceFromContext(r.Context()); svc == nil {
				t.Error("corrServiceFromContext returned nil")
			}
			if svc := docTypeServiceFromContext(r.Context()); svc == nil {
				t.Error("docTypeServiceFromContext returned nil")
			}
			if svc := tagServiceFromContext(r.Context()); svc == nil {
				t.Error("tagServiceFromContext returned nil")
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
		middleware := injectClientMiddleware("http://paperless:8000")

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
