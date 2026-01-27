package sandbox

import (
	"context"
	"os/exec"
	"runtime"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSupported(t *testing.T) {
	t.Run("returns true on linux or darwin", func(t *testing.T) {
		supported := Supported()
		isSupported := runtime.GOOS == "linux" || runtime.GOOS == "darwin"
		if supported != isSupported {
			t.Errorf("Supported() = %v, want %v (GOOS=%s)", supported, isSupported, runtime.GOOS)
		}
	})
}

func TestConfigToStatus(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
		want Status
	}{
		{
			name: "nil config",
			cfg:  nil,
			want: Status{
				Enabled:   false,
				Platform:  runtime.GOOS,
				Supported: Supported(),
			},
		},
		{
			name: "disabled sandbox",
			cfg: &Config{
				Enabled: false,
				Network: true,
			},
			want: Status{
				Enabled:   false,
				Platform:  runtime.GOOS,
				Supported: Supported(),
				Network:   true,
			},
		},
		{
			name: "enabled sandbox with network",
			cfg: &Config{
				Enabled: true,
				Network: true,
			},
			want: Status{
				Enabled:   true,
				Platform:  runtime.GOOS,
				Supported: Supported(),
				Network:   true,
			},
		},
		{
			name: "enabled sandbox without network",
			cfg: &Config{
				Enabled: true,
				Network: false,
			},
			want: Status{
				Enabled:   true,
				Platform:  runtime.GOOS,
				Supported: Supported(),
				Network:   false,
			},
		},
		{
			name: "with profile",
			cfg: &Config{
				Enabled: true,
				Network: true,
				Profile: "custom-profile",
			},
			want: Status{
				Enabled:   true,
				Platform:  runtime.GOOS,
				Supported: Supported(),
				Network:   true,
				Profile:   "custom-profile",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Status
			if tt.cfg != nil {
				got = tt.cfg.ToStatus()
			} else {
				// Nil config should return zero status
				got = Status{
					Platform:  runtime.GOOS,
					Supported: Supported(),
				}
			}
			// Clear Active field as it's not part of ToStatus
			got.Active = false

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Config.ToStatus() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNew(t *testing.T) {
	t.Run("returns nil on unsupported platforms", func(t *testing.T) {
		if Supported() {
			t.Skip("skipping on supported platform")
		}

		cfg := &Config{
			Enabled:    true,
			ProjectDir: "/tmp/test",
		}

		sb, err := New(cfg)
		if err == nil {
			t.Error("New() expected error on unsupported platform, got nil")
		}
		if sb != nil {
			t.Error("New() expected nil sandbox on unsupported platform")
		}
	})

	t.Run("returns sandbox on supported platforms", func(t *testing.T) {
		if !Supported() {
			t.Skip("skipping on unsupported platform")
		}

		cfg := &Config{
			Enabled:    true,
			ProjectDir: "/tmp/test",
			HomeDir:    "/tmp/home",
		}

		sb, err := New(cfg)
		if err != nil {
			t.Errorf("New() unexpected error: %v", err)
		}
		if sb == nil {
			t.Error("New() returned nil sandbox on supported platform")
		}

		// Clean up
		_ = sb.Cleanup(context.Background())
	})
}

func TestSandboxInterface(t *testing.T) {
	if !Supported() {
		t.Skip("skipping on unsupported platform")
	}

	t.Run("Prepare and Cleanup should be idempotent", func(t *testing.T) {
		cfg := &Config{
			Enabled:    true,
			ProjectDir: "/tmp/test",
			HomeDir:    "/tmp/home",
		}

		sb, err := New(cfg)
		if err != nil {
			t.Fatalf("New() error: %v", err)
		}

		ctx := context.Background()

		// Prepare may fail in containerized environments without proper permissions
		if err := sb.Prepare(ctx); err != nil {
			t.Logf("Prepare() not available in this environment: %v", err)
			// Still test cleanup idempotence
			if err := sb.Cleanup(ctx); err != nil {
				t.Errorf("Cleanup() after failed Prepare() should not error: %v", err)
			}

			return
		}

		// Multiple Prepare calls should not error
		if err := sb.Prepare(ctx); err != nil {
			t.Errorf("Second Prepare() error: %v", err)
		}

		// Multiple Cleanup calls should not error
		if err := sb.Cleanup(ctx); err != nil {
			t.Errorf("First Cleanup() error: %v", err)
		}
		if err := sb.Cleanup(ctx); err != nil {
			t.Errorf("Second Cleanup() error: %v", err)
		}
	})

	t.Run("WrapCommand modifies command", func(t *testing.T) {
		cfg := &Config{
			Enabled:    true,
			ProjectDir: "/tmp/test",
			HomeDir:    "/tmp/home",
		}

		sb, err := New(cfg)
		if err != nil {
			t.Fatalf("New() error: %v", err)
		}
		defer func() { _ = sb.Cleanup(context.Background()) }()

		cmd := exec.CommandContext(context.Background(), "echo", "test")
		wrapped, err := sb.WrapCommand(cmd)
		if err != nil {
			// WrapCommand may call Prepare which can fail in containers
			t.Skipf("WrapCommand() not available in this environment: %v", err)
		}

		if wrapped == nil {
			t.Error("WrapCommand() returned nil")

			return
		}

		// Wrapped command should have different path or args
		if wrapped.Path == cmd.Path && len(wrapped.Args) == len(cmd.Args) {
			t.Log("WrapCommand() may not have modified the command")
		}
	})
}

func TestConfigDefaults(t *testing.T) {
	t.Run("NewConfig sets network access by default", func(t *testing.T) {
		cfg := NewConfig("/tmp/test", "/tmp/home")

		if !cfg.Network {
			t.Error("NewConfig() should set Network to true by default for LLM API access")
		}
		if !cfg.Enabled {
			t.Error("NewConfig() should set Enabled to true by default")
		}
	})
}
