package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
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
		logrus.WithError(err).Fatal("failed to load configuration")
	}

	if err := run(cfg); err != nil {
		logrus.WithError(err).Fatal("failed to run server")
	}
}

// newSharedHTTPClient creates the shared HTTP client reused across requests.
// CheckRedirect is set to http.ErrUseLastResponse to prevent credential
// forwarding — the http.Client never follows redirects, so the token
// cannot be leaked to an external URL via a 302 response from Paperless-ngx.
func newSharedHTTPClient() *http.Client {
	return &http.Client{ //nolint:exhaustruct
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
}

// serverHandlers bundles the HTTP handlers for the main MCP server and the
// Prometheus metrics server.
type serverHandlers struct {
	main    http.Handler
	metrics http.Handler
}

// buildHandlers constructs the main MCP handler (including the full middleware
// chain and the health-check endpoint) and the Prometheus metrics handler.
func buildHandlers(cfg *config.Config, sharedHTTPClient *http.Client) serverHandlers {
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

	metricsHandler := handlers.RegisterMetricsOnRegistry(promRegistry)

	return serverHandlers{main: mux, metrics: metricsHandler}
}

// buildServers constructs the main MCP HTTP server and the Prometheus metrics
// HTTP server from the given handlers.
func buildServers(cfg *config.Config, h serverHandlers) (*http.Server, *http.Server) {
	mainServer := &http.Server{ //nolint:exhaustruct
		Addr:              cfg.ListenAddr,
		Handler:           h.main,
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       120 * time.Second,
	}

	metricsMux := http.NewServeMux()
	metricsMux.Handle("GET /metrics", h.metrics)

	metricsServer := &http.Server{ //nolint:exhaustruct
		Addr:              cfg.PrometheusMetricsAddr,
		Handler:           metricsMux,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return mainServer, metricsServer
}

// run wires the shared HTTP client, builds the servers, starts them, and blocks
// until a shutdown signal or a fatal server error is received, then shuts both
// servers down gracefully.
func run(cfg *config.Config) error {
	sharedHTTPClient := newSharedHTTPClient()

	h := buildHandlers(cfg, sharedHTTPClient)
	mainServer, metricsServer := buildServers(cfg, h)

	logrus.WithField("url", handlers.SanitizeLog(cfg.PaperlessURL)).Info("paperless-ngx url")
	logrus.WithFields(logrus.Fields{
		"version": version,
		"commit":  commit,
		"built":   date,
	}).Info("server version")

	// Channel to capture server errors (buffered to hold both if both fail).
	errCh := make(chan error, 2)

	go func() {
		logrus.WithField("addr", handlers.SanitizeLog(cfg.ListenAddr)).Info("starting mcp-paperless-ngx server")
		if err := mainServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	go func() {
		logrus.WithField("addr", handlers.SanitizeLog(cfg.PrometheusMetricsAddr)).Info("starting prometheus metrics server")
		if err := metricsServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	// Wait for SIGTERM or SIGINT for graceful shutdown.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	select {
	case sig := <-quit:
		logrus.WithField("signal", sig).Info("received signal, shutting down")
	case err := <-errCh:
		logrus.WithError(err).Error("server error")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shut down both servers in order.
	if err := mainServer.Shutdown(shutdownCtx); err != nil {
		logrus.WithError(err).Error("main server shutdown error")
	}
	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		logrus.WithError(err).Error("metrics server shutdown error")
	}

	logrus.Info("server stopped gracefully")
	return nil
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
