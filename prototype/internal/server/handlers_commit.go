package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/valksor/go-mehrhof/internal/agent"
	"github.com/valksor/go-mehrhof/internal/server/views"
	"github.com/valksor/go-mehrhof/internal/vcs"
	"github.com/valksor/go-mehrhof/internal/workflow"
)

// CommitGroupView represents a file group for the Web UI.
type CommitGroupView struct {
	Files   []string `json:"files"`
	Message string   `json:"message,omitempty"`
	Reason  string   `json:"reason,omitempty"`
}

// CommitResult represents the result of a commit operation.
type CommitResult struct {
	Hash    string `json:"hash,omitempty"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// handleCommitPage renders the commit page.
func (s *Server) handleCommitPage(w http.ResponseWriter, r *http.Request) {
	if s.renderer == nil {
		s.writeError(w, http.StatusInternalServerError, "renderer not loaded")

		return
	}

	pageData := views.ComputePageData(
		s.modeString(),
		s.config.Mode == ModeGlobal,
		s.config.AuthStore != nil,
		s.canSwitchProject(),
		s.getCurrentUser(r),
	)

	// Check if git is available
	enabled := s.config.Conductor != nil && s.config.Conductor.GetGit() != nil

	data := views.CommitData{
		PageData: pageData,
		Enabled:  enabled,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderer.RenderCommit(w, data); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to render template: "+err.Error())
	}
}

// handleCommitPlan analyzes changes and returns commit groups for preview.
func (s *Server) handleCommitPlan(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not available")

		return
	}

	git := s.config.Conductor.GetGit()
	if git == nil {
		s.writeError(w, http.StatusServiceUnavailable, "git not available")

		return
	}

	includeUnstaged := r.URL.Query().Get("all") == "true"

	analyzer := vcs.NewChangeAnalyzer(git)

	// Get agent from conductor for grouping
	agent, err := s.config.Conductor.GetAgentForStep(ctx, workflow.StepCheckpointing)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "get agent: "+err.Error())

		return
	}

	// Create adapter for vcs package
	analyzer.SetAgent(&agentAdapter{agent: agent})

	vcsGroups, err := analyzer.AnalyzeChanges(ctx, includeUnstaged)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "analyze changes: "+err.Error())

		return
	}

	// Convert to view format and generate messages
	var result []CommitGroupView
	for _, g := range vcsGroups {
		msg := s.config.Conductor.GenerateCommitMessageForGroup(ctx, g, "", nil)
		result = append(result, CommitGroupView{
			Files:   g.Files,
			Message: msg,
			Reason:  g.Reason,
		})
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"groups": result,
		"total":  len(result),
	})
}

// handleCommitExecute creates commits for the provided groups.
func (s *Server) handleCommitExecute(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not available")

		return
	}

	git := s.config.Conductor.GetGit()
	if git == nil {
		s.writeError(w, http.StatusServiceUnavailable, "git not available")

		return
	}

	var req struct {
		Groups []struct {
			Message string   `json:"message"`
			Files   []string `json:"files"`
		} `json:"groups"`
		Push bool `json:"push"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request")

		return
	}

	var results []CommitResult
	for _, g := range req.Groups {
		// Stage files
		if err := git.Add(ctx, g.Files...); err != nil {
			results = append(results, CommitResult{Error: err.Error()})

			continue
		}

		// Commit
		hash, err := git.Commit(ctx, g.Message)
		if err != nil {
			results = append(results, CommitResult{Error: err.Error()})

			continue
		}

		results = append(results, CommitResult{Hash: hash, Message: g.Message})
	}

	// Push if requested
	if req.Push && len(results) > 0 {
		branch, err := git.CurrentBranch(ctx)
		if err == nil {
			_ = git.Push(ctx, "origin", branch)
		}
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"results": results,
	})
}

// agentAdapter adapts agent.Agent to vcs.Agent interface.
type agentAdapter struct {
	agent agent.Agent
}

func (a *agentAdapter) Run(ctx context.Context, prompt string) (*vcs.AgentResponse, error) {
	resp, err := a.agent.Run(ctx, prompt)
	if err != nil {
		return nil, err
	}

	return &vcs.AgentResponse{Messages: resp.Messages}, nil
}
