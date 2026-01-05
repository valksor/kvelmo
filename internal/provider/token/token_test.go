package token

import (
	"errors"
	"testing"
)

func TestResolveToken(t *testing.T) {
	t.Run("config token is used when available", func(t *testing.T) {
		tok, err := ResolveToken(Config("TEST", "config-token"))
		if err != nil {
			t.Fatalf("ResolveToken error = %v", err)
		}
		if tok != "config-token" {
			t.Errorf("token = %q, want %q", tok, "config-token")
		}
	})

	t.Run("empty config token returns ErrNoToken", func(t *testing.T) {
		_, err := ResolveToken(Config("TEST", ""))
		if !errors.Is(err, ErrNoToken) {
			t.Errorf("error = %v, want %v", err, ErrNoToken)
		}
	})

	t.Run("CLI fallback is used when config token is empty", func(t *testing.T) {
		fallbackCalled := false
		cfg := Config("TEST", "").
			WithCLIFallback(func() string {
				fallbackCalled = true

				return "cli-token"
			})

		tok, err := ResolveToken(cfg)
		if err != nil {
			t.Fatalf("ResolveToken error = %v", err)
		}
		if tok != "cli-token" {
			t.Errorf("token = %q, want %q", tok, "cli-token")
		}
		if !fallbackCalled {
			t.Error("CLI fallback was not called")
		}
	})

	t.Run("config token takes priority over CLI fallback", func(t *testing.T) {
		fallbackCalled := false
		cfg := Config("TEST", "config-token").
			WithCLIFallback(func() string {
				fallbackCalled = true

				return "cli-token"
			})

		tok, err := ResolveToken(cfg)
		if err != nil {
			t.Fatalf("ResolveToken error = %v", err)
		}
		if tok != "config-token" {
			t.Errorf("token = %q, want %q", tok, "config-token")
		}
		if fallbackCalled {
			t.Error("CLI fallback was called when config token was available")
		}
	})

	t.Run("no token available returns ErrNoToken", func(t *testing.T) {
		_, err := ResolveToken(Config("TEST", ""))
		if !errors.Is(err, ErrNoToken) {
			t.Errorf("error = %v, want %v", err, ErrNoToken)
		}
	})

	t.Run("CLI fallback returning empty string results in ErrNoToken", func(t *testing.T) {
		cfg := Config("TEST", "").
			WithCLIFallback(func() string {
				return "" // Empty fallback
			})

		_, err := ResolveToken(cfg)
		if !errors.Is(err, ErrNoToken) {
			t.Errorf("error = %v, want %v", err, ErrNoToken)
		}
	})
}

func TestConfig(t *testing.T) {
	cfg := Config("PROVIDER", "my-token")

	if cfg.ProviderName != "PROVIDER" {
		t.Errorf("ProviderName = %q, want %q", cfg.ProviderName, "PROVIDER")
	}
	if cfg.ConfigToken != "my-token" {
		t.Errorf("ConfigToken = %q, want %q", cfg.ConfigToken, "my-token")
	}
}

func TestWithCLIFallback(t *testing.T) {
	fallback := func() string { return "cli-token" }
	cfg := Config("TEST", "").WithCLIFallback(fallback)

	if cfg.OptionalCLIFallback == nil {
		t.Error("OptionalCLIFallback was not set")
	}
}

func TestMustResolveToken(t *testing.T) {
	t.Run("panics when no token available", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustResolveToken did not panic")
			}
		}()

		_ = MustResolveToken(Config("TEST", ""))
	})

	t.Run("returns token when available in config", func(t *testing.T) {
		tok := MustResolveToken(Config("TEST", "config-token"))
		if tok != "config-token" {
			t.Errorf("token = %q, want %q", tok, "config-token")
		}
	})

	t.Run("returns token from CLI fallback when config is empty", func(t *testing.T) {
		tok := MustResolveToken(Config("TEST", "").WithCLIFallback(func() string {
			return "cli-token"
		}))
		if tok != "cli-token" {
			t.Errorf("token = %q, want %q", tok, "cli-token")
		}
	})
}
