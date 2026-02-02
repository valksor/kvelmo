package mcp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

// TestServerInvalidJSON tests handling of invalid JSON.
func TestServerInvalidJSON(t *testing.T) {
	rootCmd := &cobra.Command{Use: "test"}
	toolRegistry := NewToolRegistry(rootCmd)
	server := NewServer(toolRegistry)

	ctx := context.Background()

	// Test invalid JSON
	resp := server.handleRequest(ctx, "not json\n")

	if resp.Error == nil {
		t.Fatal("Expected error for invalid JSON")
	}

	if resp.Error.Code != ParseError {
		t.Fatalf("Expected ParseError, got %d", resp.Error.Code)
	}
}

// TestServerInvalidJSONRPCVersion tests handling of invalid JSON-RPC version.
func TestServerInvalidJSONRPCVersion(t *testing.T) {
	rootCmd := &cobra.Command{Use: "test"}
	toolRegistry := NewToolRegistry(rootCmd)
	server := NewServer(toolRegistry)

	ctx := context.Background()

	req := `{"jsonrpc":"1.0","id":1,"method":"initialize"}`
	resp := server.handleRequest(ctx, req+"\n")

	if resp.Error == nil {
		t.Fatal("Expected error for invalid JSON-RPC version")
	}

	if resp.Error.Code != InvalidRequest {
		t.Fatalf("Expected InvalidRequest, got %d", resp.Error.Code)
	}
}

// TestServerMethodNotFound tests handling of unknown methods.
func TestServerMethodNotFound(t *testing.T) {
	rootCmd := &cobra.Command{Use: "test"}
	toolRegistry := NewToolRegistry(rootCmd)
	server := NewServer(toolRegistry)

	ctx := context.Background()

	req := &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "unknown/method",
		Params:  json.RawMessage(`{}`),
	}

	reqData, _ := json.Marshal(req) //nolint:errchkjson // Test code - RawMessage contains valid JSON
	resp := server.handleRequest(ctx, string(reqData)+("\n"))

	if resp.Error == nil {
		t.Fatal("Expected error for unknown method")
	}

	if resp.Error.Code != MethodNotFound {
		t.Fatalf("Expected MethodNotFound, got %d", resp.Error.Code)
	}
}

// TestServerEmptyLines tests handling of empty lines.
func TestServerEmptyLines(t *testing.T) {
	rootCmd := &cobra.Command{Use: "test"}
	toolRegistry := NewToolRegistry(rootCmd)
	server := NewServer(toolRegistry)

	ctx := context.Background()

	// Empty line should be treated as invalid JSON by handleRequest
	// (Note: Serve() filters empty lines before calling handleRequest)
	resp := server.handleRequest(ctx, "\n")
	if resp.Error == nil {
		t.Error("Expected error for empty line")
	}

	// Should be a parse error
	if resp.Error.Code != ParseError {
		t.Errorf("Expected ParseError (%d), got %d", ParseError, resp.Error.Code)
	}

	// Multiple empty lines - same behavior
	resp = server.handleRequest(ctx, "\n\n\n")
	if resp.Error == nil {
		t.Error("Expected error for multiple empty lines")
	}
}

// TestServerToolNotFound tests calling a non-existent tool.
func TestServerToolNotFound(t *testing.T) {
	rootCmd := &cobra.Command{Use: "test"}
	toolRegistry := NewToolRegistry(rootCmd)
	server := NewServer(toolRegistry)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Initialize first
	initReq := &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodInitialize,
		Params:  json.RawMessage(`{"protocolVersion":"` + ProtocolVersion + `","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}`),
	}

	initReqData, _ := json.Marshal(initReq) //nolint:errchkjson // Test code - RawMessage contains valid JSON
	initResp := server.handleRequest(ctx, string(initReqData)+("\n"))

	if initResp.Error != nil {
		t.Fatalf("Initialize failed: %v", initResp.Error)
	}

	// Try to call non-existent tool
	req := &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`2`),
		Method:  MethodToolsCall,
		Params:  json.RawMessage(`{"name":"nonexistent_tool","arguments":{}}`),
	}

	reqData, _ := json.Marshal(req) //nolint:errchkjson // Test code - RawMessage contains valid JSON
	resp := server.handleRequest(ctx, string(reqData)+("\n"))

	if resp.Error == nil {
		t.Fatal("Expected error for non-existent tool")
	}

	if resp.Error.Code != InternalError {
		t.Fatalf("Expected InternalError, got %d", resp.Error.Code)
	}
}

// TestServerInvalidParams tests handling of invalid params.
func TestServerInvalidParams(t *testing.T) {
	rootCmd := &cobra.Command{Use: "test"}
	toolRegistry := NewToolRegistry(rootCmd)
	server := NewServer(toolRegistry)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Initialize first
	initReq := &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodInitialize,
		Params:  json.RawMessage(`{"protocolVersion":"` + ProtocolVersion + `","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}`),
	}

	initReqData, _ := json.Marshal(initReq) //nolint:errchkjson // Test code - RawMessage contains valid JSON
	initResp := server.handleRequest(ctx, string(initReqData)+("\n"))

	if initResp.Error != nil {
		t.Fatalf("Initialize failed: %v", initResp.Error)
	}

	// Test tools/call with invalid params
	req := &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`2`),
		Method:  MethodToolsCall,
		Params:  json.RawMessage(`"invalid"`), // Should be an object
	}

	reqData, _ := json.Marshal(req) //nolint:errchkjson // Test code - RawMessage contains valid JSON
	resp := server.handleRequest(ctx, string(reqData)+("\n"))

	if resp.Error == nil {
		t.Fatal("Expected error for invalid params")
	}

	if resp.Error.Code != InvalidParams {
		t.Fatalf("Expected InvalidParams, got %d", resp.Error.Code)
	}
}

// TestServerReInitialize verifies that a second initialize succeeds and the server remains functional.
func TestServerReInitialize(t *testing.T) {
	rootCmd := &cobra.Command{Use: "test"}
	toolRegistry := NewToolRegistry(rootCmd)
	server := NewServer(toolRegistry)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	initParams := json.RawMessage(`{"protocolVersion":"` + ProtocolVersion + `","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}`)

	// First initialize
	req1 := &Request{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: MethodInitialize, Params: initParams}
	req1Data, _ := json.Marshal(req1) //nolint:errchkjson // Test code - RawMessage contains valid JSON
	resp1 := server.handleRequest(ctx, string(req1Data)+"\n")

	if resp1.Error != nil {
		t.Fatalf("First initialize failed: %v", resp1.Error)
	}

	// Second initialize (re-init)
	req2 := &Request{JSONRPC: "2.0", ID: json.RawMessage(`2`), Method: MethodInitialize, Params: initParams}
	req2Data, _ := json.Marshal(req2) //nolint:errchkjson // Test code - RawMessage contains valid JSON
	resp2 := server.handleRequest(ctx, string(req2Data)+"\n")

	if resp2.Error != nil {
		t.Fatalf("Re-initialize failed: %v", resp2.Error)
	}

	// Verify tools/list still works after re-init
	listReq := &Request{JSONRPC: "2.0", ID: json.RawMessage(`3`), Method: MethodToolsList, Params: json.RawMessage(`{}`)}
	listData, _ := json.Marshal(listReq) //nolint:errchkjson // Test code - RawMessage contains valid JSON
	listResp := server.handleRequest(ctx, string(listData)+"\n")

	if listResp.Error != nil {
		t.Fatalf("tools/list after re-init failed: %v", listResp.Error)
	}
}

// TestServerProtocolVersionMismatch verifies that a wrong protocol version is rejected.
func TestServerProtocolVersionMismatch(t *testing.T) {
	rootCmd := &cobra.Command{Use: "test"}
	toolRegistry := NewToolRegistry(rootCmd)
	server := NewServer(toolRegistry)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodInitialize,
		Params:  json.RawMessage(`{"protocolVersion":"1999-01-01","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}`),
	}

	reqData, _ := json.Marshal(req) //nolint:errchkjson // Test code - RawMessage contains valid JSON
	resp := server.handleRequest(ctx, string(reqData)+"\n")

	if resp.Error == nil {
		t.Fatal("Expected error for protocol version mismatch")
	}

	if resp.Error.Code != InvalidParams {
		t.Fatalf("Expected InvalidParams (%d), got %d", InvalidParams, resp.Error.Code)
	}

	if resp.Error.Message == "" {
		t.Error("Expected non-empty error message")
	}
}
