package server

import (
	"context"

	"github.com/teran/mcp-paperless-ngx/internal/paperless"
)

type paperlessClientKey struct{}

// WithClient stores a Paperless-ngx client in the context.
func WithClient(ctx context.Context, client *paperless.Client) context.Context {
	return context.WithValue(ctx, paperlessClientKey{}, client)
}

// ClientFromContext retrieves the Paperless-ngx client from the context.
// Returns nil if no client is present.
func ClientFromContext(ctx context.Context) *paperless.Client {
	client, ok := ctx.Value(paperlessClientKey{}).(*paperless.Client)
	if !ok {
		return nil
	}
	return client
}
