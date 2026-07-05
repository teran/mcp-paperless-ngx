package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/teran/mcp-paperless-ngx/handlers"
)

// testHTTPClient is a shared HTTP client for tests that never follows redirects.
var testHTTPClient = &http.Client{ //nolint:gochecknoglobals
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

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
		{name: "removes ANSI escape leading byte", input: "hello\x1b[2Jworld", want: "hello[2Jworld"},
		{name: "removes null byte", input: "hello\x00world", want: "helloworld"},
		{name: "preserves tab", input: "hello\tworld", want: "hello\tworld"},
		{name: "removes bell", input: "hello\x07world", want: "helloworld"},
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
