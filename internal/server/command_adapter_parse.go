package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/valksor/go-mehrhof/internal/conductor/commands"
)

// --- Memory parse functions ---

func parseMemorySearchInvocation(r *http.Request) (commands.Invocation, error) {
	query := r.URL.Query().Get("q")
	if query == "" {
		return commands.Invocation{}, errors.New("q parameter is required")
	}

	limit := 5
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	return commands.Invocation{
		Source: commands.SourceAPI,
		Args:   []string{"search", query},
		Options: map[string]any{
			"limit": limit,
			"types": strings.TrimSpace(r.URL.Query().Get("types")),
			"query": query,
		},
	}, nil
}

func parseMemoryIndexInvocation(r *http.Request) (commands.Invocation, error) {
	var taskID string
	contentType := r.Header.Get("Content-Type")

	if strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
		if err := r.ParseForm(); err != nil {
			return commands.Invocation{}, errors.New("invalid form data: " + err.Error())
		}
		taskID = r.FormValue("task_id")
	} else {
		var req struct {
			TaskID string `json:"task_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return commands.Invocation{}, errors.New("invalid request body: " + err.Error())
		}
		taskID = req.TaskID
	}

	if taskID == "" {
		return commands.Invocation{}, errors.New("task_id is required")
	}

	return commands.Invocation{
		Source: commands.SourceAPI,
		Args:   []string{"index", taskID},
	}, nil
}

func parseMemoryStatsInvocation(_ *http.Request) (commands.Invocation, error) {
	return commands.Invocation{
		Source: commands.SourceAPI,
		Args:   []string{"stats"},
	}, nil
}

// --- Library parse functions ---

func parseLibraryListInvocation(r *http.Request) (commands.Invocation, error) {
	return commands.Invocation{
		Source: commands.SourceAPI,
		Args:   []string{"list"},
		Options: map[string]any{
			"shared_only":  r.URL.Query().Get("shared") == "true",
			"project_only": r.URL.Query().Get("project") == "true",
			"tag":          r.URL.Query().Get("tag"),
		},
	}, nil
}

func parseLibraryShowInvocation(r *http.Request) (commands.Invocation, error) {
	nameOrID := strings.TrimPrefix(r.URL.Path, "/api/v1/library/")
	if nameOrID == "" {
		return commands.Invocation{}, errors.New("collection name or ID required")
	}

	return commands.Invocation{
		Source: commands.SourceAPI,
		Args:   []string{"show", nameOrID},
	}, nil
}

func parseLibraryItemsInvocation(r *http.Request) (commands.Invocation, error) {
	collectionID := r.PathValue("id")
	if collectionID == "" {
		return commands.Invocation{}, errors.New("collection id is required")
	}

	return commands.Invocation{
		Source: commands.SourceAPI,
		Args:   []string{"items", collectionID},
	}, nil
}

func parseLibraryRemoveInvocation(r *http.Request) (commands.Invocation, error) {
	nameOrID := strings.TrimPrefix(r.URL.Path, "/api/v1/library/")
	if nameOrID == "" {
		return commands.Invocation{}, errors.New("collection name or ID required")
	}

	return commands.Invocation{
		Source: commands.SourceAPI,
		Args:   []string{"remove", nameOrID},
	}, nil
}

func parseLibraryStatsInvocation(_ *http.Request) (commands.Invocation, error) {
	return commands.Invocation{
		Source: commands.SourceAPI,
		Args:   []string{"stats"},
	}, nil
}

func parseLibraryPullInvocation(r *http.Request) (commands.Invocation, error) {
	if err := r.ParseForm(); err != nil {
		return commands.Invocation{}, errors.New("invalid form data: " + err.Error())
	}

	source := strings.TrimSpace(r.FormValue("source"))
	if source == "" {
		return commands.Invocation{}, errors.New("source is required")
	}

	paths := make([]string, 0)
	if rawPaths := strings.TrimSpace(r.FormValue("paths")); rawPaths != "" {
		for _, p := range strings.Split(rawPaths, ",") {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				paths = append(paths, trimmed)
			}
		}
	}

	tags := make([]string, 0)
	if rawTags := strings.TrimSpace(r.FormValue("tags")); rawTags != "" {
		for _, t := range strings.Split(rawTags, ",") {
			if trimmed := strings.TrimSpace(t); trimmed != "" {
				tags = append(tags, trimmed)
			}
		}
	}

	maxDepth := 0
	if raw := r.FormValue("max_depth"); raw != "" {
		if d, err := strconv.Atoi(raw); err == nil && d > 0 {
			maxDepth = d
		}
	}

	maxPages := 0
	if raw := r.FormValue("max_pages"); raw != "" {
		if p, err := strconv.Atoi(raw); err == nil && p > 0 {
			maxPages = p
		}
	}

	return commands.Invocation{
		Source: commands.SourceAPI,
		Args:   []string{"pull", source},
		Options: map[string]any{
			"name":           strings.TrimSpace(r.FormValue("name")),
			"mode":           strings.TrimSpace(r.FormValue("mode")),
			"shared":         r.FormValue("shared") == "on" || r.FormValue("shared") == "true",
			"paths":          paths,
			"tags":           tags,
			"max_depth":      maxDepth,
			"max_pages":      maxPages,
			"continue":       r.FormValue("continue") == "true" || r.FormValue("continue") == "on",
			"restart":        r.FormValue("restart") == "true" || r.FormValue("restart") == "on",
			"domain_scope":   strings.TrimSpace(r.FormValue("domain_scope")),
			"version_filter": r.FormValue("version_filter") == "on" || r.FormValue("version_filter") == "true",
			"version":        strings.TrimSpace(r.FormValue("version")),
		},
	}, nil
}

func parseLibraryPullPreviewInvocation(r *http.Request) (commands.Invocation, error) {
	if err := r.ParseForm(); err != nil {
		return commands.Invocation{}, errors.New("invalid form data: " + err.Error())
	}

	source := strings.TrimSpace(r.FormValue("source"))
	if source == "" {
		return commands.Invocation{}, errors.New("source is required")
	}

	maxDepth := 0
	if raw := r.FormValue("max_depth"); raw != "" {
		if d, err := strconv.Atoi(raw); err == nil && d > 0 {
			maxDepth = d
		}
	}

	maxPages := 0
	if raw := r.FormValue("max_pages"); raw != "" {
		if p, err := strconv.Atoi(raw); err == nil && p > 0 {
			maxPages = p
		}
	}

	return commands.Invocation{
		Source: commands.SourceAPI,
		Args:   []string{"pull", source},
		Options: map[string]any{
			"name":           strings.TrimSpace(r.FormValue("name")),
			"mode":           strings.TrimSpace(r.FormValue("mode")),
			"shared":         r.FormValue("shared") == "on" || r.FormValue("shared") == "true",
			"max_depth":      maxDepth,
			"max_pages":      maxPages,
			"domain_scope":   strings.TrimSpace(r.FormValue("domain_scope")),
			"version_filter": r.FormValue("version_filter") == "on" || r.FormValue("version_filter") == "true",
			"version":        strings.TrimSpace(r.FormValue("version")),
			"dry_run":        true,
		},
	}, nil
}

// --- Links parse functions ---

func parseLinksListInvocation(_ *http.Request) (commands.Invocation, error) {
	return commands.Invocation{
		Source: commands.SourceAPI,
		Args:   []string{"list"},
	}, nil
}

func parseLinksEntityInvocation(r *http.Request) (commands.Invocation, error) {
	entityID := strings.TrimPrefix(r.URL.Path, "/api/v1/links/")
	if entityID == "" {
		return commands.Invocation{}, errors.New("entity ID is required")
	}

	return commands.Invocation{
		Source: commands.SourceAPI,
		Args:   []string{entityID},
	}, nil
}

func parseLinksSearchInvocation(r *http.Request) (commands.Invocation, error) {
	query := r.URL.Query().Get("q")
	if query == "" {
		return commands.Invocation{}, errors.New("q parameter is required")
	}

	return commands.Invocation{
		Source: commands.SourceAPI,
		Args:   []string{"search", query},
		Options: map[string]any{
			"query": query,
		},
	}, nil
}

func parseLinksStatsInvocation(_ *http.Request) (commands.Invocation, error) {
	return commands.Invocation{
		Source: commands.SourceAPI,
		Args:   []string{"stats"},
	}, nil
}

func parseLinksRebuildInvocation(_ *http.Request) (commands.Invocation, error) {
	return commands.Invocation{
		Source: commands.SourceAPI,
		Args:   []string{"rebuild"},
	}, nil
}

// --- Labels parse functions ---

func parseLabelsGetInvocation(_ *http.Request) (commands.Invocation, error) {
	return commands.Invocation{
		Source: commands.SourceAPI,
		Args:   []string{"list"},
	}, nil
}

func parseLabelsPostInvocation(r *http.Request) (commands.Invocation, error) {
	var req struct {
		Action string   `json:"action"`
		Labels []string `json:"labels"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return commands.Invocation{}, errors.New("invalid request body: " + err.Error())
	}

	switch req.Action {
	case "add", "remove", "set":
	default:
		return commands.Invocation{}, errors.New("invalid action: " + req.Action)
	}

	args := []string{req.Action}
	args = append(args, req.Labels...)

	return commands.Invocation{
		Source: commands.SourceAPI,
		Args:   args,
	}, nil
}

// --- Sync/Simplify parse functions ---

func parseSyncInvocation(r *http.Request) (commands.Invocation, error) {
	var req struct {
		TaskID string `json:"task_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return commands.Invocation{}, errors.New("invalid request body: " + err.Error())
	}

	if req.TaskID == "" {
		return commands.Invocation{}, errors.New("task_id is required")
	}

	return commands.Invocation{
		Source: commands.SourceAPI,
		Args:   []string{req.TaskID},
		Options: map[string]any{
			"task_id": req.TaskID,
		},
	}, nil
}

func parseSimplifyInvocation(r *http.Request) (commands.Invocation, error) {
	inv := commands.Invocation{Source: commands.SourceAPI}

	var req struct {
		Agent        string `json:"agent"`
		NoCheckpoint bool   `json:"no_checkpoint"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		return inv, errors.New("invalid request body: " + err.Error())
	}

	inv.Options = map[string]any{
		"agent":         req.Agent,
		"no_checkpoint": req.NoCheckpoint,
	}

	return inv, nil
}

// --- Specification diff parse function ---

func parseSpecificationDiffInvocation(r *http.Request) (commands.Invocation, error) {
	taskID := strings.TrimSpace(r.PathValue("id"))
	if taskID == "" {
		return commands.Invocation{}, errors.New("task ID is required")
	}

	specNumberRaw := r.PathValue("number")
	if specNumberRaw == "" {
		return commands.Invocation{}, errors.New("specification number is required")
	}

	specNumber, err := strconv.Atoi(specNumberRaw)
	if err != nil || specNumber <= 0 {
		return commands.Invocation{}, errors.New("specification number must be a positive integer")
	}

	filePath := r.URL.Query().Get("file")
	if filePath == "" {
		return commands.Invocation{}, errors.New("file query parameter is required")
	}

	contextLines := 3
	if contextRaw := r.URL.Query().Get("context"); contextRaw != "" {
		parsed, parseErr := strconv.Atoi(contextRaw)
		if parseErr != nil || parsed < 0 {
			return commands.Invocation{}, errors.New("context must be a non-negative integer")
		}

		contextLines = parsed
	}

	return commands.Invocation{
		Source: commands.SourceAPI,
		Options: map[string]any{
			"task_id":     taskID,
			"spec_number": specNumber,
			"file":        filePath,
			"context":     contextLines,
		},
	}, nil
}

// --- Security scan parse function ---

func parseSecurityScanInvocation(r *http.Request) (commands.Invocation, error) {
	inv := commands.Invocation{Source: commands.SourceAPI}
	options := map[string]any{}

	contentType := r.Header.Get("Content-Type")
	if contentType == "application/json" {
		var req struct {
			Dir       string   `json:"dir,omitempty"`
			Scanners  []string `json:"scanners,omitempty"`
			FailLevel string   `json:"fail_level,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			options["dir"] = req.Dir
			options["fail_level"] = req.FailLevel
			if len(req.Scanners) > 0 {
				options["scanners"] = req.Scanners
			}
		}
	} else {
		if err := r.ParseForm(); err == nil {
			scanners := r.Form["scanners"]
			// Map UI values to scanner names
			var mapped []string
			for _, scanner := range scanners {
				switch scanner {
				case "sast":
					mapped = append(mapped, "gosec")
				case "secrets":
					mapped = append(mapped, "gitleaks")
				case "vulns":
					mapped = append(mapped, "govulncheck")
				default:
					mapped = append(mapped, scanner)
				}
			}
			if len(mapped) > 0 {
				options["scanners"] = mapped
			}
		}
	}

	inv.Options = options

	return inv, nil
}

// --- Agent alias parse functions ---

func parseAgentAliasListInvocation(_ *http.Request) (commands.Invocation, error) {
	return commands.Invocation{
		Source: commands.SourceAPI,
		Options: map[string]any{
			"subcommand": "list",
		},
	}, nil
}

func parseAgentAliasAddInvocation(r *http.Request) (commands.Invocation, error) {
	var req struct {
		Name        string   `json:"name"`
		Extends     string   `json:"extends"`
		Description string   `json:"description"`
		Components  []string `json:"components"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return commands.Invocation{}, errors.New("invalid JSON: " + err.Error())
	}

	return commands.Invocation{
		Source: commands.SourceAPI,
		Options: map[string]any{
			"subcommand":  "add",
			"name":        req.Name,
			"extends":     req.Extends,
			"description": req.Description,
			"components":  req.Components,
		},
	}, nil
}

func parseAgentAliasDeleteInvocation(r *http.Request) (commands.Invocation, error) {
	name := strings.TrimPrefix(r.URL.Path, "/api/v1/agents/aliases/")
	if name == "" {
		return commands.Invocation{}, errors.New("alias name is required")
	}

	return commands.Invocation{
		Source: commands.SourceAPI,
		Options: map[string]any{
			"subcommand": "delete",
			"name":       name,
		},
	}, nil
}

// --- Settings parse functions ---

func parseSettingsGetInvocation(r *http.Request) (commands.Invocation, error) {
	return commands.Invocation{
		Source: commands.SourceAPI,
		Options: map[string]any{
			"project": r.URL.Query().Get("project"),
		},
	}, nil
}

func parseConfigExplainInvocation(r *http.Request) (commands.Invocation, error) {
	step := r.URL.Query().Get("step")
	if step == "" {
		return commands.Invocation{}, errors.New("missing step parameter (planning, implementing, reviewing)")
	}

	return commands.Invocation{
		Source: commands.SourceAPI,
		Options: map[string]any{
			"step": step,
		},
	}, nil
}

// --- Quick task parse functions ---

func parseQuickCreateInvocation(r *http.Request) (commands.Invocation, error) {
	var (
		description string
		title       string
		priority    int
		labels      []string
	)

	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
		if err := r.ParseForm(); err != nil {
			return commands.Invocation{}, errors.New("invalid form data: " + err.Error())
		}
		description = r.FormValue("description")
		title = r.FormValue("title")
	} else {
		var req struct {
			Description string   `json:"description"`
			Title       string   `json:"title,omitempty"`
			Priority    int      `json:"priority"`
			Labels      []string `json:"labels,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return commands.Invocation{}, errors.New("invalid request body: " + err.Error())
		}
		description = req.Description
		title = req.Title
		priority = req.Priority
		labels = req.Labels
	}

	if description == "" {
		return commands.Invocation{}, errors.New("description is required")
	}

	return commands.Invocation{
		Source: commands.SourceAPI,
		Args:   []string{description},
		Options: map[string]any{
			"title":    title,
			"priority": priority,
			"labels":   labels,
			"queue_id": "quick-tasks",
		},
	}, nil
}

func parseSubmitSourceInvocation(r *http.Request) (commands.Invocation, error) {
	var req struct {
		Source       string   `json:"source"`
		Provider     string   `json:"provider"`
		Notes        []string `json:"notes"`
		Title        string   `json:"title,omitempty"`
		Instructions string   `json:"instructions,omitempty"`
		Labels       []string `json:"labels,omitempty"`
		QueueID      string   `json:"queue_id,omitempty"`
		Optimize     bool     `json:"optimize"`
		DryRun       bool     `json:"dry_run"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return commands.Invocation{}, errors.New("invalid request body: " + err.Error())
	}

	if strings.TrimSpace(req.Source) == "" {
		return commands.Invocation{}, errors.New("source is required")
	}
	if strings.TrimSpace(req.Provider) == "" {
		return commands.Invocation{}, errors.New("provider is required")
	}

	return commands.Invocation{
		Source: commands.SourceAPI,
		Options: map[string]any{
			"source":       req.Source,
			"provider":     req.Provider,
			"notes":        req.Notes,
			"title":        req.Title,
			"instructions": req.Instructions,
			"labels":       req.Labels,
			"queue_id":     req.QueueID,
			"optimize":     req.Optimize,
			"dry_run":      req.DryRun,
		},
	}, nil
}

func parseQuickTaskIDInvocation(r *http.Request) (commands.Invocation, error) {
	taskID := r.PathValue("taskId")
	if taskID == "" {
		return commands.Invocation{}, errors.New("task ID is required")
	}

	return commands.Invocation{
		Source: commands.SourceAPI,
		Options: map[string]any{
			"task_id": taskID,
		},
	}, nil
}

func parseQuickNoteInvocation(r *http.Request) (commands.Invocation, error) {
	taskID := r.PathValue("taskId")
	if taskID == "" {
		return commands.Invocation{}, errors.New("task ID is required")
	}

	var req struct {
		Note string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return commands.Invocation{}, errors.New("invalid request body: " + err.Error())
	}

	if req.Note == "" {
		return commands.Invocation{}, errors.New("note is required")
	}

	return commands.Invocation{
		Source: commands.SourceAPI,
		Options: map[string]any{
			"task_id": taskID,
			"note":    req.Note,
		},
	}, nil
}

func parseQuickOptimizeInvocation(r *http.Request) (commands.Invocation, error) {
	taskID := r.PathValue("taskId")
	if taskID == "" {
		return commands.Invocation{}, errors.New("task ID is required")
	}

	var agent string
	if r.Body != nil && r.ContentLength > 0 {
		var req struct {
			Agent string `json:"agent,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			agent = req.Agent
		}
	}

	return commands.Invocation{
		Source: commands.SourceAPI,
		Args:   []string{"quick-tasks/" + taskID},
		Options: map[string]any{
			"agent": agent,
		},
	}, nil
}

func parseQuickExportInvocation(r *http.Request) (commands.Invocation, error) {
	taskID := r.PathValue("taskId")
	if taskID == "" {
		return commands.Invocation{}, errors.New("task ID is required")
	}

	return commands.Invocation{
		Source: commands.SourceAPI,
		Args:   []string{"quick-tasks/" + taskID},
	}, nil
}

func parseQuickSubmitInvocation(r *http.Request) (commands.Invocation, error) {
	taskID := r.PathValue("taskId")
	if taskID == "" {
		return commands.Invocation{}, errors.New("task ID is required")
	}

	var req struct {
		Provider string   `json:"provider"`
		Labels   []string `json:"labels,omitempty"`
		DryRun   bool     `json:"dry_run,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return commands.Invocation{}, errors.New("invalid request body: " + err.Error())
	}

	if req.Provider == "" {
		return commands.Invocation{}, errors.New("provider is required")
	}

	return commands.Invocation{
		Source: commands.SourceAPI,
		Args:   []string{"quick-tasks/" + taskID, req.Provider},
		Options: map[string]any{
			"labels":  req.Labels,
			"dry_run": req.DryRun,
		},
	}, nil
}

func parseQuickStartInvocation(r *http.Request) (commands.Invocation, error) {
	taskID := r.PathValue("taskId")
	if taskID == "" {
		return commands.Invocation{}, errors.New("task ID is required")
	}

	return commands.Invocation{
		Source: commands.SourceAPI,
		Args:   []string{"queue:quick-tasks/" + taskID},
		Options: map[string]any{
			"ref": "queue:quick-tasks/" + taskID,
		},
	}, nil
}

func parseQuickDeleteInvocation(r *http.Request) (commands.Invocation, error) {
	taskID := r.PathValue("taskId")
	if taskID == "" {
		return commands.Invocation{}, errors.New("task ID is required")
	}

	return commands.Invocation{
		Source: commands.SourceAPI,
		Args:   []string{"quick-tasks/" + taskID},
	}, nil
}

// --- Commit parse functions ---

func parseCommitChangesInvocation(r *http.Request) (commands.Invocation, error) {
	return commands.Invocation{
		Source: commands.SourceAPI,
		Args:   []string{"changes"},
		Options: map[string]any{
			"include_unstaged": r.URL.Query().Get("include_unstaged") == "true",
		},
	}, nil
}

func parseCommitPlanInvocation(r *http.Request) (commands.Invocation, error) {
	return commands.Invocation{
		Source: commands.SourceAPI,
		Args:   []string{"plan"},
		Options: map[string]any{
			"include_unstaged": r.URL.Query().Get("all") == "true",
		},
	}, nil
}

func parseCommitExecuteInvocation(r *http.Request) (commands.Invocation, error) {
	var req struct {
		Groups []struct {
			Message string   `json:"message"`
			Files   []string `json:"files"`
		} `json:"groups"`
		Push bool `json:"push"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return commands.Invocation{}, errors.New("invalid request body: " + err.Error())
	}

	groups := make([]any, 0, len(req.Groups))
	for _, g := range req.Groups {
		files := make([]any, 0, len(g.Files))
		for _, f := range g.Files {
			files = append(files, f)
		}
		groups = append(groups, map[string]any{
			"message": g.Message,
			"files":   files,
		})
	}

	return commands.Invocation{
		Source: commands.SourceAPI,
		Args:   []string{"execute"},
		Options: map[string]any{
			"groups": groups,
			"push":   req.Push,
		},
	}, nil
}

// --- Stack parse functions ---

func parseStackSubcommand(sub string) func(r *http.Request) (commands.Invocation, error) {
	return func(_ *http.Request) (commands.Invocation, error) {
		return commands.Invocation{
			Source:  commands.SourceAPI,
			Options: map[string]any{"subcommand": sub},
		}, nil
	}
}

func parseStackRebasePreviewInvocation(r *http.Request) (commands.Invocation, error) {
	opts := map[string]any{"subcommand": "rebase-preview"}

	if stackID := r.URL.Query().Get("stack_id"); stackID != "" {
		opts["stack_id"] = stackID
	} else if taskID := r.URL.Query().Get("task_id"); taskID != "" {
		opts["task_id"] = taskID
	} else {
		opts["preview_all"] = true
	}

	return commands.Invocation{
		Source:  commands.SourceAPI,
		Options: opts,
	}, nil
}

func parseStackRebaseInvocation(r *http.Request) (commands.Invocation, error) {
	opts := map[string]any{"subcommand": "rebase"}

	if r.Body != nil && r.ContentLength > 0 {
		var req struct {
			StackID string `json:"stack_id,omitempty"`
			TaskID  string `json:"task_id,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			if req.TaskID != "" {
				opts["task_id"] = req.TaskID
			} else if req.StackID != "" {
				opts["stack_id"] = req.StackID
			} else {
				opts["rebase_all"] = true
			}
		}
	} else {
		opts["rebase_all"] = true
	}

	return commands.Invocation{
		Source:  commands.SourceAPI,
		Options: opts,
	}, nil
}

// --- Project parse functions ---

func parseProjectSubcommand(sub string) func(r *http.Request) (commands.Invocation, error) {
	return func(_ *http.Request) (commands.Invocation, error) {
		return commands.Invocation{
			Source: commands.SourceAPI,
			Args:   []string{sub},
		}, nil
	}
}

func parseProjectPlanInvocation(r *http.Request) (commands.Invocation, error) {
	var req struct {
		Source       string `json:"source"`
		Title        string `json:"title,omitempty"`
		Instructions string `json:"instructions,omitempty"`
		UseSchema    bool   `json:"use_schema,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return commands.Invocation{}, errors.New("invalid request body: " + err.Error())
	}

	if req.Source == "" {
		return commands.Invocation{}, errors.New("source is required")
	}

	return commands.Invocation{
		Source: commands.SourceAPI,
		Args:   []string{"plan", req.Source},
		Options: map[string]any{
			"source":       req.Source,
			"title":        req.Title,
			"instructions": req.Instructions,
			"use_schema":   req.UseSchema,
		},
	}, nil
}

func parseProjectQueueInvocation(r *http.Request) (commands.Invocation, error) {
	queueID := strings.TrimPrefix(r.URL.Path, "/api/v1/project/queue/")
	if queueID == "" {
		return commands.Invocation{}, errors.New("queue ID is required")
	}

	return commands.Invocation{
		Source: commands.SourceAPI,
		Args:   []string{"queue", queueID},
		Options: map[string]any{
			"queue_id": queueID,
		},
	}, nil
}

func parseProjectQueueDeleteInvocation(r *http.Request) (commands.Invocation, error) {
	queueID := strings.TrimPrefix(r.URL.Path, "/api/v1/project/queue/")
	if queueID == "" {
		return commands.Invocation{}, errors.New("queue ID is required")
	}

	return commands.Invocation{
		Source: commands.SourceAPI,
		Args:   []string{"queue-delete", queueID},
		Options: map[string]any{
			"queue_id": queueID,
		},
	}, nil
}

func parseProjectTasksInvocation(r *http.Request) (commands.Invocation, error) {
	return commands.Invocation{
		Source: commands.SourceAPI,
		Args:   []string{"tasks"},
		Options: map[string]any{
			"queue_id": r.URL.Query().Get("queue_id"),
			"status":   r.URL.Query().Get("status"),
		},
	}, nil
}

func parseProjectTaskEditInvocation(r *http.Request) (commands.Invocation, error) {
	taskID := strings.TrimPrefix(r.URL.Path, "/api/v1/project/tasks/")
	if taskID == "" {
		return commands.Invocation{}, errors.New("task ID is required")
	}

	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return commands.Invocation{}, errors.New("invalid request body: " + err.Error())
	}

	// Merge task ID and queue_id into options
	body["task_id"] = taskID
	if queueID := r.URL.Query().Get("queue_id"); queueID != "" {
		body["queue_id"] = queueID
	}

	return commands.Invocation{
		Source:  commands.SourceAPI,
		Args:    []string{"task-edit", taskID},
		Options: body,
	}, nil
}

func parseProjectReorderInvocation(r *http.Request) (commands.Invocation, error) {
	var req struct {
		Auto        bool   `json:"auto,omitempty"`
		TaskID      string `json:"task_id,omitempty"`
		Position    string `json:"position,omitempty"`
		ReferenceID string `json:"reference_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return commands.Invocation{}, errors.New("invalid request body: " + err.Error())
	}

	return commands.Invocation{
		Source: commands.SourceAPI,
		Args:   []string{"reorder"},
		Options: map[string]any{
			"auto":         req.Auto,
			"task_id":      req.TaskID,
			"position":     req.Position,
			"reference_id": req.ReferenceID,
			"queue_id":     r.URL.Query().Get("queue_id"),
		},
	}, nil
}

func parseProjectSubmitInvocation(r *http.Request) (commands.Invocation, error) {
	var req struct {
		QueueID    string   `json:"queue_id,omitempty"`
		Provider   string   `json:"provider"`
		CreateEpic bool     `json:"create_epic,omitempty"`
		Labels     []string `json:"labels,omitempty"`
		DryRun     bool     `json:"dry_run,omitempty"`
		Mention    string   `json:"mention,omitempty"`
		TaskIDs    []string `json:"task_ids,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return commands.Invocation{}, errors.New("invalid request body: " + err.Error())
	}

	if req.Provider == "" {
		return commands.Invocation{}, errors.New("provider is required")
	}

	return commands.Invocation{
		Source: commands.SourceAPI,
		Args:   []string{"submit", req.Provider},
		Options: map[string]any{
			"queue_id":    req.QueueID,
			"provider":    req.Provider,
			"create_epic": req.CreateEpic,
			"labels":      req.Labels,
			"dry_run":     req.DryRun,
			"mention":     req.Mention,
			"task_ids":    req.TaskIDs,
		},
	}, nil
}

func parseProjectStartInvocation(r *http.Request) (commands.Invocation, error) {
	var req struct {
		QueueID string `json:"queue_id,omitempty"`
		TaskID  string `json:"task_id,omitempty"`
		Auto    bool   `json:"auto,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return commands.Invocation{}, errors.New("invalid request body: " + err.Error())
	}

	return commands.Invocation{
		Source: commands.SourceAPI,
		Args:   []string{"start"},
		Options: map[string]any{
			"queue_id": req.QueueID,
			"auto":     req.Auto,
			"ref":      req.TaskID,
		},
	}, nil
}

// parseProjectUploadInvocation returns a server-bound ParseFn that handles
// multipart file upload, resolves the source path, and passes it to the command.
func (s *Server) parseProjectUploadInvocation() func(r *http.Request) (commands.Invocation, error) {
	return func(r *http.Request) (commands.Invocation, error) {
		taskRef, err := s.handleFileUpload(r)
		if err != nil {
			return commands.Invocation{}, err
		}

		return commands.Invocation{
			Source: commands.SourceAPI,
			Args:   []string{"upload"},
			Options: map[string]any{
				"source": taskRef,
			},
		}, nil
	}
}

// parseProjectSourceInvocation returns a server-bound ParseFn that resolves
// alternative source inputs (reference, URL, text) before passing to the command.
func (s *Server) parseProjectSourceInvocation() func(r *http.Request) (commands.Invocation, error) {
	return func(r *http.Request) (commands.Invocation, error) {
		var req struct {
			Type     string `json:"type"`
			Value    string `json:"value"`
			Filename string `json:"filename"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return commands.Invocation{}, errors.New("invalid request body: " + err.Error())
		}

		var source string
		switch req.Type {
		case "reference":
			source = req.Value
		case "text":
			taskRef, err := s.saveContentToFile(req.Value)
			if err != nil {
				return commands.Invocation{}, errors.New("failed to save content: " + err.Error())
			}
			source = taskRef
		case "url":
			resp, err := s.fetchAndSaveURL(r.Context(), req.Value)
			if err != nil {
				return commands.Invocation{}, err
			}
			source = resp
		default:
			return commands.Invocation{}, errors.New("invalid type: must be reference, url, or text")
		}

		return commands.Invocation{
			Source: commands.SourceAPI,
			Args:   []string{"source"},
			Options: map[string]any{
				"source": source,
			},
		}, nil
	}
}

// --- Template parse functions ---

func parseTemplateListInvocation(_ *http.Request) (commands.Invocation, error) {
	return commands.Invocation{
		Source: commands.SourceAPI,
		Args:   []string{"list"},
	}, nil
}

func parseTemplateGetInvocation(r *http.Request) (commands.Invocation, error) {
	name := r.PathValue("name")
	if name == "" {
		return commands.Invocation{}, errors.New("template name is required")
	}

	return commands.Invocation{
		Source: commands.SourceAPI,
		Args:   []string{"get"},
		Options: map[string]any{
			"name": name,
		},
	}, nil
}

func parseTemplateApplyInvocation(r *http.Request) (commands.Invocation, error) {
	var req struct {
		TemplateName string `json:"template_name"`
		FilePath     string `json:"file_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return commands.Invocation{}, errors.New("invalid request body: " + err.Error())
	}

	if req.TemplateName == "" {
		return commands.Invocation{}, errors.New("template_name is required")
	}
	if req.FilePath == "" {
		return commands.Invocation{}, errors.New("file_path is required")
	}

	return commands.Invocation{
		Source: commands.SourceAPI,
		Args:   []string{"apply"},
		Options: map[string]any{
			"name": req.TemplateName,
			"path": req.FilePath,
		},
	}, nil
}

// --- Interactive parse functions ---

func parseInteractiveAnswerInvocation(r *http.Request) (commands.Invocation, error) {
	var req struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return commands.Invocation{}, errors.New("invalid request body: " + err.Error())
	}

	if req.Response == "" {
		return commands.Invocation{}, errors.New("response is required")
	}

	return commands.Invocation{
		Source: commands.SourceAPI,
		Options: map[string]any{
			"response": req.Response,
		},
	}, nil
}

// --- Running task parse functions ---

func parseRunningCancelInvocation(r *http.Request) (commands.Invocation, error) {
	taskID := r.PathValue("id")
	if taskID == "" {
		return commands.Invocation{}, errors.New("task ID is required")
	}

	return commands.Invocation{
		Source: commands.SourceAPI,
		Options: map[string]any{
			"task_id": taskID,
		},
	}, nil
}

// --- Agent logs parse functions ---

func parseAgentLogsHistoryInvocation(r *http.Request) (commands.Invocation, error) {
	return commands.Invocation{
		Source: commands.SourceAPI,
		Options: map[string]any{
			"task_id": r.URL.Query().Get("task_id"),
		},
	}, nil
}

// --- Standalone review/simplify parse functions ---

func parseStandaloneReviewInvocation(r *http.Request) (commands.Invocation, error) {
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return commands.Invocation{}, errors.New("invalid request body: " + err.Error())
	}

	return commands.Invocation{
		Source:  commands.SourceAPI,
		Options: body,
	}, nil
}

func parseStandaloneSimplifyInvocation(r *http.Request) (commands.Invocation, error) {
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return commands.Invocation{}, errors.New("invalid request body: " + err.Error())
	}

	return commands.Invocation{
		Source:  commands.SourceAPI,
		Options: body,
	}, nil
}

// --- InjectFn helpers ---

// injectServerStatus returns an InjectFn that adds server status info to the invocation.
func (s *Server) injectServerStatus() func(r *http.Request, inv *commands.Invocation) {
	return func(_ *http.Request, inv *commands.Invocation) {
		if inv.Options == nil {
			inv.Options = map[string]any{}
		}

		inv.Options["mode"] = s.modeString()
		inv.Options["running"] = s.IsRunning()
		inv.Options["port"] = s.Port()
		inv.Options["can_switch_to_global"] = s.startedInGlobalMode

		if project := s.currentProjectStatusInfo(); project != nil {
			inv.Options["project"] = project
		}
	}
}

// injectServerContext returns an InjectFn that adds server context info to the invocation.
func (s *Server) injectServerContext() func(r *http.Request, inv *commands.Invocation) {
	return func(_ *http.Request, inv *commands.Invocation) {
		if inv.Options == nil {
			inv.Options = map[string]any{}
		}

		inv.Options["mode"] = s.modeString()
		inv.Options["workspace_root"] = s.config.WorkspaceRoot
	}
}

// injectSettingsMode returns an InjectFn that adds mode and workspace_root to the invocation.
func (s *Server) injectSettingsMode() func(r *http.Request, inv *commands.Invocation) {
	return func(_ *http.Request, inv *commands.Invocation) {
		if inv.Options == nil {
			inv.Options = map[string]any{}
		}

		inv.Options["mode"] = s.modeString()
		inv.Options["workspace_root"] = s.config.WorkspaceRoot
	}
}

// injectTaskRegistry returns an InjectFn that injects the server's task registry into the invocation.
func (s *Server) injectTaskRegistry() func(r *http.Request, inv *commands.Invocation) {
	return func(_ *http.Request, inv *commands.Invocation) {
		if inv.Options == nil {
			inv.Options = map[string]any{}
		}
		inv.Options["_registry"] = s.getTaskRegistry()
	}
}

// injectLibraryGlobalMode returns an InjectFn that sets shared_only=true in global mode.
func injectLibraryGlobalMode(mode Mode) func(r *http.Request, inv *commands.Invocation) {
	return func(_ *http.Request, inv *commands.Invocation) {
		if mode != ModeGlobal {
			return
		}
		if inv.Options == nil {
			inv.Options = map[string]any{}
		}
		// In global mode, default to shared_only unless explicitly set otherwise
		if shared, ok := inv.Options["shared_only"].(bool); !ok || !shared {
			if projectOnly, ok := inv.Options["project_only"].(bool); !ok || !projectOnly {
				inv.Options["shared_only"] = true
			}
		}
	}
}
