package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/valksor/go-mehrhof/internal/agent"
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

// ChangesResponse is the response for GET /api/v1/commit/changes.
type ChangesResponse struct {
	Files          []FileChange `json:"files"`
	TotalAdditions int          `json:"total_additions"`
	TotalDeletions int          `json:"total_deletions"`
	HasStaged      bool         `json:"has_staged"`
	HasUnstaged    bool         `json:"has_unstaged"`
}

// FileChange represents a changed file with diff statistics.
type FileChange struct {
	Path      string `json:"path"`
	Status    string `json:"status"` // added, modified, deleted, renamed
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
}

// handleCommitChanges returns raw git changes without AI analysis.
// GET /api/v1/commit/changes?include_unstaged=true|false.
func (s *Server) handleCommitChanges(w http.ResponseWriter, r *http.Request) {
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

	includeUnstaged := r.URL.Query().Get("include_unstaged") == "true"

	// Get file statuses
	statuses, err := git.Status(ctx)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "git status: "+err.Error())

		return
	}

	// Get diff stats for additions/deletions
	stagedStats, _ := git.DiffNumstat(ctx, true)
	unstagedStats, _ := git.DiffNumstat(ctx, false)

	// Build stats map for lookup
	statsMap := make(map[string]vcs.DiffStat)
	for _, stat := range stagedStats {
		statsMap[stat.Path] = stat
	}
	if includeUnstaged {
		for _, stat := range unstagedStats {
			if existing, ok := statsMap[stat.Path]; ok {
				existing.Additions += stat.Additions
				existing.Deletions += stat.Deletions
				statsMap[stat.Path] = existing
			} else {
				statsMap[stat.Path] = stat
			}
		}
	}

	// Build response
	resp := ChangesResponse{}
	for _, f := range statuses {
		isStaged := f.IsStaged()
		isUnstaged := f.WorkDir != ' '

		if isStaged {
			resp.HasStaged = true
		}
		if isUnstaged {
			resp.HasUnstaged = true
		}

		// Skip if only unstaged and we don't want unstaged
		if !includeUnstaged && !isStaged {
			continue
		}

		stat := statsMap[f.Path]
		resp.Files = append(resp.Files, FileChange{
			Path:      f.Path,
			Status:    mapGitStatus(f.Index, f.WorkDir),
			Additions: stat.Additions,
			Deletions: stat.Deletions,
		})
		resp.TotalAdditions += stat.Additions
		resp.TotalDeletions += stat.Deletions
	}

	s.writeJSON(w, http.StatusOK, resp)
}

// mapGitStatus converts git status bytes to a human-readable status string.
func mapGitStatus(index, workDir byte) string {
	if index == 'A' || workDir == 'A' {
		return "added"
	}
	if index == 'D' || workDir == 'D' {
		return "deleted"
	}
	if index == 'R' {
		return "renamed"
	}

	return "modified"
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
