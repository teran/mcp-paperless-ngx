package handlers

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/sirupsen/logrus"

	"github.com/teran/mcp-paperless-ngx/domain"
)

// newRequestID generates a random hex request id. On a (virtually impossible)
// entropy failure it falls back to a nanosecond timestamp.
func newRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// RequestIDMiddleware generates a per-request request_id for every incoming
// request, reusing an inbound X-Request-ID header when present (e.g. from a
// reverse proxy), and threads it through the request context (L9/G11). It also
// captures the MCP session id (Mcp-Session-Id header) into the context when
// present.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = newRequestID()
		}
		ctx := domain.WithRequestID(r.Context(), id)
		if sid := r.Header.Get("Mcp-Session-Id"); sid != "" {
			ctx = domain.WithSessionID(ctx, sid)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// requestSource derives the client origin for the request. Per L8/G10 it reads,
// in order, X-Real-IP, then X-Forwarded-For, then the peer address, comma-joining
// the values so all proxy hops are visible behind a reverse proxy.
func requestSource(r *http.Request) string {
	var parts []string
	if v := r.Header.Get("X-Real-IP"); v != "" {
		parts = append(parts, v)
	}
	if v := r.Header.Get("X-Forwarded-For"); v != "" {
		parts = append(parts, v)
	}
	if len(parts) == 0 {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			return r.RemoteAddr
		}
		return host
	}
	return strings.Join(parts, ",")
}

// checkBatchSize validates JSON-RPC batch request size.
// If the body starts with '[', it is a batch request — the function parses
// it as an array and returns an error if len(array) > MaxBatchSize.
// Non-batch requests (starting with '{' or empty) are always accepted.
func checkBatchSize(body []byte) error {
	trimmed := bytes.TrimLeftFunc(body, unicode.IsSpace)
	if len(trimmed) == 0 || trimmed[0] != '[' {
		return nil // not a batch request
	}

	var batch []json.RawMessage
	if err := json.Unmarshal(trimmed, &batch); err != nil {
		return nil // malformed JSON will be caught downstream
	}
	if len(batch) > MaxBatchSize {
		return fmt.Errorf("batch size %d exceeds maximum of %d", len(batch), MaxBatchSize)
	}
	return nil
}

type paperlessClientKey struct{}

// WithClient stores a token string in the context.
func WithClient(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, paperlessClientKey{}, token)
}

// ClientFromContext retrieves the token string stored by WithClient.
// Returns empty string if not present.
func ClientFromContext(ctx context.Context) string {
	v, _ := ctx.Value(paperlessClientKey{}).(string)
	return v
}

// DefaultMaxRequestBodySize is the maximum allowed size for a request body (1 MB).
const DefaultMaxRequestBodySize = 1 << 20

// MaxBatchSize is the maximum number of JSON-RPC requests allowed in a
// single batch. Batches larger than this are rejected to prevent
// amplification attacks where a single HTTP request triggers many
// upstream API calls to Paperless-ngx.
const MaxBatchSize = 100

// MaxTokenLength is the maximum allowed length for an API token.
// Tokens longer than this are rejected to prevent DoS via oversized
// Authorization headers forwarded to the Paperless-ngx backend.
const MaxTokenLength = 512

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
// params.name. Returns the extracted name or one of the following sentinel
// values when parsing fails:
//   - "empty_body"    — the body is nil or zero-length
//   - "parse_error"   — the body is not valid JSON
//   - "no_method"     — JSON is valid but the "method" field is empty or missing
func mcpRequestMethod(body []byte) string {
	if len(body) == 0 {
		return "empty_body"
	}
	var req struct {
		Method string `json:"method"`
		Params struct {
			Name string `json:"name"`
		} `json:"params"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return "parse_error"
	}
	if req.Method == "tools/call" && req.Params.Name != "" {
		return req.Params.Name
	}
	if req.Method != "" {
		return req.Method
	}
	return "no_method"
}

// mcpRequestArgs extracts the tool-call arguments from a JSON-RPC body for
// logging (L8). It returns nil for non-tools/call methods or when no arguments
// are present. Only non-sensitive arguments are returned — all mcp-paperless-ngx
// tools are read-only and their arguments are search filters (see S2/N2).
func mcpRequestArgs(body []byte) any {
	if len(body) == 0 {
		return nil
	}
	var req struct {
		Method string `json:"method"`
		Params struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		} `json:"params"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return nil
	}
	if req.Method != "tools/call" || len(req.Params.Arguments) == 0 {
		return nil
	}
	return SanitizeLog(string(req.Params.Arguments))
}

// SanitizeLog strips control characters from strings before logging
// to prevent log injection attacks (e.g. newlines or ANSI escape codes
// injected via JSON fields). Only printable characters and horizontal tab
// are preserved; all other control characters (0x00-0x08, 0x0b-0x1f, 0x7f
// and above) are removed.
func SanitizeLog(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\t' {
			return r
		}
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, s)
}

// RecoveryMiddleware recovers from panics in downstream handlers, logs the
// panic with a stack trace, and returns 500 Internal Server Error. Without
// this middleware any panic in an HTTP goroutine would crash the entire server.
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				logrus.WithField("error", rec).Error("panic recovered")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// LoggingMiddleware logs MCP request details at INFO level.
// Records: timestamp, MCP method name, HTTP method, request path, request
// duration, request body size, and response body size.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Read and buffer request body for method extraction.
		// r.Body is already wrapped with http.MaxBytesReader by BodyLimitMiddleware (outermost),
		// so reading is bounded to 1 MB.
		body, err := io.ReadAll(r.Body)
		if err != nil {
			logrus.WithError(err).Info("request body read failed")
			http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))

		reqSize := len(body)
		mcpMethod := SanitizeLog(mcpRequestMethod(body))
		args := mcpRequestArgs(body)
		source := requestSource(r)

		// Reject batch requests that exceed MaxBatchSize to prevent
		// amplification attacks.
		if err := checkBatchSize(body); err != nil {
			domain.WithSession(r.Context(), logrus.WithFields(logrus.Fields{
				"http_method": SanitizeLog(r.Method),
				"path":        SanitizeLog(r.URL.Path),
				"tool":        mcpMethod,
				"source":      source,
				"duration":    time.Since(start),
				"in_bytes":    reqSize,
				"out_bytes":   0,
				"outcome":     "rejected_batch",
				"status":      http.StatusBadRequest,
			})).Debug("mcp_request")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Wrap ResponseWriter to capture response size.
		lrw := &loggingResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(lrw, r)

		duration := time.Since(start)
		// Per-request MCP tool-call log line at debug level (L8/G10) carrying
		// the request_id/session_id correlation fields (L9).
		domain.WithSession(r.Context(), logrus.WithFields(logrus.Fields{
			"http_method": SanitizeLog(r.Method),
			"path":        SanitizeLog(r.URL.Path),
			"tool":        mcpMethod,
			"args":        args,
			"source":      source,
			"duration":    duration,
			"in_bytes":    reqSize,
			"out_bytes":   lrw.bodySize,
			"outcome":     outcomeForStatus(lrw.statusCode),
			"status":      lrw.statusCode,
		})).Debug("mcp_request")
	})
}

// outcomeForStatus maps an HTTP status code to a coarse outcome label used in
// the per-request log line.
func outcomeForStatus(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "success"
	case code >= 400 && code < 500:
		return "client_error"
	case code >= 500:
		return "server_error"
	default:
		return "other"
	}
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

		if len(token) > MaxTokenLength {
			http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
			return
		}

		ctx := WithClient(r.Context(), token)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
