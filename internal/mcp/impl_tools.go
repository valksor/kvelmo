package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// ToolRegistry manages MCP tools backed by Cobra commands.
type ToolRegistry struct {
	tools   map[string]*ToolWrapper
	rootCmd *cobra.Command
	mu      sync.RWMutex
}

// ToolWrapper wraps a Cobra command as an MCP tool.
type ToolWrapper struct {
	Tool      Tool
	Command   *cobra.Command
	ArgMapper func(map[string]interface{}) []string
	mu        sync.Mutex // Mutex for Cobra commands (not thread-safe)
	// For non-Cobra tools
	Executor func(ctx context.Context, args map[string]interface{}) (*ToolCallResult, error)
}

// NewToolRegistry creates a new tool registry.
func NewToolRegistry(rootCmd *cobra.Command) *ToolRegistry {
	return &ToolRegistry{
		tools:   make(map[string]*ToolWrapper),
		rootCmd: rootCmd,
	}
}

// RegisterCommand registers a Cobra command as an MCP tool.
func (r *ToolRegistry) RegisterCommand(cmd *cobra.Command, argMapper func(map[string]interface{}) []string) {
	toolName := "mehr_" + strings.ReplaceAll(cmd.CommandPath(), " ", "_")

	// Build JSON Schema for input
	inputSchema := r.buildInputSchema(cmd)

	tool := Tool{
		Name:        toolName,
		Description: cmd.Short,
		InputSchema: inputSchema,
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[toolName]; exists {
		slog.Warn("Tool already registered, overwriting", "tool", toolName)
	}

	r.tools[toolName] = &ToolWrapper{
		Tool:      tool,
		Command:   cmd,
		ArgMapper: argMapper,
	}
}

// RegisterCommands registers multiple commands.
func (r *ToolRegistry) RegisterCommands(commands []*cobra.Command, argMapper func(map[string]interface{}) []string) {
	for _, cmd := range commands {
		r.RegisterCommand(cmd, argMapper)
	}
}

// RegisterDirectTool registers a direct function tool (not backed by Cobra command).
func (r *ToolRegistry) RegisterDirectTool(name, description string, inputSchema map[string]interface{}, executor func(ctx context.Context, args map[string]interface{}) (*ToolCallResult, error)) {
	tool := Tool{
		Name:        name,
		Description: description,
		InputSchema: inputSchema,
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[name]; exists {
		slog.Warn("Tool already registered, overwriting", "tool", name)
	}

	r.tools[name] = &ToolWrapper{
		Tool:     tool,
		Executor: executor,
	}
}

// ListTools returns all registered tools.
func (r *ToolRegistry) ListTools() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]Tool, 0, len(r.tools))
	for _, wrapper := range r.tools {
		tools = append(tools, wrapper.Tool)
	}

	return tools
}

// CallTool executes a tool by name.
func (r *ToolRegistry) CallTool(ctx context.Context, name string, args map[string]interface{}) (*ToolCallResult, error) {
	// Add timeout for tool execution (5 minutes max) only if parent doesn't have a deadline
	// This respects parent context deadlines while ensuring a maximum timeout
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()
	}

	r.mu.RLock()
	wrapper, ok := r.tools[name]
	r.mu.RUnlock()

	if !ok {
		slog.Error("MCP tool not found", "tool", name)

		return nil, fmt.Errorf("tool not found: %s", name)
	}

	// Validate arguments for Cobra tools
	if wrapper.Command != nil {
		if err := r.validateRequiredArgs(wrapper.Command, args); err != nil {
			slog.Warn("MCP tool validation failed", "tool", name, "error", err)

			return nil, err
		}
		if err := r.validateNoExtraArgs(wrapper.Command, args); err != nil {
			slog.Warn("MCP tool validation failed", "tool", name, "error", err)

			return nil, err
		}
	}

	// For direct tools, validate against their schema
	if wrapper.Executor != nil && wrapper.Tool.InputSchema != nil {
		if err := r.validateNoExtraArgsForSchema(wrapper.Tool.InputSchema, args); err != nil {
			slog.Warn("MCP tool validation failed", "tool", name, "error", err)

			return nil, err
		}
	}

	// Log tool call
	slog.Info("MCP tool call", "tool", name, "args", args)

	// If it's a direct function tool, execute it
	if wrapper.Executor != nil {
		result, err := wrapper.Executor(ctx, args)
		if err != nil {
			slog.Error("MCP tool execution failed", "tool", name, "error", err)
		} else {
			slog.Info("MCP tool call succeeded", "tool", name, "is_error", result.IsError)
		}

		return result, err
	}

	// Otherwise, it's a Cobra command tool
	// Acquire mutex for this command to prevent race conditions
	// Cobra commands are not thread-safe due to mutable state (flags, output buffers, etc.)
	wrapper.mu.Lock()
	defer wrapper.mu.Unlock()

	// Map arguments to CLI args
	cliArgs := wrapper.ArgMapper(args)
	if cliArgs == nil {
		cliArgs = []string{}
	}

	// Use the command directly. The mutex above ensures only one goroutine
	// executes this command at a time. We don't clone because:
	// 1. Cobra commands don't support easy deep copying
	// 2. The mutex already provides serialization
	// 3. Resetting flags between calls is handled by Cobra
	cmd := wrapper.Command

	// Execute command with context awareness
	// Since cmd.Execute() doesn't accept a context, we run it in a goroutine
	// and select on completion vs context cancellation.
	type execResult struct {
		text string
		err  error
	}
	resultCh := make(chan execResult, 1)

	// Add timeout to command execution to prevent goroutine leaks
	execCtx, execCancel := context.WithTimeout(ctx, 30*time.Second)
	defer execCancel()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("Tool execution panic", "tool", name, "panic", r)
				resultCh <- execResult{err: fmt.Errorf("panic: %v", r)}
			}
		}()

		// Capture output
		output := &strings.Builder{}
		cmd.SetOut(output)
		cmd.SetErr(output)

		// Set args and execute
		cmd.SetArgs(cliArgs)
		err := cmd.Execute()

		resultCh <- execResult{
			text: output.String(),
			err:  err,
		}
	}()

	// Wait for either completion or context cancellation
	select {
	case res := <-resultCh:
		// Command completed
		resultText := res.text
		if res.err != nil {
			// Return error as result text (MCP style)
			result := &ToolCallResult{
				Content: []ContentBlock{
					{
						Type: ContentTypeText,
						Text: fmt.Sprintf("Error: %v\n\nOutput:\n%s", res.err, resultText),
					},
				},
				IsError: true,
			}
			slog.Error("MCP tool execution failed", "tool", name, "error", res.err)

			return result, nil
		}

		result := &ToolCallResult{
			Content: []ContentBlock{
				{
					Type: ContentTypeText,
					Text: resultText,
				},
			},
			IsError: false,
		}
		slog.Info("MCP tool call succeeded", "tool", name, "is_error", false)

		return result, nil

	case <-execCtx.Done():
		// Context was canceled (timeout or explicit cancel)
		// Note: The goroutine will complete eventually and discard the result (resultCh is buffered).
		// The timeout ensures the goroutine doesn't run forever.
		slog.Warn("MCP tool execution canceled due to context", "tool", name)

		return &ToolCallResult{
			Content: []ContentBlock{
				{
					Type: ContentTypeText,
					Text: "Tool execution canceled (timeout or interrupted)",
				},
			},
			IsError: true,
		}, nil
	}
}

// validateRequiredArgs checks if all required arguments are provided.
func (r *ToolRegistry) validateRequiredArgs(cmd *cobra.Command, args map[string]interface{}) error {
	schema := r.buildInputSchema(cmd)
	required, ok := schema["required"].([]string)
	if !ok || len(required) == 0 {
		return nil
	}

	for _, req := range required {
		if _, exists := args[req]; !exists {
			return fmt.Errorf("required argument '%s' is missing", req)
		}
	}

	return nil
}

// validateNoExtraArgs checks if provided arguments contain any keys not defined in the command's schema.
func (r *ToolRegistry) validateNoExtraArgs(cmd *cobra.Command, args map[string]interface{}) error {
	schema := r.buildInputSchema(cmd)

	return r.validateNoExtraArgsForSchema(schema, args)
}

// validateNoExtraArgsForSchema checks if provided arguments contain any keys not defined in the schema.
func (r *ToolRegistry) validateNoExtraArgsForSchema(schema map[string]interface{}, args map[string]interface{}) error {
	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		// If no properties defined, accept any args (backward compatibility)
		return nil
	}

	// Check each provided arg against the schema
	for argName := range args {
		if _, defined := properties[argName]; !defined {
			return fmt.Errorf("unexpected argument '%s' (not defined in tool schema)", argName)
		}
	}

	return nil
}

// buildInputSchema creates a JSON schema for command arguments.
func (r *ToolRegistry) buildInputSchema(cmd *cobra.Command) map[string]interface{} {
	properties := make(map[string]interface{})
	required := []string{}

	// Add local flags
	cmd.LocalFlags().VisitAll(func(flag *pflag.Flag) {
		properties[flag.Name] = r.mapFlagToSchema(flag)
		// Mark as required if it has no default value and isn't optional
		if flag.DefValue == "" && !flag.Changed {
			// Skip required check for boolean flags (they default to false)
			if flag.Value.Type() != "bool" {
				required = append(required, flag.Name)
			}
		}
	})

	// Add persistent flags
	cmd.PersistentFlags().VisitAll(func(flag *pflag.Flag) {
		properties[flag.Name] = r.mapFlagToSchema(flag)
	})

	// Check if command expects arguments
	argsSpec := cmd.Args
	if argsSpec != nil {
		// Try to validate with empty args to see if args are required
		err := argsSpec(cmd, []string{})
		if err != nil {
			// Command requires arguments
			properties["arg"] = map[string]interface{}{
				"type":        "string",
				"description": "Command argument",
			}
			required = append(required, "arg")
		} else {
			// Arguments are optional
			properties["args"] = map[string]interface{}{
				"type":        "array",
				"items":       map[string]string{"type": "string"},
				"description": "Command arguments (optional)",
			}
		}
	}

	return map[string]interface{}{
		"type":       "object",
		"properties": properties,
		"required":   required,
	}
}

// mapFlagToSchema maps a pflag to JSON schema.
func (r *ToolRegistry) mapFlagToSchema(flag *pflag.Flag) map[string]interface{} {
	schema := map[string]interface{}{
		"description": flag.Usage,
	}

	switch flag.Value.Type() {
	case "bool":
		schema["type"] = "boolean"
		if flag.DefValue != "" {
			schema["default"] = flag.DefValue == "true"
		}
	case "string", "stringArray":
		if flag.Value.Type() == "stringArray" {
			schema["type"] = "array"
			schema["items"] = map[string]string{"type": "string"}
		} else {
			schema["type"] = "string"
		}
		if flag.DefValue != "" {
			schema["default"] = flag.DefValue
		}
	case "int", "int32", "int64":
		schema["type"] = "integer"
		if flag.DefValue != "" {
			var def int
			_, _ = fmt.Sscanf(flag.DefValue, "%d", &def)
			schema["default"] = def
		}
	case "float", "float32", "float64":
		schema["type"] = "number"
		if flag.DefValue != "" {
			var def float64
			_, _ = fmt.Sscanf(flag.DefValue, "%f", &def)
			schema["default"] = def
		}
	default:
		schema["type"] = "string"
		if flag.DefValue != "" {
			schema["default"] = flag.DefValue
		}
	}

	return schema
}

// DefaultArgMapper creates a default argument mapper from MCP args to CLI args.
func DefaultArgMapper(args map[string]interface{}) []string {
	cliArgs := []string{}

	// Handle positional arg
	if arg, ok := args["arg"].(string); ok && arg != "" {
		cliArgs = append(cliArgs, arg)
	}

	// Handle args array
	if argsArray, ok := args["args"].([]interface{}); ok {
		for _, a := range argsArray {
			if str, ok := a.(string); ok {
				cliArgs = append(cliArgs, str)
			}
		}
	}

	// Handle flags
	for key, value := range args {
		if key == "arg" || key == "args" {
			continue
		}

		flagName := "--" + key

		switch v := value.(type) {
		case bool:
			if v {
				cliArgs = append(cliArgs, flagName)
			}
		case string:
			cliArgs = append(cliArgs, flagName, v)
		case float64: // JSON numbers are float64
			cliArgs = append(cliArgs, flagName, fmt.Sprintf("%v", v))
		case int: // Direct int values (not from JSON)
			cliArgs = append(cliArgs, flagName, strconv.Itoa(v))
		case int64:
			cliArgs = append(cliArgs, flagName, strconv.FormatInt(v, 10))
		case int32:
			cliArgs = append(cliArgs, flagName, strconv.Itoa(int(v)))
		case []interface{}:
			for i, item := range v {
				if str, ok := item.(string); ok {
					cliArgs = append(cliArgs, flagName, str)
				} else if item != nil {
					slog.Warn("Skipping non-string array element", "index", i, "type", fmt.Sprintf("%T", item))
				}
			}
		case []string: // Handle string arrays directly
			for _, str := range v {
				cliArgs = append(cliArgs, flagName, str)
			}
		}
	}

	return cliArgs
}
