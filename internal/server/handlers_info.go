package server

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"

	"github.com/valksor/go-toolkit/licensing"

	"github.com/valksor/go-mehrhof/internal/agent"
	"github.com/valksor/go-mehrhof/internal/server/static"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/workflow"
)

// handleGuide returns guidance on what to do next.
func (s *Server) handleGuide(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	// Check for active task
	activeTask := s.config.Conductor.GetActiveTask()
	if activeTask == nil {
		s.writeJSON(w, http.StatusOK, guideResponse{
			HasTask: false,
			NextActions: []guideAction{
				{
					Command:     "mehr start <reference>",
					Description: "Start a new task",
					Endpoint:    "POST /api/v1/workflow/start",
				},
			},
		})

		return
	}

	// Load work for task details
	work, _ := ws.LoadWork(activeTask.ID)

	// Get specifications count
	specs, _ := ws.ListSpecificationsWithStatus(activeTask.ID)

	resp := guideResponse{
		HasTask:        true,
		TaskID:         activeTask.ID,
		State:          activeTask.State,
		Specifications: len(specs),
	}

	if work != nil {
		resp.Title = work.Metadata.Title
	}

	// Check for pending question
	if ws.HasPendingQuestion(activeTask.ID) {
		q, _ := ws.LoadPendingQuestion(activeTask.ID)
		if q != nil {
			var options []string
			for _, opt := range q.Options {
				options = append(options, opt.Label)
			}
			resp.PendingQuestion = &pendingQuestionInfo{
				Question: q.Question,
				Options:  options,
			}
			resp.NextActions = []guideAction{
				{
					Command:     "mehr answer \"your answer\"",
					Description: "Respond to the question",
					Endpoint:    "POST /api/v1/workflow/answer",
				},
			}
			s.writeJSON(w, http.StatusOK, resp)

			return
		}
	}

	// Generate state-specific suggestions
	resp.NextActions = getGuideActions(workflow.State(activeTask.State), len(specs))

	s.writeJSON(w, http.StatusOK, resp)
}

// getGuideActions returns suggested actions based on workflow state.
func getGuideActions(state workflow.State, specifications int) []guideAction {
	switch state {
	case workflow.StateIdle:
		if specifications == 0 {
			return []guideAction{
				{
					Command:     "mehr plan",
					Description: "Create specifications",
					Endpoint:    "POST /api/v1/workflow/plan",
				},
				{
					Command:     "mehr note",
					Description: "Add requirements",
					Endpoint:    "POST /api/v1/tasks/{id}/notes",
				},
			}
		}
		// Check if all specs are done
		return []guideAction{
			{
				Command:     "mehr implement",
				Description: "Implement the specifications",
				Endpoint:    "POST /api/v1/workflow/implement",
			},
			{
				Command:     "mehr plan",
				Description: "Create more specifications",
				Endpoint:    "POST /api/v1/workflow/plan",
			},
		}

	case workflow.StatePlanning:
		return []guideAction{
			{
				Command:     "mehr status",
				Description: "View planning progress",
				Endpoint:    "GET /api/v1/task",
			},
		}

	case workflow.StateImplementing:
		return []guideAction{
			{
				Command:     "mehr status",
				Description: "View implementation progress",
				Endpoint:    "GET /api/v1/task",
			},
			{
				Command:     "mehr undo",
				Description: "Revert last change",
				Endpoint:    "POST /api/v1/workflow/undo",
			},
			{
				Command:     "mehr finish",
				Description: "Complete and merge",
				Endpoint:    "POST /api/v1/workflow/finish",
			},
		}

	case workflow.StateReviewing:
		return []guideAction{
			{
				Command:     "mehr finish",
				Description: "Complete and merge",
				Endpoint:    "POST /api/v1/workflow/finish",
			},
			{
				Command:     "mehr implement",
				Description: "Make more changes",
				Endpoint:    "POST /api/v1/workflow/implement",
			},
		}

	case workflow.StateDone:
		return []guideAction{
			{
				Command:     "mehr start <reference>",
				Description: "Start a new task",
				Endpoint:    "POST /api/v1/workflow/start",
			},
		}

	case workflow.StateWaiting:
		return []guideAction{
			{
				Command:     "mehr answer \"response\"",
				Description: "Respond to agent question",
				Endpoint:    "POST /api/v1/workflow/answer",
			},
		}

	case workflow.StateFailed:
		return []guideAction{
			{
				Command:     "mehr status",
				Description: "View error details",
				Endpoint:    "GET /api/v1/task",
			},
			{
				Command:     "mehr implement",
				Description: "Retry implementation",
				Endpoint:    "POST /api/v1/workflow/implement",
			},
		}

	case workflow.StateCheckpointing, workflow.StateReverting, workflow.StateRestoring:
		return []guideAction{
			{
				Command:     "mehr status",
				Description: "View operation progress",
				Endpoint:    "GET /api/v1/task",
			},
		}
	}

	return []guideAction{
		{
			Command:     "mehr status",
			Description: "View detailed status",
			Endpoint:    "GET /api/v1/task",
		},
	}
}

// handleListAgents returns a list of available agents.
func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	registry := s.config.Conductor.GetAgentRegistry()
	if registry == nil {
		s.writeJSON(w, http.StatusOK, agentsListResponse{
			Agents: []agentInfo{},
			Count:  0,
		})

		return
	}

	agentNames := registry.List()
	slices.Sort(agentNames)

	var agents []agentInfo
	for _, name := range agentNames {
		a, err := registry.Get(name)
		if err != nil {
			continue
		}

		info := agentInfo{
			Name:      name,
			Available: a.Available() == nil,
		}

		// Check if it's an alias
		if alias, ok := a.(*agent.AliasAgent); ok {
			info.Type = "alias"
			info.Description = alias.Description()
			if base := alias.BaseAgent(); base != nil {
				info.Extends = base.Name()
			}
		} else {
			info.Type = "built-in"
			// Try to get description from MetadataProvider interface
			if mp, ok := a.(agent.MetadataProvider); ok {
				meta := mp.Metadata()
				info.Description = meta.Description
				info.Version = meta.Version

				// Add capabilities
				if meta.Capabilities.Streaming || meta.Capabilities.ToolUse ||
					meta.Capabilities.FileOperations || meta.Capabilities.CodeExecution ||
					meta.Capabilities.MultiTurn || meta.Capabilities.SystemPrompt ||
					len(meta.Capabilities.AllowedTools) > 0 {
					info.Capabilities = &agentCapabilitiesInfo{
						Streaming:      meta.Capabilities.Streaming,
						ToolUse:        meta.Capabilities.ToolUse,
						FileOperations: meta.Capabilities.FileOperations,
						CodeExecution:  meta.Capabilities.CodeExecution,
						MultiTurn:      meta.Capabilities.MultiTurn,
						SystemPrompt:   meta.Capabilities.SystemPrompt,
					}
					if len(meta.Capabilities.AllowedTools) > 0 {
						info.Capabilities.AllowedTools = meta.Capabilities.AllowedTools
					}
				}

				// Add models
				for _, m := range meta.Models {
					info.Models = append(info.Models, agentModelInfo{
						ID:         m.ID,
						Name:       m.Name,
						Default:    m.Default,
						MaxTokens:  m.MaxTokens,
						InputCost:  m.InputCost,
						OutputCost: m.OutputCost,
					})
				}
			}
		}

		agents = append(agents, info)
	}

	s.writeJSON(w, http.StatusOK, agentsListResponse{
		Agents: agents,
		Count:  len(agents),
	})
}

// handleListProviders returns a list of available providers.
func (s *Server) handleListProviders(w http.ResponseWriter, r *http.Request) {
	// Static provider info (same as CLI providers list)
	providers := []providerInfo{
		{
			Scheme:      "file",
			Shorthand:   "f",
			Name:        "File",
			Description: "Single markdown file",
		},
		{
			Scheme:      "dir",
			Shorthand:   "d",
			Name:        "Directory",
			Description: "Directory with README.md",
		},
		{
			Scheme:      "github",
			Shorthand:   "gh",
			Name:        "GitHub",
			Description: "GitHub issues and pull requests",
			EnvVars:     []string{"GITHUB_TOKEN"},
		},
		{
			Scheme:      "gitlab",
			Name:        "GitLab",
			Description: "GitLab issues and merge requests",
			EnvVars:     []string{"GITLAB_TOKEN"},
		},
		{
			Scheme:      "jira",
			Name:        "Jira",
			Description: "Atlassian Jira tickets",
			EnvVars:     []string{"JIRA_TOKEN"},
		},
		{
			Scheme:      "linear",
			Name:        "Linear",
			Description: "Linear issues",
			EnvVars:     []string{"LINEAR_API_KEY"},
		},
		{
			Scheme:      "notion",
			Name:        "Notion",
			Description: "Notion pages and databases",
			EnvVars:     []string{"NOTION_TOKEN"},
		},
		{
			Scheme:      "wrike",
			Name:        "Wrike",
			Description: "Wrike tasks",
			EnvVars:     []string{"WRIKE_TOKEN"},
		},
		{
			Scheme:      "youtrack",
			Shorthand:   "yt",
			Name:        "YouTrack",
			Description: "JetBrains YouTrack issues",
			EnvVars:     []string{"YOUTRACK_TOKEN"},
		},
		{
			Scheme:      "bitbucket",
			Shorthand:   "bb",
			Name:        "Bitbucket",
			Description: "Bitbucket issues and pull requests",
			EnvVars:     []string{"BITBUCKET_TOKEN"},
		},
		{
			Scheme:      "azure",
			Name:        "Azure DevOps",
			Description: "Azure DevOps work items",
			EnvVars:     []string{"AZURE_DEVOPS_TOKEN"},
		},
		{
			Scheme:      "clickup",
			Name:        "ClickUp",
			Description: "ClickUp tasks",
			EnvVars:     []string{"CLICKUP_TOKEN"},
		},
		{
			Scheme:      "asana",
			Name:        "Asana",
			Description: "Asana tasks",
			EnvVars:     []string{"ASANA_TOKEN"},
		},
		{
			Scheme:      "monday",
			Name:        "Monday",
			Description: "Monday.com items",
			EnvVars:     []string{"MONDAY_TOKEN"},
		},
		{
			Scheme:      "trello",
			Name:        "Trello",
			Description: "Trello cards",
			EnvVars:     []string{"TRELLO_KEY", "TRELLO_TOKEN"},
		},
	}

	s.writeJSON(w, http.StatusOK, providersListResponse{
		Providers: providers,
		Count:     len(providers),
	})
}

// enhanceTaskResponseWithPendingQuestion adds pending question to task response if present.
func enhanceTaskResponseWithPendingQuestion(response map[string]any, ws *storage.Workspace, taskID string) {
	if ws.HasPendingQuestion(taskID) {
		q, err := ws.LoadPendingQuestion(taskID)
		if err == nil && q != nil {
			var options []string
			for _, opt := range q.Options {
				options = append(options, opt.Label)
			}
			response["pending_question"] = map[string]any{
				"question": q.Question,
				"options":  options,
			}
		}
	}
}

// handleLicense returns the project license text.
func (s *Server) handleLicense(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(licensing.GetProjectLicense()))
}

// handleLicenseInfo returns all dependency licenses.
// Licenses are embedded at build time via go:embed for instant access.
func (s *Server) handleLicenseInfo(w http.ResponseWriter, r *http.Request) {
	// Read embedded licenses.json
	data, err := static.FS.ReadFile("licenses.json")
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to read licenses: "+err.Error())

		return
	}

	var response licensesListResponse
	if err := json.Unmarshal(data, &response); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to parse licenses: "+err.Error())

		return
	}

	s.writeJSON(w, http.StatusOK, response)
}

// handleListAgentAliases returns all configured agent aliases.
func (s *Server) handleListAgentAliases(w http.ResponseWriter, r *http.Request) {
	ws := s.getWorkspace()
	if ws == nil {
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	cfg, err := ws.LoadConfig()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to load config: "+err.Error())

		return
	}

	// Return agents map (aliases)
	s.writeJSON(w, http.StatusOK, map[string]any{
		"agents": cfg.Agents,
	})
}

// handleCreateAgentAlias creates a new agent alias.
func (s *Server) handleCreateAgentAlias(w http.ResponseWriter, r *http.Request) {
	ws := s.getWorkspace()
	if ws == nil {
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	var req struct {
		Name        string   `json:"name"`
		Extends     string   `json:"extends"`
		Description string   `json:"description"`
		Components  []string `json:"components"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())

		return
	}

	if req.Name == "" {
		s.writeError(w, http.StatusBadRequest, "name is required")

		return
	}
	if req.Extends == "" {
		s.writeError(w, http.StatusBadRequest, "extends is required")

		return
	}

	cfg, err := ws.LoadConfig()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to load config: "+err.Error())

		return
	}

	if cfg.Agents == nil {
		cfg.Agents = make(map[string]storage.AgentAliasConfig)
	}

	cfg.Agents[req.Name] = storage.AgentAliasConfig{
		Extends:     req.Extends,
		Description: req.Description,
		Components:  req.Components,
	}

	if err := ws.SaveConfig(cfg); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to save config: "+err.Error())

		return
	}

	s.writeJSON(w, http.StatusCreated, map[string]string{
		"status":  "created",
		"message": "Agent alias created",
	})
}

// handleDeleteAgentAlias deletes an agent alias.
func (s *Server) handleDeleteAgentAlias(w http.ResponseWriter, r *http.Request) {
	ws := s.getWorkspace()
	if ws == nil {
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	// Extract alias name from path (after /api/v1/agents/aliases/)
	// The router pattern is DELETE /api/v1/agents/aliases/
	// We need to get the name from the path
	path := r.URL.Path
	prefix := "/api/v1agents/aliases/"
	if !strings.HasPrefix(path, prefix) {
		// Try with full path
		prefix = "/api/v1/agents/aliases/"
	}

	name := strings.TrimPrefix(path, prefix)
	if name == "" {
		s.writeError(w, http.StatusBadRequest, "alias name is required")

		return
	}

	cfg, err := ws.LoadConfig()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to load config: "+err.Error())

		return
	}

	if cfg.Agents == nil {
		s.writeError(w, http.StatusNotFound, "alias not found")

		return
	}

	if _, exists := cfg.Agents[name]; !exists {
		s.writeError(w, http.StatusNotFound, "alias not found: "+name)

		return
	}

	delete(cfg.Agents, name)

	if err := ws.SaveConfig(cfg); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to save config: "+err.Error())

		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{
		"status":  "deleted",
		"message": "Agent alias deleted",
	})
}

// getWorkspace returns the workspace, handling both project and global modes.
func (s *Server) getWorkspace() *storage.Workspace {
	if s.config.Mode == ModeProject && s.config.Conductor != nil {
		return s.config.Conductor.GetWorkspace()
	}

	return nil
}
