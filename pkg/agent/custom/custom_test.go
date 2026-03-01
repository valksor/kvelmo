package custom_test

import (
	"context"
	"testing"
	"time"

	"github.com/valksor/kvelmo/pkg/agent"
	"github.com/valksor/kvelmo/pkg/agent/custom"
)

func TestDefaultConfig(t *testing.T) {
	cfg := custom.DefaultConfig("myagent", []string{"echo"})
	if cfg.Name != "myagent" {
		t.Errorf("DefaultConfig().Name = %q, want %q", cfg.Name, "myagent")
	}
	if cfg.InputFormat != "json" {
		t.Errorf("DefaultConfig().InputFormat = %q, want json", cfg.InputFormat)
	}
	if cfg.OutputFormat != "ndjson" {
		t.Errorf("DefaultConfig().OutputFormat = %q, want ndjson", cfg.OutputFormat)
	}
	if cfg.Timeout == 0 {
		t.Error("DefaultConfig().Timeout should not be zero")
	}
}

func TestNew(t *testing.T) {
	a := custom.New("test", []string{"echo"})
	if a == nil {
		t.Fatal("New() returned nil")
	}
}

func TestNewWithConfig_Defaults(t *testing.T) {
	// Empty name and command should get defaults
	cfg := custom.Config{}
	a := custom.NewWithConfig(cfg)
	if a == nil {
		t.Fatal("NewWithConfig() returned nil")
	}
}

func TestName(t *testing.T) {
	a := custom.New("myagent", []string{"echo"})
	if a.Name() != "myagent" {
		t.Errorf("Name() = %q, want myagent", a.Name())
	}
}

func TestConnected_NewAgent(t *testing.T) {
	a := custom.New("test", []string{"echo"})
	if a.Connected() {
		t.Error("Connected() should return false for a new agent")
	}
}

func TestAvailable_EchoExists(t *testing.T) {
	a := custom.New("test", []string{"echo"})
	// echo is always available on Unix
	if err := a.Available(); err != nil {
		t.Errorf("Available() with 'echo' command returned unexpected error: %v", err)
	}
}

func TestAvailable_BinaryNotFound(t *testing.T) {
	a := custom.New("test", []string{"nonexistent-binary-kvelmo-test-xyz"})
	err := a.Available()
	if err == nil {
		t.Error("Available() with nonexistent command should return an error")
	}
}

func TestWithEnv(t *testing.T) {
	a := custom.New("test", []string{"echo"})
	b := a.WithEnv("MY_KEY", "my_value")
	if b == nil {
		t.Fatal("WithEnv() returned nil")
	}
	if b == a {
		t.Error("WithEnv() should return a new agent")
	}
}

func TestWithEnv_Chains(t *testing.T) {
	a := custom.New("test", []string{"echo"})
	b := a.WithEnv("K1", "v1").WithEnv("K2", "v2")
	if b == nil {
		t.Fatal("chained WithEnv() returned nil")
	}
}

func TestWithArgs(t *testing.T) {
	a := custom.New("test", []string{"echo"})
	b := a.WithArgs("--foo", "--bar")
	if b == nil {
		t.Fatal("WithArgs() returned nil")
	}
}

func TestWithWorkDir(t *testing.T) {
	a := custom.New("test", []string{"echo"})
	b := a.WithWorkDir("/tmp")
	if b == nil {
		t.Fatal("WithWorkDir() returned nil")
	}
}

func TestWithTimeout(t *testing.T) {
	a := custom.New("test", []string{"echo"})
	b := a.WithTimeout(2 * time.Minute)
	if b == nil {
		t.Fatal("WithTimeout() returned nil")
	}
}

func TestSendPrompt_NotConnected(t *testing.T) {
	a := custom.New("test", []string{"echo"})
	_, err := a.SendPrompt(context.Background(), "hello")
	if err == nil {
		t.Error("SendPrompt() on disconnected agent should return an error")
	}
}

func TestHandlePermission(t *testing.T) {
	a := custom.New("test", []string{"echo"})
	// Custom agent HandlePermission is always a no-op
	if err := a.HandlePermission("req-1", true); err != nil {
		t.Errorf("HandlePermission() returned unexpected error: %v", err)
	}
}

func TestClose_Unconnected(t *testing.T) {
	a := custom.New("test", []string{"echo"})
	if err := a.Close(); err != nil {
		t.Errorf("Close() on unconnected agent returned unexpected error: %v", err)
	}
}

func TestClose_Idempotent(t *testing.T) {
	a := custom.New("test", []string{"echo"})
	_ = a.Close()
	// Second close should not panic or error
	if err := a.Close(); err != nil {
		t.Errorf("second Close() returned unexpected error: %v", err)
	}
}

func TestConnect_WithCat(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	a := custom.NewWithConfig(custom.Config{
		Name:         "test",
		Command:      []string{"cat"},
		Timeout:      5 * time.Second,
		InputFormat:  "text",
		OutputFormat: "text",
		PermissionHandler: func(req agent.PermissionRequest) bool {
			return true
		},
	})

	if err := a.Connect(ctx); err != nil {
		t.Fatalf("Connect() with cat returned error: %v", err)
	}

	if !a.Connected() {
		t.Error("Connected() should return true after successful Connect()")
	}

	// Clean up the cat process
	if err := a.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}
