//go:build !testbinary

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

	// Verify tools were registered
	tools := registry.ListTools()
	if len(tools) == 0 {
		t.Error("registerSafeCommands() registered no tools")
	}

	// Should have a reasonable number of tools (~27 total)
	if len(tools) < 20 {
		t.Errorf("Expected at least 20 registered tools, got %d", len(tools))
	}
}

func TestRegisterSafeCommands_IncludesStatusCommand(t *testing.T) {
	registry := mcp.NewToolRegistry(rootCmd)
	registerSafeCommands(registry)

	tools := registry.ListTools()
	found := false
	for _, tool := range tools {
		if tool.Name == "status" {
			found = true

			break
		}
	}
	if !found {
		t.Error("status command should be registered")
	}
}

func TestRegisterSafeCommands_IncludesBrowserCommands(t *testing.T) {
	registry := mcp.NewToolRegistry(rootCmd)
	registerSafeCommands(registry)

	tools := registry.ListTools()

	// Check for key browser commands (uses underscores: "mehr browser status" -> "browser_status")
	requiredBrowserTools := []string{
		"browser_status",
		"browser_tabs",
		"browser_goto",
		"browser_screenshot",
		"browser_click",
	}

	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}

	for _, required := range requiredBrowserTools {
		if !toolNames[required] {
			t.Errorf("browser command %q should be registered", required)
		}
	}
}

func TestRegisterSafeCommands_ExcludesDangerousCommands(t *testing.T) {
	registry := mcp.NewToolRegistry(rootCmd)
	registerSafeCommands(registry)

	tools := registry.ListTools()

	// Commands that should NOT be registered (dangerous or not useful for MCP)
	excludedCommands := []string{
		"mcp",         // MCP server itself
		"interactive", // REPL mode
		"serve",       // Web server
		"start",       // Task workflow
		"finish",      // Task workflow
		"abandon",     // Task workflow
		"plan",        // Task workflow
		"implement",   // Task workflow
		"review",      // Task workflow
	}

	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}

	for _, excluded := range excludedCommands {
		if toolNames[excluded] {
			t.Errorf("dangerous command %q should NOT be registered", excluded)
		}
	}
}
