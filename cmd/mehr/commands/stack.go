package commands

import (
	"errors"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/valksor/go-mehrhof/internal/project"
	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-mehrhof/internal/stack"
	"github.com/valksor/go-mehrhof/internal/storage"
)

var stackCmd = &cobra.Command{
	Use:   "stack",
	Short: "Manage stacked features (dependent tasks)",
	Long: `View and manage stacked features - tasks that depend on each other.

Stacked features allow you to work on Feature B while Feature A is in code review.
When A is merged, B can be rebased onto the updated target branch.

Examples:
  mehr stack                # List all stacks with status
  mehr stack --graph        # ASCII graph visualization
  mehr stack --mermaid      # Mermaid diagram output
  mehr stack rebase         # Rebase tasks that need it`,
	RunE: runStack,
}

var stackRebaseCmd = &cobra.Command{
	Use:   "rebase [task-id]",
	Short: "Rebase stacked tasks onto their new base",
	Long: `Rebase stacked tasks after their parent has been merged.

If no task-id is provided, rebases all tasks marked as 'needs-rebase'.
Aborts entirely on conflict, leaving the repository in a clean state.

Examples:
  mehr stack rebase              # Rebase all tasks needing it
  mehr stack rebase issue-101    # Rebase specific task`,
	RunE: runStackRebase,
}

var stackSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync PR status for stacked features",
	Long: `Fetch PR status from the provider and update stack states.

This checks each stacked task's PR status and updates the local state:
- If a PR is merged, the task state becomes 'merged'
- Children of merged tasks are marked as 'needs-rebase'
- Closed/declined PRs are marked as 'abandoned'

Examples:
  mehr stack sync                # Sync all stacked feature PRs`,
	RunE: runStackSync,
}

var (
	stackGraph   bool
	stackMermaid bool
)

func init() {
	rootCmd.AddCommand(stackCmd)
	stackCmd.AddCommand(stackRebaseCmd)
	stackCmd.AddCommand(stackSyncCmd)

	stackCmd.Flags().BoolVar(&stackGraph, "graph", false, "Show ASCII graph visualization")
	stackCmd.Flags().BoolVar(&stackMermaid, "mermaid", false, "Output Mermaid diagram format")
}

func runStack(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	res, err := ResolveWorkspaceRoot(ctx)
	if err != nil {
		return err
	}

	ws, err := storage.OpenWorkspace(ctx, res.Root, nil)
	if err != nil {
		return fmt.Errorf("open workspace: %w", err)
	}

	stackStorage := stack.NewStorage(ws.DataRoot())
	if err := stackStorage.Load(); err != nil {
		return fmt.Errorf("load stacks: %w", err)
	}

	stacks := stackStorage.ListStacks()
	if len(stacks) == 0 {
		fmt.Println("No stacked features found.")
		fmt.Println("\nUse 'mehr start <task> --depends-on <parent>' to create a stacked feature.")

		return nil
	}

	// Handle visualization output
	if stackGraph || stackMermaid {
		return outputStackVisualization(stacks, stackGraph)
	}

	// Default list output
	return outputStackList(stacks)
}

func outputStackList(stacks []*stack.Stack) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	for _, s := range stacks {
		_, _ = fmt.Fprintf(w, "Stack: %s (%d tasks)\n", s.ID, s.TaskCount())

		for _, task := range s.Tasks {
			icon := getStateIcon(task.State)
			status := string(task.State)

			// Add extra context for certain states
			if task.State == stack.StateNeedsRebase {
				status += " (parent merged)"
			} else if task.State == stack.StatePendingReview && task.PRNumber > 0 {
				status += fmt.Sprintf(" (PR #%d)", task.PRNumber)
			}

			_, _ = fmt.Fprintf(w, "  %s %s\t%s\t%s\n", icon, task.ID, task.Branch, status)
		}
		_, _ = fmt.Fprintln(w)
	}

	return w.Flush()
}

func getStateIcon(state stack.StackState) string {
	switch state {
	case stack.StateMerged:
		return "✓"
	case stack.StateNeedsRebase:
		return "⟳"
	case stack.StateConflict:
		return "✗"
	case stack.StatePendingReview:
		return "◐"
	case stack.StateApproved:
		return "◉"
	case stack.StateAbandoned:
		return "⊘"
	case stack.StateActive:
		return "●"
	}

	return "●"
}

func outputStackVisualization(stacks []*stack.Stack, ascii bool) error {
	// Convert stacks to dependency graph format
	graph := stacksToGraph(stacks)

	if ascii {
		fmt.Println(project.ASCIIGraph(graph))
	} else {
		fmt.Println(project.GenerateMermaid(graph))
	}

	return nil
}

func stacksToGraph(stacks []*stack.Stack) *project.DependencyGraph {
	graph := &project.DependencyGraph{
		Nodes: make([]project.GraphNode, 0),
		Edges: make([]project.GraphEdge, 0),
	}

	for _, s := range stacks {
		for _, task := range s.Tasks {
			graph.Nodes = append(graph.Nodes, project.GraphNode{
				ID:     task.ID,
				Title:  task.Branch,
				Status: stateToGraphStatus(task.State),
			})

			if task.DependsOn != "" {
				graph.Edges = append(graph.Edges, project.GraphEdge{
					From: task.DependsOn,
					To:   task.ID,
				})
			}
		}
	}

	return graph
}

func stateToGraphStatus(state stack.StackState) string {
	switch state {
	case stack.StateMerged:
		return "done"
	case stack.StateActive:
		return "in_progress"
	case stack.StateNeedsRebase, stack.StateConflict:
		return "blocked"
	case stack.StatePendingReview, stack.StateApproved, stack.StateAbandoned:
		return "pending"
	}

	return "pending"
}

func runStackRebase(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	res, err := ResolveWorkspaceRoot(ctx)
	if err != nil {
		return err
	}

	// Require git repository
	if res.Git == nil {
		return errors.New("not in a git repository")
	}

	ws, err := storage.OpenWorkspace(ctx, res.Root, nil)
	if err != nil {
		return fmt.Errorf("open workspace: %w", err)
	}

	stackStorage := stack.NewStorage(ws.DataRoot())
	if err := stackStorage.Load(); err != nil {
		return fmt.Errorf("load stacks: %w", err)
	}

	// Create rebaser
	rebaser := stack.NewRebaser(stackStorage, res.Git)

	if len(args) > 0 {
		// Rebase specific task
		taskID := args[0]
		fmt.Printf("Rebasing task %s...\n", taskID)

		result, err := rebaser.RebaseTask(ctx, taskID)

		return handleRebaseResult(result, err)
	}

	// Rebase all tasks that need it
	// Find which stacks have tasks needing rebase
	var stacksWithRebase []*stack.Stack
	for _, s := range stackStorage.ListStacks() {
		if len(s.GetTasksNeedingRebase()) > 0 {
			stacksWithRebase = append(stacksWithRebase, s)
		}
	}

	if len(stacksWithRebase) == 0 {
		fmt.Println("No tasks need rebasing.")

		return nil
	}

	// Rebase each stack
	var totalRebased int
	for _, s := range stacksWithRebase {
		tasks := s.GetTasksNeedingRebase()
		fmt.Printf("Rebasing %d task(s) in stack %s...\n", len(tasks), s.ID)

		result, err := rebaser.RebaseAll(ctx, s.ID)
		if err != nil {
			return handleRebaseResult(result, err)
		}

		totalRebased += len(result.RebasedTasks)

		// Report results for this stack
		for _, tr := range result.RebasedTasks {
			fmt.Printf("  ✓ %s: rebased onto %s\n", tr.TaskID, tr.NewBase)
		}
	}

	if totalRebased > 0 {
		fmt.Printf("\nSuccessfully rebased %d task(s).\n", totalRebased)
	}

	return nil
}

func handleRebaseResult(result *stack.RebaseResult, err error) error {
	if result == nil {
		return err
	}

	// Report successful rebases
	for _, tr := range result.RebasedTasks {
		fmt.Printf("  ✓ %s: rebased onto %s\n", tr.TaskID, tr.NewBase)
	}

	// Report skipped tasks
	for _, st := range result.SkippedTasks {
		fmt.Printf("  - %s: skipped (%s)\n", st.TaskID, st.Reason)
	}

	// Report failure
	if result.FailedTask != nil {
		ft := result.FailedTask
		fmt.Printf("\n✗ Rebase failed for %s (%s)\n", ft.TaskID, ft.Branch)
		if ft.IsConflict {
			fmt.Println("  Reason: merge conflict detected")
			fmt.Printf("  Target: %s\n", ft.OntoBase)
			if ft.ConflictHint != "" {
				fmt.Printf("  Hint: %s\n", ft.ConflictHint)
			}
		} else {
			fmt.Printf("  Error: %v\n", ft.Error)
		}

		return fmt.Errorf("rebase failed for task %s", ft.TaskID)
	}

	return err
}

func runStackSync(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	res, err := ResolveWorkspaceRoot(ctx)
	if err != nil {
		return err
	}

	ws, err := storage.OpenWorkspace(ctx, res.Root, nil)
	if err != nil {
		return fmt.Errorf("open workspace: %w", err)
	}

	stackStorage := stack.NewStorage(ws.DataRoot())
	if err := stackStorage.Load(); err != nil {
		return fmt.Errorf("load stacks: %w", err)
	}

	stacks := stackStorage.ListStacks()
	if len(stacks) == 0 {
		fmt.Println("No stacked features to sync.")

		return nil
	}

	// Count tasks with PR numbers
	var prCount int
	for _, s := range stacks {
		for _, task := range s.Tasks {
			if task.PRNumber > 0 {
				prCount++
			}
		}
	}

	if prCount == 0 {
		fmt.Println("No stacked features have PR numbers to sync.")
		fmt.Println("\nTip: Update task PR numbers with 'mehr stack set-pr <task-id> <pr-number>'")

		return nil
	}

	fmt.Printf("Syncing PR status for %d task(s)...\n", prCount)

	// Initialize conductor to get the provider registry
	cond, err := initializeConductor(ctx)
	if err != nil {
		return fmt.Errorf("initialize conductor: %w", err)
	}

	// Get the default provider for PR fetching
	registry := cond.GetProviderRegistry()

	// Try to resolve a provider that supports PR fetching
	// For now, we'll use a simple approach - get the provider from the first task with a PR
	var prFetcher provider.PRFetcher
	for _, s := range stacks {
		for _, task := range s.Tasks {
			if task.PRNumber > 0 {
				// Try to get provider from task's source
				work, err := ws.LoadWork(task.ID)
				if err != nil {
					continue
				}

				providerInstance, _, err := registry.Resolve(ctx, work.Source.Ref, provider.NewConfig(), provider.ResolveOptions{})
				if err != nil {
					continue
				}

				if fetcher, ok := providerInstance.(provider.PRFetcher); ok {
					prFetcher = fetcher

					break
				}
			}
		}
		if prFetcher != nil {
			break
		}
	}

	if prFetcher == nil {
		return errors.New("no provider with PR fetching capability found")
	}

	// Create a tracker and sync
	tracker := stack.NewTracker(stackStorage)
	result, err := tracker.Sync(ctx, prFetcher)
	if err != nil {
		return fmt.Errorf("sync: %w", err)
	}

	// Report results
	if len(result.UpdatedTasks) == 0 {
		fmt.Println("No state changes detected.")

		return nil
	}

	fmt.Printf("\nUpdated %d task(s):\n", len(result.UpdatedTasks))
	for _, update := range result.UpdatedTasks {
		fmt.Printf("  %s: %s → %s", update.TaskID, update.OldState, update.NewState)
		if update.NewState == stack.StateMerged && update.ChildrenMarkedForRebase > 0 {
			fmt.Printf(" (%d children need rebase)", update.ChildrenMarkedForRebase)
		}
		fmt.Println()
	}

	if len(result.Errors) > 0 {
		fmt.Printf("\nWarnings (%d):\n", len(result.Errors))
		for _, err := range result.Errors {
			fmt.Printf("  - %v\n", err)
		}
	}

	return nil
}
