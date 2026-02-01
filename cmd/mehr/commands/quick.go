package commands

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-toolkit/display"
)

var (
	quickTitle    string
	quickPriority int
	quickLabels   []string
	quickQueue    string
	quickAgent    string
)

var quickCmd = &cobra.Command{
	Use:   "quick <description>",
	Short: "Create a quick task without full planning",
	Long: `Create a simple task quickly without going through the full project planning workflow.

Quick tasks are stored in a local queue and can be:
  - Iterated on with notes
  - Optimized by AI
  - Submitted to external providers
  - Exported to markdown files
  - Started directly with the standard workflow

USAGE:
  mehr quick <description>

EXAMPLES:
  mehr quick "fix typo in README.md line 42"
  mehr quick --label bug --priority 1 "investigate crash"
  mehr quick --title "Auth Fix" "users report login fails"

After creating a task, you'll be prompted for next steps:
  [d]iscuss - Enter discussion mode (add notes)
  [o]ptimize - AI optimizes task based on notes
  [s]ubmit - Submit to provider
  [tart]   - Start working on it
  [x]exit   - Done for now

See also:
  mehr optimize --task=<ref>  - AI optimize a task
  mehr note --task=<ref>      - Add notes to a task
  mehr export --task=<ref>     - Export to markdown
  mehr submit --task=<ref>     - Submit to provider`,
	Args: cobra.MinimumNArgs(1),
	RunE: runQuick,
}

func init() {
	rootCmd.AddCommand(quickCmd)

	quickCmd.Flags().StringVar(&quickTitle, "title", "", "Custom task title (auto-extracted from description)")
	quickCmd.Flags().IntVar(&quickPriority, "priority", 2, "Task priority (1=high, 2=normal, 3=low)")
	quickCmd.Flags().StringSliceVar(&quickLabels, "label", []string{}, "Task labels (can be specified multiple times)")
	quickCmd.Flags().StringVar(&quickQueue, "queue", "", "Target queue ID (default: quick-tasks)")
	quickCmd.Flags().StringVar(&quickAgent, "agent", "", "Agent to use for this task")
}

func runQuick(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	description := strings.Join(args, " ")

	// Validate description
	if strings.TrimSpace(description) == "" {
		return errors.New("description cannot be empty")
	}

	queueID := quickQueue
	if queueID == "" {
		queueID = "quick-tasks"
	}

	// Build conductor options
	opts := BuildConductorOptions(CommandOptions{
		Verbose: verbose,
	})

	if quickAgent != "" {
		opts = append(opts, conductor.WithAgent(quickAgent))
	}

	// Initialize conductor
	cond, err := initializeConductor(ctx, opts...)
	if err != nil {
		return err
	}

	// Create a quick task
	result, err := cond.CreateQuickTask(ctx, conductor.QuickTaskOptions{
		Description: description,
		Title:       quickTitle,
		Priority:    quickPriority,
		Labels:      quickLabels,
		QueueID:     queueID,
	})
	if err != nil {
		return fmt.Errorf("create quick task: %w", err)
	}

	// Display success
	displayQuickTaskSuccess(result)

	// Show an interactive menu
	return quickTaskMenu(ctx, cond, queueID, result.TaskID)
}

// displayQuickTaskSuccess shows the success message for a created task.
func displayQuickTaskSuccess(result *conductor.QuickTaskResult) {
	fmt.Println()
	fmt.Printf("✓ Created task: %s\n", display.Success(result.TaskID))
	fmt.Printf("  Title: %s\n", display.Bold(result.Title))
	fmt.Printf("  Queue: %s\n", result.QueueID)
}

// quickTaskMenu shows the interactive menu after creating a quick task.
func quickTaskMenu(ctx context.Context, cond *conductor.Conductor, queueID, taskID string) error {
	fmt.Println()
	fmt.Println("What next?")
	fmt.Println("  [d]iscuss - Enter discussion mode (add notes)")
	fmt.Println("  [o]ptimize - AI optimizes task based on notes")
	fmt.Println("  [s]ubmit - Submit to provider")
	fmt.Println("  [tart]   - Start working on it")
	fmt.Println("  [x]exit   - Done for now")

	// Read choice
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("\nYour choice [D/o/s/t/x]: ")

	input, err := reader.ReadString('\n')
	if err != nil {
		// Non-interactive mode, just exit
		return err
	}

	choice := strings.ToLower(strings.TrimSpace(input))

	// Execute choice
	return executeQuickChoice(ctx, cond, queueID, taskID, choice)
}

// executeQuickChoice executes the user's menu choice.
func executeQuickChoice(ctx context.Context, cond *conductor.Conductor, queueID, taskID, choice string) error {
	taskRef := fmt.Sprintf("%s/%s", queueID, taskID)

	switch choice {
	case "d", "":
		// Discuss mode - add notes interactively
		return enterDiscussMode(ctx, cond, queueID, taskID)

	case "o":
		// Optimize with AI
		fmt.Println("\n✨ Optimizing task with AI...")
		result, err := cond.OptimizeQueueTask(ctx, queueID, taskID)
		if err != nil {
			return fmt.Errorf("optimize task: %w", err)
		}
		displayOptimizeResult(result)

		return nil

	case "s":
		// Submit to provider
		fmt.Println("\n📤 Submit to provider")
		fmt.Print("Provider name (github, wrike, jira, etc.): ")

		reader := bufio.NewReader(os.Stdin)
		provider, _ := reader.ReadString('\n')
		provider = strings.TrimSpace(provider)

		if provider == "" {
			fmt.Println("Cancelled (no provider specified)")

			return nil
		}

		// Submit task
		submitResult, err := cond.SubmitQueueTask(ctx, queueID, taskID, conductor.SubmitOptions{
			Provider: provider,
			TaskIDs:  []string{taskID},
		})
		if err != nil {
			return fmt.Errorf("submit task: %w", err)
		}

		displaySubmitResult(submitResult)

		return nil

	case "t":
		// Start working on the task
		fmt.Println("\n▶️ Starting task...")

		return cond.Start(ctx, "queue:"+taskRef)

	case "x":
		// Exit
		fmt.Println("\nDone for now.")
		fmt.Printf("  Resume with: %s\n", display.Cyan("mehr optimize --task="+taskRef))

		return nil

	default:
		fmt.Printf("\nUnknown choice: %s\n", choice)
		fmt.Println("Done for now.")

		return nil
	}
}

// enterDiscussMode enters interactive discussion mode for adding notes.
//
//nolint:unparam // ctx is kept for a consistent signature with other command functions
func enterDiscussMode(ctx context.Context, cond *conductor.Conductor, queueID, taskID string) error {
	fmt.Println("\n💬 Discussion Mode")
	fmt.Println("Type your notes below. Type 'exit' or 'quit' when done.")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	ws := cond.GetWorkspace()

	for {
		fmt.Print("> ")

		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("read input: %w", err)
		}

		note := strings.TrimSpace(input)

		if note == "" {
			continue
		}

		if note == "exit" || note == "quit" {
			fmt.Println("\nNotes saved.")
			fmt.Printf("  Optimize with: %s\n", display.Cyan(fmt.Sprintf("mehr optimize --task=%s/%s", queueID, taskID)))

			return nil
		}

		// Add note
		if err := ws.AppendQueueNote(queueID, taskID, note); err != nil {
			return fmt.Errorf("save note: %w", err)
		}

		fmt.Println("✓ Note saved.")
	}
}

// displaySubmitResult shows the result of the task submission.
func displaySubmitResult(result *conductor.SubmitResult) {
	if len(result.Tasks) == 0 {
		fmt.Println("  No tasks submitted")

		return
	}

	task := result.Tasks[0]
	fmt.Println()
	fmt.Println("✓ Task submitted:")
	fmt.Printf("  External ID: %s\n", display.Bold(task.ExternalID))
	if task.ExternalURL != "" {
		fmt.Printf("  URL: %s\n", display.Cyan(task.ExternalURL))
	}
}
