//go:build e2e_fast
// +build e2e_fast

package e2e_test

import (
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

// TestHappyPath validates the complete workflow.
func TestHappyPath(t *testing.T) {
	dir := t.TempDir()
	h := NewHelper(t, dir)

	h.InitWithLocalConfig()

	h.WriteTask("simple.md", `---
title: Add hello function
---

Create hello.go with a function Hello() string that returns "Hello, World!"
`)

	h.Run("start", "file:simple.md", "--no-branch")
	h.AssertSuccess()
	h.AssertOutputContains("Task started")

	h.RunWithTimeout("plan", 5*time.Minute, "--auto-approve")
	h.AssertSuccess()

	h.RunWithTimeout("implement", 5*time.Minute)
	h.AssertSuccess()
	h.AssertFileExists("hello.go")
	h.AssertFileContains("hello.go", "func Hello()")

	h.RunWithTimeout("review", 3*time.Minute)
	h.AssertSuccess()

	h.Run("finish", "--yes")
	h.AssertSuccess()
}

// TestBasicCommands validates commands that don't need the agent.
func TestBasicCommands(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"version", []string{"version"}},
		{"help", []string{"help"}},
		{"init", []string{"init"}},
		{"agents list", []string{"agents", "list"}},
		{"providers list", []string{"providers", "list"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			h := NewHelper(t, dir)
			h.Run(tt.args...)
			h.AssertSuccess()
		})
	}
}

// TestStartAndPlan tests starting a task and running plan.
func TestStartAndPlan(t *testing.T) {
	dir := t.TempDir()
	h := NewHelper(t, dir)

	h.InitWithLocalConfig()

	h.WriteTask("task.md", `---
title: Create a README
---

Add a README.md file with a simple description.
`)

	h.Run("start", "file:task.md", "--no-branch")
	h.AssertSuccess()
	h.AssertOutputContains("Task started")

	h.RunWithTimeout("plan", 3*time.Minute, "--auto-approve")
	h.AssertSuccess()
}

// TestImplementDryRun tests implement with dry-run flag.
func TestImplementDryRun(t *testing.T) {
	dir := t.TempDir()
	h := NewHelper(t, dir)

	h.InitWithLocalConfig()

	h.WriteTask("task.md", `---
title: Add comment
---

Add a comment to main.go
`)

	h.Run("start", "file:task.md", "--no-branch")
	h.AssertSuccess()

	h.RunWithTimeout("plan", 3*time.Minute, "--auto-approve")
	h.AssertSuccess()

	h.RunWithTimeout("implement", 3*time.Minute, "--dry-run")
	h.AssertSuccess()
}

// TestAbsolutePathHandling verifies that files are created in the correct location
// even when the agent specification contains absolute paths.
// This is a regression test for the bug where paths like /tmp/test/hello.md
// would create nested directories like /tmp/test/tmp/test/hello.md.
func TestAbsolutePathHandling(t *testing.T) {
	dir := t.TempDir()
	h := NewHelper(t, dir)

	h.InitWithLocalConfig()

	h.WriteTask("task.md", `---
title: Create hello file
---

Create a file hello.md in the current directory with content "Hello, World!"
`)

	h.Run("start", "file:task.md", "--no-branch")
	h.AssertSuccess()

	h.RunWithTimeout("plan", 3*time.Minute, "--auto-approve")
	h.AssertSuccess()

	h.RunWithTimeout("implement", 3*time.Minute)
	h.AssertSuccess()

	// Verify file is created in the correct location (not nested)
	h.AssertFileExists("hello.md")
	h.AssertFileContains("hello.md", "Hello, World!")

	// Verify no nested directories were created (e.g., agent didn't create /tmp/test/hello.md)
	h.AssertFileNotExists("tmp")
}

// TestFindCommand tests the find command functionality.
func TestFindCommand(t *testing.T) {
	dir := t.TempDir()
	h := NewHelper(t, dir)

	h.InitWithLocalConfig()

	// Create some test files to search
	h.WriteFile("internal/test.go", `package main

func hello() string {
	return "Hello, World!"
}

func goodbye() string {
	return "Goodbye!"
}
`)

	h.WriteFile("cmd/main.go", `package main

func main() {
	println(hello())
}
`)

	h.WriteFile("README.md", `# Test Project

This is a test project for the find command.
`)

	// Test basic find
	h.Run("find", "hello")
	h.AssertSuccess()
	h.AssertOutputContains("hello")
}

// TestFindWithPath tests find command with path restriction.
func TestFindWithPath(t *testing.T) {
	dir := t.TempDir()
	h := NewHelper(t, dir)

	h.InitWithLocalConfig()

	// Create files in different directories
	h.WriteFile("internal/handler.go", "package internal\n\nfunc Handle() {}\n")
	h.WriteFile("cmd/main.go", "package main\n\nfunc main() {}\n")

	// Search only in internal directory
	h.Run("find", "func", "--path", "internal")
	h.AssertSuccess()
	h.AssertOutputContains("internal")
	// Should not contain cmd/main.go results
}

// TestFindWithPattern tests find command with file pattern.
func TestFindWithPattern(t *testing.T) {
	dir := t.TempDir()
	h := NewHelper(t, dir)

	h.InitWithLocalConfig()

	// Create different file types
	h.WriteFile("test.go", "package main\n\nfunc test() {}\n")
	h.WriteFile("test.txt", "This is a text file with the word func in it.\n")
	h.WriteFile("test.md", "# Documentation\n\nSome func documentation.\n")

	// Search for "func" only in Go files
	h.Run("find", "func", "--pattern", "*.go")
	h.AssertSuccess()
	h.AssertOutputContains("test.go")
}

// TestFindFormats tests find command with different output formats.
func TestFindFormats(t *testing.T) {
	dir := t.TempDir()
	h := NewHelper(t, dir)

	h.InitWithLocalConfig()

	h.WriteFile("test.go", "package main\n\nfunc test() {}\n")

	tests := []struct {
		name     string
		format   string
		contains []string
	}{
		{
			name:     "concise format",
			format:   "concise",
			contains: []string{"test.go"},
		},
		{
			name:     "structured format",
			format:   "structured",
			contains: []string{"Found", "match", "test.go"},
		},
		{
			name:     "json format",
			format:   "json",
			contains: []string{"{", "}", "matches"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h.Run("find", "func", "--format", tt.format)
			h.AssertSuccess()
			for _, expected := range tt.contains {
				h.AssertOutputContains(expected)
			}
		})
	}
}

// TestFindNoMatches tests find command when no matches are found.
func TestFindNoMatches(t *testing.T) {
	dir := t.TempDir()
	h := NewHelper(t, dir)

	h.InitWithLocalConfig()

	h.WriteFile("test.go", "package main\n\nfunc test() {}\n")

	// Search for something that doesn't exist
	h.Run("find", "nonexistent_function_xyz_123")
	h.AssertSuccess()
	h.AssertOutputContains("No matches")
}

// TestFindWithoutAgent tests local file search when agent is unavailable.
func TestFindWithoutAgent(t *testing.T) {
	dir := t.TempDir()
	h := NewHelper(t, dir)

	// Don't initialize with config (no agent available)
	h.WriteFile("test.go", "package main\n\nfunc test() {}\n")

	// Find should fall back to local file search
	// This test verifies the command works even without an agent configured
	h.Run("find", "test")
	// May fail if no agent available, but shouldn't panic
	// The output should either contain results or an agent error message
}

// TestFindNoQuery tests that find command properly handles missing query.
func TestFindNoQuery(t *testing.T) {
	dir := t.TempDir()
	h := NewHelper(t, dir)

	h.InitWithLocalConfig()

	// Run find without a query
	h.Run("find")
	// Should fail with error about query being required
	if h.lastExit == 0 {
		t.Error("find without query should fail")
	}
}

// WriteFile creates a file with the given content.
func (h *Helper) WriteFile(name, content string) {
	path := h.JoinDir(name)
	if err := os.MkdirAll(h.Dir(), 0o755); err != nil {
		h.t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		h.t.Fatalf("WriteFile: %v", err)
	}
}

// JoinDir joins the directory with the given path.
func (h *Helper) JoinDir(name string) string {
	return h.dir + "/" + name
}
