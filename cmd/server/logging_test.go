package main

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/teran/mcp-paperless-ngx/config"
)

func TestSetupLogging_Disabled(t *testing.T) {
	cfg := &config.Config{PaperlessURL: "http://x"} // LogLevel unset
	if err := setupLogging(cfg); err != nil {
		t.Fatalf("setupLogging() error = %v", err)
	}
	if lvl := logrus.StandardLogger().GetLevel(); lvl != logrus.PanicLevel {
		t.Errorf("expected PanicLevel (disabled), got %v", lvl)
	}
}

func TestSetupLogging_EnabledText(t *testing.T) {
	cfg := &config.Config{PaperlessURL: "http://x", LogLevel: "info"}
	if err := setupLogging(cfg); err != nil {
		t.Fatalf("setupLogging() error = %v", err)
	}
	if lvl := logrus.StandardLogger().GetLevel(); lvl != logrus.InfoLevel {
		t.Errorf("expected InfoLevel, got %v", lvl)
	}
	if _, ok := logrus.StandardLogger().Formatter.(*logrus.TextFormatter); !ok {
		t.Errorf("expected TextFormatter by default, got %T", logrus.StandardLogger().Formatter)
	}
}

func TestSetupLogging_EnabledJSON(t *testing.T) {
	cfg := &config.Config{PaperlessURL: "http://x", LogLevel: "info", LogFormat: "json"}
	if err := setupLogging(cfg); err != nil {
		t.Fatalf("setupLogging() error = %v", err)
	}
	if _, ok := logrus.StandardLogger().Formatter.(*logrus.JSONFormatter); !ok {
		t.Errorf("expected JSONFormatter, got %T", logrus.StandardLogger().Formatter)
	}
}

func TestSetupLogging_InvalidLevel(t *testing.T) {
	cfg := &config.Config{PaperlessURL: "http://x", LogLevel: "not-a-level"}
	if err := setupLogging(cfg); err == nil {
		t.Fatal("expected error for invalid LOG_LEVEL")
	}
}

func TestSetupLogging_LogFilename(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "server.log")

	cfg := &config.Config{PaperlessURL: "http://x", LogLevel: "info", LogFilename: path}
	if err := setupLogging(cfg); err != nil {
		t.Fatalf("setupLogging() error = %v", err)
	}

	logrus.Info("to file")

	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("expected log file to exist: %v", err)
	}
	if perm := fi.Mode().Perm(); perm != 0o600 {
		t.Errorf("expected log file mode 0600, got %o", perm)
	}
}

func TestSlogAdapter(t *testing.T) {
	if adapter := slogAdapter(); adapter == nil {
		t.Fatal("expected non-nil slog.Logger")
	}
}

func TestLogrusHandler_Handle(t *testing.T) {
	var buf bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&buf)
	logger.SetLevel(logrus.DebugLevel)
	logger.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true})

	h := &logrusHandler{logger: logger}
	rec := slog.NewRecord(time.Now(), slog.LevelInfo, "hello sdk", 0)
	rec.AddAttrs(slog.String("tool", "search_documents"))
	if err := h.Handle(context.Background(), rec); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "hello sdk") {
		t.Errorf("expected message in output, got %q", out)
	}
	if !strings.Contains(out, "search_documents") {
		t.Errorf("expected attr value in output, got %q", out)
	}
}

func TestLogrusHandler_WithAttrsAndGroup(t *testing.T) {
	var buf bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&buf)
	logger.SetLevel(logrus.DebugLevel)
	logger.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true})

	base := &logrusHandler{logger: logger}
	withAttrs := base.WithAttrs([]slog.Attr{slog.String("region", "eu")})
	withGroup := base.WithGroup("grp")

	_ = withAttrs.Handle(context.Background(), slog.NewRecord(time.Now(), slog.LevelInfo, "attrs msg", 0))
	_ = withGroup.Handle(context.Background(), slog.NewRecord(time.Now(), slog.LevelInfo, "group msg", 0))

	out := buf.String()
	if !strings.Contains(out, "region=eu") {
		t.Errorf("expected WithAttrs to carry the attr, got %q", out)
	}
	if !strings.Contains(out, "group msg") {
		t.Errorf("expected WithGroup handler to log, got %q", out)
	}
}

func TestSlogLevelToLogrus(t *testing.T) {
	tests := []struct {
		slogLevel slog.Level
		want      logrus.Level
	}{
		{slog.LevelError, logrus.ErrorLevel},
		{slog.LevelWarn, logrus.WarnLevel},
		{slog.LevelInfo, logrus.InfoLevel},
		{slog.LevelDebug, logrus.DebugLevel},
		{slog.Level(-8), logrus.DebugLevel},
	}
	for _, tt := range tests {
		if got := slogLevelToLogrus(tt.slogLevel); got != tt.want {
			t.Errorf("slogLevelToLogrus(%d) = %v, want %v", tt.slogLevel, got, tt.want)
		}
	}
}
