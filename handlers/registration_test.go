package handlers

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teran/mcp-paperless-ngx/application"
)

// ---------------------------------------------------------------------------
// *FromContext tests
// ---------------------------------------------------------------------------

func TestContextWithServices(t *testing.T) {
	t.Parallel()

	t.Run("stores and retrieves all services", func(t *testing.T) {
		ctx := context.Background()

		docSvc := application.NewDocumentService(&mockDocRepo{}) //nolint:exhaustruct
		corrSvc := newTestCorrSvc()
		docTypeSvc := newTestDocTypeSvc()
		tagSvc := application.NewTagService(&mockTagRepo{}) //nolint:exhaustruct

		ctx = ContextWithServices(ctx, docSvc, corrSvc, docTypeSvc, tagSvc)

		if got := DocServiceFromContext(ctx); got != docSvc {
			t.Error("DocServiceFromContext returned unexpected value")
		}
		if got := CorrServiceFromContext(ctx); got != corrSvc {
			t.Error("CorrServiceFromContext returned unexpected value")
		}
		if got := DocTypeServiceFromContext(ctx); got != docTypeSvc {
			t.Error("DocTypeServiceFromContext returned unexpected value")
		}
		if got := TagServiceFromContext(ctx); got != tagSvc {
			t.Error("TagServiceFromContext returned unexpected value")
		}
	})
}

func TestDocServiceFromContext(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when not set", func(t *testing.T) {
		if got := DocServiceFromContext(context.Background()); got != nil {
			t.Error("expected nil from empty context")
		}
	})

	t.Run("returns nil when wrong type stored", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), docServiceCtxKey{}, "not a service")
		if got := DocServiceFromContext(ctx); got != nil {
			t.Error("expected nil for wrong type")
		}
	})
}

func TestCorrServiceFromContext(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when not set", func(t *testing.T) {
		if got := CorrServiceFromContext(context.Background()); got != nil {
			t.Error("expected nil from empty context")
		}
	})

	t.Run("returns nil when wrong type stored", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), corrServiceCtxKey{}, "not a service")
		if got := CorrServiceFromContext(ctx); got != nil {
			t.Error("expected nil for wrong type")
		}
	})
}

func TestDocTypeServiceFromContext(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when not set", func(t *testing.T) {
		if got := DocTypeServiceFromContext(context.Background()); got != nil {
			t.Error("expected nil from empty context")
		}
	})

	t.Run("returns nil when wrong type stored", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), docTypeServiceCtxKey{}, "not a service")
		if got := DocTypeServiceFromContext(ctx); got != nil {
			t.Error("expected nil for wrong type")
		}
	})
}

func TestTagServiceFromContext(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when not set", func(t *testing.T) {
		if got := TagServiceFromContext(context.Background()); got != nil {
			t.Error("expected nil from empty context")
		}
	})

	t.Run("returns nil when wrong type stored", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), tagServiceCtxKey{}, "not a service")
		if got := TagServiceFromContext(ctx); got != nil {
			t.Error("expected nil for wrong type")
		}
	})
}

// ---------------------------------------------------------------------------
// RegisterTools tests
// ---------------------------------------------------------------------------

func TestRegisterTools(t *testing.T) {
	t.Parallel()

	srv := mcp.NewServer(&mcp.Implementation{ //nolint:exhaustruct
		Name:    "mcp-paperless-ngx-test",
		Version: "test",
	}, &mcp.ServerOptions{ //nolint:exhaustruct
		Capabilities: &mcp.ServerCapabilities{ //nolint:exhaustruct
			Tools: &mcp.ToolCapabilities{ListChanged: false},
		},
	})

	RegisterTools(srv)
}
