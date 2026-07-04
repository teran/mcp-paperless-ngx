package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teran/mcp-paperless-ngx/application"
	"github.com/teran/mcp-paperless-ngx/handlers"
	infra "github.com/teran/mcp-paperless-ngx/infrastructure/paperless"
)

// Build-time variables injected by goreleaser (via ldflags).
var (
	version = "dev"
	commit  = "none"    //nolint:gochecknoglobals
	date    = "unknown" //nolint:gochecknoglobals
)

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
		Version: version,
	}, &mcp.ServerOptions{ //nolint:exhaustruct
		Capabilities: &mcp.ServerCapabilities{ //nolint:exhaustruct
			Tools: &mcp.ToolCapabilities{ListChanged: false},
		},
	})

	// Register tools via handler factories.
	handlers.RegisterTools(srv)

	// Create the Streamable HTTP handler.
	mcpHandler := mcp.NewStreamableHTTPHandler(
		func(r *http.Request) *mcp.Server {
			return srv
		},
		&mcp.StreamableHTTPOptions{ //nolint:exhaustruct
			Stateless: true,
		},
	)

	// Wrap with middlewares: body limit → token extraction → client injection.
	handler := handlers.TokenMiddleware(
		handlers.BodyLimitMiddleware(handlers.DefaultMaxRequestBodySize)(
			injectClientMiddleware(paperlessURL)(mcpHandler),
		),
	)

	//nolint:gosec // env vars are server-side config
	log.Printf("Paperless-ngx URL: %s", sanitizeLog(paperlessURL))
	log.Printf("Version: %s, commit: %s, built: %s", version, commit, date)

	httpServer := &http.Server{ //nolint:exhaustruct
		Addr:              listenAddr,
		Handler:           handler,
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      0, // streaming MCP responses
		IdleTimeout:       120 * time.Second,
	}

	// Channel to capture server errors.
	errCh := make(chan error, 1)
	go func() {
		log.Printf("Starting mcp-paperless-ngx server on %s", sanitizeLog(listenAddr))
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	// Wait for SIGTERM or SIGINT for graceful shutdown.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	select {
	case sig := <-quit:
		log.Printf("Received signal %v, shutting down...", sig)
	case err := <-errCh:
		log.Printf("Server error: %v", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		cancel()
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	cancel()

	log.Println("Server stopped gracefully")
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

			// Build application services using adapters and store in context.
			docSvc := application.NewDocumentService(client)
			corrSvc := application.NewCorrespondentService(infra.NewCorrespondentRepo(client))
			docTypeSvc := application.NewDocumentTypeService(infra.NewDocumentTypeRepo(client))
			tagSvc := application.NewTagService(infra.NewTagRepo(client))

			ctx := r.Context()
			ctx = handlers.ContextWithServices(ctx, docSvc, corrSvc, docTypeSvc, tagSvc)
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
