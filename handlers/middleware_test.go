package handlers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTokenMiddleware(t *testing.T) { //nolint:gocognit,gocyclo,maintidx
	t.Parallel()

	t.Run("Bearer token passes through", func(t *testing.T) {
		t.Parallel()

		var capturedToken string
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedToken = ClientFromContext(r.Context())
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
			capturedToken = ClientFromContext(r.Context())
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
			capturedToken = ClientFromContext(r.Context())
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
			capturedToken = ClientFromContext(r.Context())
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
			capturedToken = ClientFromContext(r.Context())
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

	t.Run("token exceeding MaxTokenLength returns 401", func(t *testing.T) {
		t.Parallel()

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("next handler should not be called with oversized token")
		})

		handler := TokenMiddleware(next)

		longToken := strings.Repeat("A", MaxTokenLength+1)
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+longToken)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", rr.Code)
		}
	})

	t.Run("token at MaxTokenLength passes through", func(t *testing.T) {
		t.Parallel()

		var capturedToken string
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedToken = ClientFromContext(r.Context())
			w.WriteHeader(http.StatusOK)
		})

		handler := TokenMiddleware(next)

		token := strings.Repeat("B", MaxTokenLength)
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
		if capturedToken != token {
			t.Errorf("expected token of length %d, got length %d", MaxTokenLength, len(capturedToken))
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

// ============================================================
// loggingResponseWriter tests
// ============================================================

func TestLoggingResponseWriter(t *testing.T) {
	t.Parallel()

	t.Run("captures status code and body size", func(t *testing.T) {
		t.Parallel()

		rr := httptest.NewRecorder()
		lrw := &loggingResponseWriter{
			ResponseWriter: rr,
			statusCode:     http.StatusOK,
		}

		lrw.WriteHeader(http.StatusCreated)
		n, err := lrw.Write([]byte("hello"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n != 5 {
			t.Errorf("expected written bytes=5, got %d", n)
		}
		if lrw.statusCode != http.StatusCreated {
			t.Errorf("expected status code %d, got %d", http.StatusCreated, lrw.statusCode)
		}
		if lrw.bodySize != 5 {
			t.Errorf("expected body size 5, got %d", lrw.bodySize)
		}
	})

	t.Run("accumulates body size across multiple writes", func(t *testing.T) {
		t.Parallel()

		rr := httptest.NewRecorder()
		lrw := &loggingResponseWriter{
			ResponseWriter: rr,
			statusCode:     http.StatusOK,
		}

		_, _ = lrw.Write([]byte("abc"))
		_, _ = lrw.Write([]byte("def"))
		_, _ = lrw.Write([]byte("gh"))

		if lrw.bodySize != 8 {
			t.Errorf("expected body size 8, got %d", lrw.bodySize)
		}
	})

	t.Run("default status code is 200", func(t *testing.T) {
		t.Parallel()

		rr := httptest.NewRecorder()
		lrw := &loggingResponseWriter{
			ResponseWriter: rr,
			statusCode:     http.StatusOK,
		}

		_, _ = lrw.Write([]byte("data"))

		if lrw.statusCode != http.StatusOK {
			t.Errorf("expected default status %d, got %d", http.StatusOK, lrw.statusCode)
		}
	})
}

// ============================================================
// mcpRequestMethod tests
// ============================================================

func TestMCPRequestMethod(t *testing.T) {
	t.Parallel()

	t.Run("tools/call with tool name", func(t *testing.T) {
		t.Parallel()

		body := []byte(`{"jsonrpc":"2.0","method":"tools/call","params":{"name":"search_documents","arguments":{"query":"test"}},"id":"1"}`)
		method := mcpRequestMethod(body)
		if method != "search_documents" {
			t.Errorf("expected 'search_documents', got %q", method)
		}
	})

	t.Run("initialize method", func(t *testing.T) {
		t.Parallel()

		body := []byte(`{"jsonrpc":"2.0","method":"initialize","params":{},"id":"1"}`)
		method := mcpRequestMethod(body)
		if method != "initialize" {
			t.Errorf("expected 'initialize', got %q", method)
		}
	})

	t.Run("tools/list method", func(t *testing.T) {
		t.Parallel()

		body := []byte(`{"jsonrpc":"2.0","method":"tools/list","id":"1"}`)
		method := mcpRequestMethod(body)
		if method != "tools/list" {
			t.Errorf("expected 'tools/list', got %q", method)
		}
	})

	t.Run("empty body returns unknown", func(t *testing.T) {
		t.Parallel()

		method := mcpRequestMethod([]byte{})
		if method != "unknown" {
			t.Errorf("expected 'unknown', got %q", method)
		}
	})

	t.Run("invalid JSON returns unknown", func(t *testing.T) {
		t.Parallel()

		method := mcpRequestMethod([]byte(`not json`))
		if method != "unknown" {
			t.Errorf("expected 'unknown', got %q", method)
		}
	})
}

// ============================================================
// LoggingMiddleware tests
// ============================================================

// ============================================================
// checkBatchSize tests
// ============================================================

func TestCheckBatchSize(t *testing.T) {
	t.Parallel()

	t.Run("single request not a batch", func(t *testing.T) {
		t.Parallel()

		body := []byte(`{"jsonrpc":"2.0","method":"ping","id":1}`)
		if err := checkBatchSize(body); err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("batch within limit", func(t *testing.T) {
		t.Parallel()

		items := make([]string, 50)
		for i := range items {
			items[i] = `{"jsonrpc":"2.0","method":"ping","id":` + fmt.Sprintf("%d", i+1) + `}`
		}
		body := []byte("[" + strings.Join(items, ",") + "]")
		if err := checkBatchSize(body); err != nil {
			t.Errorf("expected nil for batch size 50, got %v", err)
		}
	})

	t.Run("batch exactly at limit", func(t *testing.T) {
		t.Parallel()

		items := make([]string, MaxBatchSize)
		for i := range items {
			items[i] = `{"jsonrpc":"2.0","method":"ping","id":` + fmt.Sprintf("%d", i+1) + `}`
		}
		body := []byte("[" + strings.Join(items, ",") + "]")
		if err := checkBatchSize(body); err != nil {
			t.Errorf("expected nil for batch size %d, got %v", MaxBatchSize, err)
		}
	})

	t.Run("batch exceeding limit", func(t *testing.T) {
		t.Parallel()

		items := make([]string, MaxBatchSize+1)
		for i := range items {
			items[i] = `{"jsonrpc":"2.0","method":"ping","id":` + fmt.Sprintf("%d", i+1) + `}`
		}
		body := []byte("[" + strings.Join(items, ",") + "]")
		if err := checkBatchSize(body); err == nil {
			t.Errorf("expected error for batch size %d, got nil", MaxBatchSize+1)
		}
	})

	t.Run("empty body", func(t *testing.T) {
		t.Parallel()

		if err := checkBatchSize([]byte{}); err != nil {
			t.Errorf("expected nil for empty body, got %v", err)
		}
	})

	t.Run("malformed batch JSON", func(t *testing.T) {
		t.Parallel()

		body := []byte(`[invalid json`)
		if err := checkBatchSize(body); err != nil {
			t.Errorf("expected nil for malformed batch (caught downstream), got %v", err)
		}
	})
}

func TestLoggingMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("logs tools/call method with payload sizes", func(t *testing.T) {
		t.Parallel()

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"1","result":{"content":"ok"}}`))
		})

		handler := LoggingMiddleware(next)

		body := `{"jsonrpc":"2.0","method":"tools/call","params":{"name":"search_documents","arguments":{"query":"test"}},"id":"1"}`
		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", strings.NewReader(body))
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
		if rr.Body.String() != `{"id":"1","result":{"content":"ok"}}` {
			t.Errorf("unexpected response body: %q", rr.Body.String())
		}
	})

	t.Run("preserves request body for downstream handlers", func(t *testing.T) {
		t.Parallel()

		var capturedBody []byte
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			capturedBody = body
			w.WriteHeader(http.StatusOK)
		})

		handler := LoggingMiddleware(next)

		body := `{"jsonrpc":"2.0","method":"initialize","params":{},"id":"1"}`
		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", strings.NewReader(body))
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if string(capturedBody) != body {
			t.Errorf("expected body %q, got %q", body, string(capturedBody))
		}
	})

	t.Run("passes through on body read error", func(t *testing.T) {
		t.Parallel()

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler := LoggingMiddleware(next)

		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
	})

	t.Run("logs unknown method for non-JSON body", func(t *testing.T) {
		t.Parallel()

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		})

		handler := LoggingMiddleware(next)

		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", strings.NewReader("plain text body"))
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
	})
}

// ============================================================
// RecoveryMiddleware tests
// ============================================================

func TestRecoveryMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("passes through normal request", func(t *testing.T) {
		t.Parallel()

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		})

		handler := RecoveryMiddleware(next)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
		if rr.Body.String() != "ok" {
			t.Errorf("expected body %q, got %q", "ok", rr.Body.String())
		}
	})

	t.Run("recovers from panic", func(t *testing.T) {
		t.Parallel()

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("test panic")
		})

		handler := RecoveryMiddleware(next)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", rr.Code)
		}
		if rr.Body.String() != "Internal Server Error\n" {
			t.Errorf("expected body %q, got %q", "Internal Server Error\n", rr.Body.String())
		}
	})

	t.Run("recovers from nil panic", func(t *testing.T) {
		t.Parallel()

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var p *int
			_ = *p // intentional nil dereference
		})

		handler := RecoveryMiddleware(next)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", rr.Code)
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
