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
