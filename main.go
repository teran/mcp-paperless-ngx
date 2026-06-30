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

	"github.com/teran/mcp-paperless-ngx/internal/paperless"
	"github.com/teran/mcp-paperless-ngx/internal/server"
)

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

	// Register all tools.
	server.RegisterTools(srv)

	// Create the Streamable HTTP handler.
	mcpHandler := mcp.NewStreamableHTTPHandler(
		func(r *http.Request) *mcp.Server {
			return srv
		},
		&mcp.StreamableHTTPOptions{ //nolint:exhaustruct
			Stateless: true,
		},
	)

	// Wrap with token extraction middleware.
	handler := tokenMiddleware(mcpHandler, paperlessURL)

	//nolint:gosec // log injection not applicable — env vars are server-side config
	log.Printf("Starting mcp-paperless-ngx server on %s", sanitizeLog(listenAddr))
	//nolint:gosec // log injection not applicable — env vars are server-side config
	log.Printf("Paperless-ngx URL: %s", sanitizeLog(paperlessURL))

	server := &http.Server{ //nolint:exhaustruct
		Addr:              listenAddr,
		Handler:           handler,
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      0, // streaming MCP responses
		IdleTimeout:       120 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// sanitizeLog removes newlines and truncates long strings for safe logging.
func sanitizeLog(s string) string {
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	if len(s) > 500 {
		s = s[:500] + "..."
	}
	return s
}

// tokenMiddleware extracts the bearer token from the Authorization header,
// creates a Paperless-ngx client, and stores it in the request context.
//
// The client passes the token to Paperless-ngx using the "Token" auth scheme
// (as required by the Paperless-ngx API). The MCP client must send the token
// as "Bearer <token>" in the Authorization header.
func tokenMiddleware(next http.Handler, paperlessURL string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
			return
		}

		// Support both "Bearer <token>" and "Token <token>" schemes.
		var token string
		if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			token = strings.TrimSpace(authHeader[len("bearer "):])
		} else if strings.HasPrefix(strings.ToLower(authHeader), "token ") {
			token = strings.TrimSpace(authHeader[len("token "):])
		}

		if token == "" {
			http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
			return
		}

		client := paperless.NewClient(paperlessURL, token)

		// Sanity check: verify the token by calling a simple API endpoint.
		if err := verifyToken(r.Context(), client); err != nil {
			log.Printf("Token verification failed: %v", err)
			http.Error(w, fmt.Sprintf("Token verification failed: %v", err), http.StatusUnauthorized)
			return
		}

		ctx := server.WithClient(r.Context(), client)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// verifyToken checks if the token is valid by calling the correspondents endpoint.
func verifyToken(ctx context.Context, client *paperless.Client) error {
	_, err := client.SearchCorrespondents(ctx, "", 1, 1)
	if err != nil {
		return fmt.Errorf("%w: %w", errTokenVerification, err)
	}
	return nil
}
