package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/valksor/go-mehrhof/internal/conductor"
)

func init() {
	Register(Command{
		Info: CommandInfo{
			Name:         "fs-browse",
			Description:  "Browse filesystem directories for project selection",
			Category:     "info",
			RequiresTask: false,
		},
		Handler: handleFSBrowse,
	})
}

// handleFSBrowse lists directories at the given path for the folder browser UI.
// If no path is provided, defaults to the user's home directory.
// Only returns directories (not files) and excludes hidden directories.
func handleFSBrowse(_ context.Context, _ *conductor.Conductor, inv Invocation) (*Result, error) {
	path := GetString(inv.Options, "path")
	if path == "" {
		// Default to home directory
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("cannot determine home directory: %w", err)
		}
		path = home
	}

	// Security: resolve to absolute path, clean it
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}
	absPath = filepath.Clean(absPath)

	// Verify path exists and is a directory
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("path does not exist: %s", absPath)
		}
		if os.IsPermission(err) {
			return nil, fmt.Errorf("permission denied: %s", absPath)
		}

		return nil, fmt.Errorf("cannot access path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", absPath)
	}

	// Read directory entries
	entries, err := os.ReadDir(absPath)
	if err != nil {
		if os.IsPermission(err) {
			return nil, fmt.Errorf("permission denied reading directory: %s", absPath)
		}

		return nil, fmt.Errorf("cannot read directory: %w", err)
	}

	// Filter to only directories, exclude hidden
	dirs := make([]map[string]string, 0)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		// Skip hidden directories (starting with .)
		if strings.HasPrefix(name, ".") {
			continue
		}
		dirs = append(dirs, map[string]string{
			"name": name,
			"type": "dir",
		})
	}

	// Calculate parent directory
	parent := filepath.Dir(absPath)

	return NewResult(fmt.Sprintf("%d directories", len(dirs))).WithData(map[string]any{
		"path":    absPath,
		"parent":  parent,
		"entries": dirs,
	}), nil
}
