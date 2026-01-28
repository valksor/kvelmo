package vcs

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectDefaultBranch(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Initialize git repo with 'main' branch
	ctx := context.Background()
	_, err := runGitCommandContext(ctx, tmpDir, "init", "-b", "main")
	if err != nil {
		t.Fatalf("git init: %v", err)
	}

	// Set user config for commits
	_, _ = runGitCommandContext(ctx, tmpDir, "config", "user.email", "test@test.com")
	_, _ = runGitCommandContext(ctx, tmpDir, "config", "user.name", "Test User")

	// Create a file and commit
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	_, _ = runGitCommandContext(ctx, tmpDir, "add", ".")
	_, _ = runGitCommandContext(ctx, tmpDir, "commit", "-m", "initial")

	g, err := New(ctx, tmpDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	branch, err := g.DetectDefaultBranch(ctx)
	if err != nil {
		t.Errorf("DetectDefaultBranch: %v", err)
	}

	if branch != "main" {
		t.Errorf("DetectDefaultBranch = %q, want %q", branch, "main")
	}
}

func TestDiffUncommitted(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	ctx := context.Background()

	// Initialize git repo
	_, err := runGitCommandContext(ctx, tmpDir, "init", "-b", "main")
	if err != nil {
		t.Fatalf("git init: %v", err)
	}

	// Set user config
	_, _ = runGitCommandContext(ctx, tmpDir, "config", "user.email", "test@test.com")
	_, _ = runGitCommandContext(ctx, tmpDir, "config", "user.name", "Test User")

	// Create a file and commit
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("original content\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	_, _ = runGitCommandContext(ctx, tmpDir, "add", ".")
	_, _ = runGitCommandContext(ctx, tmpDir, "commit", "-m", "initial")

	g, err := New(ctx, tmpDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	t.Run("no uncommitted changes", func(t *testing.T) {
		diff, err := g.DiffUncommitted(ctx, 3)
		if err != nil {
			t.Errorf("DiffUncommitted: %v", err)
		}
		if diff != "" {
			t.Errorf("DiffUncommitted with no changes = %q, want empty", diff)
		}
	})

	t.Run("with unstaged changes", func(t *testing.T) {
		// Modify the file
		if err := os.WriteFile(testFile, []byte("modified content\n"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}

		diff, err := g.DiffUncommitted(ctx, 3)
		if err != nil {
			t.Errorf("DiffUncommitted: %v", err)
		}
		if !strings.Contains(diff, "modified content") {
			t.Errorf("DiffUncommitted should contain 'modified content'")
		}
		if !strings.Contains(diff, "# Unstaged changes") {
			t.Errorf("DiffUncommitted should contain unstaged header")
		}
	})

	t.Run("with staged changes", func(t *testing.T) {
		// Stage the changes
		_, _ = runGitCommandContext(ctx, tmpDir, "add", ".")

		diff, err := g.DiffUncommitted(ctx, 3)
		if err != nil {
			t.Errorf("DiffUncommitted: %v", err)
		}
		if !strings.Contains(diff, "# Staged changes") {
			t.Errorf("DiffUncommitted should contain staged header")
		}
	})
}

func TestDiffBranch(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	ctx := context.Background()

	// Initialize git repo
	_, err := runGitCommandContext(ctx, tmpDir, "init", "-b", "main")
	if err != nil {
		t.Fatalf("git init: %v", err)
	}

	// Set user config
	_, _ = runGitCommandContext(ctx, tmpDir, "config", "user.email", "test@test.com")
	_, _ = runGitCommandContext(ctx, tmpDir, "config", "user.name", "Test User")

	// Create initial commit on main
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("main content\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	_, _ = runGitCommandContext(ctx, tmpDir, "add", ".")
	_, _ = runGitCommandContext(ctx, tmpDir, "commit", "-m", "initial")

	// Create and switch to feature branch
	_, _ = runGitCommandContext(ctx, tmpDir, "checkout", "-b", "feature")

	// Make changes on feature branch
	if err := os.WriteFile(testFile, []byte("feature content\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	_, _ = runGitCommandContext(ctx, tmpDir, "add", ".")
	_, _ = runGitCommandContext(ctx, tmpDir, "commit", "-m", "feature change")

	g, err := New(ctx, tmpDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	t.Run("diff against main", func(t *testing.T) {
		diff, err := g.DiffBranch(ctx, "main", 3)
		if err != nil {
			t.Errorf("DiffBranch: %v", err)
		}
		if !strings.Contains(diff, "feature content") {
			t.Errorf("DiffBranch should contain 'feature content'")
		}
	})

	t.Run("auto-detect base branch", func(t *testing.T) {
		diff, err := g.DiffBranch(ctx, "", 3)
		if err != nil {
			t.Errorf("DiffBranch with auto-detect: %v", err)
		}
		// Should work since main exists
		if diff == "" {
			t.Error("DiffBranch with auto-detect should return diff")
		}
	})
}

func TestDiffRange(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	// Initialize git repo
	_, err := runGitCommandContext(ctx, tmpDir, "init", "-b", "main")
	if err != nil {
		t.Fatalf("git init: %v", err)
	}

	// Set user config
	_, _ = runGitCommandContext(ctx, tmpDir, "config", "user.email", "test@test.com")
	_, _ = runGitCommandContext(ctx, tmpDir, "config", "user.name", "Test User")

	// Create multiple commits
	testFile := filepath.Join(tmpDir, "test.txt")

	for i := 1; i <= 3; i++ {
		content := []byte("commit " + string(rune('0'+i)) + "\n")
		if err := os.WriteFile(testFile, content, 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
		_, _ = runGitCommandContext(ctx, tmpDir, "add", ".")
		_, _ = runGitCommandContext(ctx, tmpDir, "commit", "-m", "commit "+string(rune('0'+i)))
	}

	g, err := New(ctx, tmpDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	diff, err := g.DiffRange(ctx, "HEAD~2..HEAD", 3)
	if err != nil {
		t.Errorf("DiffRange: %v", err)
	}
	if diff == "" {
		t.Error("DiffRange should return diff for commit range")
	}
}

func TestDiffFiles(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	// Initialize git repo
	_, err := runGitCommandContext(ctx, tmpDir, "init", "-b", "main")
	if err != nil {
		t.Fatalf("git init: %v", err)
	}

	// Set user config
	_, _ = runGitCommandContext(ctx, tmpDir, "config", "user.email", "test@test.com")
	_, _ = runGitCommandContext(ctx, tmpDir, "config", "user.name", "Test User")

	// Create files and commit
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	if err := os.WriteFile(file1, []byte("file1 content\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := os.WriteFile(file2, []byte("file2 content\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	_, _ = runGitCommandContext(ctx, tmpDir, "add", ".")
	_, _ = runGitCommandContext(ctx, tmpDir, "commit", "-m", "initial")

	// Modify file1 only
	if err := os.WriteFile(file1, []byte("file1 modified\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	g, err := New(ctx, tmpDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	t.Run("diff specific file", func(t *testing.T) {
		diff, err := g.DiffFiles(ctx, []string{"file1.txt"}, 3)
		if err != nil {
			t.Errorf("DiffFiles: %v", err)
		}
		if !strings.Contains(diff, "file1 modified") {
			t.Errorf("DiffFiles should contain 'file1 modified'")
		}
	})

	t.Run("diff file with no changes", func(t *testing.T) {
		diff, err := g.DiffFiles(ctx, []string{"file2.txt"}, 3)
		if err != nil {
			t.Errorf("DiffFiles: %v", err)
		}
		if diff != "" {
			t.Errorf("DiffFiles for unchanged file should be empty")
		}
	})
}
