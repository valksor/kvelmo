package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/storage"
)

// Project command flags.
var (
	// plan flags.
	projectPlanTitle        string
	projectPlanInstructions string

	// tasks flags.
	projectTasksStatus   string
	projectTasksShowDeps bool

	// edit flags.
	projectEditTitle       string
	projectEditDescription string
	projectEditPriority    int
	projectEditStatus      string
	projectEditDependsOn   string
	projectEditLabels      string
	projectEditAssignee    string

	// reorder flags.
	projectReorderBefore string
	projectReorderAfter  string
	projectReorderAuto   bool

	// submit flags.
	projectSubmitProvider   string
	projectSubmitCreateEpic bool
	projectSubmitLabels     string
	projectSubmitDryRun     bool

	// start flags.
	projectStartAuto bool
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Project planning and task management",
	Long: `Project planning workflow that creates local task queues from any source.

The project workflow:
1. Create a plan from any source (dir, file, provider reference)
2. Review and edit tasks locally
3. Submit tasks to an external provider
4. Start implementing tasks in order

COMMANDS:
  plan      Create a project plan from a source
  tasks     View and filter local task queue
  edit      Edit a task in the queue
  reorder   Reorder tasks in the queue
  submit    Submit tasks to an external provider
  start     Start implementing tasks from the queue

EXAMPLES:
  mehr project plan dir:/workspace/.final/ --title "Q1 Features"
  mehr project tasks --status ready
  mehr project edit task-2 --priority 1
  mehr project submit --provider wrike
  mehr project start`,
}

var projectPlanCmd = &cobra.Command{
	Use:   "plan <source>",
	Short: "Create a project plan from a source",
	Long: `Create a task breakdown from a source using AI.

SOURCES:
  dir:/path/to/dir        Directory of files to analyze
  file:/path/to/file.md   Single file to analyze
  github:123              GitHub issue
  jira:PROJ-123           Jira issue
  wrike:abc123            Wrike task

The AI will analyze the source content and create a structured
task breakdown with dependencies, priorities, and labels.

EXAMPLES:
  mehr project plan dir:/workspace/.final/
  mehr project plan file:requirements.md --title "Auth System"
  mehr project plan github:123`,
	Args: cobra.ExactArgs(1),
	RunE: runProjectPlan,
}

var projectTasksCmd = &cobra.Command{
	Use:   "tasks [queue-id]",
	Short: "View tasks in a queue",
	Long: `View and filter tasks in a project queue.

If no queue-id is provided, shows tasks from the most recent queue.

FLAGS:
  --status      Filter by status (pending, ready, blocked, submitted)
  --show-deps   Show dependency relationships

EXAMPLES:
  mehr project tasks
  mehr project tasks --status ready
  mehr project tasks my-queue --show-deps`,
	Args: cobra.MaximumNArgs(1),
	RunE: runProjectTasks,
}

var projectEditCmd = &cobra.Command{
	Use:   "edit <task-id>",
	Short: "Edit a task in the queue",
	Long: `Edit a task's properties in the local queue.

FLAGS:
  --title         New title for the task
  --description   New description
  --priority      Priority (1 = highest)
  --status        Status (pending, ready, blocked)
  --depends-on    Dependencies (comma-separated task IDs)
  --labels        Labels (comma-separated)
  --assignee      Assignee identifier

EXAMPLES:
  mehr project edit task-2 --priority 1
  mehr project edit task-2 --depends-on task-1,task-3
  mehr project edit task-2 --status ready`,
	Args: cobra.ExactArgs(1),
	RunE: runProjectEdit,
}

var projectReorderCmd = &cobra.Command{
	Use:   "reorder [task-id]",
	Short: "Reorder tasks in the queue",
	Long: `Reorder tasks in the queue manually or using AI.

FLAGS:
  --before    Move task before another task
  --after     Move task after another task
  --auto      Use AI to suggest optimal order

EXAMPLES:
  mehr project reorder task-3 --before task-1
  mehr project reorder task-3 --after task-5
  mehr project reorder --auto`,
	Args: cobra.MaximumNArgs(1),
	RunE: runProjectReorder,
}

var projectSubmitCmd = &cobra.Command{
	Use:   "submit [queue-id]",
	Short: "Submit tasks to an external provider",
	Long: `Submit tasks from a queue to an external provider.

FLAGS:
  --provider      Provider name (wrike, github, jira, etc.) [required]
  --create-epic   Create an epic/project to group tasks
  --labels        Labels to apply to all tasks (comma-separated)
  --dry-run       Preview only, don't actually submit

EXAMPLES:
  mehr project submit --provider wrike
  mehr project submit --provider github --create-epic
  mehr project submit --dry-run --provider jira`,
	Args: cobra.MaximumNArgs(1),
	RunE: runProjectSubmit,
}

var projectStartCmd = &cobra.Command{
	Use:   "start [queue-id]",
	Short: "Start implementing tasks from the queue",
	Long: `Start implementing the next ready task from a queue.

FLAGS:
  --auto    Auto-chain through all tasks

EXAMPLES:
  mehr project start
  mehr project start my-queue
  mehr project start --auto`,
	Args: cobra.MaximumNArgs(1),
	RunE: runProjectStart,
}

func init() {
	// plan flags
	projectPlanCmd.Flags().StringVar(&projectPlanTitle, "title", "", "Project title")
	projectPlanCmd.Flags().StringVar(&projectPlanInstructions, "instructions", "", "Custom instructions for AI")

	// tasks flags
	projectTasksCmd.Flags().StringVar(&projectTasksStatus, "status", "", "Filter by status")
	projectTasksCmd.Flags().BoolVar(&projectTasksShowDeps, "show-deps", false, "Show dependency relationships")

	// edit flags
	projectEditCmd.Flags().StringVar(&projectEditTitle, "title", "", "New title")
	projectEditCmd.Flags().StringVar(&projectEditDescription, "description", "", "New description")
	projectEditCmd.Flags().IntVar(&projectEditPriority, "priority", 0, "Priority (1 = highest)")
	projectEditCmd.Flags().StringVar(&projectEditStatus, "status", "", "Status (pending, ready, blocked)")
	projectEditCmd.Flags().StringVar(&projectEditDependsOn, "depends-on", "", "Dependencies (comma-separated)")
	projectEditCmd.Flags().StringVar(&projectEditLabels, "labels", "", "Labels (comma-separated)")
	projectEditCmd.Flags().StringVar(&projectEditAssignee, "assignee", "", "Assignee")

	// reorder flags
	projectReorderCmd.Flags().StringVar(&projectReorderBefore, "before", "", "Move task before this task")
	projectReorderCmd.Flags().StringVar(&projectReorderAfter, "after", "", "Move task after this task")
	projectReorderCmd.Flags().BoolVar(&projectReorderAuto, "auto", false, "AI-suggested order")

	// submit flags
	projectSubmitCmd.Flags().StringVar(&projectSubmitProvider, "provider", "", "Provider name (required)")
	projectSubmitCmd.Flags().BoolVar(&projectSubmitCreateEpic, "create-epic", false, "Create epic/project")
	projectSubmitCmd.Flags().StringVar(&projectSubmitLabels, "labels", "", "Labels (comma-separated)")
	projectSubmitCmd.Flags().BoolVar(&projectSubmitDryRun, "dry-run", false, "Preview only")

	// start flags
	projectStartCmd.Flags().BoolVar(&projectStartAuto, "auto", false, "Auto-chain through all tasks")

	// Add subcommands
	projectCmd.AddCommand(projectPlanCmd)
	projectCmd.AddCommand(projectTasksCmd)
	projectCmd.AddCommand(projectEditCmd)
	projectCmd.AddCommand(projectReorderCmd)
	projectCmd.AddCommand(projectSubmitCmd)
	projectCmd.AddCommand(projectStartCmd)

	// Add project command to root
	rootCmd.AddCommand(projectCmd)
}

func runProjectPlan(cmd *cobra.Command, args []string) error {
	source := args[0]
	ctx := context.Background()

	// Initialize conductor
	cond, err := initializeConductor(ctx)
	if err != nil {
		return fmt.Errorf("initialize: %w", err)
	}

	// Create project plan
	opts := conductor.ProjectPlanOptions{
		Title:              projectPlanTitle,
		CustomInstructions: projectPlanInstructions,
	}

	result, err := cond.CreateProjectPlan(ctx, source, opts)
	if err != nil {
		return fmt.Errorf("create plan: %w", err)
	}

	// Display results
	fmt.Printf("Created queue: %s\n", result.Queue.ID)
	fmt.Printf("  %d tasks identified\n", len(result.Tasks))
	if len(result.Questions) > 0 {
		fmt.Printf("  %d questions to resolve\n", len(result.Questions))
	}
	if len(result.Blockers) > 0 {
		fmt.Printf("  %d blockers noted\n", len(result.Blockers))
	}

	return nil
}

func runProjectTasks(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Initialize conductor
	cond, err := initializeConductor(ctx)
	if err != nil {
		return fmt.Errorf("initialize: %w", err)
	}

	// Get queue ID
	var queueID string
	if len(args) > 0 {
		queueID = args[0]
	} else {
		// Get most recent queue
		queues, err := cond.GetWorkspace().ListQueues()
		if err != nil {
			return fmt.Errorf("list queues: %w", err)
		}
		if len(queues) == 0 {
			return errors.New("no queues found")
		}
		queueID = queues[len(queues)-1] // Most recent
	}

	// Load queue
	queue, err := storage.LoadTaskQueue(cond.GetWorkspace(), queueID)
	if err != nil {
		return fmt.Errorf("load queue: %w", err)
	}

	// Filter tasks
	var tasks []*storage.QueuedTask
	for _, task := range queue.Tasks {
		if projectTasksStatus != "" && string(task.Status) != projectTasksStatus {
			continue
		}
		tasks = append(tasks, task)
	}

	// Display tasks
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "ID\tTitle\tStatus\tPriority\tDepends On\n")
	for _, task := range tasks {
		deps := "-"
		if len(task.DependsOn) > 0 {
			deps = strings.Join(task.DependsOn, ", ")
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n",
			task.ID, truncate(task.Title, 40), task.Status, task.Priority, deps)
	}
	_ = w.Flush()

	// Show dependency graph if requested
	if projectTasksShowDeps {
		fmt.Println("\nDependency Graph:")
		for _, task := range tasks {
			if len(task.Blocks) > 0 {
				fmt.Printf("  %s blocks: %s\n", task.ID, strings.Join(task.Blocks, ", "))
			}
		}
	}

	// Show questions
	if len(queue.Questions) > 0 {
		fmt.Println("\nQuestions:")
		for i, q := range queue.Questions {
			fmt.Printf("  %d. %s\n", i+1, q)
		}
	}

	return nil
}

func runProjectEdit(cmd *cobra.Command, args []string) error {
	taskID := args[0]
	ctx := context.Background()

	// Initialize conductor
	cond, err := initializeConductor(ctx)
	if err != nil {
		return fmt.Errorf("initialize: %w", err)
	}

	// Get most recent queue
	queues, err := cond.GetWorkspace().ListQueues()
	if err != nil {
		return fmt.Errorf("list queues: %w", err)
	}
	if len(queues) == 0 {
		return errors.New("no queues found")
	}
	queueID := queues[len(queues)-1]

	// Load queue
	queue, err := storage.LoadTaskQueue(cond.GetWorkspace(), queueID)
	if err != nil {
		return fmt.Errorf("load queue: %w", err)
	}

	// Update task
	err = queue.UpdateTask(taskID, func(task *storage.QueuedTask) {
		if projectEditTitle != "" {
			task.Title = projectEditTitle
		}
		if projectEditDescription != "" {
			task.Description = projectEditDescription
		}
		if projectEditPriority != 0 {
			task.Priority = projectEditPriority
		}
		if projectEditStatus != "" {
			task.Status = storage.TaskStatus(projectEditStatus)
		}
		if projectEditDependsOn != "" {
			task.DependsOn = strings.Split(projectEditDependsOn, ",")
			for i, dep := range task.DependsOn {
				task.DependsOn[i] = strings.TrimSpace(dep)
			}
		}
		if projectEditLabels != "" {
			task.Labels = strings.Split(projectEditLabels, ",")
			for i, label := range task.Labels {
				task.Labels[i] = strings.TrimSpace(label)
			}
		}
		if projectEditAssignee != "" {
			task.Assignee = projectEditAssignee
		}
	})
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	// Recompute relationships
	queue.ComputeBlocksRelations()
	queue.ComputeTaskStatuses()

	// Save queue
	if err := queue.Save(); err != nil {
		return fmt.Errorf("save queue: %w", err)
	}

	fmt.Printf("Updated task: %s\n", taskID)

	return nil
}

func runProjectReorder(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Initialize conductor
	cond, err := initializeConductor(ctx)
	if err != nil {
		return fmt.Errorf("initialize: %w", err)
	}

	// Get most recent queue
	queues, err := cond.GetWorkspace().ListQueues()
	if err != nil {
		return fmt.Errorf("list queues: %w", err)
	}
	if len(queues) == 0 {
		return errors.New("no queues found")
	}
	queueID := queues[len(queues)-1]

	// Load queue
	queue, err := storage.LoadTaskQueue(cond.GetWorkspace(), queueID)
	if err != nil {
		return fmt.Errorf("load queue: %w", err)
	}

	if projectReorderAuto {
		result, err := cond.AutoReorderTasks(cmd.Context(), queueID)
		if err != nil {
			return fmt.Errorf("auto reorder: %w", err)
		}

		fmt.Println("Tasks reordered by AI:")
		fmt.Println()
		fmt.Println("New order:")
		for i, taskID := range result.NewOrder {
			fmt.Printf("  %d. %s\n", i+1, taskID)
		}
		fmt.Println()
		if result.Reasoning != "" {
			fmt.Println("Reasoning:")
			fmt.Println(result.Reasoning)
		}

		return nil
	}

	if len(args) == 0 {
		return errors.New("task ID required for manual reorder")
	}
	taskID := args[0]

	// Find target index
	var targetIndex int
	if projectReorderBefore != "" {
		for i, task := range queue.Tasks {
			if task.ID == projectReorderBefore {
				targetIndex = i

				break
			}
		}
	} else if projectReorderAfter != "" {
		for i, task := range queue.Tasks {
			if task.ID == projectReorderAfter {
				targetIndex = i + 1

				break
			}
		}
	} else {
		return errors.New("--before or --after required")
	}

	// Reorder
	if err := queue.ReorderTask(taskID, targetIndex); err != nil {
		return fmt.Errorf("reorder: %w", err)
	}

	// Save queue
	if err := queue.Save(); err != nil {
		return fmt.Errorf("save queue: %w", err)
	}

	fmt.Printf("Moved task %s to position %d\n", taskID, targetIndex+1)

	return nil
}

func runProjectSubmit(cmd *cobra.Command, args []string) error {
	if projectSubmitProvider == "" {
		return errors.New("--provider is required")
	}
	ctx := context.Background()

	// Initialize conductor
	cond, err := initializeConductor(ctx)
	if err != nil {
		return fmt.Errorf("initialize: %w", err)
	}

	// Get queue ID
	var queueID string
	if len(args) > 0 {
		queueID = args[0]
	} else {
		queues, err := cond.GetWorkspace().ListQueues()
		if err != nil {
			return fmt.Errorf("list queues: %w", err)
		}
		if len(queues) == 0 {
			return errors.New("no queues found")
		}
		queueID = queues[len(queues)-1]
	}

	// Parse labels
	var labels []string
	if projectSubmitLabels != "" {
		labels = strings.Split(projectSubmitLabels, ",")
		for i, label := range labels {
			labels[i] = strings.TrimSpace(label)
		}
	}

	// Submit
	opts := conductor.SubmitOptions{
		Provider:   projectSubmitProvider,
		CreateEpic: projectSubmitCreateEpic,
		Labels:     labels,
		DryRun:     projectSubmitDryRun,
	}

	result, err := cond.SubmitProjectTasks(ctx, queueID, opts)
	if err != nil {
		return fmt.Errorf("submit: %w", err)
	}

	// Display results
	if result.DryRun {
		fmt.Println("Dry run - no tasks submitted")
	}
	fmt.Printf("Submitted %d tasks to %s:\n", len(result.Tasks), projectSubmitProvider)
	for _, task := range result.Tasks {
		fmt.Printf("  %s -> %s (%s)\n", task.LocalID, task.ExternalID, task.Title)
	}

	return nil
}

func runProjectStart(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Initialize conductor
	cond, err := initializeConductor(ctx)
	if err != nil {
		return fmt.Errorf("initialize: %w", err)
	}

	// Get queue ID
	var queueID string
	if len(args) > 0 {
		queueID = args[0]
	} else {
		queues, err := cond.GetWorkspace().ListQueues()
		if err != nil {
			return fmt.Errorf("list queues: %w", err)
		}
		if len(queues) == 0 {
			return errors.New("no queues found")
		}
		queueID = queues[len(queues)-1]
	}

	if projectStartAuto {
		// Run full automation
		opts := conductor.ProjectAutoOptions{}
		result, err := cond.RunProjectAuto(ctx, "", opts)
		if err != nil {
			return fmt.Errorf("auto: %w", err)
		}
		fmt.Printf("Completed %d/%d tasks\n", result.TasksCompleted, result.TasksPlanned)

		return nil
	}

	// Start next task
	task, err := cond.StartNextTask(ctx, queueID)
	if err != nil {
		return fmt.Errorf("start: %w", err)
	}

	fmt.Printf("Started task: %s - %s\n", task.ID, task.Title)

	return nil
}

// truncate truncates a string to maxLen length with ellipsis.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	return s[:maxLen-3] + "..."
}
