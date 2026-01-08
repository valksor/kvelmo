//go:build !no_mcp
// +build !no_mcp

package mcp

import (
	"context"
	"errors"
	"log/slog"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/provider"
)

// RegisterRegistryTools registers agent and provider registry tools.
// Requires a conductor with registered providers and agents.
func RegisterRegistryTools(registry *ToolRegistry, cond *conductor.Conductor) {
	agentReg := cond.GetAgentRegistry()
	providerReg := cond.GetProviderRegistry()

	// agents_list
	registry.RegisterDirectTool(
		"agents_list",
		"List all registered AI agents",
		map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
			"required":   []string{},
		},
		func(ctx context.Context, args map[string]interface{}) (*ToolCallResult, error) {
			agents := agentReg.List()

			result := make([]map[string]interface{}, 0, len(agents))
			for _, name := range agents {
				a, err := agentReg.Get(name)
				if err != nil {
					// Skip agents that can't be loaded
					slog.Warn("Failed to get agent, skipping", "agent", name, "error", err)

					continue
				}
				// Check if agent has metadata provider for description
				description := name
				result = append(result, map[string]interface{}{
					"name":        name,
					"description": description,
					"available":   a.Available() == nil,
				})
			}

			return jsonResult(map[string]interface{}{"agents": result}), nil
		},
	)

	// agents_get_default
	registry.RegisterDirectTool(
		"agents_get_default",
		"Get the default agent",
		map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
			"required":   []string{},
		},
		func(ctx context.Context, args map[string]interface{}) (*ToolCallResult, error) {
			def, err := agentReg.GetDefault()
			if err != nil {
				return errorResult(err), nil
			}

			return jsonResult(map[string]interface{}{
				"name": def.Name(),
			}), nil
		},
	)

	// providers_list
	registry.RegisterDirectTool(
		"providers_list",
		"List all registered task providers",
		map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
			"required":   []string{},
		},
		func(ctx context.Context, args map[string]interface{}) (*ToolCallResult, error) {
			providers := providerReg.List()

			result := make([]map[string]interface{}, 0, len(providers))
			for _, p := range providers {
				result = append(result, map[string]interface{}{
					"name":         p.Name,
					"description":  p.Description,
					"schemes":      p.Schemes,
					"capabilities": p.Capabilities,
					"priority":     p.Priority,
				})
			}

			return jsonResult(map[string]interface{}{"providers": result}), nil
		},
	)

	// providers_resolve
	registry.RegisterDirectTool(
		"providers_resolve",
		"Resolve a provider from a task reference",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"reference": map[string]interface{}{
					"type":        "string",
					"description": "Task reference (e.g., 'file:task.md', 'github:123')",
				},
				"default_provider": map[string]interface{}{
					"type":        "string",
					"description": "Default provider to use if no scheme specified",
				},
			},
			"required": []string{"reference"},
		},
		func(ctx context.Context, args map[string]interface{}) (*ToolCallResult, error) {
			ref, _ := args["reference"].(string)
			defaultProvider, _ := args["default_provider"].(string)

			if ref == "" {
				return errorResult(errors.New("reference is required")), nil
			}

			// Create empty config for resolution
			cfg := provider.Config{}

			// Try to resolve
			_, providerName, err := providerReg.Resolve(ctx, ref, cfg, provider.ResolveOptions{
				DefaultProvider: defaultProvider,
			})
			if err != nil {
				return errorResult(err), nil
			}

			return jsonResult(map[string]interface{}{
				"reference": ref,
				"provider":  providerName,
			}), nil
		},
	)
}
