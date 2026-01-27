//go:build e2e_fast
// +build e2e_fast

package e2e_test

import (
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	if os.Getenv("ZAI_API_KEY") == "" {
		println("Skipping e2e-fast: ZAI_API_KEY not set")
		os.Exit(0)
	}
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
	h.AssertOutputContains("Started task")

	h.RunWithTimeout("plan", 5*time.Minute)
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
	h.AssertOutputContains("Started task")

	h.RunWithTimeout("plan", 3*time.Minute)
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

	h.RunWithTimeout("plan", 3*time.Minute)
	h.AssertSuccess()

	h.RunWithTimeout("implement", 3*time.Minute, "--dry-run")
	h.AssertSuccess()
}
