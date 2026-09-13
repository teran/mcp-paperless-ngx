package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/teran/mcp-paperless-ngx/config"
)

// setupLogging configures the process-wide logrus logger according to L2-L4:
//   - gated by LOG_LEVEL (when unset, logging is a no-op / disabled);
//   - LOG_FILENAME overrides the output file (stdout is the default for the
//     remote HTTP/SSE transport per the 12-factor methodology);
//   - LOG_FORMAT=json switches to JSON output, otherwise text with a full
//     absolute timestamp.
//
// When logging is disabled, the standard logger is pointed at io.Discard at
// Panic level so every log call becomes a no-op.
func setupLogging(cfg *config.Config) error {
	if cfg.LogLevel == "" {
		logrus.SetOutput(io.Discard)
		logrus.SetLevel(logrus.PanicLevel)
		return nil
	}

	lvl, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		return err
	}

	logger := logrus.StandardLogger()
	logger.SetLevel(lvl)

	var out io.Writer = os.Stdout
	if cfg.LogFilename != "" {
		f, err := os.OpenFile(cfg.LogFilename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		if err != nil {
			return err
		}
		// Enforce chmod 600 even if the file already existed with looser perms.
		if err := f.Chmod(0o600); err != nil {
			return err
		}
		out = f
	}
	logger.SetOutput(out)

	if strings.EqualFold(cfg.LogFormat, "json") {
		logger.SetFormatter(&logrus.JSONFormatter{}) //nolint:exhaustruct
	} else {
		logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	}

	return nil
}

// logrusHandler adapts a log/slog handler to write into a logrus logger. It
// bridges the MCP go-sdk's slog logger into the server's logrus logger (L7),
// so SDK-level MCP events (session connect/end, tool-call results/errors,
// protocol warnings) appear in the server logs at the configured level.
type logrusHandler struct {
	logger *logrus.Logger
	attrs  []slog.Attr
}

// slogAdapter returns a slog.Logger backed by the process-wide logrus logger,
// suitable for mcp.ServerOptions.Logger.
func slogAdapter() *slog.Logger {
	return slog.New(&logrusHandler{logger: logrus.StandardLogger()})
}

// Enabled reports whether the record should be handled. logrus filters by its
// own configured level, so this always reports true.
func (h *logrusHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

// Handle converts a slog record into a logrus entry and logs it.
func (h *logrusHandler) Handle(_ context.Context, rec slog.Record) error {
	entry := h.logger.WithTime(rec.Time)
	for _, a := range h.attrs {
		entry = entry.WithField(a.Key, a.Value.Any())
	}
	rec.Attrs(func(a slog.Attr) bool {
		entry = entry.WithField(a.Key, a.Value.Any())
		return true
	})
	entry.Log(slogLevelToLogrus(rec.Level), rec.Message)
	return nil
}

// WithAttrs returns a copy of the handler carrying the additional attributes.
func (h *logrusHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	merged := make([]slog.Attr, 0, len(h.attrs)+len(attrs))
	merged = append(merged, h.attrs...)
	merged = append(merged, attrs...)
	return &logrusHandler{logger: h.logger, attrs: merged}
}

// WithGroup returns a copy of the handler scoped to a group. Groups are
// flattened into dotted attribute keys for logrus.
func (h *logrusHandler) WithGroup(name string) slog.Handler {
	return &logrusHandler{logger: h.logger, attrs: h.attrs}
}

// slogLevelToLogrus maps a slog level to the corresponding logrus level.
func slogLevelToLogrus(l slog.Level) logrus.Level {
	switch {
	case l >= slog.LevelError:
		return logrus.ErrorLevel
	case l >= slog.LevelWarn:
		return logrus.WarnLevel
	case l >= slog.LevelInfo:
		return logrus.InfoLevel
	default:
		return logrus.DebugLevel
	}
}
