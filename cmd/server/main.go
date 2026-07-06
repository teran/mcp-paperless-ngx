package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/time/rate"

	"github.com/teran/mcp-paperless-ngx/application"
	"github.com/teran/mcp-paperless-ngx/config"
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
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

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

	// Create Prometheus registry and metrics collectors.
	promRegistry := prometheus.NewRegistry()
	metrics := handlers.NewMetrics(promRegistry)

	// Register tools via handler factories.
	handlers.RegisterTools(srv, metrics)

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
	// recovery → metrics → rate limit → body limit → logging → token → client injection → MCP handler.
	// RecoveryMiddleware is outermost so that any panic anywhere in the chain
	// is caught and the server stays alive.
	// MetricsMiddleware tracks the active-requests gauge only (no body reads).
	// RateLimitMiddleware is third because it is the cheapest check (no body reading).
	// BodyLimitMiddleware bounds the body for everything after it.
	handler := handlers.RecoveryMiddleware(
		handlers.MetricsMiddleware(metrics)(
			handlers.RateLimitMiddleware(handlers.RateLimiterConfig{
				GlobalLimit:    rate.Limit(cfg.RateLimitGlobal),
				GlobalBurst:    cfg.RateLimitGlobal * 2,
				PerClientLimit: rate.Limit(cfg.RateLimitPerClient),
				PerClientBurst: cfg.RateLimitPerClient * 2,
			})(
				handlers.BodyLimitMiddleware(handlers.DefaultMaxRequestBodySize)(
					handlers.LoggingMiddleware(
						handlers.TokenMiddleware(
							injectClientMiddleware(cfg.PaperlessURL, sharedHTTPClient)(mcpHandler),
						),
					),
				),
			),
		),
	)

	// Health-check endpoint — bypasses all middleware (auth, rate limit, etc.)
	// so that load balancers and orchestrators always get a 200 when the server
	// is alive, regardless of token state.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.Handle("/", handler)

	log.Printf("Paperless-ngx URL: %s", handlers.SanitizeLog(cfg.PaperlessURL))
	log.Printf("Version: %s, commit: %s, built: %s", version, commit, date)

	// ---- Main MCP HTTP server ----
	mainServer := &http.Server{ //nolint:exhaustruct
		Addr:              cfg.ListenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       120 * time.Second,
	}

	// ---- Metrics HTTP server ----
	metricsHandler := handlers.RegisterMetricsOnRegistry(promRegistry)
	metricsMux := http.NewServeMux()
	metricsMux.Handle("GET /metrics", metricsHandler)

	metricsServer := &http.Server{ //nolint:exhaustruct
		Addr:              cfg.PrometheusMetricsAddr,
		Handler:           metricsMux,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Channel to capture server errors (buffered to hold both if both fail).
	errCh := make(chan error, 2)

	go func() {
		log.Printf("Starting mcp-paperless-ngx server on %s", handlers.SanitizeLog(cfg.ListenAddr))
		if err := mainServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	go func() {
		log.Printf("Starting Prometheus metrics server on %s", handlers.SanitizeLog(cfg.PrometheusMetricsAddr))
		if err := metricsServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
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
	defer cancel()

	// Shut down both servers in order.
	if err := mainServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("Main server shutdown error: %v", err)
	}
	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("Metrics server shutdown error: %v", err)
	}

	log.Println("Server stopped gracefully")
}

// injectClientMiddleware creates the Paperless-ngx client and attaches
// application services to the context.
func injectClientMiddleware(paperlessURL string, sharedHTTPClient *http.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := handlers.ClientFromContext(r.Context())
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
