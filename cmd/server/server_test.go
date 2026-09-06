package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/teran/mcp-paperless-ngx/config"
)

func testConfig() *config.Config {
	return &config.Config{
		PaperlessURL:          "http://paperless:8000",
		ListenAddr:            "127.0.0.1:0",
		PrometheusMetricsAddr: "127.0.0.1:0",
		RateLimitGlobal:       100,
		RateLimitPerClient:    10,
		WriteTimeout:          30 * time.Second,
	}
}

// ---------------------------------------------------------------------------
// newSharedHTTPClient tests
// ---------------------------------------------------------------------------

func TestNewSharedHTTPClient(t *testing.T) {
	t.Parallel()

	client := newSharedHTTPClient()
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.Timeout != 30*time.Second {
		t.Errorf("expected 30s timeout, got %s", client.Timeout)
	}
	if client.CheckRedirect == nil {
		t.Fatal("expected CheckRedirect to be set")
	}
	if client.Transport == nil {
		t.Fatal("expected Transport to be set")
	}

	// Verify redirect protection: the client must not follow redirects.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://evil.example/steal", http.StatusFound)
	}))
	defer ts.Close()

	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, ts.URL, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusFound {
		t.Errorf("expected 302 (redirect not followed), got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// buildHandlers tests
// ---------------------------------------------------------------------------

func TestBuildHandlers_Healthz(t *testing.T) {
	t.Parallel()

	h := buildHandlers(testConfig(), newSharedHTTPClient())

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	h.main.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected application/json, got %q", ct)
	}
	if rr.Body.String() != `{"status":"ok"}` {
		t.Errorf("unexpected body %q", rr.Body.String())
	}
}

func TestBuildHandlers_MissingToken_Returns401(t *testing.T) {
	t.Parallel()

	h := buildHandlers(testConfig(), newSharedHTTPClient())

	// A request to the MCP route without an Authorization header must be
	// rejected by TokenMiddleware before reaching the MCP handler.
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(`{"jsonrpc":"2.0","method":"tools/list","id":1}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.main.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without token, got %d", rr.Code)
	}
}

func TestBuildHandlers_WithToken_ReachesMCP(t *testing.T) {
	t.Parallel()

	h := buildHandlers(testConfig(), newSharedHTTPClient())

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(`{"jsonrpc":"2.0","method":"tools/list","id":1}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()
	h.main.ServeHTTP(rr, req)

	// With a valid token the request should pass auth and reach the MCP
	// streamable handler, which returns 200 (not 401).
	if rr.Code == http.StatusUnauthorized {
		t.Fatal("expected request to pass auth, got 401")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected MCP handler to return 200, got %d", rr.Code)
	}

	// The MCP streamable handler responds with SSE (text/event-stream).
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
		t.Errorf("expected text/event-stream content type, got %q", ct)
	}
}

func TestBuildHandlers_MetricsHandler(t *testing.T) {
	t.Parallel()

	h := buildHandlers(testConfig(), newSharedHTTPClient())

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	h.metrics.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	// go_* runtime metrics (registered via the Go collector) are always present.
	if !strings.Contains(body, "go_goroutines") {
		t.Errorf("expected go_goroutines (Go collector) in metrics output")
	}
}

// ---------------------------------------------------------------------------
// buildServers tests
// ---------------------------------------------------------------------------

func TestBuildServers(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	h := buildHandlers(cfg, newSharedHTTPClient())
	mainServer, metricsServer := buildServers(cfg, h)

	if mainServer == nil || metricsServer == nil {
		t.Fatal("expected non-nil servers")
	}

	if mainServer.Addr != cfg.ListenAddr {
		t.Errorf("expected main Addr %q, got %q", cfg.ListenAddr, mainServer.Addr)
	}
	if mainServer.Handler == nil {
		t.Error("expected main server handler to be set")
	}
	if mainServer.WriteTimeout != cfg.WriteTimeout {
		t.Errorf("expected write timeout %s, got %s", cfg.WriteTimeout, mainServer.WriteTimeout)
	}
	if mainServer.ReadHeaderTimeout != 30*time.Second {
		t.Errorf("expected read header timeout 30s, got %s", mainServer.ReadHeaderTimeout)
	}

	if metricsServer.Addr != cfg.PrometheusMetricsAddr {
		t.Errorf("expected metrics Addr %q, got %q", cfg.PrometheusMetricsAddr, metricsServer.Addr)
	}
	if metricsServer.Handler == nil {
		t.Error("expected metrics server handler to be set")
	}

	// The metrics handler mux should expose GET /metrics.
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	metricsServer.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected /metrics to return 200, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// run tests
// ---------------------------------------------------------------------------

// TestRun_ShutsDownOnServerError verifies that run() exits and performs a
// graceful shutdown when the main server fails to start (e.g. an invalid
// listen address produces an immediate ListenAndServe error).
func TestRun_ShutsDownOnServerError(t *testing.T) {
	cfg := testConfig()
	cfg.ListenAddr = "invalid-addr" // causes ListenAndServe to fail immediately
	cfg.PrometheusMetricsAddr = "127.0.0.1:0"

	done := make(chan error, 1)
	go func() {
		done <- run(cfg)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run returned unexpected error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("run did not exit after server error")
	}
}

// TestRun_ShutsDownOnMetricsError verifies that run() exits and performs a
// graceful shutdown when the metrics server fails to start.
func TestRun_ShutsDownOnMetricsError(t *testing.T) {
	cfg := testConfig()
	cfg.ListenAddr = "127.0.0.1:0"
	cfg.PrometheusMetricsAddr = "invalid-addr" // causes ListenAndServe to fail immediately

	done := make(chan error, 1)
	go func() {
		done <- run(cfg)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run returned unexpected error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("run did not exit after metrics server error")
	}
}

// TestRun_ShutsDownOnSignal verifies that run() exits and performs a graceful
// shutdown when a SIGINT is received.
func TestRun_ShutsDownOnSignal(t *testing.T) {
	if os.Getenv("MCP_SKIP_SIGNAL_TEST") != "" {
		t.Skip("signal test disabled")
	}

	cfg := testConfig()

	done := make(chan error, 1)
	go func() {
		done <- run(cfg)
	}()

	// Give the servers time to start and signal.Notify to register.
	time.Sleep(300 * time.Millisecond)

	if err := syscall.Kill(os.Getpid(), syscall.SIGINT); err != nil {
		t.Fatalf("failed to send SIGINT: %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run returned unexpected error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("run did not exit after signal")
	}
}

// TestRun_InvalidConfig_DoesNotCrash ensures run never panics and returns nil
// even when the config is empty (servers fail to bind but run handles it).
func TestRun_EmptyConfig(t *testing.T) {
	cfg := &config.Config{} // zero value

	done := make(chan error, 1)
	go func() {
		done <- run(cfg)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run returned unexpected error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("run did not exit")
	}
}

// ---------------------------------------------------------------------------
// integration: /mcp route via the full middleware chain
// ---------------------------------------------------------------------------

func TestBuildHandlers_Integration_ToolsList(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	cfg.ListenAddr = "127.0.0.1:0"
	h := buildHandlers(cfg, newSharedHTTPClient())

	srv := httptest.NewServer(h.main)
	defer srv.Close()

	client := srv.Client()

	// 1. No token -> 401.
	noTokReq, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, srv.URL,
		strings.NewReader(`{"jsonrpc":"2.0","method":"tools/list","id":1}`))
	noTokReq.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(noTokReq)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 without token, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// 2. With token -> 200 and a JSON-RPC response listing the tools.
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, srv.URL,
		strings.NewReader(`{"jsonrpc":"2.0","method":"tools/list","id":1}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Authorization", "Bearer test-token")
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}

	// The response is SSE-formatted: lines of "event: ..." and "data: <json>".
	// Extract the data payload and unmarshal the JSON-RPC response.
	dataLine := ""
	for _, line := range strings.Split(string(body), "\n") {
		if strings.HasPrefix(line, "data:") {
			dataLine = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			break
		}
	}
	if dataLine == "" {
		t.Fatalf("no SSE data payload found in response: %q", body)
	}

	var msg struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Result  struct {
			Tools []map[string]interface{} `json:"tools"`
		} `json:"result"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(dataLine), &msg); err != nil {
		t.Fatalf("failed to unmarshal response %q: %v", dataLine, err)
	}

	if msg.Error != nil {
		t.Fatalf("tools/list returned error: %+v", msg.Error)
	}
	if len(msg.Result.Tools) == 0 {
		t.Fatal("expected at least one tool in tools/list result")
	}
}

// TestMain_ConfigLoadFailureExits verifies that main() exits with a non-zero
// code when configuration loading fails (e.g. an invalid Paperless URL).
func TestMain_ConfigLoadFailureExits(t *testing.T) {
	if os.Getenv("MCP_RUN_MAIN") == "1" {
		// In the subprocess: PAPERLESS_URL is empty/invalid, so config.Load
		// returns an error and main() calls log.Fatalf -> os.Exit(1).
		main()
		return
	}

	cmd := exec.CommandContext(context.Background(), os.Args[0], "-test.run=TestMain_ConfigLoadFailureExits") //nolint:gosec // fixed, non-tainted args in a test helper
	cmd.Env = append(os.Environ(), "MCP_RUN_MAIN=1", "PAPERLESS_URL=")
	err := cmd.Run()

	var ee *exec.ExitError
	if !errors.As(err, &ee) {
		t.Fatalf("expected the subprocess to exit with an error, got %v", err)
	}
	if ee.ExitCode() != 1 {
		t.Errorf("expected exit code 1, got %d", ee.ExitCode())
	}
}
