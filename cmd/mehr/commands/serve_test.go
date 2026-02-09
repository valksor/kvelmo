//go:build !testbinary

package commands

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/valksor/go-mehrhof/internal/server"
)

func TestServeCommand_Properties(t *testing.T) {
	if serveCmd.Use != "serve" {
		t.Errorf("Use = %q, want %q", serveCmd.Use, "serve")
	}

	if serveCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if serveCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if serveCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestServeCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		shorthand    string
		defaultValue string
	}{
		{
			name:         "port flag",
			flagName:     "port",
			shorthand:    "p",
			defaultValue: "0",
		},
		{
			name:         "global flag",
			flagName:     "global",
			shorthand:    "",
			defaultValue: "false",
		},
		{
			name:         "open flag",
			flagName:     "open",
			shorthand:    "",
			defaultValue: "false",
		},
		{
			name:         "api flag",
			flagName:     "api",
			shorthand:    "",
			defaultValue: "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := serveCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("flag %q not found", tt.flagName)

				return
			}

			if flag.DefValue != tt.defaultValue {
				t.Errorf("flag %q default value = %q, want %q", tt.flagName, flag.DefValue, tt.defaultValue)
			}

			if tt.shorthand != "" {
				shorthand := serveCmd.Flags().ShorthandLookup(tt.shorthand)
				if shorthand == nil {
					t.Errorf("shorthand %q not found for flag %q", tt.shorthand, tt.flagName)
				}
			}
		})
	}
}

func TestServeCommand_RegisteredInRoot(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "serve" {
			found = true

			break
		}
	}
	if !found {
		t.Error("serve command not registered in root command")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// portAvailable behavioral tests
// ──────────────────────────────────────────────────────────────────────────────

func TestPortAvailable_UnusedPort(t *testing.T) {
	// Use port 0 to get a random available port, then check that port is available
	lc := net.ListenConfig{}
	ln, err := lc.Listen(context.Background(), "tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to get free port: %v", err)
	}
	tcpAddr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatal("failed to get TCP address")
	}
	port := tcpAddr.Port
	_ = ln.Close()

	// After closing, the port should be available
	if !portAvailable("localhost", port) {
		t.Error("portAvailable returned false for unused port")
	}
}

func TestPortAvailable_UsedPort(t *testing.T) {
	// Bind to a port
	lc := net.ListenConfig{}
	ln, err := lc.Listen(context.Background(), "tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to bind port: %v", err)
	}
	defer func() { _ = ln.Close() }()

	tcpAddr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatal("failed to get TCP address")
	}
	port := tcpAddr.Port

	// While bound, the port should NOT be available
	if portAvailable("localhost", port) {
		t.Error("portAvailable returned true for used port")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// resolveServePort behavioral tests
// ──────────────────────────────────────────────────────────────────────────────

func TestResolveServePort_ExplicitPort(t *testing.T) {
	opts := serveOptions{port: 8080, preferredPort: 6337}
	// Mock checker that always returns false (simulating ports in use)
	checkPort := func(string, int) bool { return false }

	port := resolveServePort(opts, checkPort)

	// Should use explicit port regardless of availability
	if port != 8080 {
		t.Errorf("port = %d, want 8080", port)
	}
}

func TestResolveServePort_PreferredPortAvailable(t *testing.T) {
	opts := serveOptions{port: 0, preferredPort: 6337}
	// Mock checker that says preferred port is available
	checkPort := func(host string, p int) bool {
		return p == 6337
	}

	port := resolveServePort(opts, checkPort)

	// Should use preferred port
	if port != 6337 {
		t.Errorf("port = %d, want 6337", port)
	}
}

func TestResolveServePort_PreferredPortUnavailable(t *testing.T) {
	opts := serveOptions{port: 0, preferredPort: 6337}
	// Mock checker that says preferred port is NOT available
	checkPort := func(string, int) bool { return false }

	port := resolveServePort(opts, checkPort)

	// Should fall back to random (0)
	if port != 0 {
		t.Errorf("port = %d, want 0 (random)", port)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// buildBaseServerConfig behavioral tests
// ──────────────────────────────────────────────────────────────────────────────

func TestBuildBaseServerConfig_ProjectMode(t *testing.T) {
	opts := serveOptions{global: false, apiOnly: false}

	cfg := buildBaseServerConfig(opts, 8080)

	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want 8080", cfg.Port)
	}
	if cfg.Host != "localhost" {
		t.Errorf("Host = %q, want %q", cfg.Host, "localhost")
	}
	if cfg.Mode != server.ModeProject {
		t.Errorf("Mode = %v, want ModeProject", cfg.Mode)
	}
	if cfg.APIOnly {
		t.Error("APIOnly = true, want false")
	}
}

func TestBuildBaseServerConfig_GlobalMode(t *testing.T) {
	opts := serveOptions{global: true, apiOnly: false}

	cfg := buildBaseServerConfig(opts, 6337)

	if cfg.Mode != server.ModeGlobal {
		t.Errorf("Mode = %v, want ModeGlobal", cfg.Mode)
	}
}

func TestBuildBaseServerConfig_APIOnly(t *testing.T) {
	opts := serveOptions{global: false, apiOnly: true}

	cfg := buildBaseServerConfig(opts, 8080)

	if !cfg.APIOnly {
		t.Error("APIOnly = false, want true")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Integration: actual port checking (uses real network)
// ──────────────────────────────────────────────────────────────────────────────

func TestPortAvailable_Integration(t *testing.T) {
	// Skip in short mode as it touches the network
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	// Start a listener on a random port
	lc := net.ListenConfig{}
	ln, err := lc.Listen(context.Background(), "tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}

	tcpAddr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatal("failed to get TCP address")
	}
	port := tcpAddr.Port

	// Port should be in use
	if portAvailable("localhost", port) {
		t.Errorf("port %d reported available while listener active", port)
	}

	// Close listener
	_ = ln.Close()

	// Give OS time to release the port (may need small delay on some systems)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	<-ctx.Done()

	// Port should now be available
	if !portAvailable("localhost", port) {
		t.Errorf("port %d reported unavailable after listener closed", port)
	}
}
