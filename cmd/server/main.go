package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/time/rate"

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

	rateLimitGlobal := parseRateLimit(os.Getenv("RATE_LIMIT_GLOBAL"), 100)
	rateLimitPerClient := parseRateLimit(os.Getenv("RATE_LIMIT_PER_CLIENT"), 10)

	// sharedHTTPClient is reused across requests for connection pooling.
	// CheckRedirect is set to http.ErrUseLastResponse to prevent credential
	// forwarding — the http.Client never follows redirects, so the token
	// cannot be leaked to an external URL via a 302 response from Paperless-ngx.
	sharedHTTPClient := &http.Client{ //nolint:exhaustruct
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: &http.Transport{ //nolint:exhaustruct
			MaxIdleConns:       100,
			IdleConnTimeout:    90 * time.Second,
			DisableCompression: false,
			DisableKeepAlives:  false,
		},
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

	// Wrap with middlewares (outermost to innermost):
	// rate limit → body limit → logging → token extraction → client injection → MCP handler.
	// RateLimitMiddleware is outermost because it is the cheapest check (no body reading).
	// BodyLimitMiddleware is second so that io.ReadAll in LoggingMiddleware
	// is bounded by the 1 MB limit — a large malicious body is rejected before
	// the logging middleware reads it into memory.
	handler := handlers.RateLimitMiddleware(handlers.RateLimiterConfig{
		GlobalLimit:    rate.Limit(rateLimitGlobal),
		GlobalBurst:    rateLimitGlobal * 2,
		PerClientLimit: rate.Limit(rateLimitPerClient),
		PerClientBurst: rateLimitPerClient * 2,
	})(
		handlers.BodyLimitMiddleware(handlers.DefaultMaxRequestBodySize)(
			handlers.LoggingMiddleware(
				handlers.TokenMiddleware(
					injectClientMiddleware(paperlessURL, sharedHTTPClient)(mcpHandler),
				),
			),
		),
	)

	//nolint:gosec // env vars are server-side config
	log.Printf("Paperless-ngx URL: %s", handlers.SanitizeLog(paperlessURL))
	log.Printf("Version: %s, commit: %s, built: %s", version, commit, date)

	writeTimeout := parseDurationSeconds(os.Getenv("WRITE_TIMEOUT"), 300)

	httpServer := &http.Server{ //nolint:exhaustruct
		Addr:              listenAddr,
		Handler:           handler,
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       120 * time.Second,
	}

	// Channel to capture server errors.
	errCh := make(chan error, 1)
	go func() {
		log.Printf("Starting mcp-paperless-ngx server on %s", handlers.SanitizeLog(listenAddr)) //nolint:gosec // sanitized above
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
func injectClientMiddleware(paperlessURL string, sharedHTTPClient *http.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw, _ := handlers.ClientFromContext(r.Context()).(string)
			if raw == "" {
				http.Error(w, "Missing token in context", http.StatusUnauthorized)
				return
			}

			client := infra.NewClient(paperlessURL, raw, sharedHTTPClient)

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

// parseRateLimit parses an integer rate limit from an environment variable.
// Returns the default value if the env var is empty or invalid.
func parseRateLimit(val string, defaultVal int) int {
	if val == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(val)
	if err != nil || n <= 0 {
		return defaultVal
	}
	return n
}

// parseDurationSeconds parses an integer number of seconds from an
// environment variable. Returns the default value if the env var is empty
// or invalid. A value of 0 disables the timeout for streaming use cases.
func parseDurationSeconds(val string, defaultVal int) time.Duration {
	if val == "" {
		return time.Duration(defaultVal) * time.Second
	}
	n, err := strconv.Atoi(val)
	if err != nil || n < 0 {
		return time.Duration(defaultVal) * time.Second
	}
	return time.Duration(n) * time.Second
}

// sanitizeLog is deprecated; use handlers.SanitizeLog instead.
var sanitizeLog = handlers.SanitizeLog
