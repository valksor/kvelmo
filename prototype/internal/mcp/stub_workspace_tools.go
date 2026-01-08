//go:build no_mcp
// +build no_mcp

package mcp

// RegisterWorkspaceTools is a no-op when MCP is disabled.
func RegisterWorkspaceTools(registry *ToolRegistry) {
	// No-op in stub
}
