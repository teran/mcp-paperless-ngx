package handlers

import (
	"context"
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
