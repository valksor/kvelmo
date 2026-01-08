//go:build no_mcp
// +build no_mcp

package mcp

import (
	"context"
)

// ServerOption stub when MCP is disabled.
type ServerOption func(*Server)

// WithRateLimit stub returns empty function.
func WithRateLimit(ratePerSec float64, burst int) ServerOption {
	return func(s *Server) {}
}

// Server stub when MCP is disabled.
type Server struct {
	toolRegistry *ToolRegistry
}

// NewServer creates a stub server when MCP is disabled.
func NewServer(toolRegistry *ToolRegistry, opts ...ServerOption) *Server {
	return &Server{
		toolRegistry: toolRegistry,
	}
}

// Serve returns an error - MCP is disabled.
func (s *Server) Serve(ctx context.Context) error {
	return ErrDisabled
}
