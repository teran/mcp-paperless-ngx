package handlers

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/time/rate"
)

// ---------------------------------------------------------------------------
// SanitizeLog
// ---------------------------------------------------------------------------

func TestSanitizeLog(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty string", "", ""},
		{"normal text preserved", "hello world", "hello world"},
		{"newline stripped", "line1\nline2", "line1line2"},
		{"carriage return stripped", "a\rb", "ab"},
		{"tab preserved", "a\tb", "a\tb"},
		{"DEL control char stripped", "a\x7fb", "ab"},
		{"control char 0x01 stripped", "a\x01b", "ab"},
		{"high unicode preserved", "привет", "привет"},
		{"only control chars removed", "\x00\x01\x02", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := SanitizeLog(tt.in); got != tt.want {
				t.Errorf("SanitizeLog(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// LoggingMiddleware: read-body error and batch-size exceeded
// ---------------------------------------------------------------------------

// errReader is a reader that fails on the first Read.
type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }

func TestLoggingMiddleware_ReadBodyError_413(t *testing.T) {
	t.Parallel()

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called on body read error")
	})

	handler := LoggingMiddleware(next)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", nil)
	req.Body = io.NopCloser(errReader{})
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413, got %d", rr.Code)
	}
}

func TestLoggingMiddleware_BatchTooLarge_400(t *testing.T) {
	t.Parallel()

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called for oversized batch")
	})

	handler := LoggingMiddleware(next)

	// Build a batch array with MaxBatchSize+1 elements.
	var b strings.Builder
	b.WriteByte('[')
	for i := 0; i <= MaxBatchSize; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(`{"jsonrpc":"2.0","method":"x","id":` + string(rune('0'+i%10)) + `}`)
	}
	b.WriteByte(']')

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(b.String()))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// Rate limiter: default burst / per-client burst
// ---------------------------------------------------------------------------

func TestNewRateLimiter_DefaultGlobalBurst(t *testing.T) {
	t.Parallel()

	// GlobalBurst <= 0 should default to 2x the rate.
	rl := NewRateLimiter(RateLimiterConfig{
		GlobalLimit:    rate.Limit(50),
		GlobalBurst:    0,
		PerClientLimit: rate.Limit(10),
		PerClientBurst: 10,
	})
	defer rl.Stop()

	if rl.global.Burst() != 100 {
		t.Errorf("expected default global burst 100 (2x50), got %d", rl.global.Burst())
	}
}

func TestRateLimiter_Allow_DefaultPerClientBurst(t *testing.T) {
	t.Parallel()

	// PerClientBurst <= 0 should default to 2x the per-client rate when a
	// new client limiter is created.
	rl := NewRateLimiter(RateLimiterConfig{
		GlobalLimit:    rate.Limit(1000),
		GlobalBurst:    1000,
		PerClientLimit: rate.Limit(10),
		PerClientBurst: 0,
	})
	defer rl.Stop()

	if !rl.Allow("10.0.0.1") {
		t.Fatal("expected Allow to succeed")
	}

	rl.mu.Lock()
	cl := rl.clients["10.0.0.1"]
	rl.mu.Unlock()
	if cl == nil {
		t.Fatal("expected client limiter to be created")
	}
	if cl.limiter.Burst() != 20 {
		t.Errorf("expected default per-client burst 20 (2x10), got %d", cl.limiter.Burst())
	}
}

// ---------------------------------------------------------------------------
// extractClientIP
// ---------------------------------------------------------------------------

func TestExtractClientIP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		want       string
	}{
		{"x-client-ip with port strips port", map[string]string{"X-Client-Ip": "10.1.2.3:9999"}, "10.9.9.9:80", "10.1.2.3"},
		{"x-client-ip without port returned as-is", map[string]string{"X-Client-Ip": "10.1.2.3"}, "10.9.9.9:80", "10.1.2.3"},
		{"x-forwarded-for first of comma list", map[string]string{"X-Forwarded-For": "10.2.2.2, 10.3.3.3"}, "10.9.9.9:80", "10.2.2.2"},
		{"x-forwarded-for with port strips port", map[string]string{"X-Forwarded-For": "10.4.4.4:1234"}, "10.9.9.9:80", "10.4.4.4"},
		{"x-forwarded-for without port returned as-is", map[string]string{"X-Forwarded-For": "10.5.5.5"}, "10.9.9.9:80", "10.5.5.5"},
		{"fallback to remote addr strips port", nil, "10.6.6.6:8080", "10.6.6.6"},
		{"fallback remote addr without port returned as-is", nil, "10.7.7.7", "10.7.7.7"},
		{"prefers x-client-ip over xff", map[string]string{"X-Client-Ip": "10.1.1.1", "X-Forwarded-For": "10.2.2.2"}, "10.9.9.9:80", "10.1.1.1"},
		{"prefers xff over remote addr", map[string]string{"X-Forwarded-For": "10.8.8.8"}, "10.9.9.9:80", "10.8.8.8"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			req.RemoteAddr = tt.remoteAddr
			if got := extractClientIP(req); got != tt.want {
				t.Errorf("extractClientIP = %q, want %q", got, tt.want)
			}
		})
	}
}
