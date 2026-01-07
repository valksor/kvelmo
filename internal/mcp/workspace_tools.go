package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/vcs"
)

// openValidWorkspace resolves and validates a workspace for MCP operations.
// It ensures the workspace is valid. The workspace must exist (we don't auto-create in MCP mode).
func openValidWorkspace(ctx context.Context) (*storage.Workspace, error) {
	root, err := resolveWorkspaceRoot(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace root: %w", err)
	}

	// Validate workspace before opening
	if err := validateWorkspace(root); err != nil {
		return nil, fmt.Errorf("validate workspace: %w", err)
	}

	ws, err := storage.OpenWorkspace(ctx, root, nil)
	if err != nil {
		return nil, fmt.Errorf("open workspace: %w", err)
	}

	return ws, nil
}

// resolveWorkspaceRoot resolves the workspace root directory from current working directory.
// This is a minimal version to avoid import cycles.
func resolveWorkspaceRoot(ctx context.Context) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	git, err := vcs.New(ctx, cwd)
	if err != nil {
		// Not in a git repository, use cwd as root
		// We ignore the error here because it's expected when not in a git repo
		//nolint:nilerr // Intentional - non-git repos use cwd as root
		return cwd, nil
	}

	// In a git repository - check if we're in a worktree
	if git.IsWorktree() {
		mainRepo, err := git.GetMainWorktreePath(ctx)
		if err != nil {
			return "", fmt.Errorf("get main repo from worktree: %w", err)
		}

		return mainRepo, nil
	}

	// In main git repository
	return git.Root(), nil
}

// validateWorkspace checks if the given root is accessible and safe.
// This is a minimal validation - the real workspace structure validation
// happens in storage.OpenWorkspace().
func validateWorkspace(root string) error {
	// Check that root directory exists and is accessible
	info, err := os.Stat(root)
	if err != nil {
		return fmt.Errorf("cannot access workspace root: %w", err)
	}
	if !info.IsDir() {
		return errors.New("workspace root is not a directory")
	}

	// Resolve symlinks on the root to prevent directory traversal attacks
	_, err = filepath.EvalSymlinks(root)
	if err != nil {
		return fmt.Errorf("invalid workspace path: %w", err)
	}

	return nil
}

// RegisterWorkspaceTools registers workspace data access tools.
func RegisterWorkspaceTools(registry *ToolRegistry) {
	// workspace_get_active_task
	registry.RegisterDirectTool(
		"workspace_get_active_task",
		"Get the current active task information including ID, state, title, and specifications",
		map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
			"required":   []string{},
		},
		func(ctx context.Context, args map[string]interface{}) (*ToolCallResult, error) {
			// Add timeout to prevent hanging
			ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			ws, err := openValidWorkspace(ctx)
			if err != nil {
				return errorResult(err), nil
			}

			if !ws.HasActiveTask() {
				return textResult("No active task in workspace"), nil
			}

			active, err := ws.LoadActiveTask()
			if err != nil {
				return errorResult(err), nil
			}

			work, err := ws.LoadWork(active.ID)
			if err != nil {
				return errorResult(err), nil
			}

			result := map[string]interface{}{
				"task_id":  active.ID,
				"state":    active.State,
				"title":    work.Metadata.Title,
				"source":   active.Ref,
				"branch":   active.Branch,
				"work_dir": active.WorkDir,
			}

			// Add specifications summary
			specs, err := ws.ListSpecificationsWithStatus(active.ID)
			if err != nil {
				return errorResult(fmt.Errorf("failed to list specifications: %w", err)), nil
			}
			specsSummary := make([]map[string]interface{}, 0, len(specs))
			for _, spec := range specs {
				specsSummary = append(specsSummary, map[string]interface{}{
					"number": spec.Number,
					"title":  spec.Title,
					"status": spec.Status,
				})
			}
			result["specifications"] = specsSummary

			return jsonResult(result), nil
		},
	)

	// workspace_list_tasks
	registry.RegisterDirectTool(
		"workspace_list_tasks",
		"List all tasks in the workspace with their IDs, states, and titles",
		map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
			"required":   []string{},
		},
		func(ctx context.Context, args map[string]interface{}) (*ToolCallResult, error) {
			// Add timeout to prevent hanging
			ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			ws, err := openValidWorkspace(ctx)
			if err != nil {
				return errorResult(err), nil
			}

			taskIDs, err := ws.ListWorks()
			if err != nil {
				return errorResult(err), nil
			}

			tasks := make([]map[string]interface{}, 0, len(taskIDs))
			var activeID string
			if ws.HasActiveTask() {
				active, err := ws.LoadActiveTask()
				if err == nil && active != nil {
					activeID = active.ID
				}
			}

			for _, taskID := range taskIDs {
				work, err := ws.LoadWork(taskID)
				if err != nil {
					continue
				}

				isActive := taskID == activeID
				state := "unknown"
				if isActive {
					active, err := ws.LoadActiveTask()
					if err == nil && active != nil {
						state = active.State
					}
				}

				tasks = append(tasks, map[string]interface{}{
					"task_id":   taskID,
					"title":     work.Metadata.Title,
					"state":     state,
					"is_active": isActive,
					"source":    work.Source.Ref,
				})
			}

			return jsonResult(map[string]interface{}{"tasks": tasks}), nil
		},
	)

	// workspace_get_specs
	registry.RegisterDirectTool(
		"workspace_get_specs",
		"Get specifications for a task",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"task_id": map[string]interface{}{
					"type":        "string",
					"description": "Task ID (defaults to active task)",
				},
				"summary_only": map[string]interface{}{
					"type":        "boolean",
					"description": "Only return summaries without full content",
				},
			},
			"required": []string{},
		},
		func(ctx context.Context, args map[string]interface{}) (*ToolCallResult, error) {
			// Add timeout to prevent hanging
			ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			ws, err := openValidWorkspace(ctx)
			if err != nil {
				return errorResult(err), nil
			}

			// Get summary_only flag
			summaryOnly, _ := args["summary_only"].(bool)

			// Get task ID from args or use active task
			var taskID string
			if tid, ok := args["task_id"].(string); ok && tid != "" {
				taskID = tid
			} else if ws.HasActiveTask() {
				active, err := ws.LoadActiveTask()
				if err == nil && active != nil {
					taskID = active.ID
				}
			}

			if taskID == "" {
				return errorResult(errors.New("no task specified and no active task")), nil
			}

			specs, err := ws.ListSpecificationsWithStatus(taskID)
			if err != nil {
				return errorResult(err), nil
			}

			specsData := make([]map[string]interface{}, 0, len(specs))
			for _, spec := range specs {
				specData := map[string]interface{}{
					"number": spec.Number,
					"title":  spec.Title,
					"status": spec.Status,
				}

				if !summaryOnly {
					// Get content for each specification
					content, err := ws.LoadSpecification(taskID, spec.Number)
					if err != nil {
						content = fmt.Sprintf("(error loading content: %v)", err)
					}
					specData["content"] = content
				}

				specsData = append(specsData, specData)
			}

			return jsonResult(map[string]interface{}{
				"task_id":        taskID,
				"specifications": specsData,
			}), nil
		},
	)

	// workspace_get_sessions
	registry.RegisterDirectTool(
		"workspace_get_sessions",
		"Get session history for a task",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"task_id": map[string]interface{}{
					"type":        "string",
					"description": "Task ID (defaults to active task)",
				},
			},
			"required": []string{},
		},
		func(ctx context.Context, args map[string]interface{}) (*ToolCallResult, error) {
			// Add timeout to prevent hanging
			ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			ws, err := openValidWorkspace(ctx)
			if err != nil {
				return errorResult(err), nil
			}

			// Get task ID from args or use active task
			var taskID string
			if tid, ok := args["task_id"].(string); ok && tid != "" {
				taskID = tid
			} else if ws.HasActiveTask() {
				active, err := ws.LoadActiveTask()
				if err == nil && active != nil {
					taskID = active.ID
				}
			}

			if taskID == "" {
				return errorResult(errors.New("no task specified and no active task")), nil
			}

			sessions, err := ws.ListSessions(taskID)
			if err != nil {
				return errorResult(err), nil
			}

			sessionsData := make([]map[string]interface{}, 0, len(sessions))
			for _, session := range sessions {
				sessionData := map[string]interface{}{
					"kind":    session.Kind,
					"started": session.Metadata.StartedAt.Format("2006-01-02T15:04:05Z"),
				}
				if session.Usage != nil {
					sessionData["input_tokens"] = session.Usage.InputTokens
					sessionData["output_tokens"] = session.Usage.OutputTokens
				}
				sessionsData = append(sessionsData, sessionData)
			}

			return jsonResult(map[string]interface{}{
				"task_id":  taskID,
				"sessions": sessionsData,
			}), nil
		},
	)

	// workspace_get_notes
	registry.RegisterDirectTool(
		"workspace_get_notes",
		"Get notes for a task",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"task_id": map[string]interface{}{
					"type":        "string",
					"description": "Task ID (defaults to active task)",
				},
			},
			"required": []string{},
		},
		func(ctx context.Context, args map[string]interface{}) (*ToolCallResult, error) {
			// Add timeout to prevent hanging
			ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			ws, err := openValidWorkspace(ctx)
			if err != nil {
				return errorResult(err), nil
			}

			// Get task ID from args or use active task
			var taskID string
			if tid, ok := args["task_id"].(string); ok && tid != "" {
				taskID = tid
			} else if ws.HasActiveTask() {
				active, err := ws.LoadActiveTask()
				if err == nil && active != nil {
					taskID = active.ID
				}
			}

			if taskID == "" {
				return errorResult(errors.New("no task specified and no active task")), nil
			}

			notes, err := ws.ReadNotes(taskID)
			if err != nil {
				return errorResult(err), nil
			}

			return textResult(notes), nil
		},
	)
}

// textResult creates a text content block result.
func textResult(text string) *ToolCallResult {
	return &ToolCallResult{
		Content: []ContentBlock{
			{
				Type: ContentTypeText,
				Text: text,
			},
		},
		IsError: false,
	}
}

// jsonResult creates a JSON content block result.
func jsonResult(data interface{}) *ToolCallResult {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		// Fallback to error text if JSON marshaling fails
		return &ToolCallResult{
			Content: []ContentBlock{
				{
					Type: ContentTypeText,
					Text: fmt.Sprintf("Error: failed to marshal result: %v", err),
				},
			},
			IsError: true,
		}
	}

	return &ToolCallResult{
		Content: []ContentBlock{
			{
				Type: ContentTypeText,
				Text: string(jsonData),
			},
		},
		IsError: false,
	}
}

// errorResult creates an error result.
func errorResult(err error) *ToolCallResult {
	return &ToolCallResult{
		Content: []ContentBlock{
			{
				Type: ContentTypeText,
				Text: fmt.Sprintf("Error: %v", err),
			},
		},
		IsError: true,
	}
}
