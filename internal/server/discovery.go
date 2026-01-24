package server

import (
	"os"
	"path/filepath"
	"time"

	"github.com/valksor/go-mehrhof/internal/storage"
)

// DiscoveredProject represents a project found in the workspaces directory.
type DiscoveredProject struct {
	ID              string    `json:"id"`
	TaskCount       int       `json:"task_count"`
	ActiveTask      string    `json:"active_task,omitempty"`
	ActiveWorktrees int       `json:"active_worktrees"`
	LastModified    time.Time `json:"last_modified"`
}

// DiscoverProjects scans the workspaces directory to find all projects.
// It looks in ~/.valksor/mehrhof/workspaces/ for project directories.
func DiscoverProjects() ([]DiscoveredProject, error) {
	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	// Build workspaces path
	workspacesDir := filepath.Join(homeDir, storage.MehrhofHomeDir, storage.WorkspacesDir)

	// Check if workspaces directory exists
	if _, err := os.Stat(workspacesDir); os.IsNotExist(err) {
		// No workspaces directory, return empty list
		return []DiscoveredProject{}, nil
	}

	// List all project directories
	entries, err := os.ReadDir(workspacesDir)
	if err != nil {
		return nil, err
	}

	var projects []DiscoveredProject
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		projectID := entry.Name()
		projectDir := filepath.Join(workspacesDir, projectID)

		project := DiscoveredProject{
			ID: projectID,
		}

		// Get last modified time from directory
		info, err := entry.Info()
		if err == nil {
			project.LastModified = info.ModTime()
		}

		// Count tasks in work/ subdirectory
		workDir := filepath.Join(projectDir, "work")
		if workEntries, err := os.ReadDir(workDir); err == nil {
			for _, we := range workEntries {
				if we.IsDir() {
					project.TaskCount++

					// Check if task has a worktree (by looking for worktree marker)
					taskDir := filepath.Join(workDir, we.Name())
					workFile := filepath.Join(taskDir, "work.yaml")
					if data, err := os.ReadFile(workFile); err == nil {
						// Simple check for worktree path in work.yaml
						if containsWorktreePath(data) {
							project.ActiveWorktrees++
						}
					}
				}
			}
		}

		// Check for active task
		activeTaskFile := filepath.Join(projectDir, ".active_task")
		if data, err := os.ReadFile(activeTaskFile); err == nil {
			project.ActiveTask = string(data)
		}

		projects = append(projects, project)
	}

	return projects, nil
}

// GetProjectWorkspacePath returns the filesystem path for a project's workspace directory.
func GetProjectWorkspacePath(projectID string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, storage.MehrhofHomeDir, storage.WorkspacesDir, projectID), nil
}

// containsWorktreePath checks if YAML data contains a worktree_path field with a value.
// This is a simple check that doesn't require full YAML parsing.
func containsWorktreePath(data []byte) bool {
	// Look for "worktree_path:" followed by a non-empty value
	content := string(data)
	for i := range len(content) - 15 {
		if content[i:i+14] == "worktree_path:" {
			// Check if there's a value after the colon (not just whitespace)
			rest := content[i+14:]
			for j := 0; j < len(rest) && j < 100; j++ {
				c := rest[j]
				if c == '\n' || c == '\r' {
					break
				}
				if c != ' ' && c != '\t' && c != '"' && c != '\'' {
					return true
				}
			}
		}
	}

	return false
}
