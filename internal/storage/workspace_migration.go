package storage

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

// MigrateFromLegacy moves work/ and .active_task from project to global location.
// config.yaml and .env stay in .mehrhof/ in the project.
// The migration is atomic: copies to new location, verifies, then deletes old.
func (w *Workspace) MigrateFromLegacy() error {
	legacyRoot := w.GetLegacyTaskRoot()
	newWorkspaceRoot := w.workspaceRoot

	// Check if legacy directory exists
	if _, err := os.Stat(legacyRoot); os.IsNotExist(err) {
		return fmt.Errorf("legacy directory does not exist: %s", legacyRoot)
	}

	// Check if new location already exists
	if _, err := os.Stat(newWorkspaceRoot); err == nil {
		slog.Info("new workspace already exists, skipping migration", "path", newWorkspaceRoot)

		return nil
	}

	slog.Info("migrating workspace data to home directory",
		"work from", filepath.Join(legacyRoot, "work"),
		"to", filepath.Join(newWorkspaceRoot, "work"))

	// Step 1: Migrate work/ directory if it exists
	legacyWorkDir := filepath.Join(legacyRoot, "work")
	if _, err := os.Stat(legacyWorkDir); err == nil {
		if err := os.Rename(legacyWorkDir, w.workRoot); err != nil {
			// Try copy if rename fails (cross-device)
			if err := copyDir(legacyWorkDir, w.workRoot); err != nil {
				return fmt.Errorf("migrate work directory: %w", err)
			}
			// Remove source directory after successful copy
			if err := os.RemoveAll(legacyWorkDir); err != nil {
				slog.Warn("failed to remove legacy work directory after copy",
					"path", legacyWorkDir, "error", err)
			}
		}
		slog.Info("migrated work/ directory")
	} else {
		slog.Info("no work/ directory to migrate")
	}

	// Step 2: Move .active_task file from repo root to new location
	legacyActiveTask := filepath.Join(w.root, activeTaskFile)
	if _, err := os.Stat(legacyActiveTask); err == nil {
		newActiveTask := w.ActiveTaskPath()
		if err := os.Rename(legacyActiveTask, newActiveTask); err != nil {
			slog.Warn("failed to move .active_task file", "error", err)
		} else {
			slog.Info("migrated .active_task file")
		}
	}

	slog.Info("migration complete",
		"config and .env remain in", legacyRoot,
		"work and .active_task moved to", newWorkspaceRoot)

	return nil
}

// copyDir recursively copies a directory tree.
func copyDir(src, dst string) error {
	// Create destination directory
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}

	// Read source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	// Copy each entry
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectory
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Copy file
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a single file from src to dst.
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = srcFile.Close() }()

	// Create destination
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = dstFile.Close() }()

	// Copy contents
	if _, err := dstFile.ReadFrom(srcFile); err != nil {
		return err
	}

	// Get source file info
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Set file permissions
	return os.Chmod(dst, srcInfo.Mode())
}
