package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"
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
