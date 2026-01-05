package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUrlToProjectID(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "HTTPS URL",
			url:      "https://github.com/user/repo.git",
			expected: "github.com-user-repo",
		},
		{
			name:     "SSH URL",
			url:      "git@github.com:user/repo.git",
			expected: "github.com-user-repo",
		},
		{
			name:     "Nested groups",
			url:      "https://gitlab.com/group/subgroup/project.git",
			expected: "gitlab.com-group-subgroup-project",
		},
		{
			name:     "Without .git suffix",
			url:      "https://github.com/user/repo",
			expected: "github.com-user-repo",
		},
		{
			name:     "Deeply nested",
			url:      "https://gitlab.com/group/subgroup/subsubgroup/project.git",
			expected: "gitlab.com-group-subgroup-subsubgroup-project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := urlToProjectID(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHashPathToFallbackID(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		prefixLen int // Should be "local-" + 10 hex chars = 16 total
	}{
		{
			name:      "Simple path",
			path:      "/home/user/projects/myproject",
			prefixLen: 16, // "local-" + 10 hex chars
		},
		{
			name:      "Relative path",
			path:      "../myproject",
			prefixLen: 16,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hashPathToFallbackID(tt.path)
			assert.Equal(t, "local-", result[:6])
			assert.Len(t, result, tt.prefixLen)
			// Verify it's hex after the prefix
			for _, c := range result[6:] {
				assert.True(t, (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'),
					"Expected hex character, got: %c", c)
			}
		})
	}
}

func TestMigrateFromLegacy(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	legacyDir := filepath.Join(tempDir, "repo", ".mehrhof")
	newWorkspaceDir := filepath.Join(tempDir, "new-workspace")
	repoRoot := filepath.Join(tempDir, "repo")

	// Create legacy workspace structure
	require.NoError(t, os.MkdirAll(legacyDir, 0o755))
	configContent := "git:\n  auto_commit: true\n"
	require.NoError(t, os.WriteFile(filepath.Join(legacyDir, "config.yaml"),
		[]byte(configContent), 0o644))

	// Create work/ directory with some data
	legacyWorkDir := filepath.Join(legacyDir, "work")
	require.NoError(t, os.MkdirAll(legacyWorkDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(legacyWorkDir, "test.txt"), []byte("test data"), 0o644))

	// Create active task file in repo root
	activeTaskContent := "id: abc123\nref: file:test.md\n"
	require.NoError(t, os.WriteFile(filepath.Join(repoRoot, ".active_task"),
		[]byte(activeTaskContent), 0o644))

	// Create workspace pointing to new location
	ws := &Workspace{
		root:          repoRoot,
		taskRoot:      legacyDir,       // .mehrhof in project
		workspaceRoot: newWorkspaceDir, // home dir location
		workRoot:      filepath.Join(newWorkspaceDir, "work"),
	}

	// Run migration
	err := ws.MigrateFromLegacy()
	require.NoError(t, err)

	// Verify config.yaml stays in project (not moved)
	_, err = os.Stat(filepath.Join(legacyDir, "config.yaml"))
	assert.NoError(t, err, "config.yaml should remain in project")

	// Verify work/ was moved to home directory
	_, err = os.Stat(filepath.Join(newWorkspaceDir, "work", "test.txt"))
	assert.NoError(t, err, "work/ should be moved to home directory")

	// Verify .active_task was moved to home directory
	_, err = os.Stat(filepath.Join(newWorkspaceDir, ".active_task"))
	assert.NoError(t, err, ".active_task should be moved to home directory")

	// Verify .active_task was removed from repo root
	_, err = os.Stat(filepath.Join(repoRoot, ".active_task"))
	assert.True(t, os.IsNotExist(err), ".active_task should be removed from repo root")

	// Verify legacy work/ directory was removed
	_, err = os.Stat(legacyWorkDir)
	assert.True(t, os.IsNotExist(err), "legacy work/ directory should be removed")
}

func TestMigrateFromLegacy_NoWorkDirectory(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	legacyDir := filepath.Join(tempDir, "repo", ".mehrhof")
	newWorkspaceDir := filepath.Join(tempDir, "new-workspace")
	repoRoot := filepath.Join(tempDir, "repo")

	// Create legacy workspace with only config.yaml (no work/ directory)
	require.NoError(t, os.MkdirAll(legacyDir, 0o755))
	configContent := "git:\n  auto_commit: true\n"
	require.NoError(t, os.WriteFile(filepath.Join(legacyDir, "config.yaml"),
		[]byte(configContent), 0o644))

	// Create workspace pointing to new location
	ws := &Workspace{
		root:          repoRoot,
		taskRoot:      legacyDir,
		workspaceRoot: newWorkspaceDir,
		workRoot:      filepath.Join(newWorkspaceDir, "work"),
	}

	// Run migration - should succeed even without work/ directory
	err := ws.MigrateFromLegacy()
	require.NoError(t, err)

	// Verify config.yaml stays in project
	_, err = os.Stat(filepath.Join(legacyDir, "config.yaml"))
	assert.NoError(t, err, "config.yaml should remain in project")
}

func TestNeedsMigration(t *testing.T) {
	t.Run("Legacy directory exists", func(t *testing.T) {
		tempDir := t.TempDir()
		legacyDir := filepath.Join(tempDir, ".mehrhof")
		require.NoError(t, os.MkdirAll(legacyDir, 0o755))

		ws := &Workspace{
			root:     tempDir,
			taskRoot: filepath.Join(tempDir, ".mehrhof"),
		}

		assert.True(t, ws.NeedsMigration())
	})

	t.Run("Legacy directory does not exist", func(t *testing.T) {
		tempDir := t.TempDir()

		ws := &Workspace{
			root:     tempDir,
			taskRoot: filepath.Join(tempDir, ".mehrhof"),
		}

		assert.False(t, ws.NeedsMigration())
	})
}

func TestMigrateFromLegacy_NewWorkspaceAlreadyExists(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	legacyDir := filepath.Join(tempDir, "repo", ".mehrhof")
	newWorkspaceDir := filepath.Join(tempDir, "new-workspace")
	repoRoot := filepath.Join(tempDir, "repo")

	// Create legacy workspace structure
	require.NoError(t, os.MkdirAll(legacyDir, 0o755))
	configContent := "git:\n  auto_commit: true\n"
	require.NoError(t, os.WriteFile(filepath.Join(legacyDir, "config.yaml"),
		[]byte(configContent), 0o644))

	// Create new workspace directory (simulating it already exists from previous task)
	require.NoError(t, os.MkdirAll(newWorkspaceDir, 0o755))
	existingFile := filepath.Join(newWorkspaceDir, "existing.txt")
	require.NoError(t, os.WriteFile(existingFile, []byte("existing data"), 0o644))

	// Create legacy work/ directory that won't be migrated
	legacyWorkDir := filepath.Join(legacyDir, "work")
	require.NoError(t, os.MkdirAll(legacyWorkDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(legacyWorkDir, "test.txt"), []byte("test data"), 0o644))

	// Create workspace pointing to new location
	ws := &Workspace{
		root:          repoRoot,
		taskRoot:      legacyDir,
		workspaceRoot: newWorkspaceDir,
		workRoot:      filepath.Join(newWorkspaceDir, "work"),
	}

	// Run migration - should succeed and skip migration since new workspace exists
	err := ws.MigrateFromLegacy()
	require.NoError(t, err)

	// Verify new workspace existing file is still there (not overwritten)
	_, err = os.Stat(existingFile)
	assert.NoError(t, err, "existing file in new workspace should remain")

	// Verify legacy work/ was NOT moved (migration was skipped)
	_, err = os.Stat(legacyWorkDir)
	assert.NoError(t, err, "legacy work/ directory should still exist since migration was skipped")
}

func TestGetLegacyTaskRoot(t *testing.T) {
	repoRoot := "/home/user/project"
	ws := &Workspace{
		root: repoRoot,
	}

	expected := filepath.Join(repoRoot, ".mehrhof")
	assert.Equal(t, expected, ws.GetLegacyTaskRoot())
}

func TestCopyDir(t *testing.T) {
	tempDir := t.TempDir()
	srcDir := filepath.Join(tempDir, "src")
	dstDir := filepath.Join(tempDir, "dst")

	// Create source directory structure
	require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "subdir"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("content1"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "subdir", "file2.txt"), []byte("content2"), 0o644))

	// Copy directory
	err := copyDir(srcDir, dstDir)
	require.NoError(t, err)

	// Verify files were copied
	content1, err := os.ReadFile(filepath.Join(dstDir, "file1.txt"))
	require.NoError(t, err)
	assert.Equal(t, "content1", string(content1))

	content2, err := os.ReadFile(filepath.Join(dstDir, "subdir", "file2.txt"))
	require.NoError(t, err)
	assert.Equal(t, "content2", string(content2))
}
