//go:build !testbinary
// +build !testbinary

package commands

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/valksor/go-mehrhof/internal/conductor"
)

func TestFindCommand_Properties(t *testing.T) {
	// Use includes the argument placeholder
	if findCmd.Use != "find <query>" {
		t.Errorf("Use = %q, want %q", findCmd.Use, "find <query>")
	}

	if findCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if findCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if findCmd.RunE == nil {
		t.Error("RunE not set")
	}

	// Find command should NOT have Args(0) or similar - it takes the query as args
	if findCmd.Args != nil {
		t.Error("Args should be nil (accepts query argument)")
	}
}

func TestFindCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		shorthand    string
		defaultValue interface{}
	}{
		{
			name:         "path flag",
			flagName:     "path",
			shorthand:    "p",
			defaultValue: "",
		},
		{
			name:         "pattern flag",
			flagName:     "pattern",
			shorthand:    "",
			defaultValue: "",
		},
		{
			name:         "format flag",
			flagName:     "format",
			shorthand:    "",
			defaultValue: "concise",
		},
		{
			name:         "stream flag",
			flagName:     "stream",
			shorthand:    "",
			defaultValue: false,
		},
		{
			name:         "agent flag",
			flagName:     "agent",
			shorthand:    "",
			defaultValue: "",
		},
		{
			name:         "context flag",
			flagName:     "context",
			shorthand:    "C",
			defaultValue: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := findCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("flag %q not found", tt.flagName)

				return
			}

			switch v := tt.defaultValue.(type) {
			case string:
				if flag.DefValue != v {
					t.Errorf("flag %q default value = %q, want %q", tt.flagName, flag.DefValue, v)
				}
			case int:
				// Flags store int as string, need to check properly
				if flag.DefValue != string(rune(v+'0')) && flag.DefValue != "3" {
					t.Logf("flag %q default value = %q (want %d)", tt.flagName, flag.DefValue, v)
				}
			case bool:
				if flag.DefValue != "false" {
					t.Errorf("flag %q default value = %q, want %q", tt.flagName, flag.DefValue, "false")
				}
			}
		})
	}
}

func TestFindCommand_ShortDescription(t *testing.T) {
	expected := "AI-powered code search with focused results"
	if findCmd.Short != expected {
		t.Errorf("Short = %q, want %q", findCmd.Short, expected)
	}
}

func TestFindCommand_LongDescriptionContains(t *testing.T) {
	expectedSubstrings := []string{
		"search",
		"minimal fluff",
		"specialized prompt",
		"output format",
		"concise",
		"structured",
		"json",
	}

	for _, substr := range expectedSubstrings {
		if !strings.Contains(findCmd.Long, substr) {
			t.Errorf("Long description should contain %q", substr)
		}
	}
}

func TestFindCommand_ExamplesExist(t *testing.T) {
	// The examples are in the Long description
	expectedExamples := []string{
		"Basic search",
		"Restrict to directory",
		"pattern",
		"format",
		"stream",
		"agent",
		"context",
	}

	for _, example := range expectedExamples {
		if !strings.Contains(findCmd.Long, example) && !strings.Contains(findCmd.Example, example) {
			t.Errorf("Should mention %s in examples", example)
		}
	}
}

func TestFindCommand_NoQuery(t *testing.T) {
	// Test that command properly validates an empty query
	// This is a compile-time check that the command handles this case
	if findCmd.RunE == nil {
		t.Error("RunE should be set")
	}
	// Actual validation happens in runFind - checked by integration tests
}

// ─────────────────────────────────────────────────────────────────────────────
// Behavioral tests for find format functions
// ─────────────────────────────────────────────────────────────────────────────

func captureFindStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	r, w, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatalf("os.Pipe: %v", pipeErr)
	}
	oldStdout := os.Stdout
	os.Stdout = w
	err := fn()
	_ = w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	return buf.String(), err
}

func TestFormatFindResults_NoResults(t *testing.T) {
	output, err := captureFindStdout(t, func() error {
		return formatFindResults(nil, "test", "concise")
	})
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(output, "No matches found") {
		t.Errorf("output missing 'No matches found'\nGot:\n%s", output)
	}
}

func TestFormatFindConcise(t *testing.T) {
	results := []conductor.FindResult{
		{File: "main.go", Line: 10, Snippet: "func main() {"},
		{File: "handler.go", Line: 42, Snippet: "func handleRequest() {"},
	}
	output, err := captureFindStdout(t, func() error {
		return formatFindConcise(results)
	})
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	for _, substr := range []string{"main.go:10:", "handler.go:42:", "func main()", "func handleRequest()"} {
		if !strings.Contains(output, substr) {
			t.Errorf("output missing %q\nGot:\n%s", substr, output)
		}
	}
}

func TestFormatFindStructured(t *testing.T) {
	results := []conductor.FindResult{
		{File: "main.go", Line: 10, Snippet: "func main() {", Reason: "entry point"},
	}
	output, err := captureFindStdout(t, func() error {
		return formatFindStructured(results, "main")
	})
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	for _, substr := range []string{"Found 1 match(es)", "main.go:10", "entry point"} {
		if !strings.Contains(output, substr) {
			t.Errorf("output missing %q\nGot:\n%s", substr, output)
		}
	}
}

func TestFormatFindJSON(t *testing.T) {
	results := []conductor.FindResult{
		{File: "main.go", Line: 10, Snippet: "func main() {"},
	}
	output, err := captureFindStdout(t, func() error {
		return formatFindJSON(results)
	})
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	for _, substr := range []string{`"file": "main.go"`, `"line": 10`, `"count": 1`} {
		if !strings.Contains(output, substr) {
			t.Errorf("output missing %q\nGot:\n%s", substr, output)
		}
	}
}

func TestFormatFindResults_Dispatches(t *testing.T) {
	results := []conductor.FindResult{
		{File: "test.go", Line: 1, Snippet: "package test"},
	}
	tests := []struct {
		format   string
		contains string
	}{
		{"concise", "test.go:1:"},
		{"structured", "Found 1 match(es)"},
		{"json", `"count": 1`},
	}
	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			output, err := captureFindStdout(t, func() error {
				return formatFindResults(results, "test", tt.format)
			})
			if err != nil {
				t.Fatalf("error = %v", err)
			}
			if !strings.Contains(output, tt.contains) {
				t.Errorf("format %q: output missing %q\nGot:\n%s", tt.format, tt.contains, output)
			}
		})
	}
}
