package handlers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTokenMiddleware(t *testing.T) { //nolint:gocognit,maintidx
	t.Parallel()

	t.Run("Bearer token passes through", func(t *testing.T) {
		t.Parallel()

		var capturedToken string
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := ClientFromContext(r.Context())
			if raw != nil {
				var ok bool
				capturedToken, ok = raw.(string)
				if !ok {
					t.Error("expected string type from context")
				}
			}
			w.WriteHeader(http.StatusOK)
		})

		handler := TokenMiddleware(next)

		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer my-secret-token")

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
		if capturedToken != "my-secret-token" {
			t.Errorf("expected token %q, got %q", "my-secret-token", capturedToken)
		}
	})

	t.Run("Token scheme (backwards compat) passes through", func(t *testing.T) {
		t.Parallel()

		var capturedToken string
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := ClientFromContext(r.Context())
			if raw != nil {
				var ok bool
				capturedToken, ok = raw.(string)
				if !ok {
					t.Error("expected string type from context")
				}
			}
			w.WriteHeader(http.StatusOK)
		})

		handler := TokenMiddleware(next)

		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Token token-via-old-scheme")

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
		if capturedToken != "token-via-old-scheme" { //nolint:gosec
			t.Errorf("expected token %q, got %q", "token-via-old-scheme", capturedToken)
		}
	})

	t.Run("case-insensitive Bearer prefix", func(t *testing.T) {
		t.Parallel()

		var capturedToken string
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := ClientFromContext(r.Context())
			if raw != nil {
				var ok bool
				capturedToken, ok = raw.(string)
				if !ok {
					t.Error("expected string type from context")
				}
			}
			w.WriteHeader(http.StatusOK)
		})

		handler := TokenMiddleware(next)

		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "BEARER uppercase-bearer")

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
		if capturedToken != "uppercase-bearer" {
			t.Errorf("expected token %q, got %q", "uppercase-bearer", capturedToken)
		}
	})

	t.Run("case-insensitive Token prefix", func(t *testing.T) {
		t.Parallel()

		var capturedToken string
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := ClientFromContext(r.Context())
			if raw != nil {
				var ok bool
				capturedToken, ok = raw.(string)
				if !ok {
					t.Error("expected string type from context")
				}
			}
			w.WriteHeader(http.StatusOK)
		})

		handler := TokenMiddleware(next)

		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "TOKEN upper-token")

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
		if capturedToken != "upper-token" {
			t.Errorf("expected token %q, got %q", "upper-token", capturedToken)
		}
	})

	t.Run("Bearer token with whitespace trimmed", func(t *testing.T) {
		t.Parallel()

		var capturedToken string
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := ClientFromContext(r.Context())
			if raw != nil {
				var ok bool
				capturedToken, ok = raw.(string)
				if !ok {
					t.Error("expected string type from context")
				}
			}
			w.WriteHeader(http.StatusOK)
		})

		handler := TokenMiddleware(next)

		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer   token-with-spaces   ")

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
		if capturedToken != "token-with-spaces" {
			t.Errorf("expected token %q, got %q", "token-with-spaces", capturedToken)
		}
	})

	t.Run("no Authorization header returns 401", func(t *testing.T) {
		t.Parallel()

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("next handler should not be called when header is missing")
		})

		handler := TokenMiddleware(next)

		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		// No Authorization header set.

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", rr.Code)
		}
		body := rr.Body.String()
		if body != "Missing Authorization header\n" {
			t.Errorf("unexpected body: %q", body)
		}
	})

	t.Run("empty token returns 401", func(t *testing.T) {
		t.Parallel()

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("next handler should not be called with empty token")
		})

		handler := TokenMiddleware(next)

		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer ")

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", rr.Code)
		}
		body := rr.Body.String()
		if body != "Invalid Authorization header format\n" {
			t.Errorf("unexpected body: %q", body)
		}
	})

	t.Run("just the word Bearer with no token returns 401", func(t *testing.T) {
		t.Parallel()

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("next handler should not be called")
		})

		handler := TokenMiddleware(next)

		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer")

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", rr.Code)
		}
	})

	t.Run("invalid scheme returns 401", func(t *testing.T) {
		t.Parallel()

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("next handler should not be called with invalid scheme")
		})

		handler := TokenMiddleware(next)

		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", rr.Code)
		}
		body := rr.Body.String()
		if body != "Invalid Authorization header format\n" {
			t.Errorf("unexpected body: %q", body)
		}
	})

	t.Run("malformed header returns 401", func(t *testing.T) {
		t.Parallel()

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("next handler should not be called with malformed header")
		})

		handler := TokenMiddleware(next)

		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		req.Header.Set("Authorization", " Bearer leading-whitespace")

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", rr.Code)
		}
	})
}

// ============================================================
// BodyLimitMiddleware tests
// ============================================================

func TestBodyLimitMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("small body passes through", func(t *testing.T) {
		t.Parallel()

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("unexpected error reading body: %v", err)
			}
			if string(body) != "hello" {
				t.Errorf("expected body %q, got %q", "hello", string(body))
			}
			w.WriteHeader(http.StatusOK)
		})

		handler := BodyLimitMiddleware(1024)(next)

		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", strings.NewReader("hello"))
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
	})

	t.Run("body exceeding limit returns 413", func(t *testing.T) {
		t.Parallel()

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, err := io.ReadAll(r.Body)
			if err == nil {
				t.Error("expected error reading body exceeding limit, got nil")
			}
			if !MaxBytesError(err) {
				t.Errorf("expected MaxBytesError, got %T: %v", err, err)
			}
			w.WriteHeader(http.StatusRequestEntityTooLarge)
		})

		handler := BodyLimitMiddleware(5)(next)

		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", strings.NewReader("hello world"))
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusRequestEntityTooLarge {
			t.Errorf("expected status 413, got %d", rr.Code)
		}
	})

	t.Run("zero limit allows zero-length body", func(t *testing.T) {
		t.Parallel()

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("unexpected error reading body: %v", err)
			}
			if len(body) != 0 {
				t.Errorf("expected empty body, got %q", string(body))
			}
			w.WriteHeader(http.StatusOK)
		})

		handler := BodyLimitMiddleware(0)(next)

		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", http.NoBody)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
	})

	t.Run("GET request with no body passes through", func(t *testing.T) {
		t.Parallel()

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler := BodyLimitMiddleware(1024)(next)

		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
	})
}

func TestMaxBytesError(t *testing.T) {
	t.Parallel()

	t.Run("with MaxBytesError", func(t *testing.T) {
		t.Parallel()

		err := &http.MaxBytesError{Limit: 100}
		if !MaxBytesError(err) {
			t.Error("expected true for MaxBytesError")
		}
	})

	t.Run("with other error", func(t *testing.T) {
		t.Parallel()

		err := errors.New("some other error")
		if MaxBytesError(err) {
			t.Error("expected false for other error")
		}
	})

	t.Run("with nil", func(t *testing.T) {
		t.Parallel()

		if MaxBytesError(nil) {
			t.Error("expected false for nil")
		}
	})
}
