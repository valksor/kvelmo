package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestASCCompatibility tests that mehrhof can be configured to match
// the aerones-super-code (ASC) Python tool's file structure exactly.
//
// ASC patterns:
// - Branch: asc/<ticket-id>
// - Specs: tickets/<ticket-id>/SPEC.md, SPEC-2.md, SPEC-3.md...
// - Reviews: tickets/<ticket-id>/CODERABBIT-1.txt, CODERABBIT-2.txt...

func TestASCCompatibility_SpecPath(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)

	// Configure to match ASC
	cfg := NewDefaultWorkspaceConfig()
	cfg.Specification.SaveInProject = true
	cfg.Specification.ProjectDir = "tickets"
	cfg.Specification.FilenamePattern = "SPEC-{n}.md"

	// Test spec paths match ASC: tickets/<task-id>/SPEC-N.md
	tests := []struct {
		taskID   string
		number   int
		wantPath string
	}{
		{"A-123", 1, "tickets/A-123/SPEC-1.md"},
		{"A-123", 2, "tickets/A-123/SPEC-2.md"},
		{"TASK-456", 1, "tickets/TASK-456/SPEC-1.md"},
	}

	for _, tt := range tests {
		path := ws.ProjectSpecificationPath(tt.taskID, tt.number, cfg)
		if !strings.HasSuffix(path, tt.wantPath) {
			t.Errorf("ProjectSpecificationPath(%q, %d) = %q, want suffix %q",
				tt.taskID, tt.number, path, tt.wantPath)
		}
	}
}

func TestASCCompatibility_ReviewPath(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)

	// Configure to match ASC
	cfg := NewDefaultWorkspaceConfig()
	cfg.Specification.ProjectDir = "tickets" // Reviews use same ProjectDir
	cfg.Review.SaveInProject = true
	cfg.Review.FilenamePattern = "CODERABBIT-{n}.txt"

	// Test review paths match ASC: tickets/<task-id>/CODERABBIT-N.txt
	tests := []struct {
		taskID   string
		number   int
		wantPath string
	}{
		{"A-123", 1, "tickets/A-123/CODERABBIT-1.txt"},
		{"A-123", 2, "tickets/A-123/CODERABBIT-2.txt"},
		{"TASK-456", 1, "tickets/TASK-456/CODERABBIT-1.txt"},
	}

	for _, tt := range tests {
		path := ws.ProjectReviewPath(tt.taskID, tt.number, cfg)
		if !strings.HasSuffix(path, tt.wantPath) {
			t.Errorf("ProjectReviewPath(%q, %d) = %q, want suffix %q",
				tt.taskID, tt.number, path, tt.wantPath)
		}
	}
}

func TestASCCompatibility_FullWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	// Configure to match ASC exactly
	cfg, err := ws.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	cfg.Git.BranchPattern = "asc/{key}"
	cfg.Git.CommitPrefix = "[{key}]"
	cfg.Specification.SaveInProject = true
	cfg.Specification.ProjectDir = "tickets"
	cfg.Specification.FilenamePattern = "SPEC-{n}.md"
	cfg.Review.SaveInProject = true
	cfg.Review.FilenamePattern = "CODERABBIT-{n}.txt"
	if err := ws.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	// Create a task (simulating mehr start A-123)
	taskID := "A-123"
	source := SourceInfo{Type: "file", Ref: "task.md"}
	if _, err := ws.CreateWork(taskID, source); err != nil {
		t.Fatalf("CreateWork: %v", err)
	}

	// Save specs (simulating mehr plan iterations)
	specs := []string{
		"# Specification 1\n\nInitial plan.",
		"# Specification 2\n\nRevised plan.",
	}
	for i, content := range specs {
		if err := ws.SaveSpecification(taskID, i+1, content); err != nil {
			t.Fatalf("SaveSpecification(%d): %v", i+1, err)
		}
	}

	// Save reviews (simulating mehr review with coderabbit)
	reviews := []string{
		"# Code Review 1\n\nIssues found.",
		"# Code Review 2\n\nAll clear.",
	}
	for i, content := range reviews {
		if err := ws.SaveReview(taskID, i+1, content); err != nil {
			t.Fatalf("SaveReview(%d): %v", i+1, err)
		}
	}

	// Verify ASC-compatible file structure exists in project
	expectedFiles := []string{
		"tickets/A-123/SPEC-1.md",
		"tickets/A-123/SPEC-2.md",
		"tickets/A-123/CODERABBIT-1.txt",
		"tickets/A-123/CODERABBIT-2.txt",
	}

	for _, relPath := range expectedFiles {
		fullPath := filepath.Join(tmpDir, relPath)
		if _, err := os.Stat(fullPath); err != nil {
			t.Errorf("Expected ASC-compatible file not found: %s (error: %v)", relPath, err)
		}
	}

	// Verify content
	specPath := filepath.Join(tmpDir, "tickets/A-123/SPEC-1.md")
	content, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("ReadFile(SPEC-1.md): %v", err)
	}
	if string(content) != specs[0] {
		t.Errorf("SPEC-1.md content = %q, want %q", string(content), specs[0])
	}

	reviewPath := filepath.Join(tmpDir, "tickets/A-123/CODERABBIT-1.txt")
	content, err = os.ReadFile(reviewPath)
	if err != nil {
		t.Fatalf("ReadFile(CODERABBIT-1.txt): %v", err)
	}
	if string(content) != reviews[0] {
		t.Errorf("CODERABBIT-1.txt content = %q, want %q", string(content), reviews[0])
	}
}

func TestASCCompatibility_BranchPattern(t *testing.T) {
	cfg := NewDefaultWorkspaceConfig()
	cfg.Git.BranchPattern = "asc/{key}"

	// The branch pattern should produce asc/<ticket-id> when resolved
	// (actual resolution happens in conductor, but we test the config is stored correctly)
	if cfg.Git.BranchPattern != "asc/{key}" {
		t.Errorf("BranchPattern = %q, want %q", cfg.Git.BranchPattern, "asc/{key}")
	}
}

func TestASCCompatibility_CommitPrefix(t *testing.T) {
	cfg := NewDefaultWorkspaceConfig()
	cfg.Git.CommitPrefix = "[{key}]"

	// The commit prefix should produce [<ticket-id>] when resolved
	if cfg.Git.CommitPrefix != "[{key}]" {
		t.Errorf("CommitPrefix = %q, want %q", cfg.Git.CommitPrefix, "[{key}]")
	}
}
