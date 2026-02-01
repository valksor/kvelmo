//go:build !testbinary
// +build !testbinary

package commands

import (
	"testing"

	"github.com/valksor/go-mehrhof/internal/mcp"
)

func TestMCPCommand_Properties(t *testing.T) {
	if mcpCmd.Use != "mcp" {
		t.Errorf("Use = %q, want %q", mcpCmd.Use, "mcp")
	}

	if mcpCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if mcpCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if mcpCmd.RunE == nil {
		t.Error("RunE not set")
	}

	if mcpCmd.Hidden {
		t.Error("MCP command should not be hidden")
	}
}

func TestMCPCommand_RegisteredInRoot(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "mcp" {
			found = true

			break
		}
	}
	if !found {
		t.Error("mcp command not registered in root command")
	}
}

func TestMCPCommand_ShortDescription(t *testing.T) {
	expected := "Start MCP server (for AI agents)"
	if mcpCmd.Short != expected {
		t.Errorf("Short = %q, want %q", mcpCmd.Short, expected)
	}
}

func TestRegisterSafeCommands(t *testing.T) {
	// Create a tool registry with the root command
	registry := mcp.NewToolRegistry(rootCmd)

	// Register safe commands
	registerSafeCommands(registry)

	// Verify tools were registered (should have at least the known safe commands)
	tools := registry.ListTools()
	if len(tools) == 0 {
		t.Error("registerSafeCommands() registered no tools")
	}

	// Should have a reasonable number of tools (status, list, guide, browser*, config, providers, etc.)
	if len(tools) < 5 {
		t.Errorf("Expected at least 5 registered tools, got %d", len(tools))
	}
}
