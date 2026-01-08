//go:build no_mcp
// +build no_mcp

package mcp

import (
	"context"
	"sync"

	"github.com/spf13/cobra"
)

// ToolRegistry stub when MCP is disabled.
type ToolRegistry struct {
	tools   map[string]*ToolWrapper
	rootCmd *cobra.Command
	mu      sync.RWMutex
}

// ToolWrapper stub when MCP is disabled.
type ToolWrapper struct {
	Tool      Tool
	Command   *cobra.Command
	ArgMapper func(map[string]interface{}) []string
	mu        sync.Mutex
	Executor  func(ctx context.Context, args map[string]interface{}) (*ToolCallResult, error)
}

// NewToolRegistry creates a stub tool registry.
func NewToolRegistry(rootCmd *cobra.Command) *ToolRegistry {
	return &ToolRegistry{
		tools:   make(map[string]*ToolWrapper),
		rootCmd: rootCmd,
	}
}

// RegisterCommand is a no-op when MCP is disabled.
func (r *ToolRegistry) RegisterCommand(cmd *cobra.Command, argMapper func(map[string]interface{}) []string) {
	// No-op in stub
}

// RegisterCommands is a no-op when MCP is disabled.
func (r *ToolRegistry) RegisterCommands(commands []*cobra.Command, argMapper func(map[string]interface{}) []string) {
	// No-op in stub
}

// RegisterDirectTool is a no-op when MCP is disabled.
func (r *ToolRegistry) RegisterDirectTool(name, description string, inputSchema map[string]interface{}, executor func(ctx context.Context, args map[string]interface{}) (*ToolCallResult, error)) {
	// No-op in stub
}

// ListTools returns empty list when MCP is disabled.
func (r *ToolRegistry) ListTools() []Tool {
	return []Tool{}
}

// CallTool returns an error when MCP is disabled.
func (r *ToolRegistry) CallTool(ctx context.Context, name string, args map[string]interface{}) (*ToolCallResult, error) {
	return nil, ErrDisabled
}

// DefaultArgMapper returns nil when MCP is disabled.
func DefaultArgMapper(args map[string]interface{}) []string {
	return nil
}
