//go:build !no_mcp
// +build !no_mcp

package mcp

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

// TestServerInitialize tests the initialize request/response flow.
func TestServerInitialize(t *testing.T) {
	// Create server
	rootCmd := &cobra.Command{Use: "test"}
	toolRegistry := NewToolRegistry(rootCmd)
	server := NewServer(toolRegistry)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test initialize request
	req := &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodInitialize,
		Params:  json.RawMessage(`{"protocolVersion":"` + ProtocolVersion + `","capabilities":{},"clientInfo":{"name":"test-client","version":"1.0"}}`),
	}

	reqData, _ := json.Marshal(req) //nolint:errchkjson // Test code - RawMessage contains valid JSON
	resp := server.handleRequest(ctx, string(reqData)+("\n"))

	if resp == nil {
		t.Fatal("Expected non-nil response")
	}

	if resp.Error != nil {
		t.Fatalf("Initialize failed: %v", resp.Error)
	}

	// Verify response
	if resp.JSONRPC != "2.0" {
		t.Errorf("JSON-RPC version mismatch: got %s", resp.JSONRPC)
	}

	if string(resp.ID) != "1" {
		t.Errorf("ID mismatch: got %s", string(resp.ID))
	}

	if resp.Result == nil {
		t.Fatal("No result in response")
	}

	// Unmarshal and verify result
	var result InitializeResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if result.ProtocolVersion != "2024-11-05" {
		t.Errorf("Protocol version mismatch: got %s", result.ProtocolVersion)
	}

	if result.ServerInfo.Name != "go-mehrhof" {
		t.Errorf("Server name mismatch: got %s", result.ServerInfo.Name)
	}
}

// TestServerToolsList tests the tools/list request.
func TestServerToolsList(t *testing.T) {
	rootCmd := &cobra.Command{Use: "test"}
	toolRegistry := NewToolRegistry(rootCmd)

	// Register a test tool
	toolRegistry.RegisterDirectTool(
		"test_tool",
		"A test tool",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"arg1": map[string]interface{}{
					"type":        "string",
					"description": "First argument",
				},
			},
			"required": []string{"arg1"},
		},
		func(ctx context.Context, args map[string]interface{}) (*ToolCallResult, error) {
			return textResult("test result"), nil
		},
	)

	server := NewServer(toolRegistry)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test tools/list before initialize (should fail)
	req := &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  MethodToolsList,
		Params:  json.RawMessage(`{}`),
	}

	reqData, _ := json.Marshal(req) //nolint:errchkjson // Test code - RawMessage contains valid JSON
	resp := server.handleRequest(ctx, string(reqData)+("\n"))

	if resp == nil {
		t.Fatal("Expected error response for tools/list before initialize")
	}

	if resp.Error == nil {
		t.Fatal("Expected error for tools/list before initialize")
	}

	if resp.Error.Code != InvalidRequest {
		t.Fatalf("Expected InvalidRequest error, got %d", resp.Error.Code)
	}

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

	// Now test tools/list
	req2 := &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`2`),
		Method:  MethodToolsList,
		Params:  json.RawMessage(`{}`),
	}

	req2Data, _ := json.Marshal(req2) //nolint:errchkjson // Test code - RawMessage contains valid JSON
	resp2 := server.handleRequest(ctx, string(req2Data)+("\n"))

	if resp2.Error != nil {
		t.Fatalf("tools/list failed: %v", resp2.Error)
	}

	var result ToolsListResult
	if err := json.Unmarshal(resp2.Result, &result); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if len(result.Tools) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(result.Tools))
	}

	if result.Tools[0].Name != "test_tool" {
		t.Errorf("Expected tool name 'test_tool', got '%s'", result.Tools[0].Name)
	}
}

// TestServerToolsCall tests the tools/call request.
func TestServerToolsCall(t *testing.T) {
	rootCmd := &cobra.Command{Use: "test"}
	toolRegistry := NewToolRegistry(rootCmd)

	// Register a test tool
	called := false
	toolRegistry.RegisterDirectTool(
		"test_tool",
		"A test tool",
		map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
			"required":   []string{},
		},
		func(ctx context.Context, args map[string]interface{}) (*ToolCallResult, error) {
			called = true

			return textResult("test result"), nil
		},
	)

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

	// Test tools/call
	req := &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`2`),
		Method:  MethodToolsCall,
		Params:  json.RawMessage(`{"name":"test_tool","arguments":{}}`),
	}

	reqData, _ := json.Marshal(req) //nolint:errchkjson // Test code - RawMessage contains valid JSON
	resp := server.handleRequest(ctx, string(reqData)+("\n"))

	if resp.Error != nil {
		t.Fatalf("tools/call failed: %v", resp.Error)
	}

	if !called {
		t.Error("Tool executor was not called")
	}

	var result ToolCallResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if len(result.Content) != 1 {
		t.Fatalf("Expected 1 content block, got %d", len(result.Content))
	}

	if result.Content[0].Text != "test result" {
		t.Errorf("Expected text 'test result', got '%s'", result.Content[0].Text)
	}
}

// TestServerConcurrentCalls tests concurrent tool calls for thread safety.
func TestServerConcurrentCalls(t *testing.T) {
	rootCmd := &cobra.Command{Use: "test"}
	toolRegistry := NewToolRegistry(rootCmd)

	// Register multiple test tools
	for i := range 5 {
		idx := i
		toolRegistry.RegisterDirectTool(
			"test_tool_"+string(rune('0'+idx)),
			"A test tool",
			map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
				"required":   []string{},
			},
			func(ctx context.Context, args map[string]interface{}) (*ToolCallResult, error) {
				// Simulate some work
				time.Sleep(10 * time.Millisecond)

				return textResult("result"), nil
			},
		)
	}

	server := NewServer(toolRegistry)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Initialize
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

	// Make concurrent calls
	var wg sync.WaitGroup
	errors := make(chan error, 5)

	for i := range 5 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			req := &Request{
				JSONRPC: "2.0",
				ID:      json.RawMessage(string(rune('2' + idx))),
				Method:  MethodToolsCall,
				Params:  json.RawMessage(`{"name":"test_tool_` + string(rune('0'+idx)) + `","arguments":{}}`),
			}

			reqData, _ := json.Marshal(req) //nolint:errchkjson // Test code - RawMessage contains valid JSON
			resp := server.handleRequest(ctx, string(reqData)+("\n"))

			if resp.Error != nil {
				errors <- resp.Error

				return
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Fatalf("Concurrent call error: %v", err)
	}
}

// TestServerInvalidJSON tests handling of invalid JSON.
func TestServerInvalidJSON(t *testing.T) {
	rootCmd := &cobra.Command{Use: "test"}
	toolRegistry := NewToolRegistry(rootCmd)
	server := NewServer(toolRegistry)

	ctx := context.Background()

	// Test invalid JSON
	resp := server.handleRequest(ctx, "not json\n")

	if resp == nil {
		t.Fatal("Expected error response for invalid JSON")
	}

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

	if resp == nil {
		t.Fatal("Expected error response for invalid JSON-RPC version")
	}

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

	if resp == nil {
		t.Fatal("Expected error response for unknown method")
	}

	if resp.Error == nil {
		t.Fatal("Expected error for unknown method")
	}

	if resp.Error.Code != MethodNotFound {
		t.Fatalf("Expected MethodNotFound, got %d", resp.Error.Code)
	}
}

// TestServerShutdown tests the shutdown request.
func TestServerShutdown(t *testing.T) {
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

	// Test shutdown
	req := &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`2`),
		Method:  MethodShutdown,
		Params:  json.RawMessage(`{}`),
	}

	reqData, _ := json.Marshal(req) //nolint:errchkjson // Test code - RawMessage contains valid JSON
	resp := server.handleRequest(ctx, string(reqData)+("\n"))

	if resp.Error != nil {
		t.Fatalf("Shutdown failed: %v", resp.Error)
	}

	// Server should no longer be initialized
	req2 := &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`3`),
		Method:  MethodToolsList,
		Params:  json.RawMessage(`{}`),
	}

	req2Data, _ := json.Marshal(req2) //nolint:errchkjson // Test code - RawMessage contains valid JSON
	resp2 := server.handleRequest(ctx, string(req2Data)+("\n"))

	if resp2.Error == nil {
		t.Fatal("Expected error after shutdown")
	}

	if resp2.Error.Code != InvalidRequest {
		t.Fatalf("Expected InvalidRequest after shutdown, got %d", resp2.Error.Code)
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
	if resp == nil {
		t.Fatal("Expected error response for empty line")
	}

	if resp.Error == nil {
		t.Error("Expected error for empty line")
	}

	// Should be a parse error
	if resp.Error.Code != ParseError {
		t.Errorf("Expected ParseError (%d), got %d", ParseError, resp.Error.Code)
	}

	// Multiple empty lines - same behavior
	resp = server.handleRequest(ctx, "\n\n\n")
	if resp == nil {
		t.Fatal("Expected error response for multiple empty lines")
	}

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
