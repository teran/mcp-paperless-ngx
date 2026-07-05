package config

import (
	"testing"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

func TestLoad(t *testing.T) {

	t.Run("all env vars set correctly", func(t *testing.T) {
		t.Setenv("PAPERLESS_URL", "http://paperless:8000")
		t.Setenv("LISTEN_ADDR", ":9090")
		t.Setenv("RATE_LIMIT_GLOBAL", "200")
		t.Setenv("RATE_LIMIT_PER_CLIENT", "50")
		t.Setenv("WRITE_TIMEOUT", "600s")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() returned error: %v", err)
		}

		if cfg.PaperlessURL != "http://paperless:8000" {
			t.Errorf("PaperlessURL = %q, want %q", cfg.PaperlessURL, "http://paperless:8000")
		}
		if cfg.ListenAddr != ":9090" {
			t.Errorf("ListenAddr = %q, want %q", cfg.ListenAddr, ":9090")
		}
		if cfg.RateLimitGlobal != 200 {
			t.Errorf("RateLimitGlobal = %d, want 200", cfg.RateLimitGlobal)
		}
		if cfg.RateLimitPerClient != 50 {
			t.Errorf("RateLimitPerClient = %d, want 50", cfg.RateLimitPerClient)
		}
		if cfg.WriteTimeout != 600*time.Second {
			t.Errorf("WriteTimeout = %v, want %v", cfg.WriteTimeout, 600*time.Second)
		}
	})

	t.Run("defaults when env vars are not set", func(t *testing.T) {
		t.Setenv("PAPERLESS_URL", "http://paperless:8000")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() returned error: %v", err)
		}

		if cfg.ListenAddr != ":8080" {
			t.Errorf("ListenAddr = %q, want %q", cfg.ListenAddr, ":8080")
		}
		if cfg.RateLimitGlobal != 100 {
			t.Errorf("RateLimitGlobal = %d, want 100", cfg.RateLimitGlobal)
		}
		if cfg.RateLimitPerClient != 10 {
			t.Errorf("RateLimitPerClient = %d, want 10", cfg.RateLimitPerClient)
		}
		if cfg.WriteTimeout != 300*time.Second {
			t.Errorf("WriteTimeout = %v, want %v", cfg.WriteTimeout, 300*time.Second)
		}
	})
}

func TestLoad_Errors(t *testing.T) {

	t.Run("PAPERLESS_URL is required", func(t *testing.T) {
		t.Setenv("PAPERLESS_URL", "")

		_, err := Load()
		if err == nil {
			t.Fatal("Load() expected error for missing PAPERLESS_URL")
		}
	})

	t.Run("PAPERLESS_URL invalid URL", func(t *testing.T) {
		t.Setenv("PAPERLESS_URL", "://invalid")

		_, err := Load()
		if err == nil {
			t.Fatal("Load() expected error for invalid URL")
		}
	})

	t.Run("PAPERLESS_URL must have http/https scheme", func(t *testing.T) {
		t.Setenv("PAPERLESS_URL", "ftp://paperless:8000")

		_, err := Load()
		if err == nil {
			t.Fatal("Load() expected error for non-http scheme")
		}
	})

	t.Run("PAPERLESS_URL must have host", func(t *testing.T) {
		t.Setenv("PAPERLESS_URL", "http://")

		_, err := Load()
		if err == nil {
			t.Fatal("Load() expected error for empty host")
		}
	})

	t.Run("RATE_LIMIT_GLOBAL must be >= 1", func(t *testing.T) {
		t.Setenv("PAPERLESS_URL", "http://paperless:8000")
		t.Setenv("RATE_LIMIT_GLOBAL", "0")

		_, err := Load()
		if err == nil {
			t.Fatal("Load() expected error for RATE_LIMIT_GLOBAL=0")
		}
	})

	t.Run("RATE_LIMIT_PER_CLIENT must be >= 1", func(t *testing.T) {
		t.Setenv("PAPERLESS_URL", "http://paperless:8000")
		t.Setenv("RATE_LIMIT_PER_CLIENT", "-5")

		_, err := Load()
		if err == nil {
			t.Fatal("Load() expected error for RATE_LIMIT_PER_CLIENT=-5")
		}
	})

	t.Run("envconfig error on invalid env var value", func(t *testing.T) {
		t.Setenv("PAPERLESS_URL", "http://paperless:8000")
		t.Setenv("RATE_LIMIT_GLOBAL", "not-a-number")

		_, err := Load()
		if err == nil {
			t.Fatal("Load() expected error for invalid RATE_LIMIT_GLOBAL value")
		}
	})

	t.Run("non-int value for validatePositiveInt", func(t *testing.T) {
		// Directly test the !ok branch of validatePositiveInt
		err := validation.Validate("not-an-int", validation.By(validatePositiveInt))
		if err == nil {
			t.Fatal("validatePositiveInt expected error for non-int value")
		}
	})

	t.Run("validateURLHost parse error", func(t *testing.T) {
		// Direct call to cover the url.Parse error branch (dead code from Load()
		// because validateURLScheme catches it first).
		err := validateURLHost("://invalid")
		if err == nil {
			t.Fatal("validateURLHost expected error for unparseable URL")
		}
	})
}
