package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/teran/mcp-paperless-ngx/handlers"
	infra "github.com/teran/mcp-paperless-ngx/infrastructure/paperless"
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
// verifyToken tests
// ---------------------------------------------------------------------------

func TestVerifyToken(t *testing.T) {
	t.Parallel()

	t.Run("valid token passes", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"count":1,"results":[{"id":1,"name":"Test","slug":"test","matching_algorithm":1,"is_insensitive":false,"document_count":0,"last_correspondence":"","user_can_change":true}]}`))
		}))
		defer srv.Close()

		client := infra.NewClient(srv.URL, "valid-token")
		err := verifyToken(context.Background(), client)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
	})

	t.Run("invalid token returns error wrapping errTokenVerification", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"detail":"Invalid token"}`))
		}))
		defer srv.Close()

		client := infra.NewClient(srv.URL, "bad-token")
		err := verifyToken(context.Background(), client)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, errTokenVerification) {
			t.Errorf("expected error to wrap errTokenVerification, got: %v", err)
		}
	})

	t.Run("unreachable server returns error wrapping errTokenVerification", func(t *testing.T) {
		client := infra.NewClient("http://127.0.0.1:1", "some-token")
		err := verifyToken(context.Background(), client)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, errTokenVerification) {
			t.Errorf("expected error to wrap errTokenVerification, got: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// injectClientMiddleware tests
// ---------------------------------------------------------------------------

func TestInjectClientMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("with valid token in context creates services and passes through", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"count":1,"results":[{"id":1,"name":"Test","slug":"test","matching_algorithm":1,"is_insensitive":false,"document_count":0,"last_correspondence":"","user_can_change":true}]}`))
		}))
		defer srv.Close()

		middleware := injectClientMiddleware(srv.URL)

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
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("paperless handler should not be called when token is missing")
		}))
		defer srv.Close()

		middleware := injectClientMiddleware(srv.URL)

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

	t.Run("invalid token returns 401", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"detail":"Invalid token"}`))
		}))
		defer srv.Close()

		middleware := injectClientMiddleware(srv.URL)

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("next handler should not be called when token is invalid")
		})

		req := httptest.NewRequestWithContext(
			handlers.WithClient(context.Background(), "bad-token"),
			http.MethodGet, "/", nil,
		)
		rr := httptest.NewRecorder()
		middleware(next).ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", rr.Code)
		}
	})
}
