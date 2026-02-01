//go:build !testbinary

package commands

import (
	"testing"
)

// TestBrowserCommand_StrictCertsFlag tests that the --strict-certs flag is properly configured.
func TestBrowserCommand_StrictCertsFlag(t *testing.T) {
	flag := browserCmd.PersistentFlags().Lookup("strict-certs")
	if flag == nil {
		t.Fatal("strict-certs flag not found")

		return
	}

	// Check the default value is false (meaning IgnoreCertErrors is true by default)
	if flag.DefValue != "false" {
		t.Errorf("strict-certs flag default value = %q, want 'false'", flag.DefValue)
	}

	// Verify the flag is a bool flag
	if flag.Value.Type() != "bool" {
		t.Errorf("strict-certs flag type = %q, want 'bool'", flag.Value.Type())
	}
}

// TestBrowserCommand_HostFlag tests that the --host flag is properly configured.
func TestBrowserCommand_HostFlag(t *testing.T) {
	flag := browserCmd.PersistentFlags().Lookup("host")
	if flag == nil {
		t.Fatal("host flag not found")

		return
	}

	if flag.DefValue != "localhost" {
		t.Errorf("host flag default value = %q, want 'localhost'", flag.DefValue)
	}
}

// TestBrowserCommand_PortFlag tests that the --port flag is properly configured.
func TestBrowserCommand_PortFlag(t *testing.T) {
	flag := browserCmd.PersistentFlags().Lookup("port")
	if flag == nil {
		t.Fatal("port flag not found")

		return
	}

	if flag.DefValue != "0" {
		t.Errorf("port flag default value = %q, want '0'", flag.DefValue)
	}
}

// TestBrowserCommand_HeadlessFlag tests that the --headless flag is properly configured.
func TestBrowserCommand_HeadlessFlag(t *testing.T) {
	flag := browserCmd.PersistentFlags().Lookup("headless")
	if flag == nil {
		t.Fatal("headless flag not found")

		return
	}

	if flag.DefValue != "false" {
		t.Errorf("headless flag default value = %q, want 'false'", flag.DefValue)
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"truncated", "hello world", 8, "hello..."},
		{"very short max", "hello world", 6, "hel..."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateString(tt.input, tt.maxLen)
			if got != tt.expected {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.expected)
			}
		})
	}
}

// TestBrowserCommand_KeepAliveFlag tests that the --keep-alive flag is properly configured.
func TestBrowserCommand_KeepAliveFlag(t *testing.T) {
	flag := browserCmd.PersistentFlags().Lookup("keep-alive")
	if flag == nil {
		t.Fatal("keep-alive flag not found")

		return
	}

	if flag.DefValue != "false" {
		t.Errorf("keep-alive flag default value = %q, want 'false'", flag.DefValue)
	}

	if flag.Value.Type() != "bool" {
		t.Errorf("keep-alive flag type = %q, want 'bool'", flag.Value.Type())
	}
}
