package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/valksor/go-mehrhof/internal/agent"
	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/vcs"
	"github.com/valksor/go-mehrhof/internal/workflow"
)

// commitAgentAdapter bridges agent.Agent to the vcs.Agent interface.
type commitAgentAdapter struct {
	agent agent.Agent
}

func (a *commitAgentAdapter) Run(ctx context.Context, prompt string) (*vcs.AgentResponse, error) {
	resp, err := a.agent.Run(ctx, prompt)
	if err != nil {
		return nil, err
	}

	return &vcs.AgentResponse{Messages: resp.Messages}, nil
}

// commitFileChange represents a changed file with diff statistics.
type commitFileChange struct {
	Path      string `json:"path"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
}

// commitGroupView represents a file group with its generated message.
type commitGroupView struct {
	Files   []string `json:"files"`
	Message string   `json:"message,omitempty"`
	Reason  string   `json:"reason,omitempty"`
}

// commitExecuteResult represents the result of a single commit operation.
type commitExecuteResult struct {
	Hash    string `json:"hash,omitempty"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

func init() {
	Register(Command{
		Info: CommandInfo{
			Name:         "commit",
			Description:  "Analyze, plan, and execute git commits",
			Category:     "git",
			RequiresTask: false,
			MutatesState: false,
			Subcommands:  []string{"changes", "plan", "execute"},
		},
		Handler: handleCommit,
	})
}

// handleCommit dispatches to the appropriate commit subcommand.
func handleCommit(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	if len(inv.Args) == 0 {
		return nil, fmt.Errorf("%w: commit requires a subcommand (changes, plan, execute)", ErrBadRequest)
	}

	subcommand := inv.Args[0]
	sub := Invocation{
		Args:    inv.Args[1:],
		Options: inv.Options,
		Source:  inv.Source,
	}

	switch subcommand {
	case "changes":
		return handleCommitChanges(ctx, cond, sub)
	case "plan":
		return handleCommitPlan(ctx, cond, sub)
	case "execute":
		return handleCommitExecute(ctx, cond, sub)
	default:
		return nil, fmt.Errorf("%w: unknown commit subcommand %q (use changes, plan, execute)", ErrBadRequest, subcommand)
	}
}

// handleCommitChanges returns raw git changes without AI analysis.
func handleCommitChanges(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	git := cond.GetGit()
	if git == nil {
		return nil, errors.New("git not available")
	}

	includeUnstaged := GetBool(inv.Options, "include_unstaged")

	statuses, err := git.Status(ctx)
	if err != nil {
		return nil, fmt.Errorf("git status: %w", err)
	}

	stagedStats, _ := git.DiffNumstat(ctx, true)
	unstagedStats, _ := git.DiffNumstat(ctx, false)

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

	var (
		files          []commitFileChange
		totalAdditions int
		totalDeletions int
		hasStaged      bool
		hasUnstaged    bool
	)

	for _, f := range statuses {
		isStaged := f.IsStaged()
		isUnstagedFile := f.WorkDir != ' '

		if isStaged {
			hasStaged = true
		}
		if isUnstagedFile {
			hasUnstaged = true
		}

		if !includeUnstaged && !isStaged {
			continue
		}

		stat := statsMap[f.Path]
		files = append(files, commitFileChange{
			Path:      f.Path,
			Status:    commitMapGitStatus(f.Index, f.WorkDir),
			Additions: stat.Additions,
			Deletions: stat.Deletions,
		})
		totalAdditions += stat.Additions
		totalDeletions += stat.Deletions
	}

	return NewResult("Changes retrieved").WithData(map[string]any{
		"files":           files,
		"total_additions": totalAdditions,
		"total_deletions": totalDeletions,
		"has_staged":      hasStaged,
		"has_unstaged":    hasUnstaged,
	}), nil
}

// handleCommitPlan analyzes changes and returns commit groups with AI-generated messages.
func handleCommitPlan(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	git := cond.GetGit()
	if git == nil {
		return nil, errors.New("git not available")
	}

	includeUnstaged := GetBool(inv.Options, "include_unstaged")

	agentInstance, err := cond.GetAgentForStep(ctx, workflow.StepCheckpointing)
	if err != nil {
		return nil, fmt.Errorf("get agent: %w", err)
	}

	analyzer := vcs.NewChangeAnalyzer(git)
	analyzer.SetAgent(&commitAgentAdapter{agent: agentInstance})

	vcsGroups, err := analyzer.AnalyzeChanges(ctx, includeUnstaged)
	if err != nil {
		return nil, fmt.Errorf("analyze changes: %w", err)
	}

	groups := make([]commitGroupView, 0, len(vcsGroups))
	for _, g := range vcsGroups {
		msg := cond.GenerateCommitMessageForGroup(ctx, g, "", nil)
		groups = append(groups, commitGroupView{
			Files:   g.Files,
			Message: msg,
			Reason:  g.Reason,
		})
	}

	return NewResult("Commit plan generated").WithData(map[string]any{
		"groups": groups,
		"total":  len(groups),
	}), nil
}

// handleCommitExecute creates commits for the provided groups.
func handleCommitExecute(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	git := cond.GetGit()
	if git == nil {
		return nil, errors.New("git not available")
	}

	groups, err := parseCommitGroups(inv.Options)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrBadRequest, err.Error())
	}

	if len(groups) == 0 {
		return nil, fmt.Errorf("%w: no groups provided", ErrBadRequest)
	}

	push := GetBool(inv.Options, "push")

	var results []commitExecuteResult
	for _, g := range groups {
		if err := git.Add(ctx, g.files...); err != nil {
			results = append(results, commitExecuteResult{Error: err.Error()})

			continue
		}

		hash, err := git.Commit(ctx, g.message)
		if err != nil {
			results = append(results, commitExecuteResult{Error: err.Error()})

			continue
		}

		results = append(results, commitExecuteResult{Hash: hash, Message: g.message})
	}

	if push && len(results) > 0 {
		branch, err := git.CurrentBranch(ctx)
		if err == nil {
			_ = git.Push(ctx, "origin", branch)
		}
	}

	return NewResult("Commits executed").WithData(map[string]any{
		"results": results,
	}), nil
}

// commitGroup holds parsed commit group data from options.
type commitGroup struct {
	message string
	files   []string
}

// parseCommitGroups extracts commit groups from the options map.
func parseCommitGroups(opts map[string]any) ([]commitGroup, error) {
	if opts == nil {
		return nil, errors.New("options required")
	}

	rawGroups, ok := opts["groups"]
	if !ok {
		return nil, errors.New("groups field required")
	}

	groupSlice, ok := rawGroups.([]any)
	if !ok {
		return nil, errors.New("groups must be an array")
	}

	groups := make([]commitGroup, 0, len(groupSlice))
	for i, raw := range groupSlice {
		m, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("group %d: expected object", i)
		}

		message := GetString(m, "message")
		if message == "" {
			return nil, fmt.Errorf("group %d: message required", i)
		}

		files, err := parseStringSlice(m, "files")
		if err != nil {
			return nil, fmt.Errorf("group %d: %w", i, err)
		}
		if len(files) == 0 {
			return nil, fmt.Errorf("group %d: files required", i)
		}

		groups = append(groups, commitGroup{
			message: message,
			files:   files,
		})
	}

	return groups, nil
}

// parseStringSlice extracts a string slice from a map field.
func parseStringSlice(m map[string]any, key string) ([]string, error) {
	raw, ok := m[key]
	if !ok {
		return nil, fmt.Errorf("%s field required", key)
	}

	slice, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("%s must be an array", key)
	}

	result := make([]string, 0, len(slice))
	for _, v := range slice {
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("%s must contain strings", key)
		}

		result = append(result, s)
	}

	return result, nil
}

// commitMapGitStatus converts git status bytes to a human-readable status string.
func commitMapGitStatus(index, workDir byte) string {
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
