package mcp

import (
	"fmt"
	"log/slog"
	"strconv"
)

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

// FilterTools removes tools not in the allowList.
// If allowList is empty, all tools are kept (no filtering).
func (r *ToolRegistry) FilterTools(allowList []string) {
	if len(allowList) == 0 {
		return
	}

	allowed := make(map[string]bool, len(allowList))
	for _, name := range allowList {
		allowed[name] = true
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for name := range r.tools {
		if !allowed[name] {
			slog.Info("MCP tool filtered out by allowlist", "tool", name)
			delete(r.tools, name)
		}
	}
}
