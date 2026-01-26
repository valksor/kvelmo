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

func TestBuildInputSchema_MarkFlagRequired(t *testing.T) {
	rootCmd := &cobra.Command{Use: "test"}
	registry := NewToolRegistry(rootCmd)

	cmd := &cobra.Command{Use: "test-cmd"}
	cmd.Flags().String("required-flag", "", "A required flag")
	cmd.Flags().String("optional-flag", "", "An optional flag")
	cmd.Flags().String("with-default", "default-value", "Has a default value")
	_ = cmd.MarkFlagRequired("required-flag")

	schema := registry.buildInputSchema(cmd)
	required, ok := schema["required"].([]string)
	if !ok {
		required = []string{}
	}

	hasRequired := false
	hasOptional := false
	hasWithDefault := false
	for _, r := range required {
		if r == "required-flag" {
			hasRequired = true
		}
		if r == "optional-flag" {
			hasOptional = true
		}
		if r == "with-default" {
			hasWithDefault = true
		}
	}

	if !hasRequired {
		t.Error("Flag marked with MarkFlagRequired should be in required array")
	}
	if hasOptional {
		t.Error("Optional flag (no MarkFlagRequired) should NOT be in required array")
	}
	if hasWithDefault {
		t.Error("Flag with default value should NOT be in required array")
	}
}

func TestBuildInputSchema_PositionalArgRequired(t *testing.T) {
	rootCmd := &cobra.Command{Use: "test"}
	registry := NewToolRegistry(rootCmd)

	cmd := &cobra.Command{
		Use:  "test-cmd <required-arg>",
		Args: cobra.ExactArgs(1),
	}
	cmd.Flags().String("optional-flag", "", "An optional flag")

	schema := registry.buildInputSchema(cmd)
	required, ok := schema["required"].([]string)
	if !ok {
		required = []string{}
	}

	hasArg := false
	hasOptional := false
	for _, r := range required {
		if r == "arg" {
			hasArg = true
		}
		if r == "optional-flag" {
			hasOptional = true
		}
	}

	if !hasArg {
		t.Error("Positional argument should be in required array")
	}
	if hasOptional {
		t.Error("Optional flag should NOT be in required array")
	}
}

func TestBuildInputSchema_NoRequiredFlags(t *testing.T) {
	rootCmd := &cobra.Command{Use: "test"}
	registry := NewToolRegistry(rootCmd)

	cmd := &cobra.Command{
		Use:  "test-cmd",
		Args: cobra.NoArgs,
	}
	cmd.Flags().String("optional1", "", "Optional 1")
	cmd.Flags().String("optional2", "", "Optional 2")
	cmd.Flags().Bool("verbose", false, "Verbose output")

	schema := registry.buildInputSchema(cmd)
	required, ok := schema["required"].([]string)
	if !ok {
		required = []string{}
	}

	if len(required) != 0 {
		t.Errorf("Expected no required parameters, got %d: %v", len(required), required)
	}
}

func TestBuildInputSchema_BrowserScreenshot(t *testing.T) {
	// This test verifies the fix for the issue where browser_screenshot --output
	// was incorrectly marked as required in the MCP schema
	rootCmd := &cobra.Command{Use: "mehr"}
	registry := NewToolRegistry(rootCmd)

	// Recreate browser screenshot command structure
	screenshotOutput := ""
	screenshotFormat := "png"
	screenshotQuality := 80
	screenshotFullPage := false

	browserScreenshotCmd := &cobra.Command{
		Use:   "screenshot [url]",
		Short: "Capture screenshot",
		Long:  "Capture a screenshot of the current tab or navigate to URL first.",
		Args:  cobra.MaximumNArgs(1),
	}
	browserScreenshotCmd.Flags().StringVarP(&screenshotOutput, "output", "o", "", "Output file path")
	browserScreenshotCmd.Flags().StringVarP(&screenshotFormat, "format", "f", "png", "Format (png, jpeg)")
	browserScreenshotCmd.Flags().IntVar(&screenshotQuality, "quality", 80, "JPEG quality (1-100)")
	browserScreenshotCmd.Flags().BoolVarP(&screenshotFullPage, "full-page", "F", false, "Capture full scrollable page")

	schema := registry.buildInputSchema(browserScreenshotCmd)
	required, ok := schema["required"].([]string)
	if !ok {
		required = []string{}
	}

	// Verify NO flags are marked as required
	// The --output flag has an empty default but is optional (has fallback in code)
	// The [url] argument is optional (MaximumNArgs(1))
	for _, req := range required {
		if req == "output" || req == "format" || req == "quality" || req == "full-page" {
			t.Errorf("Flag '%s' should NOT be marked as required", req)
		}
	}

	// Verify the schema has the expected properties
	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Properties not found in schema")
	}

	// Check that output flag exists
	if _, exists := props["output"]; !exists {
		t.Error("output flag should be in properties")
	}

	// Check that format flag exists with default
	formatSchema, exists := props["format"].(map[string]interface{})
	if !exists {
		t.Error("format flag should be in properties")
	} else {
		if formatSchema["type"] != "string" {
			t.Errorf("format type should be string, got %v", formatSchema["type"])
		}
		if formatSchema["default"] != "png" {
			t.Errorf("format default should be png, got %v", formatSchema["default"])
		}
	}
}

func TestBuildInputSchema_BrowserClick(t *testing.T) {
	// This test verifies that flags marked with MarkFlagRequired()
	// (like --selector for browser click) ARE correctly marked as required
	rootCmd := &cobra.Command{Use: "mehr"}
	registry := NewToolRegistry(rootCmd)

	// Recreate browser click command structure
	clickSelector := ""

	browserClickCmd := &cobra.Command{
		Use:   "click --selector <css>",
		Short: "Click an element",
		Long:  "Click an element using CSS selector.",
	}
	browserClickCmd.Flags().StringVar(&clickSelector, "selector", "", "CSS selector")
	_ = browserClickCmd.MarkFlagRequired("selector")

	schema := registry.buildInputSchema(browserClickCmd)
	required, ok := schema["required"].([]string)
	if !ok {
		required = []string{}
	}

	// Verify selector IS marked as required
	hasSelector := false
	for _, req := range required {
		if req == "selector" {
			hasSelector = true

			break
		}
	}

	if !hasSelector {
		t.Error("selector flag (MarkFlagRequired) should be in required array")
	}
}
