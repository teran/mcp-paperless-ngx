package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTokenMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("Bearer token passes through", func(t *testing.T) {
		t.Parallel()

		var capturedToken string
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := ClientFromContext(r.Context())
			if raw != nil {
				capturedToken = raw.(string)
			}
			w.WriteHeader(http.StatusOK)
		})

		handler := TokenMiddleware(next)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
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
				capturedToken = raw.(string)
			}
			w.WriteHeader(http.StatusOK)
		})

		handler := TokenMiddleware(next)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Token token-via-old-scheme")

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
		if capturedToken != "token-via-old-scheme" {
			t.Errorf("expected token %q, got %q", "token-via-old-scheme", capturedToken)
		}
	})

	t.Run("case-insensitive Bearer prefix", func(t *testing.T) {
		t.Parallel()

		var capturedToken string
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := ClientFromContext(r.Context())
			if raw != nil {
				capturedToken = raw.(string)
			}
			w.WriteHeader(http.StatusOK)
		})

		handler := TokenMiddleware(next)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
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
				capturedToken = raw.(string)
			}
			w.WriteHeader(http.StatusOK)
		})

		handler := TokenMiddleware(next)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
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
				capturedToken = raw.(string)
			}
			w.WriteHeader(http.StatusOK)
		})

		handler := TokenMiddleware(next)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
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

		req := httptest.NewRequest(http.MethodGet, "/", nil)
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

		req := httptest.NewRequest(http.MethodGet, "/", nil)
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

		req := httptest.NewRequest(http.MethodGet, "/", nil)
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

		req := httptest.NewRequest(http.MethodGet, "/", nil)
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

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", " Bearer leading-whitespace")

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", rr.Code)
		}
	})
}
