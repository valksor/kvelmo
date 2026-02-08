package commands

import (
	"context"
	"fmt"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/display"
	"github.com/valksor/go-mehrhof/internal/storage"
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

// handleProjectsList returns all registered projects.
func handleProjectsList(_ context.Context, _ *conductor.Conductor, _ Invocation) (*Result, error) {
	registry, err := storage.LoadRegistry()
	if err != nil {
		return nil, fmt.Errorf("failed to load registry: %w", err)
	}

	projects := registry.List()
	result := make([]map[string]any, 0, len(projects))

	for _, p := range projects {
		result = append(result, map[string]any{
			"id":          storage.SanitizeRemoteURL(p.ID),
			"name":        p.Name,
			"path":        p.Path,
			"remote_url":  storage.SanitizeRemoteURL(p.RemoteURL),
			"last_access": p.LastAccess,
		})
	}

	return NewResult(fmt.Sprintf("%d project(s)", len(result))).WithData(map[string]any{
		"projects": result,
		"count":    len(result),
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
