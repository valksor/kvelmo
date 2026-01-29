package quality

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/valksor/go-mehrhof/internal/storage"
)

// CustomLinter implements Linter for user-defined commands.
// Supports arbitrary CLI tools like phpstan, psalm, mypy, etc.
type CustomLinter struct {
	name       string
	command    []string
	extensions []string
	// JSON parsing configuration
	jsonOutput bool
	// For parsing tool-specific JSON formats
	parseFunc func([]byte) (*Result, error)
}

// NewCustomLinter creates a new custom linter from configuration.
func NewCustomLinter(name string, cfg storage.LinterConfig) *CustomLinter {
	l := &CustomLinter{
		name:       name,
		command:    cfg.Command,
		extensions: cfg.Extensions,
		// Assume JSON output by default if command is configured
		jsonOutput: true,
	}

	// Append additional args from config
	if len(cfg.Args) > 0 {
		l.command = append(l.command, cfg.Args...)
	}

	return l
}

// Name returns the linter identifier.
func (c *CustomLinter) Name() string {
	return c.name
}

// Available checks if the custom linter binary is installed.
func (c *CustomLinter) Available() bool {
	if len(c.command) == 0 {
		return false
	}
	_, err := exec.LookPath(c.command[0])

	return err == nil
}

// Run executes the custom linter on the specified files.
func (c *CustomLinter) Run(ctx context.Context, workDir string, files []string) (*Result, error) {
	// Build command: base command + files
	args := make([]string, 0, len(c.command)+len(files))
	if len(c.command) > 1 {
		args = append(args, c.command[1:]...)
	}

	// Filter files by extension if configured
	var filesToCheck []string
	if len(c.extensions) > 0 {
		for _, f := range files {
			ext := strings.ToLower(filepath.Ext(f))
			for _, allowed := range c.extensions {
				if ext == strings.ToLower(allowed) {
					filesToCheck = append(filesToCheck, f)

					break
				}
			}
		}
	} else {
		filesToCheck = files
	}

	// Add files to command
	if len(filesToCheck) > 0 {
		args = append(args, filesToCheck...)
	}

	cmd := exec.CommandContext(ctx, c.command[0], args...)
	cmd.Dir = workDir

	output, err := cmd.CombinedOutput()

	// Try parsing result
	if c.parseFunc != nil {
		return c.parseFunc(output)
	}

	result, parseErr := c.parseOutput(output)
	if parseErr != nil {
		// If parsing fails but we got an error, include the raw output
		if err != nil {
			//nolint:nilerr // Error embedded in Result struct
			return &Result{
				Linter:  c.Name(),
				Passed:  false,
				Summary: "Linter failed: " + string(output),
				Error:   err,
			}, nil
		}

		return nil, fmt.Errorf("parse %s output: %w", c.Name(), parseErr)
	}

	return result, nil
}

// parseOutput attempts to parse the linter output as JSON or text.
func (c *CustomLinter) parseOutput(output []byte) (*Result, error) {
	// Empty output means success
	if len(output) == 0 {
		return &Result{
			Linter:  c.Name(),
			Passed:  true,
			Summary: "No issues found",
		}, nil
	}

	// Try parsing as JSON first
	var parsed any
	if err := json.Unmarshal(output, &parsed); err == nil {
		return c.parseJSONOutput(parsed)
	}

	// Check for common success messages in text output
	outputStr := string(output)
	if strings.Contains(outputStr, "no errors found") ||
		strings.Contains(outputStr, "no issues found") ||
		strings.Contains(outputStr, "0 errors") ||
		strings.Contains(outputStr, "0 warnings") {
		return &Result{
			Linter:  c.Name(),
			Passed:  true,
			Summary: "No issues found",
		}, nil
	}

	// Treat raw output as an error/result
	return &Result{
		Linter:  c.Name(),
		Passed:  false, // Assume failure if we can't parse
		Summary: strings.TrimSpace(outputStr),
		Issues: []Issue{
			{
				Path:     "",
				Line:     0,
				Column:   0,
				Message:  strings.TrimSpace(outputStr),
				Severity: SeverityWarning,
			},
		},
	}, nil
}

// parseJSONOutput attempts to parse various JSON formats from linters.
func (c *CustomLinter) parseJSONOutput(parsed any) (*Result, error) {
	issues := make([]Issue, 0)

	// Handle different JSON structures
	switch v := parsed.(type) {
	case map[string]any:
		// Single object - look for common keys
		issues = c.parseJSONObject(v)

	case []any:
		// Array of objects
		for _, item := range v {
			if obj, ok := item.(map[string]any); ok {
				issues = append(issues, c.parseJSONObject(obj)...)
			}
		}
	}

	if len(issues) == 0 {
		return &Result{
			Linter:  c.Name(),
			Passed:  true,
			Summary: "No issues found",
		}, nil
	}

	// Determine pass status
	passed := true
	for _, issue := range issues {
		if issue.Severity == SeverityError {
			passed = false

			break
		}
	}

	return &Result{
		Linter:  c.Name(),
		Issues:  issues,
		Passed:  passed,
		Summary: fmt.Sprintf("%d issues found", len(issues)),
	}, nil
}

// parseJSONObject parses a single JSON object into issues.
func (c *CustomLinter) parseJSONObject(obj map[string]any) []Issue {
	issues := make([]Issue, 0)

	// Common field names for file paths
	fileKeys := []string{"file", "filename", "path", "filePath"}
	// Common field names for line numbers
	lineKeys := []string{"line", "row", "lineNumber"}
	// Common field names for column
	columnKeys := []string{"column", "col"}
	// Common field names for messages
	msgKeys := []string{"message", "msg", "text", "description"}
	// Common field names for severity
	severityKeys := []string{"severity", "level"}
	// Common field names for rules
	ruleKeys := []string{"rule", "ruleId", "code", "rule_id"}

	// Extract file path
	var path string
	for _, key := range fileKeys {
		if val, ok := obj[key].(string); ok {
			path = val

			break
		}
	}

	// Extract line number
	line := 0
	for _, key := range lineKeys {
		if val, ok := obj[key].(float64); ok {
			line = int(val)

			break
		}
	}

	// Extract column
	column := 0
	for _, key := range columnKeys {
		if val, ok := obj[key].(float64); ok {
			column = int(val)

			break
		}
	}

	// Extract message
	message := ""
	for _, key := range msgKeys {
		if val, ok := obj[key].(string); ok {
			message = val

			break
		}
	}

	// Extract severity
	severity := SeverityWarning
	for _, key := range severityKeys {
		if val, ok := obj[key].(string); ok {
			switch strings.ToLower(val) {
			case "error", "err", "critical":
				severity = SeverityError
			case "warning", "warn":
				severity = SeverityWarning
			default:
				severity = SeverityInfo
			}

			break
		}
	}

	// Extract rule
	rule := ""
	for _, key := range ruleKeys {
		if val, ok := obj[key].(string); ok {
			rule = val

			break
		}
	}

	// If we have at least a message, create an issue
	if message != "" {
		issues = append(issues, Issue{
			Path:     path,
			Line:     line,
			Column:   column,
			Message:  message,
			Severity: severity,
			Rule:     rule,
		})
	}

	return issues
}
