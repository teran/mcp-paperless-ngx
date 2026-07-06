package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/prometheus/client_golang/prometheus"
)

// ============================================================
// statusCodeToClass tests
// ============================================================

func TestStatusCodeToClass(t *testing.T) {
	t.Parallel()

	tests := []struct {
		code int
		want string
	}{
		{200, "2xx"},
		{201, "2xx"},
		{299, "2xx"},
		{300, "3xx"},
		{301, "3xx"},
		{399, "3xx"},
		{400, "4xx"},
		{401, "4xx"},
		{404, "4xx"},
		{499, "4xx"},
		{500, "5xx"},
		{502, "5xx"},
		{599, "5xx"},
		{600, "5xx"},
		{100, "unknown"},
		{199, "unknown"},
		{0, "unknown"},
		{-1, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()

			got := statusCodeToClass(tt.code)
			if got != tt.want {
				t.Errorf("statusCodeToClass(%d) = %q, want %q", tt.code, got, tt.want)
			}
		})
	}
}

// ============================================================
// metricsResponseWriter tests
// ============================================================

func TestMetricsResponseWriter(t *testing.T) {
	t.Parallel()

	t.Run("captures status code", func(t *testing.T) {
		t.Parallel()

		rr := httptest.NewRecorder()
		mrw := &metricsResponseWriter{
			ResponseWriter: rr,
			statusCode:     http.StatusOK,
		}

		mrw.WriteHeader(http.StatusBadRequest)

		if mrw.statusCode != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, mrw.statusCode)
		}
	})

	t.Run("passes through Write to underlying writer", func(t *testing.T) {
		t.Parallel()

		rr := httptest.NewRecorder()
		mrw := &metricsResponseWriter{
			ResponseWriter: rr,
			statusCode:     http.StatusOK,
		}

		n, err := mrw.Write([]byte("hello"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n != 5 {
			t.Errorf("expected written bytes=5, got %d", n)
		}
		if rr.Body.String() != "hello" {
			t.Errorf("expected body %q, got %q", "hello", rr.Body.String())
		}
	})

	t.Run("default status code is 200", func(t *testing.T) {
		t.Parallel()

		rr := httptest.NewRecorder()
		mrw := &metricsResponseWriter{
			ResponseWriter: rr,
			statusCode:     http.StatusOK,
		}

		_, _ = mrw.Write([]byte("data"))

		if mrw.statusCode != http.StatusOK {
			t.Errorf("expected default status %d, got %d", http.StatusOK, mrw.statusCode)
		}
	})
}

// ============================================================
// MetricsMiddleware tests
// ============================================================

func TestMetricsMiddleware_ActiveRequestsGauge(t *testing.T) {
	t.Parallel()

	reg := prometheus.NewRegistry()
	metrics := NewMetrics(reg)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check active gauge is 1 inside handler.
		families, err := reg.Gather()
		if err != nil {
			t.Errorf("Gather() error: %v", err)
		}
		for _, f := range families {
			if f.GetName() == "mcp_active_requests" {
				if v := f.GetMetric()[0].GetGauge().GetValue(); v != 1 {
					t.Errorf("expected active_requests=1 inside handler, got %f", v)
				}
			}
		}
		w.WriteHeader(http.StatusOK)
	})

	handler := MetricsMiddleware(metrics)(next)

	body := `{"jsonrpc":"2.0","method":"tools/list","id":"1"}`
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(body))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	// After handler returns, active_requests should be back to 0.
	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather() error: %v", err)
	}
	for _, f := range families {
		if f.GetName() == "mcp_active_requests" {
			if v := f.GetMetric()[0].GetGauge().GetValue(); v != 0 {
				t.Errorf("expected active_requests=0 after handler, got %f", v)
			}
		}
	}
}

func TestMetricsMiddleware_DecrementsOnPanic(t *testing.T) {
	t.Parallel()

	reg := prometheus.NewRegistry()
	metrics := NewMetrics(reg)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	handler := RecoveryMiddleware(MetricsMiddleware(metrics)(next))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rr.Code)
	}

	// After recovery from panic, active_requests should still be 0.
	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather() error: %v", err)
	}
	for _, f := range families {
		if f.GetName() == "mcp_active_requests" {
			if v := f.GetMetric()[0].GetGauge().GetValue(); v != 0 {
				t.Errorf("expected active_requests=0 after panic, got %f", v)
			}
		}
	}
}

// ============================================================
// WrapToolHandler tests
// ============================================================

func TestWrapToolHandler_RecordsMetrics(t *testing.T) { //nolint:gocognit,gocyclo
	t.Parallel()

	t.Run("success records 2xx with counter and histogram", func(t *testing.T) {
		t.Parallel()

		reg := prometheus.NewRegistry()
		metrics := NewMetrics(reg)

		inner := func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, struct{}, error) {
			return nil, struct{}{}, nil
		}

		wrapped := WrapToolHandler(metrics, "success_tool", inner)
		_, _, err := wrapped(context.Background(), nil, struct{}{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		families, err := reg.Gather()
		if err != nil {
			t.Fatalf("Gather() error: %v", err)
		}

		var foundCounter, foundHistogram bool
		for _, f := range families {
			switch f.GetName() {
			case "mcp_tool_requests_total":
				for _, m := range f.GetMetric() {
					labels := make(map[string]string)
					for _, l := range m.GetLabel() {
						labels[l.GetName()] = l.GetValue()
					}
					if labels["tool"] == "success_tool" && labels["status_class"] == "2xx" {
						if m.GetCounter().GetValue() == 1 {
							foundCounter = true
						}
					}
				}
			case "mcp_tool_duration_seconds":
				for _, m := range f.GetMetric() {
					labels := make(map[string]string)
					for _, l := range m.GetLabel() {
						labels[l.GetName()] = l.GetValue()
					}
					if labels["tool"] == "success_tool" && m.GetHistogram().GetSampleCount() == 1 {
						foundHistogram = true
					}
				}
			}
		}

		if !foundCounter {
			t.Error("expected mcp_tool_requests_total with tool=success_tool,status_class=2xx and value=1")
		}
		if !foundHistogram {
			t.Error("expected mcp_tool_duration_seconds with tool=success_tool and sample_count=1")
		}
	})

	t.Run("error records 4xx status class", func(t *testing.T) {
		t.Parallel()

		reg := prometheus.NewRegistry()
		metrics := NewMetrics(reg)

		inner := func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, struct{}, error) {
			return nil, struct{}{}, errors.New("tool error")
		}

		wrapped := WrapToolHandler(metrics, "error_tool", inner)
		_, _, err := wrapped(context.Background(), nil, struct{}{})
		if err == nil {
			t.Fatal("expected error")
		}

		families, err := reg.Gather()
		if err != nil {
			t.Fatalf("Gather() error: %v", err)
		}

		var foundCounter bool
		for _, f := range families {
			if f.GetName() != "mcp_tool_requests_total" {
				continue
			}
			for _, m := range f.GetMetric() {
				labels := make(map[string]string)
				for _, l := range m.GetLabel() {
					labels[l.GetName()] = l.GetValue()
				}
				if labels["tool"] == "error_tool" && labels["status_class"] == "4xx" {
					if m.GetCounter().GetValue() == 1 {
						foundCounter = true
					}
				}
			}
		}

		if !foundCounter {
			t.Error("expected mcp_tool_requests_total with tool=error_tool,status_class=4xx and value=1")
		}
	})

	t.Run("records positive duration", func(t *testing.T) {
		t.Parallel()

		reg := prometheus.NewRegistry()
		metrics := NewMetrics(reg)

		inner := func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, struct{}, error) {
			time.Sleep(time.Millisecond)
			return nil, struct{}{}, nil
		}

		wrapped := WrapToolHandler(metrics, "slow_tool", inner)
		_, _, err := wrapped(context.Background(), nil, struct{}{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		families, err := reg.Gather()
		if err != nil {
			t.Fatalf("Gather() error: %v", err)
		}

		for _, f := range families {
			if f.GetName() != "mcp_tool_duration_seconds" {
				continue
			}
			for _, m := range f.GetMetric() {
				labels := make(map[string]string)
				for _, l := range m.GetLabel() {
					labels[l.GetName()] = l.GetValue()
				}
				if labels["tool"] != "slow_tool" {
					continue
				}
				h := m.GetHistogram()
				if h.GetSampleCount() != 1 {
					t.Errorf("expected sample_count=1, got %d", h.GetSampleCount())
				}
				if h.GetSampleSum() <= 0 {
					t.Errorf("expected positive sample_sum, got %f", h.GetSampleSum())
				}
			}
		}
	})
}

// ============================================================
// RegisterMetricsOnRegistry test
// ============================================================

func TestRegisterMetricsOnRegistry(t *testing.T) {
	t.Parallel()

	reg := prometheus.NewRegistry()

	// Create and register MCP metrics first (as main.go does).
	_ = NewMetrics(reg)

	// Now register Go + process collectors and get handler.
	handler := RegisterMetricsOnRegistry(reg)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	body := rr.Body.String()

	// Go runtime metrics: go_goroutines always has a value.
	if !strings.Contains(body, "go_goroutines") {
		t.Error("expected go_goroutines in metrics output")
	}
	// Custom MCP metrics: gauges are always emitted (even at zero).
	if !strings.Contains(body, "mcp_active_requests") {
		t.Error("expected mcp_active_requests in metrics output")
	}
	// Counter and histogram metrics are only emitted after they have been
	// observed at least once (Prometheus default behaviour). They are tested
	// in TestMetricsMiddleware where actual requests are processed.
}
