package domain

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestRequestIDContext(t *testing.T) {
	t.Parallel()

	t.Run("round-trips a request id", func(t *testing.T) {
		t.Parallel()

		ctx := WithRequestID(context.Background(), "req-123")
		if got := RequestIDFromContext(ctx); got != "req-123" {
			t.Errorf("RequestIDFromContext = %q, want %q", got, "req-123")
		}
	})

	t.Run("returns empty when not set", func(t *testing.T) {
		t.Parallel()

		if got := RequestIDFromContext(context.Background()); got != "" {
			t.Errorf("RequestIDFromContext = %q, want empty", got)
		}
	})

	t.Run("overwrites existing id", func(t *testing.T) {
		t.Parallel()

		ctx := WithRequestID(context.Background(), "first")
		ctx = WithRequestID(ctx, "second")
		if got := RequestIDFromContext(ctx); got != "second" {
			t.Errorf("RequestIDFromContext = %q, want %q", got, "second")
		}
	})
}

func TestSessionIDContext(t *testing.T) {
	t.Parallel()

	t.Run("round-trips a session id", func(t *testing.T) {
		t.Parallel()

		ctx := WithSessionID(context.Background(), "sess-1")
		if got := SessionIDFromContext(ctx); got != "sess-1" {
			t.Errorf("SessionIDFromContext = %q, want %q", got, "sess-1")
		}
	})

	t.Run("returns empty when not set", func(t *testing.T) {
		t.Parallel()

		if got := SessionIDFromContext(context.Background()); got != "" {
			t.Errorf("SessionIDFromContext = %q, want empty", got)
		}
	})
}

func TestWithSession(t *testing.T) {
	t.Parallel()

	entry := logrus.NewEntry(logrus.New())

	t.Run("adds request_id and session_id fields", func(t *testing.T) {
		t.Parallel()

		ctx := WithSessionID(WithRequestID(context.Background(), "req-9"), "sess-9")
		got := WithSession(ctx, entry)

		if got.Data["request_id"] != "req-9" {
			t.Errorf("request_id field = %v, want req-9", got.Data["request_id"])
		}
		if got.Data["session_id"] != "sess-9" {
			t.Errorf("session_id field = %v, want sess-9", got.Data["session_id"])
		}
	})

	t.Run("adds only request_id when no session", func(t *testing.T) {
		t.Parallel()

		got := WithSession(WithRequestID(context.Background(), "req-only"), entry)

		if got.Data["request_id"] != "req-only" {
			t.Errorf("request_id field = %v, want req-only", got.Data["request_id"])
		}
		if _, ok := got.Data["session_id"]; ok {
			t.Error("session_id field should be absent")
		}
	})

	t.Run("adds nothing when no ids in context", func(t *testing.T) {
		t.Parallel()

		got := WithSession(context.Background(), entry)
		if _, ok := got.Data["request_id"]; ok {
			t.Error("request_id field should be absent")
		}
	})
}
