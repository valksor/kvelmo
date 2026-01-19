// Package helper_test provides domain-specific testing utilities for go-mehrhof tests.
// Generic utilities (WriteFile, git helpers, assertions) should use github.com/valksor/go-toolkit/helper_test.
package helper_test

import (
	"os"
	"path/filepath"
	"testing"
)

// CreateTaskFile creates a task markdown file with the specified content.
func CreateTaskFile(t *testing.T, dir, filename, title, description string) string {
	t.Helper()
	content := "---\ntitle: " + title + "\n---\n\n" + description + "\n"
	path := filepath.Join(dir, filename)

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}

	return path
}

// CreateTaskDir creates a task directory with a README and optional subtask files.
func CreateTaskDir(t *testing.T, baseDir, taskName, readmeContent string, subtasks []string) string {
	t.Helper()
	taskDir := filepath.Join(baseDir, taskName)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", taskDir, err)
	}

	// Create README
	readmePath := filepath.Join(taskDir, "README.md")
	if err := os.MkdirAll(filepath.Dir(readmePath), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", filepath.Dir(readmePath), err)
	}
	if err := os.WriteFile(readmePath, []byte(readmeContent), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", readmePath, err)
	}

	// Create subtask files
	for i, subtask := range subtasks {
		filename := filepath.Join(taskDir, subtask)
		content := "---\ntitle: Subtask " + string(rune('1'+i)) + "\n---\n\n" + subtask + "\n"
		if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
			t.Fatalf("MkdirAll(%s): %v", filepath.Dir(filename), err)
		}
		if err := os.WriteFile(filename, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", filename, err)
		}
	}

	return taskDir
}

// CreateMakefile creates a Makefile with optional quality target.
func CreateMakefile(t *testing.T, dir string, hasQualityTarget bool) {
	t.Helper()
	content := ".PHONY: build test\n\nbuild:\n\t@echo building\n\ntest:\n\t@echo testing\n"
	if hasQualityTarget {
		content += "\n.PHONY: quality\nquality:\n\t@echo running quality checks\n"
	}

	path := filepath.Join(dir, "Makefile")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}
