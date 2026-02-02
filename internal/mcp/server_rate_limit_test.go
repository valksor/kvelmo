package mcp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

// TestServerRateLimitExceeded verifies that rate limiting rejects requests when the limit is exhausted.
func TestServerRateLimitExceeded(t *testing.T) {
	rootCmd := &cobra.Command{Use: "test"}
	toolRegistry := NewToolRegistry(rootCmd)

	// Register a simple tool
	toolRegistry.RegisterDirectTool(
		"echo",
		"Echo tool",
		map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
			"required":   []string{},
		},
		func(ctx context.Context, args map[string]interface{}) (*ToolCallResult, error) {
			return textResult("ok"), nil
		},
	)

	// Create server with extremely low rate limit: 0.001 req/s, burst 1
	// This means after the first request uses the burst, subsequent requests must wait ~1000s
	server := NewServer(toolRegistry, WithRateLimit(0.001, 1))

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
	initResp := server.handleRequest(ctx, string(initReqData)+"\n")

	if initResp.Error != nil {
		t.Fatalf("Initialize failed: %v", initResp.Error)
	}

	// First call should succeed (uses burst token)
	req := &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`2`),
		Method:  MethodToolsCall,
		Params:  json.RawMessage(`{"name":"echo","arguments":{}}`),
	}

	reqData, _ := json.Marshal(req) //nolint:errchkjson // Test code - RawMessage contains valid JSON
	resp := server.handleRequest(ctx, string(reqData)+"\n")

	if resp.Error != nil {
		t.Fatalf("First call should succeed: %v", resp.Error)
	}

	// Second call with a tight deadline should fail — burst is exhausted and rate is too low
	tightCtx, tightCancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer tightCancel()

	req2 := &Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`3`),
		Method:  MethodToolsCall,
		Params:  json.RawMessage(`{"name":"echo","arguments":{}}`),
	}

	req2Data, _ := json.Marshal(req2) //nolint:errchkjson // Test code - RawMessage contains valid JSON
	resp2 := server.handleRequest(tightCtx, string(req2Data)+"\n")

	if resp2.Error == nil {
		t.Fatal("Expected rate limit error for second call with tight deadline")
	}

	if resp2.Error.Code != InternalError {
		t.Errorf("Expected InternalError (%d), got %d", InternalError, resp2.Error.Code)
	}
}
