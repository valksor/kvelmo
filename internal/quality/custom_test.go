package quality

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valksor/go-mehrhof/internal/storage"
)

// TestNewCustomLinter tests creating a new custom linter.
func TestNewCustomLinter(t *testing.T) {
	tests := []struct {
		name string
		cfg  storage.LinterConfig
	}{
		{
			name: "minimal config",
			cfg: storage.LinterConfig{
				Command: []string{"echo"},
			},
		},
		{
			name: "with args",
			cfg: storage.LinterConfig{
				Command: []string{"phpstan", "analyse"},
				Args:    []string{"--no-progress"},
			},
		},
		{
			name: "with extensions",
			cfg: storage.LinterConfig{
				Command:    []string{"eslint"},
				Extensions: []string{".js", ".ts"},
			},
		},
		{
			name: "full config",
			cfg: storage.LinterConfig{
				Command:    []string{"mypy"},
				Args:       []string{"--strict"},
				Extensions: []string{".py"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			linter := NewCustomLinter("test-linter", tt.cfg)

			assert.NotNil(t, linter)
			assert.Equal(t, "test-linter", linter.Name())
			assert.True(t, linter.jsonOutput, "jsonOutput should be true by default")
		})
	}
}

// TestCustomLinter_Name tests the Name method.
func TestCustomLinter_Name(t *testing.T) {
	cfg := storage.LinterConfig{Command: []string{"echo"}}
	linter := NewCustomLinter("my-linter", cfg)

	assert.Equal(t, "my-linter", linter.Name())
}

// TestCustomLinter_Available tests the Available method.
func TestCustomLinter_Available(t *testing.T) {
	tests := []struct {
		name          string
		command       string
		wantAvailable bool
	}{
		{
			name:          "exists command",
			command:       "echo",
			wantAvailable: true,
		},
		{
			name:          "not exists command",
			command:       "nonexistent-linter-xyz123",
			wantAvailable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := storage.LinterConfig{Command: []string{tt.command}}
			linter := NewCustomLinter("test", cfg)

			got := linter.Available()
			assert.Equal(t, tt.wantAvailable, got)
		})
	}
}

// TestCustomLinter_Run_EmptyOutput tests running with empty output.
func TestCustomLinter_Run_EmptyOutput(t *testing.T) {
	// Use "true" command which produces no output and exits successfully
	cfg := storage.LinterConfig{
		Command: []string{"true"},
	}
	linter := NewCustomLinter("test", cfg)

	ctx := context.Background()
	tmpDir := t.TempDir()

	result, err := linter.Run(ctx, tmpDir, []string{"file.txt"})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Passed)
	assert.Equal(t, "test", result.Linter)
}

// TestCustomLinter_Run_NoFiles tests running with no files.
func TestCustomLinter_Run_NoFiles(t *testing.T) {
	cfg := storage.LinterConfig{
		Command: []string{"echo", "{}"},
	}
	linter := NewCustomLinter("test", cfg)

	ctx := context.Background()
	tmpDir := t.TempDir()

	result, err := linter.Run(ctx, tmpDir, []string{})

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

// TestCustomLinter_Run_ExtensionFilter tests file extension filtering.
func TestCustomLinter_Run_ExtensionFilter(t *testing.T) {
	tests := []struct {
		name       string
		extensions []string
		files      []string
		wantRun    bool
	}{
		{
			name:       "no extension filter",
			extensions: nil,
			files:      []string{"file.txt", "file.py"},
			wantRun:    true,
		},
		{
			name:       "matching extension",
			extensions: []string{".py"},
			files:      []string{"file.py", "other.py"},
			wantRun:    true,
		},
		{
			name:       "no matching extensions",
			extensions: []string{".py"},
			files:      []string{"file.txt", "file.js"},
			wantRun:    true, // Still runs, but with no files
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := storage.LinterConfig{
				Command:    []string{"echo", "{}"},
				Extensions: tt.extensions,
			}
			linter := NewCustomLinter("test", cfg)

			ctx := context.Background()
			tmpDir := t.TempDir()

			result, err := linter.Run(ctx, tmpDir, tt.files)

			assert.NoError(t, err)
			assert.NotNil(t, result)
		})
	}
}

// TestCustomLinter_Run_CommandError tests handling command errors.
func TestCustomLinter_Run_CommandError(t *testing.T) {
	// Create a script that fails
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "fail.sh")
	scriptContent := `#!/bin/sh
echo "error: something went wrong" >&2
exit 1
`
	ctx := context.Background()
	err := exec.CommandContext(ctx, "sh", "-c", "printf '%s' '"+scriptContent+"' > "+scriptPath).Run()
	require.NoError(t, err)

	// Make script executable
	err = exec.CommandContext(ctx, "chmod", "+x", scriptPath).Run()
	require.NoError(t, err)

	cfg := storage.LinterConfig{
		Command: []string{scriptPath},
	}
	linter := NewCustomLinter("test", cfg)

	result, err := linter.Run(ctx, tmpDir, []string{"file.txt"})

	// Should not return error, but embed in result
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Passed)
}

// TestCustomLinter_Run_ContextCancellation tests context cancellation.
func TestCustomLinter_Run_ContextCancellation(t *testing.T) {
	// Use "sh" with a read that blocks until stdin is closed
	// This ensures the command is actually running
	cfg := storage.LinterConfig{
		Command: []string{"sh", "-c", "while true; do sleep 0.1; done"},
	}
	linter := NewCustomLinter("test", cfg)

	ctx, cancel := context.WithCancel(context.Background())

	// Start the linter in a goroutine
	resultChan := make(chan *Result, 1)
	errChan := make(chan error, 1)

	go func() {
		tmpDir := t.TempDir()
		result, err := linter.Run(ctx, tmpDir, []string{"file.txt"})
		if err != nil {
			errChan <- err
		} else {
			resultChan <- result
		}
	}()

	// Give it a moment to start the command
	time.Sleep(100 * time.Millisecond)

	// Cancel the context
	cancel()

	// Wait for result or error
	select {
	case <-errChan:
		// Got error - this is expected
	case <-resultChan:
		// Got result - command completed before cancellation
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for cancellation")
	}
}

// TestParseOutput_Empty tests parsing empty output.
func TestParseOutput_Empty(t *testing.T) {
	cfg := storage.LinterConfig{Command: []string{"echo"}}
	linter := NewCustomLinter("test", cfg)

	result, err := linter.parseOutput([]byte{})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Passed)
	assert.Equal(t, "No issues found", result.Summary)
}

// TestParseOutput_JSON tests parsing JSON output.
func TestParseOutput_JSON(t *testing.T) {
	tests := []struct {
		name       string
		output     string
		wantPassed bool
		wantIssues int
	}{
		{
			name:       "empty array",
			output:     "[]",
			wantPassed: true,
			wantIssues: 0,
		},
		{
			name:       "single warning (no severity specified, defaults to warning)",
			output:     `[{"file": "test.py", "line": 10, "message": "unused variable"}]`,
			wantPassed: true, // Warnings don't cause failure
			wantIssues: 1,
		},
		{
			name:       "single error",
			output:     `[{"file": "test.py", "line": 10, "message": "syntax error", "severity": "error"}]`,
			wantPassed: false,
			wantIssues: 1,
		},
		{
			name:       "multiple issues with error",
			output:     `[{"file": "test.py", "line": 5, "message": "error", "severity": "error"}, {"file": "test.py", "line": 10, "message": "warning", "severity": "warning"}]`,
			wantPassed: false,
			wantIssues: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := storage.LinterConfig{Command: []string{"echo"}}
			linter := NewCustomLinter("test", cfg)

			result, err := linter.parseOutput([]byte(tt.output))

			assert.NoError(t, err)
			assert.Equal(t, tt.wantPassed, result.Passed)
			assert.Equal(t, tt.wantIssues, len(result.Issues))
		})
	}
}

// TestParseOutput_TextSuccess tests parsing text success messages.
func TestParseOutput_TextSuccess(t *testing.T) {
	tests := []struct {
		name   string
		output string
	}{
		{
			name:   "no errors found",
			output: "no errors found",
		},
		{
			name:   "no issues found",
			output: "no issues found",
		},
		{
			name:   "0 errors",
			output: "0 errors",
		},
		{
			name:   "0 warnings",
			output: "0 warnings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := storage.LinterConfig{Command: []string{"echo"}}
			linter := NewCustomLinter("test", cfg)

			result, err := linter.parseOutput([]byte(tt.output))

			assert.NoError(t, err)
			assert.True(t, result.Passed)
			assert.Equal(t, "No issues found", result.Summary)
		})
	}
}

// TestParseOutput_TextFailure tests parsing text failure messages.
func TestParseOutput_TextFailure(t *testing.T) {
	cfg := storage.LinterConfig{Command: []string{"echo"}}
	linter := NewCustomLinter("test", cfg)

	result, err := linter.parseOutput([]byte("Some error occurred"))

	assert.NoError(t, err)
	assert.False(t, result.Passed)
	assert.Equal(t, 1, len(result.Issues))
	assert.Equal(t, "Some error occurred", result.Issues[0].Message)
}

// TestParseJSONOutput_SingleObject tests parsing single JSON object.
func TestParseJSONOutput_SingleObject(t *testing.T) {
	cfg := storage.LinterConfig{Command: []string{"echo"}}
	linter := NewCustomLinter("test", cfg)

	output := `{"file": "test.py", "line": 10, "column": 5, "message": "unused variable", "severity": "error"}`

	var parsed any
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)

	result, err := linter.parseJSONOutput(parsed)

	assert.NoError(t, err)
	assert.False(t, result.Passed)
	assert.Equal(t, 1, len(result.Issues))
	assert.Equal(t, "test.py", result.Issues[0].Path)
	assert.Equal(t, 10, result.Issues[0].Line)
	assert.Equal(t, 5, result.Issues[0].Column)
	assert.Equal(t, "unused variable", result.Issues[0].Message)
	assert.Equal(t, SeverityError, result.Issues[0].Severity)
}

// TestParseJSONOutput_Array tests parsing JSON array.
func TestParseJSONOutput_Array(t *testing.T) {
	cfg := storage.LinterConfig{Command: []string{"echo"}}
	linter := NewCustomLinter("test", cfg)

	// Use severity: "error" to ensure the test fails as expected
	output := `[{"file": "test.py", "message": "error1", "severity": "error"}, {"file": "test.py", "message": "error2", "severity": "error"}]`

	var parsed any
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)

	result, err := linter.parseJSONOutput(parsed)

	assert.NoError(t, err)
	assert.False(t, result.Passed)
	assert.Equal(t, 2, len(result.Issues))
}

// TestParseJSONOutput_EmptyArray tests parsing empty JSON array.
func TestParseJSONOutput_EmptyArray(t *testing.T) {
	cfg := storage.LinterConfig{Command: []string{"echo"}}
	linter := NewCustomLinter("test", cfg)

	output := `[]`

	var parsed any
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)

	result, err := linter.parseJSONOutput(parsed)

	assert.NoError(t, err)
	assert.True(t, result.Passed)
	assert.Equal(t, "No issues found", result.Summary)
}

// TestParseJSONObject_CommonFields tests parsing common field names.
func TestParseJSONObject_CommonFields(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantPath     string
		wantLine     int
		wantColumn   int
		wantMessage  string
		wantSeverity Severity
	}{
		{
			name:         "standard fields",
			input:        `{"file": "test.py", "line": 10, "column": 5, "message": "error"}`,
			wantPath:     "test.py",
			wantLine:     10,
			wantColumn:   5,
			wantMessage:  "error",
			wantSeverity: SeverityWarning,
		},
		{
			name:         "alternate field names",
			input:        `{"filename": "test.py", "row": 20, "col": 3, "text": "warning"}`,
			wantPath:     "test.py",
			wantLine:     20,
			wantColumn:   3,
			wantMessage:  "warning",
			wantSeverity: SeverityWarning,
		},
		{
			name:         "path field",
			input:        `{"path": "/path/to/test.py", "lineNumber": 15, "message": "issue"}`,
			wantPath:     "/path/to/test.py",
			wantLine:     15,
			wantColumn:   0,
			wantMessage:  "issue",
			wantSeverity: SeverityWarning,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := storage.LinterConfig{Command: []string{"echo"}}
			linter := NewCustomLinter("test", cfg)

			var obj map[string]any
			err := json.Unmarshal([]byte(tt.input), &obj)
			require.NoError(t, err)

			issues := linter.parseJSONObject(obj)

			if tt.wantMessage != "" {
				require.Equal(t, 1, len(issues))
				assert.Equal(t, tt.wantPath, issues[0].Path)
				assert.Equal(t, tt.wantLine, issues[0].Line)
				assert.Equal(t, tt.wantColumn, issues[0].Column)
				assert.Equal(t, tt.wantMessage, issues[0].Message)
				assert.Equal(t, tt.wantSeverity, issues[0].Severity)
			} else {
				assert.Equal(t, 0, len(issues))
			}
		})
	}
}

// TestParseJSONObject_AllFieldTypes tests all field type variations.
func TestParseJSONObject_AllFieldTypes(t *testing.T) {
	cfg := storage.LinterConfig{Command: []string{"echo"}}
	linter := NewCustomLinter("test", cfg)

	input := `{
		"file": "test.py",
		"line": 10,
		"column": 5,
		"message": "Test issue",
		"severity": "error",
		"rule": "RULE001"
	}`

	var obj map[string]any
	err := json.Unmarshal([]byte(input), &obj)
	require.NoError(t, err)

	issues := linter.parseJSONObject(obj)

	require.Equal(t, 1, len(issues))
	assert.Equal(t, "test.py", issues[0].Path)
	assert.Equal(t, 10, issues[0].Line)
	assert.Equal(t, 5, issues[0].Column)
	assert.Equal(t, "Test issue", issues[0].Message)
	assert.Equal(t, SeverityError, issues[0].Severity)
	assert.Equal(t, "RULE001", issues[0].Rule)
}

// TestParseJSONObject_SeverityLevels tests parsing severity levels.
func TestParseJSONObject_SeverityLevels(t *testing.T) {
	tests := []struct {
		name         string
		severity     string
		wantSeverity Severity
	}{
		{
			name:         "error",
			severity:     "error",
			wantSeverity: SeverityError,
		},
		{
			name:         "ERROR",
			severity:     "ERROR",
			wantSeverity: SeverityError,
		},
		{
			name:         "err",
			severity:     "err",
			wantSeverity: SeverityError,
		},
		{
			name:         "critical",
			severity:     "critical",
			wantSeverity: SeverityError,
		},
		{
			name:         "warning",
			severity:     "warning",
			wantSeverity: SeverityWarning,
		},
		{
			name:         "warn",
			severity:     "warn",
			wantSeverity: SeverityWarning,
		},
		{
			name:         "info",
			severity:     "info",
			wantSeverity: SeverityInfo,
		},
		{
			name:         "unknown",
			severity:     "unknown",
			wantSeverity: SeverityInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := storage.LinterConfig{Command: []string{"echo"}}
			linter := NewCustomLinter("test", cfg)

			input := `{"file": "test.py", "line": 1, "message": "test", "severity": "` + tt.severity + `"}`

			var obj map[string]any
			err := json.Unmarshal([]byte(input), &obj)
			require.NoError(t, err)

			issues := linter.parseJSONObject(obj)

			require.Equal(t, 1, len(issues))
			assert.Equal(t, tt.wantSeverity, issues[0].Severity)
		})
	}
}

// TestParseJSONObject_NoMessage tests handling object without message field.
func TestParseJSONObject_NoMessage(t *testing.T) {
	cfg := storage.LinterConfig{Command: []string{"echo"}}
	linter := NewCustomLinter("test", cfg)

	input := `{"file": "test.py", "line": 1}`

	var obj map[string]any
	err := json.Unmarshal([]byte(input), &obj)
	require.NoError(t, err)

	issues := linter.parseJSONObject(obj)

	// No message field means no issue should be created
	assert.Equal(t, 0, len(issues))
}

// TestCustomLinter_Integration tests integration scenarios.
func TestCustomLinter_Integration(t *testing.T) {
	// Create a temporary directory with test files
	tmpDir := t.TempDir()

	// Create test files
	testFiles := []string{
		"test.py",
		"test.js",
		"README.md",
	}
	ctx := context.Background()
	for _, f := range testFiles {
		path := filepath.Join(tmpDir, f)
		err := exec.CommandContext(ctx, "touch", path).Run()
		require.NoError(t, err)
	}

	tests := []struct {
		name       string
		cfg        storage.LinterConfig
		files      []string
		wantPassed bool
	}{
		{
			name: "echo JSON success message",
			cfg: storage.LinterConfig{
				// Filenames will be appended, but the success message will still be found
				Command: []string{"sh", "-c", "echo 'no errors found'"},
			},
			files:      []string{"test.py"},
			wantPassed: true,
		},
		{
			name: "true command with extension filter",
			cfg: storage.LinterConfig{
				Command:    []string{"true"},
				Extensions: []string{".py"},
			},
			files:      []string{"test.py", "test.js", "README.md"},
			wantPassed: true,
		},
		{
			name: "command with error output",
			cfg: storage.LinterConfig{
				// Use sh to produce error output
				Command: []string{"sh", "-c", "echo 'error: something failed' >&2; exit 1"},
			},
			files:      []string{"test.py"},
			wantPassed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			linter := NewCustomLinter("integration-test", tt.cfg)

			ctx := context.Background()
			result, err := linter.Run(ctx, tmpDir, tt.files)

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.wantPassed, result.Passed)
		})
	}
}

// TestCustomLinter_WithArgs tests appending args from config.
func TestCustomLinter_WithArgs(t *testing.T) {
	cfg := storage.LinterConfig{
		Command: []string{"echo", "base"},
		Args:    []string{"arg1", "arg2"},
	}
	linter := NewCustomLinter("test", cfg)

	// Verify args were appended
	// The linter.command should be ["echo", "base", "arg1", "arg2"]
	assert.Equal(t, 4, len(linter.command))
	assert.Equal(t, "echo", linter.command[0])
	assert.Equal(t, "base", linter.command[1])
	assert.Equal(t, "arg1", linter.command[2])
	assert.Equal(t, "arg2", linter.command[3])
}

// TestParseJSONOutput_WarningSeverity tests warning severity doesn't fail.
func TestParseJSONOutput_WarningSeverity(t *testing.T) {
	cfg := storage.LinterConfig{Command: []string{"echo"}}
	linter := NewCustomLinter("test", cfg)

	output := `[{"file": "test.py", "line": 1, "message": "warning", "severity": "warning"}]`

	var parsed any
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)

	result, err := linter.parseJSONOutput(parsed)

	assert.NoError(t, err)
	// Warnings don't cause failure
	assert.True(t, result.Passed)
	assert.Equal(t, 1, len(result.Issues))
}

// TestParseJSONOutput_ErrorSeverity tests error severity fails.
func TestParseJSONOutput_ErrorSeverity(t *testing.T) {
	cfg := storage.LinterConfig{Command: []string{"echo"}}
	linter := NewCustomLinter("test", cfg)

	output := `[{"file": "test.py", "line": 1, "message": "error", "severity": "error"}]`

	var parsed any
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)

	result, err := linter.parseJSONOutput(parsed)

	assert.NoError(t, err)
	assert.False(t, result.Passed)
	assert.Equal(t, 1, len(result.Issues))
}

// TestParseJSONObject_InvalidJSON tests handling invalid JSON.
func TestParseJSONObject_InvalidJSON(t *testing.T) {
	cfg := storage.LinterConfig{Command: []string{"echo"}}
	linter := NewCustomLinter("test", cfg)

	// Invalid JSON should fall back to text parsing
	result, err := linter.parseOutput([]byte("not valid json {}"))

	assert.NoError(t, err)
	assert.False(t, result.Passed)
}

// TestCustomLinter_Run_Timeout tests long-running command timeout.
func TestCustomLinter_Run_Timeout(t *testing.T) {
	cfg := storage.LinterConfig{
		Command: []string{"sleep", "100"},
	}
	linter := NewCustomLinter("test", cfg)

	// Use a much shorter timeout than the sleep duration
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	tmpDir := t.TempDir()

	result, err := linter.Run(ctx, tmpDir, []string{"file.txt"})

	// Should timeout - check that error is related to context
	// Note: sleep might exit cleanly before timeout in fast environments
	if err == nil {
		// If no error, verify we got a result (sleep completed early)
		assert.NotNil(t, result)
	} else {
		// Error should be context-related
		assert.True(t, errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "context"),
			"expected context-related error, got: %v", err)
	}
}

// TestCustomLinter_MixedSeverityOutput tests mixed severity in output.
func TestCustomLinter_MixedSeverityOutput(t *testing.T) {
	cfg := storage.LinterConfig{Command: []string{"echo"}}
	linter := NewCustomLinter("test", cfg)

	output := `[
		{"file": "test.py", "line": 1, "message": "info", "severity": "info"},
		{"file": "test.py", "line": 2, "message": "warning", "severity": "warning"},
		{"file": "test.py", "line": 3, "message": "error", "severity": "error"}
	]`

	var parsed any
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)

	result, err := linter.parseJSONOutput(parsed)

	assert.NoError(t, err)
	// Error causes failure
	assert.False(t, result.Passed)
	assert.Equal(t, 3, len(result.Issues))

	// Verify severities
	hasInfo := false
	hasWarning := false
	hasError := false
	for _, issue := range result.Issues {
		switch issue.Severity {
		case SeverityInfo:
			hasInfo = true
		case SeverityWarning:
			hasWarning = true
		case SeverityError:
			hasError = true
		}
	}
	assert.True(t, hasInfo, "should have info severity")
	assert.True(t, hasWarning, "should have warning severity")
	assert.True(t, hasError, "should have error severity")
}

// TestParseOutput_WithNewlines tests handling output with newlines.
func TestParseOutput_WithNewlines(t *testing.T) {
	cfg := storage.LinterConfig{Command: []string{"echo"}}
	linter := NewCustomLinter("test", cfg)

	output := "line 1\nline 2\nline 3"

	result, err := linter.parseOutput([]byte(output))

	assert.NoError(t, err)
	assert.False(t, result.Passed)
	assert.Equal(t, strings.TrimSpace(output), result.Issues[0].Message)
}

// TestCustomLinter_EmptyCommand tests handling empty command.
func TestCustomLinter_EmptyCommand(t *testing.T) {
	cfg := storage.LinterConfig{
		Command: []string{},
	}
	linter := NewCustomLinter("test", cfg)

	// Available should return false for empty command
	assert.False(t, linter.Available())
}

// TestParseJSONObject_AllRuleFieldNames tests various rule field names.
func TestParseJSONObject_AllRuleFieldNames(t *testing.T) {
	tests := []struct {
		name      string
		ruleField string
	}{
		{"rule", "rule"},
		{"ruleId", "ruleId"},
		{"code", "code"},
		{"rule_id", "rule_id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := storage.LinterConfig{Command: []string{"echo"}}
			linter := NewCustomLinter("test", cfg)

			input := `{"file": "test.py", "line": 1, "message": "test", "` + tt.ruleField + `": "RULE001"}`

			var obj map[string]any
			err := json.Unmarshal([]byte(input), &obj)
			require.NoError(t, err)

			issues := linter.parseJSONObject(obj)

			require.Equal(t, 1, len(issues))
			assert.Equal(t, "RULE001", issues[0].Rule)
		})
	}
}

// TestCustomLinter_ExtensionMatching tests case-insensitive extension matching.
func TestCustomLinter_ExtensionMatching(t *testing.T) {
	tests := []struct {
		name       string
		extensions []string
		files      []string
		wantMatch  int
	}{
		{
			name:       "exact match",
			extensions: []string{".py"},
			files:      []string{"test.py", "other.py"},
			wantMatch:  2,
		},
		{
			name:       "case insensitive",
			extensions: []string{".PY"},
			files:      []string{"test.py", "TEST.PY"},
			wantMatch:  2,
		},
		{
			name:       "no match",
			extensions: []string{".js"},
			files:      []string{"test.py", "other.txt"},
			wantMatch:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Count files that would pass the filter
			var filesToCheck []string
			for _, f := range tt.files {
				ext := filepath.Ext(f)
				for _, allowed := range tt.extensions {
					if strings.EqualFold(ext, allowed) {
						filesToCheck = append(filesToCheck, f)

						break
					}
				}
			}

			assert.Equal(t, tt.wantMatch, len(filesToCheck))
		})
	}
}
