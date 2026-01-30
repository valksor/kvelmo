package commands

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/valksor/go-mehrhof/internal/agent"
	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/vcs"
	"github.com/valksor/go-mehrhof/internal/workflow"
	"github.com/valksor/go-toolkit/display"
)

var (
	commitPush   bool
	commitAll    bool
	commitDryRun bool
	commitNote   string // User hint for steering grouping
	commitAgent  string // Agent to use for commit message generation
)

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Create logical commits from uncommitted changes",
	Long: `Create logically grouped commits from uncommitted changes using AI.

The commit command analyzes your changes and groups them into logical
commits based on semantic relationships (same feature, bugfix, refactor, etc.).

Commit messages are generated to MATCH THE STYLE of existing commits in
your repository - it learns from your git history!

FLAGS:
  --push, -p    Push commits to remote after creating
  --all, -a     Include unstaged changes
  --dry-run, -n Show what would be committed without creating
  --note, -m    Hint to guide AI grouping when re-running

EXAMPLES:
  mehr commit                 # Create commits from staged changes
  mehr commit --all           # Include unstaged changes
  mehr commit --dry-run       # Preview commits without creating
  mehr commit --note "group 1 and 3 are the same feature"
`,
	RunE: runCommit,
}

func init() {
	rootCmd.AddCommand(commitCmd)
	commitCmd.Flags().BoolVarP(&commitPush, "push", "p", false, "Push commits to remote after creating")
	commitCmd.Flags().BoolVarP(&commitAll, "all", "a", false, "Include unstaged changes")
	commitCmd.Flags().BoolVarP(&commitDryRun, "dry-run", "n", false, "Show what would be committed without creating")
	commitCmd.Flags().StringVarP(&commitNote, "note", "m", "", "Hint to guide AI grouping (e.g., 'group 1 and 3 are same feature')")
	commitCmd.Flags().StringVar(&commitAgent, "agent-commit", "", "Agent to use for commit message generation")
}

func runCommit(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	res, err := ResolveWorkspaceRoot(ctx)
	if err != nil {
		return err
	}

	if res.Git == nil {
		return errors.New("not in a git repository")
	}

	// Open workspace for history persistence
	ws, err := storage.OpenWorkspace(ctx, res.Root, nil)
	if err != nil {
		return fmt.Errorf("open workspace: %w", err)
	}

	history := storage.NewCommitHistory(ws.DataRoot())
	previousAttempts, _ := history.LoadAttempts() // Ignore error if no history yet

	// Calculate current file hash to detect if files changed
	currentHash, _ := res.Git.HashChangedFiles(ctx, commitAll)

	// Check if we can reuse the most recent attempt (optimization)
	var reusedGroups []storage.ChangeGroup
	if len(previousAttempts) > 0 {
		lastAttempt := previousAttempts[len(previousAttempts)-1]
		// Reuse if: same files, last was dry-run (user approved), and no new note
		if lastAttempt.FileHash == currentHash && lastAttempt.IsDryRun && commitNote == lastAttempt.Note {
			// User saw dry-run and now wants to commit - reuse the groups!
			reusedGroups = lastAttempt.Groups
			fmt.Printf("Reusing grouping from dry-run (%s)\n",
				lastAttempt.Timestamp.Format("15:04:05"))
		}
	}

	// Show attempt number if we have history and not reusing
	if len(reusedGroups) == 0 && len(previousAttempts) > 0 {
		lastAttempt := previousAttempts[len(previousAttempts)-1]
		fmt.Printf("Attempt #%d (refining attempt from %s)\n",
			len(previousAttempts)+1,
			lastAttempt.Timestamp.Format("15:04:05"))
	}

	var groups []storage.ChangeGroup

	if len(reusedGroups) > 0 {
		// Reuse from dry-run - skip AI analysis!
		groups = reusedGroups
	} else {
		// Analyze changes with AI
		analyzer := vcs.NewChangeAnalyzer(res.Git)

		// We need to get the agent from conductor for the analyzer
		var opts []conductor.Option
		if commitAgent != "" {
			opts = append(opts, conductor.WithStepAgent("checkpointing", commitAgent))
		}
		cond, err := initializeConductor(ctx, opts...)
		if err != nil {
			return fmt.Errorf("initialize conductor: %w", err)
		}

		// Get the checkpointing agent for grouping
		aiAgent, err := cond.GetAgentForStep(ctx, workflow.StepCheckpointing)
		if err != nil {
			return fmt.Errorf("get agent: %w", err)
		}

		// Create an adapter that wraps the agent for the vcs package
		analyzer.SetAgent(&agentAdapter{agent: aiAgent})

		vcsGroups, err := analyzer.AnalyzeChanges(ctx, commitAll)
		if err != nil {
			return fmt.Errorf("analyze changes: %w", err)
		}

		if len(vcsGroups) == 0 {
			fmt.Println(display.InfoMsg("No changes to commit"))

			return nil
		}

		// Convert []vcs.FileGroup to []storage.ChangeGroup
		for _, g := range vcsGroups {
			groups = append(groups, storage.ChangeGroup{
				Files: g.Files,
			})
		}

		// Save this attempt for potential future refinements
		attempt := storage.CommitAttempt{
			Timestamp: time.Now(),
			Groups:    groups,
			Note:      commitNote,
			FileHash:  currentHash,
			IsDryRun:  commitDryRun,
		}
		if err := history.SaveAttempt(attempt); err != nil {
			// Non-fatal: just log warning
			fmt.Printf("Warning: could not save attempt history: %v\n", err)
		}
	}

	// Use conductor to generate commit messages for each group
	var msgOpts []conductor.Option
	if commitAgent != "" {
		msgOpts = append(msgOpts, conductor.WithStepAgent("checkpointing", commitAgent))
	}
	cond, err := initializeConductor(ctx, msgOpts...)
	if err != nil {
		return err
	}

	// Generate messages and show preview
	for i, group := range groups {
		// Convert storage.ChangeGroup to vcs.FileGroup for the conductor
		vcsGroup := vcs.FileGroup{
			Files: group.Files,
		}

		// Generate commit message using agent with context
		msg := cond.GenerateCommitMessageForGroup(ctx, vcsGroup, commitNote, previousAttempts)
		group.Message = msg

		fmt.Printf("[%d] %s\n", i+1, display.Bold(msg))
		for _, f := range group.Files {
			fmt.Printf("    %s\n", display.Muted(f))
		}
		fmt.Println()
	}

	if commitDryRun {
		fmt.Println(display.Muted("(dry run: no commits created)"))

		return nil
	}

	// Confirm before committing (unless --push is used, implying confidence)
	if !commitPush {
		fmt.Printf("Create %d commits? [y/N] ", len(groups))
		var response string
		_, _ = fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Aborted")

			return nil
		}
	}

	// Execute commits
	for _, group := range groups {
		// Stage files
		if err := res.Git.Add(ctx, group.Files...); err != nil {
			return fmt.Errorf("stage files: %w", err)
		}

		// Commit
		hash, err := res.Git.Commit(ctx, group.Message)
		if err != nil {
			return fmt.Errorf("commit: %w", err)
		}

		fmt.Printf("Created %s\n", display.SuccessMsg("%s", hash[:8]+" "+group.Message))
	}

	// Clear history after successful commits
	_ = history.Clear()

	// Push if requested
	if commitPush {
		branch, err := res.Git.CurrentBranch(ctx)
		if err != nil {
			return fmt.Errorf("get current branch: %w", err)
		}

		if err := res.Git.Push(ctx, "origin", branch); err != nil {
			return fmt.Errorf("push: %w", err)
		}
		fmt.Println(display.SuccessMsg("Pushed to remote"))
	}

	return nil
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
