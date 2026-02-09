package commands

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/display"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/vcs"
	"github.com/valksor/go-toolkit/version"
)

func init() {
	Register(Command{
		Info: CommandInfo{
			Name:         "server-status",
			Description:  "Get server status including mode and workflow state",
			Category:     "info",
			RequiresTask: false,
		},
		Handler: handleServerStatus,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "server-context",
			Description:  "Get server context including workspace and active task",
			Category:     "info",
			RequiresTask: false,
		},
		Handler: handleServerContext,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "projects-list",
			Description:  "List registered projects",
			Category:     "info",
			RequiresTask: false,
		},
		Handler: handleProjectsList,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "docs-url",
			Description:  "Get documentation URL for current build",
			Category:     "info",
			RequiresTask: false,
		},
		Handler: handleDocsURL,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "projects-toggle-favorite",
			Description:  "Toggle favorite status for a project",
			Category:     "info",
			RequiresTask: false,
		},
		Handler: handleProjectsToggleFavorite,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "projects-remove",
			Description:  "Remove a project from tracking",
			Category:     "info",
			RequiresTask: false,
		},
		Handler: handleProjectsRemove,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "projects-add",
			Description:  "Add a new project to tracking",
			Category:     "info",
			RequiresTask: false,
		},
		Handler: handleProjectsAdd,
	})
}

// handleServerStatus returns server status info.
// Server-level fields (mode, running, port, canSwitchToGlobal, project) are injected
// via InjectFn from the server handler before this command runs.
func handleServerStatus(_ context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	response := map[string]any{
		"mode":              GetString(inv.Options, "mode"),
		"running":           GetBool(inv.Options, "running"),
		"port":              GetInt(inv.Options, "port"),
		"canSwitchToGlobal": GetBool(inv.Options, "can_switch_to_global"),
	}

	// Project info is injected by InjectFn as a pre-built map
	if project, ok := inv.Options["project"]; ok && project != nil {
		response["project"] = project
	}

	// Add workflow state from conductor if available.
	// We read machine.State() which has its own lock (m.mu), independent of the
	// conductor's c.mu. Setting result.State here prevents Execute()'s post-handler
	// enrichment from calling GetActiveTask() which would block on c.mu during Finish.
	var stateStr string
	if cond != nil {
		if machine := cond.GetMachine(); machine != nil {
			stateStr = string(machine.State())
			response["state"] = stateStr
		}
	}

	result := NewResult("Server status loaded").WithData(response)
	if stateStr != "" {
		result = result.WithState(stateStr)
	}

	return result, nil
}

// handleServerContext returns server context info.
// Server-level fields (mode, workspace_root) are injected via InjectFn.
func handleServerContext(_ context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	response := map[string]any{
		"mode":           GetString(inv.Options, "mode"),
		"workspace_root": GetString(inv.Options, "workspace_root"),
	}

	var activeTask *storage.ActiveTask
	if cond != nil {
		activeTask = cond.GetActiveTask()
	}
	if activeTask != nil {
		response["current_task"] = map[string]any{
			"id":            activeTask.ID,
			"state":         activeTask.State,
			"ref":           activeTask.Ref,
			"branch":        activeTask.Branch,
			"worktree_path": activeTask.WorktreePath,
			"started":       activeTask.Started,
		}
	}

	return NewResult("Server context loaded").WithData(response), nil
}

// handleProjectsList returns all registered projects from the registry.
// Projects are ordered by last access (most recent first) and include remote URLs.
func handleProjectsList(_ context.Context, _ *conductor.Conductor, _ Invocation) (*Result, error) {
	registry, err := storage.LoadRegistry()
	if err != nil {
		return nil, fmt.Errorf("failed to load registry: %w", err)
	}

	projects := registry.ListRecent(storage.MaxRecentProjects)
	favorites := registry.GetFavorites()

	result := make([]map[string]any, 0, len(projects))
	for _, p := range projects {
		result = append(result, map[string]any{
			"id":          p.Path, // Use path as ID for backwards compatibility
			"name":        p.Name,
			"path":        p.Path,
			"remote_url":  p.RemoteURL, // Already sanitized on registry load
			"last_access": p.LastAccess,
			"is_favorite": p.IsFavorite,
		})
	}

	return NewResult(fmt.Sprintf("%d project(s)", len(result))).WithData(map[string]any{
		"projects":  result,
		"favorites": favorites,
		"count":     len(result),
	}), nil
}

// handleDocsURL returns the documentation URL appropriate for the current build.
// Stable releases (v*) link to /docs/latest, all others to /docs/nightly.
func handleDocsURL(_ context.Context, _ *conductor.Conductor, _ Invocation) (*Result, error) {
	return NewResult("Documentation URL").WithData(map[string]any{
		"url":     display.DocsURL(),
		"version": version.Version,
	}), nil
}

// handleProjectsToggleFavorite toggles the favorite status for a project.
func handleProjectsToggleFavorite(_ context.Context, _ *conductor.Conductor, inv Invocation) (*Result, error) {
	path := GetString(inv.Options, "path")
	if path == "" {
		return nil, errors.New("path is required")
	}

	registry, err := storage.LoadRegistry()
	if err != nil {
		return nil, fmt.Errorf("failed to load registry: %w", err)
	}

	isFavorite, err := registry.ToggleFavorite(path)
	if err != nil {
		return nil, fmt.Errorf("failed to toggle favorite: %w", err)
	}

	if err := registry.Save(); err != nil {
		return nil, fmt.Errorf("failed to save registry: %w", err)
	}

	action := "removed from"
	if isFavorite {
		action = "added to"
	}

	return NewResult(fmt.Sprintf("Project %s favorites", action)).WithData(map[string]any{
		"path":        path,
		"is_favorite": isFavorite,
	}), nil
}

// handleProjectsRemove removes a project from the registry.
func handleProjectsRemove(_ context.Context, _ *conductor.Conductor, inv Invocation) (*Result, error) {
	path := GetString(inv.Options, "path")
	if path == "" {
		return nil, errors.New("path is required")
	}

	registry, err := storage.LoadRegistry()
	if err != nil {
		return nil, fmt.Errorf("failed to load registry: %w", err)
	}

	// Find project by path to get its ID
	meta, found := registry.FindByPath(path)
	if !found {
		return nil, fmt.Errorf("project not found: %s", path)
	}

	registry.Unregister(meta.ID)

	if err := registry.Save(); err != nil {
		return nil, fmt.Errorf("failed to save registry: %w", err)
	}

	return NewResult("Project removed from registry").WithData(map[string]any{
		"path": path,
	}), nil
}

// handleProjectsAdd adds a new project to the registry by path.
func handleProjectsAdd(ctx context.Context, _ *conductor.Conductor, inv Invocation) (*Result, error) {
	path := GetString(inv.Options, "path")
	if path == "" {
		return nil, errors.New("path is required")
	}

	// Generate project ID and get remote URL
	projectID, err := storage.GenerateProjectID(ctx, path)
	if err != nil {
		// Fallback to path-based ID
		projectID = filepath.Base(path)
	}

	// Extract project name from path
	name := filepath.Base(path)

	// Try to get remote URL for the project
	remoteURL := getRemoteURL(ctx, path)

	registry, err := storage.LoadRegistry()
	if err != nil {
		return nil, fmt.Errorf("failed to load registry: %w", err)
	}

	if err := registry.Register(projectID, path, remoteURL, name); err != nil {
		return nil, fmt.Errorf("failed to register project: %w", err)
	}

	if err := registry.Save(); err != nil {
		return nil, fmt.Errorf("failed to save registry: %w", err)
	}

	return NewResult("Project added to registry").WithData(map[string]any{
		"path": path,
		"name": name,
	}), nil
}

// getRemoteURL attempts to get the git remote URL for a project path.
func getRemoteURL(ctx context.Context, path string) string {
	git, err := vcs.New(ctx, path)
	if err != nil {
		return ""
	}

	remote, err := git.GetDefaultRemote(ctx)
	if err != nil || remote == "" {
		return ""
	}

	url, err := git.RemoteURL(ctx, remote)
	if err != nil {
		return ""
	}

	return url
}
