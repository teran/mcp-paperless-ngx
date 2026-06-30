package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teran/mcp-paperless-ngx/application"
	"github.com/teran/mcp-paperless-ngx/handlers"
	infra "github.com/teran/mcp-paperless-ngx/infrastructure/paperless"
)

// Context keys for dependency injection.
type (
	docServiceCtxKey  struct{}
	corrServiceCtxKey struct{}
	tagServiceCtxKey  struct{}
)

func contextWithServices(ctx context.Context, docSvc *application.DocumentService, corrSvc *application.CorrespondentService, tagSvc *application.TagService) context.Context {
	ctx = context.WithValue(ctx, docServiceCtxKey{}, docSvc)
	ctx = context.WithValue(ctx, corrServiceCtxKey{}, corrSvc)
	ctx = context.WithValue(ctx, tagServiceCtxKey{}, tagSvc)
	return ctx
}

func docServiceFromContext(ctx context.Context) *application.DocumentService {
	v, _ := ctx.Value(docServiceCtxKey{}).(*application.DocumentService)
	return v
}

func corrServiceFromContext(ctx context.Context) *application.CorrespondentService {
	v, _ := ctx.Value(corrServiceCtxKey{}).(*application.CorrespondentService)
	return v
}

func tagServiceFromContext(ctx context.Context) *application.TagService {
	v, _ := ctx.Value(tagServiceCtxKey{}).(*application.TagService)
	return v
}

var errTokenVerification = errors.New("token verification failed")

func main() {
	paperlessURL := os.Getenv("PAPERLESS_URL")
	if paperlessURL == "" {
		log.Fatal("PAPERLESS_URL environment variable is required")
	}
	paperlessURL = strings.TrimRight(paperlessURL, "/")

	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":8080"
	}

	// Create the MCP server instance.
	srv := mcp.NewServer(&mcp.Implementation{ //nolint:exhaustruct
		Name:    "mcp-paperless-ngx",
		Version: "v1.0.0",
	}, &mcp.ServerOptions{ //nolint:exhaustruct
		Capabilities: &mcp.ServerCapabilities{ //nolint:exhaustruct
			Tools: &mcp.ToolCapabilities{ListChanged: false},
		},
	})

	// Register tools via handler factories.
	registerTools(srv)

	// Create the Streamable HTTP handler.
	mcpHandler := mcp.NewStreamableHTTPHandler(
		func(r *http.Request) *mcp.Server {
			return srv
		},
		&mcp.StreamableHTTPOptions{ //nolint:exhaustruct
			Stateless: true,
		},
	)

	// Wrap with token extraction middleware, then with client injection.
	handler := handlers.TokenMiddleware(injectClientMiddleware(paperlessURL)(mcpHandler))

	//nolint:gosec // env vars are server-side config
	log.Printf("Starting mcp-paperless-ngx server on %s", sanitizeLog(listenAddr))
	//nolint:gosec // env vars are server-side config
	log.Printf("Paperless-ngx URL: %s", sanitizeLog(paperlessURL))

	httpServer := &http.Server{ //nolint:exhaustruct
		Addr:              listenAddr,
		Handler:           handler,
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      0, // streaming MCP responses
		IdleTimeout:       120 * time.Second,
	}

	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// registerTools registers all MCP tools on the server.
// Handlers extract services from context at runtime.
func registerTools(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{ //nolint:exhaustruct
		Name:        "search_documents",
		Description: "Search documents with filters (query, correspondent, tags, date range).",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in handlers.SearchDocumentsInput) (*mcp.CallToolResult, handlers.SearchDocumentsOutput, error) {
		return handlers.NewSearchDocumentsHandler(docServiceFromContext(ctx))(ctx, nil, in)
	})

	mcp.AddTool(s, &mcp.Tool{ //nolint:exhaustruct
		Name:        "get_document_content",
		Description: "Get full OCR text and metadata of a document.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in handlers.GetDocumentContentInput) (*mcp.CallToolResult, handlers.DocumentDetail, error) {
		return handlers.NewGetDocumentContentHandler(docServiceFromContext(ctx))(ctx, nil, in)
	})

	mcp.AddTool(s, &mcp.Tool{ //nolint:exhaustruct
		Name:        "search_correspondents",
		Description: "Search correspondents by name.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in handlers.SearchCorrespondentsInput) (*mcp.CallToolResult, handlers.SearchCorrespondentsOutput, error) {
		return handlers.NewSearchCorrespondentsHandler(corrServiceFromContext(ctx))(ctx, nil, in)
	})

	mcp.AddTool(s, &mcp.Tool{ //nolint:exhaustruct
		Name:        "get_documents_by_correspondent",
		Description: "List documents for a correspondent.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in handlers.GetDocumentsByCorrespondentInput) (*mcp.CallToolResult, handlers.SearchDocumentsOutput, error) {
		return handlers.NewGetDocumentsByCorrespondentHandler(docServiceFromContext(ctx))(ctx, nil, in)
	})

	mcp.AddTool(s, &mcp.Tool{ //nolint:exhaustruct
		Name:        "list_tags",
		Description: "List all tags.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in handlers.ListTagsInput) (*mcp.CallToolResult, handlers.ListTagsOutput, error) {
		return handlers.NewListTagsHandler(tagServiceFromContext(ctx))(ctx, nil, in)
	})

	mcp.AddTool(s, &mcp.Tool{ //nolint:exhaustruct
		Name:        "get_documents_by_tag",
		Description: "List documents for a tag.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in handlers.GetDocumentsByTagInput) (*mcp.CallToolResult, handlers.SearchDocumentsOutput, error) {
		return handlers.NewGetDocumentsByTagHandler(docServiceFromContext(ctx))(ctx, nil, in)
	})

	mcp.AddTool(s, &mcp.Tool{ //nolint:exhaustruct
		Name:        "fulltext_search",
		Description: "Full-text search across all documents.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in handlers.FulltextSearchInput) (*mcp.CallToolResult, handlers.FulltextSearchOutput, error) {
		return handlers.NewFulltextSearchHandler(docServiceFromContext(ctx))(ctx, nil, in)
	})
}

// injectClientMiddleware creates the Paperless-ngx client and attaches
// application services to the context.
func injectClientMiddleware(paperlessURL string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw, _ := handlers.ClientFromContext(r.Context()).(string)
			if raw == "" {
				http.Error(w, "Missing token in context", http.StatusUnauthorized)
				return
			}

			client := infra.NewClient(paperlessURL, raw)

			// Verify the token before proceeding.
			if err := verifyToken(r.Context(), client); err != nil {
				log.Printf("Token verification failed: %v", err)
				http.Error(w, fmt.Sprintf("Token verification failed: %v", err), http.StatusUnauthorized)
				return
			}

			// Build application services using adapters and store in context.
			docSvc := application.NewDocumentService(client)
			corrSvc := application.NewCorrespondentService(infra.NewCorrespondentRepo(client))
			tagSvc := application.NewTagService(infra.NewTagRepo(client))

			ctx := r.Context()
			ctx = contextWithServices(ctx, docSvc, corrSvc, tagSvc)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// sanitizeLog removes newlines and truncates long strings.
func sanitizeLog(s string) string {
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	if len(s) > 500 {
		s = s[:500] + "..."
	}
	return s
}

// verifyToken checks if the token is valid.
func verifyToken(ctx context.Context, client *infra.Client) error {
	_, err := client.SearchCorrespondents(ctx, "", 1, 1)
	if err != nil {
		return fmt.Errorf("%w: %w", errTokenVerification, err)
	}
	return nil
}
