package mcp

import (
	"context"
	"testing"

	"github.com/spf13/cobra"
)

func TestNewToolRegistry(t *testing.T) {
	rootCmd := &cobra.Command{Use: "test"}
	registry := NewToolRegistry(rootCmd)

	if registry.tools == nil {
		t.Error("tools map not initialized")
	}
	if registry.rootCmd != rootCmd {
		t.Error("rootCmd not set correctly")
	}
}

func TestRegisterDirectTool(t *testing.T) {
	rootCmd := &cobra.Command{Use: "test"}
	registry := NewToolRegistry(rootCmd)

	executed := false
	executor := func(ctx context.Context, args map[string]interface{}) (*ToolCallResult, error) {
		executed = true

		return textResult("test result"), nil
	}

	inputSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"test": map[string]interface{}{
				"type": "string",
			},
		},
	}

	registry.RegisterDirectTool("test_tool", "A test tool", inputSchema, executor)

	// Check tool is registered
	tools := registry.ListTools()
	if len(tools) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(tools))
	}

	if tools[0].Name != "test_tool" {
		t.Errorf("Tool name mismatch: got %s", tools[0].Name)
	}

	// Call the tool
	result, err := registry.CallTool(context.Background(), "test_tool", map[string]interface{}{})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	if !executed {
		t.Error("Executor was not called")
	}

	if result == nil {
		t.Fatal("Result is nil")
	}
}

func TestRegisterCommand(t *testing.T) {
	rootCmd := &cobra.Command{Use: "root"}
	testCmd := &cobra.Command{
		Use:   "test",
		Short: "Test command",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println("test output")
		},
	}
	rootCmd.AddCommand(testCmd)

	registry := NewToolRegistry(rootCmd)
	registry.RegisterCommand(testCmd, DefaultArgMapper)

	tools := registry.ListTools()
	if len(tools) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(tools))
	}

	// Command path is "root test" -> tool name is "test" (root command stripped)
	if tools[0].Name != "test" {
		t.Errorf("Tool name mismatch: got %s", tools[0].Name)
	}

	// Check input schema was created
	if tools[0].InputSchema == nil {
		t.Error("InputSchema is nil")
	}
}

func TestListTools(t *testing.T) {
	rootCmd := &cobra.Command{Use: "test"}
	registry := NewToolRegistry(rootCmd)

	// Register multiple tools
	executor := func(ctx context.Context, args map[string]interface{}) (*ToolCallResult, error) {
		return textResult("test"), nil
	}

	registry.RegisterDirectTool("tool1", "Tool 1", map[string]interface{}{}, executor)
	registry.RegisterDirectTool("tool2", "Tool 2", map[string]interface{}{}, executor)

	tools := registry.ListTools()
	if len(tools) != 2 {
		t.Fatalf("Expected 2 tools, got %d", len(tools))
	}
}

func TestCallToolNotFound(t *testing.T) {
	rootCmd := &cobra.Command{Use: "test"}
	registry := NewToolRegistry(rootCmd)

	_, err := registry.CallTool(context.Background(), "nonexistent", map[string]interface{}{})
	if err == nil {
		t.Error("Expected error for nonexistent tool")
	}

	if err.Error() != "tool not found: nonexistent" {
		t.Errorf("Error message mismatch: got %s", err.Error())
	}
}

func TestDefaultArgMapper(t *testing.T) {
	args := map[string]interface{}{
		"arg":   "test-arg",
		"flag1": "value1",
		"flag2": true,
		"flag3": 42,
	}

	result := DefaultArgMapper(args)

	// Check positional arg
	found := false
	for _, arg := range result {
		if arg == "test-arg" {
			found = true

			break
		}
	}
	if !found {
		t.Error("Positional arg not found in result")
	}

	// Check flags - the result should contain: test-arg, --flag1, value1, --flag2, --flag3, 42
	expectedItems := map[string]bool{
		"test-arg": false,
		"--flag1":  false,
		"value1":   false,
		"--flag2":  false,
		"--flag3":  false,
		"42":       false,
	}

	for _, arg := range result {
		if _, exists := expectedItems[arg]; exists {
			expectedItems[arg] = true
		}
	}

	for item, found := range expectedItems {
		if !found {
			t.Errorf("Expected item %s not found in result: %v", item, result)
		}
	}
}

func TestDefaultArgMapperWithArray(t *testing.T) {
	args := map[string]interface{}{
		"args": []interface{}{"arg1", "arg2", "arg3"},
		"flag": "value",
	}

	result := DefaultArgMapper(args)

	// Check args array
	if len(result) != 5 { // 3 args + flag (name + value)
		t.Fatalf("Expected 5 elements, got %d", len(result))
	}

	if result[0] != "arg1" {
		t.Errorf("First arg mismatch: got %s", result[0])
	}
	if result[1] != "arg2" {
		t.Errorf("Second arg mismatch: got %s", result[1])
	}
	if result[2] != "arg3" {
		t.Errorf("Third arg mismatch: got %s", result[2])
	}
}
