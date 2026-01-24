// Package mcp implements the Model Context Protocol (MCP) server for Mehrhof.
// MCP allows AI agents to discover and call tools exposed by the server.
package mcp

import (
	"encoding/json"
	"errors"
)

const (
	// ProtocolVersion is the MCP protocol version supported by this server.
	// Must match assern's protocol version for compatibility.
	ProtocolVersion = "2025-06-18"
)

// ErrDisabled is returned when MCP is disabled.
var ErrDisabled = errors.New("MCP server is disabled in this build")

// Protocol-specific types for MCP (Model Context Protocol).
// Based on JSON-RPC 2.0 with MCP-specific extensions.

// Request represents a JSON-RPC 2.0 request (MCP compatible).
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"` // Can be string, number, or null
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response represents a JSON-RPC 2.0 response (MCP compatible).
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"` // Must match request ID
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// Error represents a JSON-RPC 2.0 error.
type Error struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *Error) Error() string {
	return e.Message
}

// Standard JSON-RPC error codes.
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

// MCP-specific method names.
const (
	MethodInitialize    = "initialize"
	MethodToolsList     = "tools/list"
	MethodToolsCall     = "tools/call"
	MethodResourcesList = "resources/list"
	MethodResourcesRead = "resources/read"
	MethodPromptsList   = "prompts/list"
	MethodPromptsGet    = "prompts/get"
	MethodShutdown      = "shutdown"
	MethodNotifications = "notifications/"
)

// ServerInfo describes the MCP server.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ServerCapabilities describes server capabilities.
type ServerCapabilities struct {
	Tools     *ToolsCapabilities     `json:"tools,omitempty"`
	Resources *ResourcesCapabilities `json:"resources,omitempty"`
	Prompts   *PromptsCapabilities   `json:"prompts,omitempty"`
}

// ToolsCapabilities describes tools capability.
type ToolsCapabilities struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourcesCapabilities describes resources capability.
type ResourcesCapabilities struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// PromptsCapabilities describes prompts capability.
type PromptsCapabilities struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ClientCapabilities describes client capabilities.
type ClientCapabilities struct {
	Experimental map[string]interface{} `json:"experimental,omitempty"`
	Roots        *RootsCapability       `json:"roots,omitempty"`
	Sampling     *SamplingCapability    `json:"sampling,omitempty"`
}

// RootsCapability describes client's roots capability.
type RootsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// SamplingCapability describes client's sampling capability.
type SamplingCapability struct{}

// InitializeParams contains initialization parameters from client.
type InitializeParams struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ClientCapabilities `json:"capabilities"`
	ClientInfo      ClientInfo         `json:"clientInfo"`
}

// ClientInfo describes the MCP client.
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// InitializeResult contains initialization response from server.
type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
}

// Tool describes an available MCP tool.
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// ToolsListParams contains parameters for listing tools.
type ToolsListParams struct {
	Cursor string `json:"cursor,omitempty"`
}

// ToolsListResult contains the result of listing tools.
type ToolsListResult struct {
	Tools []Tool `json:"tools"`
	// NextCursor string `json:"nextCursor,omitempty"` // For pagination
}

// ToolCallParams contains parameters for calling a tool.
type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
	// RetryThreshold map[string]interface{} `json:"_meta,omitempty"` // For progress tracking
}

// ToolCallResult contains the result of calling a tool.
type ToolCallResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

// ContentBlock represents a block of content in a tool result.
type ContentBlock struct {
	Type     string                 `json:"type"`
	Text     string                 `json:"text,omitempty"`
	Data     string                 `json:"data,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Content types.
const (
	ContentTypeText     = "text"
	ContentTypeImage    = "image"
	ContentTypeResource = "resource"
)

// EmptyResult is an empty result for shutdown.
type EmptyResult struct{}
