//go:build no_mcp
// +build no_mcp

package mcp

// RegisterRegistryTools is a no-op when MCP is disabled.
func RegisterRegistryTools(registry *ToolRegistry, cond interface{}) {
	// No-op in stub
}
