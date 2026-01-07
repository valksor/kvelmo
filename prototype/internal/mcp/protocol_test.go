package mcp

import (
	"encoding/json"
	"testing"
)

func TestRequestMarshal(t *testing.T) {
	req := Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params:  json.RawMessage(`{"test":"value"}`),
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var unmarshaled Request
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if unmarshaled.JSONRPC != "2.0" {
		t.Errorf("JSONRPC version mismatch: got %s", unmarshaled.JSONRPC)
	}
	if unmarshaled.Method != "initialize" {
		t.Errorf("Method mismatch: got %s", unmarshaled.Method)
	}
}

func TestResponseMarshal(t *testing.T) {
	result := json.RawMessage(`{"test":"value"}`)
	resp := Response{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Result:  result,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var unmarshaled Response
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if unmarshaled.JSONRPC != "2.0" {
		t.Errorf("JSONRPC version mismatch: got %s", unmarshaled.JSONRPC)
	}
}

func TestResponseError(t *testing.T) {
	errResp := &Error{
		Code:    MethodNotFound,
		Message: "Method not found",
	}

	resp := Response{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Error:   errResp,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var unmarshaled Response
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if unmarshaled.Error == nil {
		t.Fatal("Error field is nil")
	}
	if unmarshaled.Error.Code != MethodNotFound {
		t.Errorf("Error code mismatch: got %d", unmarshaled.Error.Code)
	}
}

func TestToolSchema(t *testing.T) {
	tool := Tool{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"arg1": map[string]interface{}{
					"type":        "string",
					"description": "First argument",
				},
			},
			"required": []string{"arg1"},
		},
	}

	data, err := json.Marshal(tool)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var unmarshaled Tool
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if unmarshaled.Name != "test_tool" {
		t.Errorf("Name mismatch: got %s", unmarshaled.Name)
	}
}

func TestContentBlock(t *testing.T) {
	block := ContentBlock{
		Type: ContentTypeText,
		Text: "Test content",
	}

	data, err := json.Marshal(block)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var unmarshaled ContentBlock
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if unmarshaled.Type != ContentTypeText {
		t.Errorf("Type mismatch: got %s", unmarshaled.Type)
	}
	if unmarshaled.Text != "Test content" {
		t.Errorf("Text mismatch: got %s", unmarshaled.Text)
	}
}

func TestToolCallResult(t *testing.T) {
	result := ToolCallResult{
		Content: []ContentBlock{
			{
				Type: ContentTypeText,
				Text: "Result text",
			},
		},
		IsError: false,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var unmarshaled ToolCallResult
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(unmarshaled.Content) != 1 {
		t.Fatalf("Content length mismatch: got %d", len(unmarshaled.Content))
	}
	if unmarshaled.Content[0].Text != "Result text" {
		t.Errorf("Content text mismatch: got %s", unmarshaled.Content[0].Text)
	}
	if unmarshaled.IsError {
		t.Error("IsError should be false")
	}
}
