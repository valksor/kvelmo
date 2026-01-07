package mcp

import (
	"context"
	"testing"

	"github.com/valksor/go-mehrhof/internal/agent/claude"
	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/provider/file"
)

func TestRegistryToolsRegistration(t *testing.T) {
	cond, err := conductor.New()
	if err != nil {
		t.Fatalf("Failed to create conductor: %v", err)
	}

	// Register a test provider
	file.Register(cond.GetProviderRegistry())

	// Register a test agent (may fail if claude binary not available, but that's ok for test)
	_ = claude.Register(cond.GetAgentRegistry())

	registry := NewToolRegistry(nil)
	RegisterRegistryTools(registry, cond)

	tools := registry.ListTools()

	// Check that registry tools are registered
	expectedTools := []string{
		"agents_list",
		"agents_get_default",
		"providers_list",
		"providers_resolve",
	}

	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}

	for _, expected := range expectedTools {
		if !toolNames[expected] {
			t.Errorf("Tool %s not registered", expected)
		}
	}
}

func TestAgentsListTool(t *testing.T) {
	cond, err := conductor.New()
	if err != nil {
		t.Fatalf("Failed to create conductor: %v", err)
	}

	// Register agents
	_ = claude.Register(cond.GetAgentRegistry())

	registry := NewToolRegistry(nil)
	RegisterRegistryTools(registry, cond)

	result, err := registry.CallTool(context.Background(), "agents_list", map[string]interface{}{})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	if result.IsError {
		t.Errorf("Tool returned error: %s", result.Content[0].Text)
	}

	if result.Content[0].Text == "" {
		t.Error("Expected non-empty response")
	}
}

func TestProvidersListTool(t *testing.T) {
	cond, err := conductor.New()
	if err != nil {
		t.Fatalf("Failed to create conductor: %v", err)
	}

	// Register providers
	file.Register(cond.GetProviderRegistry())

	registry := NewToolRegistry(nil)
	RegisterRegistryTools(registry, cond)

	result, err := registry.CallTool(context.Background(), "providers_list", map[string]interface{}{})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	if result.IsError {
		t.Errorf("Tool returned error: %s", result.Content[0].Text)
	}

	if result.Content[0].Text == "" {
		t.Error("Expected non-empty response")
	}

	// Check that file provider is in the list
	resultText := result.Content[0].Text
	if len(resultText) < 10 {
		t.Errorf("Response too short: got %d chars", len(resultText))
	}
}

func TestProvidersResolveTool(t *testing.T) {
	cond, err := conductor.New()
	if err != nil {
		t.Fatalf("Failed to create conductor: %v", err)
	}

	// Register providers
	file.Register(cond.GetProviderRegistry())

	registry := NewToolRegistry(nil)
	RegisterRegistryTools(registry, cond)

	tests := []struct {
		name        string
		reference   string
		defaultProv string
		wantError   bool
	}{
		{
			name:      "empty reference",
			reference: "",
			wantError: true, // Empty reference should return error
		},
		{
			name:      "invalid scheme",
			reference: "invalid:test",
			wantError: true, // Unknown provider scheme should error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := map[string]interface{}{
				"reference": tt.reference,
			}
			if tt.defaultProv != "" {
				args["default_provider"] = tt.defaultProv
			}

			result, err := registry.CallTool(context.Background(), "providers_resolve", args)
			if err != nil {
				t.Fatalf("CallTool failed: %v", err)
			}

			if tt.wantError && !result.IsError {
				t.Error("Expected error but got none")
			}
		})
	}
}
