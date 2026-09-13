package domain

import (
	"context"

	"github.com/sirupsen/logrus"
)

type (
	requestIDCtxKey struct{}
	sessionIDCtxKey struct{}
)

// WithRequestID returns a copy of ctx carrying the given request_id. It is
// used to thread a per-request correlation id through the context so that any
// component handling the request can read it without threading it through
// signatures (see RequestIDFromContext).
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDCtxKey{}, id)
}

// RequestIDFromContext returns the request_id stored in ctx, or "" if none.
func RequestIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(requestIDCtxKey{}).(string)
	return v
}

// WithSessionID returns a copy of ctx carrying the given session_id (e.g. the
// MCP session identifier received from the client).
func WithSessionID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, sessionIDCtxKey{}, id)
}

// SessionIDFromContext returns the session_id stored in ctx, or "" if none.
func SessionIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(sessionIDCtxKey{}).(string)
	return v
}

// WithSession decorates a logrus entry with the request-scoped correlation
// fields (request_id and, when present, session_id) read from ctx. It is used
// as the base for every request-scoped log line so that a single request's
// logs are correlatable.
func WithSession(ctx context.Context, entry *logrus.Entry) *logrus.Entry {
	if id := RequestIDFromContext(ctx); id != "" {
		entry = entry.WithField("request_id", id)
	}
	if sid := SessionIDFromContext(ctx); sid != "" {
		entry = entry.WithField("session_id", sid)
	}
	return entry
}
