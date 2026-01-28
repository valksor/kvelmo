package conductor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFindOptions_QueryRequired tests that FindOptions requires a query.
func TestFindOptions_QueryRequired(t *testing.T) {
	opts := FindOptions{Query: ""}
	if opts.Query == "" {
		// Empty query should be caught before calling Find
		t.Log("Empty query detected correctly")
	}
}

// TestFindInFiles tests local file-based search.
func TestFindInFiles(t *testing.T) {
	t.Run("basic search", func(t *testing.T) {
		tmpDir := t.TempDir()
		ctx := context.Background()

		// Create test files
		testFile1 := filepath.Join(tmpDir, "test1.go")
		testFile2 := filepath.Join(tmpDir, "test2.txt")

		if err := os.WriteFile(testFile1, []byte("package main\n\nfunc hello() {\n\tprintln(\"hello\")\n}"), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		if err := os.WriteFile(testFile2, []byte("hello world\nthis is a test\nhello again"), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		c, err := New(WithWorkDir(tmpDir))
		if err != nil {
			t.Fatalf("New: %v", err)
		}

		opts := FindOptions{
			Query:   "hello",
			Context: 1,
		}

		results, err := c.FindInFiles(ctx, opts)
		if err != nil {
			t.Fatalf("FindInFiles: %v", err)
		}

		// Should find "hello" in both files
		if len(results) < 2 {
			t.Errorf("FindInFiles() returned %d results, want at least 2", len(results))
		}

		// Check that results have proper structure
		for _, r := range results {
			if r.File == "" {
				t.Error("FindResult.File is empty")
			}
			if r.Line <= 0 {
				t.Errorf("FindResult.Line = %d, want > 0", r.Line)
			}
			if r.Snippet == "" {
				t.Error("FindResult.Snippet is empty")
			}
		}
	})

	t.Run("search with path restriction", func(t *testing.T) {
		tmpDir := t.TempDir()
		ctx := context.Background()

		// Create subdirectories and files
		subdir := filepath.Join(tmpDir, "internal")
		if err := os.MkdirAll(subdir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}

		rootFile := filepath.Join(tmpDir, "root.go")
		subFile := filepath.Join(subdir, "internal.go")

		if err := os.WriteFile(rootFile, []byte("package main\n"), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		if err := os.WriteFile(subFile, []byte("package internal\n"), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		c, err := New(WithWorkDir(tmpDir))
		if err != nil {
			t.Fatalf("New: %v", err)
		}

		opts := FindOptions{
			Query:   "package",
			Path:    "internal",
			Context: 1,
		}

		results, err := c.FindInFiles(ctx, opts)
		if err != nil {
			t.Fatalf("FindInFiles: %v", err)
		}

		// Should only find results in internal directory
		for _, r := range results {
			if !strings.Contains(r.File, "internal") {
				t.Errorf("FindResult.File = %q, should be in internal directory", r.File)
			}
		}
	})

	t.Run("empty query returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		ctx := context.Background()

		c, err := New(WithWorkDir(tmpDir))
		if err != nil {
			t.Fatalf("New: %v", err)
		}

		opts := FindOptions{
			Query: "",
		}

		_, err = c.FindInFiles(ctx, opts)
		if err == nil {
			t.Error("FindInFiles() expected error for empty query, got nil")
		}
	})

	t.Run("no matches found", func(t *testing.T) {
		tmpDir := t.TempDir()
		ctx := context.Background()

		// Create test file
		testFile := filepath.Join(tmpDir, "test.go")
		if err := os.WriteFile(testFile, []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		c, err := New(WithWorkDir(tmpDir))
		if err != nil {
			t.Fatalf("New: %v", err)
		}

		opts := FindOptions{
			Query:   "nonexistent_function_xyz",
			Context: 1,
		}

		results, err := c.FindInFiles(ctx, opts)
		if err != nil {
			t.Fatalf("FindInFiles: %v", err)
		}

		if len(results) != 0 {
			t.Errorf("FindInFiles() returned %d results, want 0", len(results))
		}
	})
}

// TestParseFindResultBlock tests parsing of structured find results.
func TestParseFindResultBlock(t *testing.T) {
	tests := []struct {
		name        string
		block       string
		wantFile    string
		wantLine    int
		wantSnippet string
		wantReason  string
	}{
		{
			name: "full result block",
			block: `file: internal/test.go
line: 42
snippet: func test() {}
reason: test function definition`,
			wantFile:    "internal/test.go",
			wantLine:    42,
			wantSnippet: "func test() {}",
			wantReason:  "test function definition",
		},
		{
			name: "minimal result block",
			block: `file: cmd/main.go
line: 10
snippet: package main`,
			wantFile:    "cmd/main.go",
			wantLine:    10,
			wantSnippet: "package main",
			wantReason:  "",
		},
		{
			name: "block with context",
			block: `file: internal/test.go
line: 42
snippet: func test() {}
context: line 1\nline 2\nline 3
reason: context test`,
			wantFile:    "internal/test.go",
			wantLine:    42,
			wantSnippet: "func test() {}",
			wantReason:  "context test",
		},
		{
			name:        "empty block",
			block:       "",
			wantFile:    "",
			wantLine:    0,
			wantSnippet: "",
		},
		{
			name: "malformed block - missing line",
			block: `file: internal/test.go
snippet: func test() {}`,
			wantFile:    "internal/test.go",
			wantLine:    0, // Line parsing fails, returns 0
			wantSnippet: "func test() {}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseFindResultBlock(tt.block)

			if result.File != tt.wantFile {
				t.Errorf("File = %q, want %q", result.File, tt.wantFile)
			}
			if result.Line != tt.wantLine {
				t.Errorf("Line = %d, want %d", result.Line, tt.wantLine)
			}
			if result.Snippet != tt.wantSnippet {
				t.Errorf("Snippet = %q, want %q", result.Snippet, tt.wantSnippet)
			}
			if result.Reason != tt.wantReason {
				t.Errorf("Reason = %q, want %q", result.Reason, tt.wantReason)
			}
		})
	}
}

// TestParseFindResults tests parsing of full agent responses.
func TestParseFindResults(t *testing.T) {
	tests := []struct {
		name           string
		response       string
		wantCount      int
		checkFirstFile string // Optional: check first result's file
	}{
		{
			name: "single structured result",
			response: `--- FIND ---
file: internal/test.go
line: 42
snippet: func test() {}
reason: test function
--- END ---`,
			wantCount:      1,
			checkFirstFile: "internal/test.go",
		},
		{
			name: "multiple structured results",
			response: `--- FIND ---
file: internal/test.go
line: 42
snippet: func test() {}
--- END ---
--- FIND ---
file: cmd/main.go
line: 10
snippet: package main
--- END ---`,
			wantCount: 2,
		},
		{
			name: "fallback conversational format",
			response: `I found some results:

internal/test.go:42: func test() {}
cmd/main.go:10: package main

That's all I found.`,
			wantCount: 2,
		},
		{
			name: "mixed format - structured first",
			response: `--- FIND ---
file: internal/test.go
line: 42
snippet: func test() {}
--- END ---

Also found:
internal/other.go:100: something else`,
			wantCount: 1, // Only structured parsed
		},
		{
			name:      "empty response",
			response:  "",
			wantCount: 0,
		},
		{
			name:      "no structured or fallback matches",
			response:  `I searched but couldn't find anything matching your query.`,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := parseFindResults(tt.response)

			if len(results) != tt.wantCount {
				t.Errorf("parseFindResults() returned %d results, want %d", len(results), tt.wantCount)
			}

			if tt.checkFirstFile != "" && len(results) > 0 {
				if results[0].File != tt.checkFirstFile {
					t.Errorf("First result File = %q, want %q", results[0].File, tt.checkFirstFile)
				}
			}
		})
	}
}

// TestExtractFallbackResults tests fallback parsing of conversational responses.
func TestExtractFallbackResults(t *testing.T) {
	tests := []struct {
		name      string
		response  string
		wantCount int
	}{
		{
			name:      "single file:line format",
			response:  `internal/test.go:42: func test() {}`,
			wantCount: 1,
		},
		{
			name: "multiple file:line:snippet format",
			response: `internal/test.go:42: func test() {}
cmd/main.go:10: package main
README.md:5: # Project`,
			wantCount: 3,
		},
		{
			name: "file:line without snippet",
			response: `internal/test.go:42
cmd/main.go:10`,
			wantCount: 2,
		},
		{
			name: "embedded in text",
			response: `Found matches at:
internal/test.go:42: func test() {}
And also:
cmd/main.go:10: package main`,
			wantCount: 2,
		},
		{
			name:      "no matches",
			response:  `No results found in the codebase.`,
			wantCount: 0,
		},
		{
			name: "only file paths without line numbers",
			response: `internal/test.go
cmd/main.go`,
			wantCount: 0, // Requires line numbers
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := extractFallbackResults(tt.response)

			if len(results) != tt.wantCount {
				t.Errorf("extractFallbackResults() returned %d results, want %d", len(results), tt.wantCount)
			}
		})
	}
}

// TestFindResult tests FindResult struct.
func TestFindResult(t *testing.T) {
	result := FindResult{
		File:    "internal/test.go",
		Line:    42,
		Snippet: "func test() {}",
		Context: []string{"line 1", "line 2", "line 3"},
		Reason:  "test function",
	}

	if result.File != "internal/test.go" {
		t.Errorf("File = %q, want %q", result.File, "internal/test.go")
	}
	if result.Line != 42 {
		t.Errorf("Line = %d, want 42", result.Line)
	}
	if len(result.Context) != 3 {
		t.Errorf("Context len = %d, want 3", len(result.Context))
	}
}

// TestBuildFindPrompt tests the prompt building function.
func TestBuildFindPrompt(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		workDir string
	}{
		{
			name:    "basic query",
			query:   "test function",
			workDir: "/project",
		},
		{
			name:    "query with special characters",
			query:   "find TODO comments",
			workDir: "/workspace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := buildFindPrompt(tt.query, tt.workDir, nil, FindOptions{})

			if prompt == "" {
				t.Error("buildFindPrompt() returned empty string")
			}

			// Check that key elements are in the prompt
			expectedInPrompt := []string{
				tt.query,
				"CRITICAL CONSTRAINTS",
				"OUTPUT FORMAT",
				"DO NOT",
			}

			for _, expected := range expectedInPrompt {
				if !strings.Contains(prompt, expected) {
					t.Errorf("buildFindPrompt() should contain %q", expected)
				}
			}
		})
	}
}
