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
	"resty.dev/v3"

	"github.com/teran/mcp-paperless-ngx/application"
	"github.com/teran/mcp-paperless-ngx/config"
	"github.com/teran/mcp-paperless-ngx/domain"
	"github.com/teran/mcp-paperless-ngx/handlers"
	infra "github.com/teran/mcp-paperless-ngx/infrastructure/paperless"
)

// Build-time variables injected by goreleaser (via ldflags). These follow the
// B2 convention: appName, appVersion, appCommitHash, appTimestamp.
var (
	appName       = "mcp-paperless-ngx" //nolint:gochecknoglobals
	appVersion    = "dev"               //nolint:gochecknoglobals
	appCommitHash = "none"              //nolint:gochecknoglobals
	appTimestamp  = "unknown"           //nolint:gochecknoglobals
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

// newSharedHTTPClient creates the shared resty client reused across requests.
// RedirectNoPolicy is set to prevent credential forwarding — the client never
// follows redirects, so the token cannot be leaked to an external URL via a 302
// response from Paperless-ngx. Timeouts, retries and connection pooling are
// configured explicitly here (see G9).
func newSharedHTTPClient() *resty.Client {
	return resty.New().
		SetTimeout(30 * time.Second).
		SetRetryCount(2).
		SetRetryWaitTime(200 * time.Millisecond).
		SetRedirectPolicy(resty.RedirectNoPolicy()).
		SetTransport(&http.Transport{ //nolint:exhaustruct
			MaxIdleConns:       100,
			IdleConnTimeout:    90 * time.Second,
			DisableCompression: false,
			DisableKeepAlives:  false,
		})
}

// serverHandlers bundles the HTTP handlers for the main MCP server and the
// Prometheus metrics server.
type serverHandlers struct {
	main    http.Handler
	metrics http.Handler
}

// buildHandlers constructs the main MCP handler (including the full middleware
// chain and the health-check endpoint) and the Prometheus metrics handler.
func buildHandlers(cfg *config.Config, sharedHTTPClient *resty.Client) serverHandlers {
	// Create the MCP server instance. When logging is enabled, wire the SDK's
	// slog logger into logrus (L7/N25) so SDK-level events are visible.
	sdkLogger := slogAdapter()
	srv := mcp.NewServer(&mcp.Implementation{ //nolint:exhaustruct
		Name:    appName,
		Version: appVersion,
	}, &mcp.ServerOptions{ //nolint:exhaustruct
		Capabilities: &mcp.ServerCapabilities{ //nolint:exhaustruct
			Tools: &mcp.ToolCapabilities{ListChanged: false},
		},
		Logger: sdkLogger,
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
	// recovery → request-id → metrics → rate limit → body limit → logging → token → client injection → MCP handler.
	// RecoveryMiddleware is outermost so that any panic anywhere in the chain
	// is caught and the server stays alive.
	// RequestIDMiddleware generates a per-request request_id (reusing an inbound
	// X-Request-ID) and threads it through the context (L9/G11).
	// MetricsMiddleware tracks the active-requests gauge only (no body reads).
	// RateLimitMiddleware is third because it is the cheapest check (no body reading).
	// BodyLimitMiddleware bounds the body for everything after it.
	handler := handlers.RecoveryMiddleware(
		handlers.RequestIDMiddleware(
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
	// Configure logging per L2-L4: gated by LOG_LEVEL (disabled by default),
	// LOG_FILENAME override, LOG_FORMAT=json. When disabled, logrus is a no-op.
	if err := setupLogging(cfg); err != nil {
		return err
	}

	// B5/L6: when logging is enabled, the startup banner is the very first log
	// line, using the build metadata embedded at build time (B2).
	if cfg.LogLevel != "" {
		logrus.Infof("Starting %s/%s (commit: %s; built at %s)",
			appName, appVersion, appCommitHash, appTimestamp)
	}

	sharedHTTPClient := newSharedHTTPClient()

	h := buildHandlers(cfg, sharedHTTPClient)
	mainServer, metricsServer := buildServers(cfg, h)

	logrus.WithField("url", handlers.SanitizeLog(cfg.PaperlessURL)).Info("paperless-ngx url")
	logrus.WithFields(logrus.Fields{
		"appName":      appName,
		"appVersion":   appVersion,
		"appCommit":    appCommitHash,
		"appTimestamp": appTimestamp,
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
func injectClientMiddleware(paperlessURL string, sharedHTTPClient *resty.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := handlers.ClientFromContext(r.Context())
			if raw == "" {
				http.Error(w, "Missing token in context", http.StatusUnauthorized)
				return
			}

			client := infra.NewClient(paperlessURL, raw, sharedHTTPClient)
			// Tag outbound client logs with the request_id/session_id from the
			// request context (L9/G11).
			client.SetLogger(domain.WithSession(r.Context(), logrus.NewEntry(logrus.StandardLogger())))

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
