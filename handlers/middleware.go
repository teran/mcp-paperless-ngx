package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type paperlessClientKey struct{}

// WithClient stores a value (typically *paperless.Client) in the context.
func WithClient(ctx context.Context, client any) context.Context {
	return context.WithValue(ctx, paperlessClientKey{}, client)
}

// ClientFromContext retrieves a value stored by WithClient.
// Returns nil if not present.
func ClientFromContext(ctx context.Context) any {
	return ctx.Value(paperlessClientKey{})
}

// DefaultMaxRequestBodySize is the maximum allowed size for a request body (1 MB).
const DefaultMaxRequestBodySize = 1 << 20

// BodyLimitMiddleware limits the request body size to maxBytes.
// The size limit is enforced before the body reaches downstream handlers,
// preventing resource exhaustion from large requests.
func BodyLimitMiddleware(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}

// MaxBytesError returns true if the error is an http.MaxBytesError.
func MaxBytesError(err error) bool {
	var maxBytesErr *http.MaxBytesError
	return err != nil && errors.As(err, &maxBytesErr)
}

// loggingResponseWriter wraps http.ResponseWriter to capture
// the HTTP status code and response body size.
type loggingResponseWriter struct {
	http.ResponseWriter

	statusCode int
	bodySize   int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	n, err := lrw.ResponseWriter.Write(b)
	lrw.bodySize += n
	return n, err
}

// mcpRequestMethod attempts to extract the MCP method name from a JSON-RPC
// request body. For "tools/call" it additionally extracts the tool name from
// params.name. Returns the extracted name or "unknown" if parsing fails.
func mcpRequestMethod(body []byte) string {
	var req struct {
		Method string `json:"method"`
		Params struct {
			Name string `json:"name"`
		} `json:"params"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return "unknown"
	}
	if req.Method == "tools/call" && req.Params.Name != "" {
		return req.Params.Name
	}
	if req.Method != "" {
		return req.Method
	}
	return "unknown"
}

// sanitizeLog strips control characters from strings before logging
// to prevent log injection attacks (e.g. newlines injected via JSON fields).
func sanitizeLog(s string) string {
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}

// LoggingMiddleware logs MCP request details at INFO level.
// Records: timestamp, MCP method name, request duration, request body size,
// and response body size.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Read and buffer request body for method extraction.
		// r.Body is already wrapped with http.MaxBytesReader by BodyLimitMiddleware (outermost),
		// so reading is bounded to 1 MB.
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("INFO mcp_log request_error=read_body error=%v", err)
			http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))

		reqSize := len(body)
		mcpMethod := sanitizeLog(mcpRequestMethod(body))

		// Wrap ResponseWriter to capture response size.
		lrw := &loggingResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(lrw, r)

		duration := time.Since(start)
		log.Printf("INFO mcp_log method=%s duration=%v req_size=%d resp_size=%d status=%d",
			mcpMethod, duration, reqSize, lrw.bodySize, lrw.statusCode)
	})
}

// TokenMiddleware extracts the bearer token from the Authorization header
// and stores it in the request context.
func TokenMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
			return
		}

		// Support both "Bearer <token>" and "Token <token>" schemes.
		var token string
		if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			token = strings.TrimSpace(authHeader[len("bearer "):])
		} else if strings.HasPrefix(strings.ToLower(authHeader), "token ") {
			token = strings.TrimSpace(authHeader[len("token "):])
		}

		if token == "" {
			http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
			return
		}

		ctx := WithClient(r.Context(), token)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
