package conductor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/valksor/go-mehrhof/internal/agent"
	"github.com/valksor/go-mehrhof/internal/events"
)

// DeleteFileSentinel is a special marker that indicates a file should be deleted
// This provides an alternative to setting operation: delete in YAML blocks.
const DeleteFileSentinel = "__DELETE_FILE__"

// ensureDirExists creates the directory for the given file path if it doesn't exist.
// This is a helper to avoid code duplication when writing files.
func ensureDirExists(path string) error {
	dir := filepath.Dir(path)
	if dir == "" || dir == "." {
		return nil
	}

	return os.MkdirAll(dir, 0o755)
}

// validatePathInWorkspace ensures that a resolved path is within the workspace root.
// This prevents path traversal attacks when applying file changes from AI agent output.
func validatePathInWorkspace(resolved, root string) error {
	// Get the relative path from root to resolved
	rel, err := filepath.Rel(root, resolved)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	// Check if the relative path starts with ".." which would escape the root
	if strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
		return fmt.Errorf("path outside workspace: %s", resolved)
	}

	return nil
}

// normalizeAgentPath normalizes a file path from an agent to prevent path nesting.
// Agents may return paths that are:
// - Prefixed with ./
// - Absolute paths within the root (when they shouldn't be)
// - Already relative to root
//
// The function ensures the path is relative before joining with root.
func normalizeAgentPath(agentPath, root string) string {
	// If path is empty, return as-is
	if agentPath == "" {
		return agentPath
	}

	// Strip leading ./ if present
	agentPath = strings.TrimPrefix(agentPath, "."+string(filepath.Separator))

	// If path is absolute, check if it starts with root
	if filepath.IsAbs(agentPath) {
		// If the path is within root, make it relative
		if rel, err := filepath.Rel(root, agentPath); err == nil {
			// Check if rel doesn't start with .. (path is within root)
			if !strings.HasPrefix(rel, "..") {
				return rel
			}
		}
		// Path is outside root or error - return as-is (validation will catch it)
		return agentPath
	}

	// Path is already relative, return as-is
	return agentPath
}

// applyFiles writes agent file changes to disk.
func applyFiles(_ context.Context, c *Conductor, files []agent.FileChange) error {
	root := c.opts.WorkDir
	if c.git != nil {
		root = c.git.Root()
	}

	// Resolve symlinks in root path for accurate validation (handles macOS /var -> /private/var symlinks)
	resolvedRoot := root
	if res, err := filepath.EvalSymlinks(root); err == nil {
		resolvedRoot = res
	}
	// If root doesn't exist yet, EvalSymlinks will fail, so we use root as-is

	var stats struct {
		created int
		updated int
		deleted int
	}

	for _, fc := range files {
		// Normalize the path to prevent nesting when agent returns absolute paths
		normalizedPath := normalizeAgentPath(fc.Path, root)
		path := filepath.Join(root, normalizedPath)

		// Validate the path is within workspace (prevent path traversal attacks)
		// Resolve symlinks in the target path and validate it stays within root
		resolvedPath := path
		if res, err := filepath.EvalSymlinks(path); err == nil {
			resolvedPath = res
		}
		// Validate against both the original root and resolved root to handle symlinked paths
		if err := validatePathInWorkspace(resolvedPath, root); err != nil {
			if err := validatePathInWorkspace(resolvedPath, resolvedRoot); err != nil {
				return fmt.Errorf("invalid file path %q: %w", fc.Path, err)
			}
		}

		// Check for delete sentinel in content (alternative to operation: delete)
		if fc.Content == DeleteFileSentinel {
			fc.Operation = agent.FileOpDelete
		}

		switch fc.Operation {
		case agent.FileOpCreate:
			// Ensure directory exists
			if err := ensureDirExists(path); err != nil {
				return fmt.Errorf("create directory for %s: %w", path, err)
			}

			// Write file
			if err := os.WriteFile(path, []byte(fc.Content), 0o644); err != nil {
				return fmt.Errorf("write file %s: %w", path, err)
			}
			stats.created++

			c.eventBus.PublishRaw(events.Event{
				Type: events.TypeFileChanged,
				Data: map[string]any{
					"path":      normalizedPath,
					"operation": "create",
				},
			})

		case agent.FileOpUpdate:
			// Ensure directory exists
			if err := ensureDirExists(path); err != nil {
				return fmt.Errorf("create directory for %s: %w", path, err)
			}

			// Write file
			if err := os.WriteFile(path, []byte(fc.Content), 0o644); err != nil {
				return fmt.Errorf("write file %s: %w", path, err)
			}
			stats.updated++

			c.eventBus.PublishRaw(events.Event{
				Type: events.TypeFileChanged,
				Data: map[string]any{
					"path":      normalizedPath,
					"operation": "update",
				},
			})

		case agent.FileOpDelete:
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("delete file %s: %w", path, err)
			}
			stats.deleted++

			c.eventBus.PublishRaw(events.Event{
				Type: events.TypeFileChanged,
				Data: map[string]any{
					"path":      normalizedPath,
					"operation": "delete",
				},
			})
		}
	}

	// Publish summary of file operations
	if stats.created > 0 || stats.updated > 0 || stats.deleted > 0 {
		c.eventBus.PublishRaw(events.Event{
			Type: events.TypeProgress,
			Data: map[string]any{
				"message": fmt.Sprintf("Files: %d created, %d updated, %d deleted",
					stats.created, stats.updated, stats.deleted),
			},
		})
	}

	return nil
}
