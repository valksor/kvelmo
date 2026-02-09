package commands

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/display"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/workflow"
	kitdisplay "github.com/valksor/go-toolkit/display"
)

var (
	noteTask    string // Add a note to queue task (format: <queue-id>/<task-id>)
	noteRunning string // Add a note to running the parallel task (running task ID)
)

var noteListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all notes for the active task",
	Long: `List all notes with timestamps and workflow state.

Each note shows its number, timestamp, and the workflow state when it was added.

Examples:
  mehr note list    # List all notes for the active task`,
	RunE: runNoteList,
}

var noteViewCmd = &cobra.Command{
	Use:   "view [number]",
	Short: "View note content",
	Long: `Display the full content of notes.

Without a number, shows all notes. With a number, shows only that note.

Examples:
  mehr note view       # View all notes
  mehr note view 1     # View note #1
  mehr note view 3     # View note #3`,
	Args: cobra.MaximumNArgs(1),
	RunE: runNoteView,
}

var noteCmd = &cobra.Command{
	Use:     "note [message]",
	Aliases: []string{"answer"},
	Short:   "Add notes to the task or answer agent questions",
	Long: `Add notes, context, or requirements to the current task.

This command saves your input directly to notes.md in the work directory.
Notes are included when the agent runs during plan/implement/review phases.

You can also add notes to queue tasks (without starting them) using --task:
  mehr note --task=quick-tasks/task-1 "Add requirement"

You can send notes to running parallel tasks using --running:
  mehr note --running=abc123 "Consider this edge case"

WHEN TO USE:
  • You want to add requirements or context before running plan/implement
  • The agent is waiting for your answer to a question
  • You want to provide clarification or additional information
  • You want to add notes to a quick task before optimizing or starting it

USE THIS COMMAND FOR:
  Adding context, requirements, or answering agent questions

RELATED COMMANDS:
  plan      - Create implementation specifications (sees your notes)
  implement - Execute specifications (sees your notes)
  guide     - Check if an agent question is pending
  optimize  - AI optimizes task based on notes

Examples:
  mehr note                           # Enter interactive mode
  mehr note "Use PostgreSQL"          # Add a note
  mehr note "Add error handling"      # Add context before planning
  mehr note --task=quick-tasks/task-1 "Add requirement"  # Add to queue task
  mehr note --running=abc123 "edge case"  # Send note to parallel running task
  mehr answer "Yes, proceed"          # Answer agent question (alias)`,
	RunE: runNote,
}

func init() {
	rootCmd.AddCommand(noteCmd)
	noteCmd.AddCommand(noteListCmd)
	noteCmd.AddCommand(noteViewCmd)
	noteCmd.Flags().StringVar(&noteTask, "task", "", "Queue task ID (format: <queue-id>/<task-id>)")
	noteCmd.Flags().StringVar(&noteRunning, "running", "", "Running parallel task ID (from 'mehr list --running')")
}

func runNote(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Build conductor options
	opts := []conductor.Option{
		conductor.WithVerbose(verbose),
	}

	// Initialize conductor with standard providers and agents
	cond, err := initializeConductor(ctx, opts...)
	if err != nil {
		return err
	}

	// If --running specified, add a note to running the parallel task
	if noteRunning != "" {
		return addNoteToRunningTask(ctx, noteRunning, args)
	}

	// If --task specified, add note to a queue task
	if noteTask != "" {
		return addNoteToQueueTask(ctx, cond, noteTask, args)
	}

	// Check for an active task
	if cond.GetActiveTask() == nil {
		fmt.Print(display.NoActiveTaskError())

		return errors.New("no active task")
	}

	taskID := cond.GetActiveTask().ID
	ws := cond.GetWorkspace()

	// Helper function to save a regular note (not for answering questions)
	saveNote := func(message string) error {
		return ws.AppendNote(taskID, message, cond.GetActiveTask().State)
	}

	// If a message provided as argument, save it and exit
	if len(args) > 0 {
		message := strings.Join(args, " ")

		// Case 1: Pending question exists - use AnswerQuestion for full handling
		if ws.HasPendingQuestion(taskID) {
			q, _ := ws.LoadPendingQuestion(taskID)
			fmt.Printf("Answering: %s\n", q.Question)

			if err := cond.AnswerQuestion(ctx, message); err != nil {
				return fmt.Errorf("answer question: %w", err)
			}

			fmt.Println("Answer submitted.")
			fmt.Println("\nNext steps:")
			fmt.Printf("  %s\n", kitdisplay.Cyan("mehr plan")+"      "+kitdisplay.Muted("- Continue planning with your answer"))
			fmt.Printf("  %s\n", kitdisplay.Cyan("mehr implement")+" "+kitdisplay.Muted("- Start implementation"))

			return nil
		}

		// Case 2: No pending question but stuck in waiting state
		// (edge case: old code cleared question but didn't transition state)
		status, _ := cond.Status(ctx)
		if status.State == string(workflow.StateWaiting) {
			// Save note first
			if err := saveNote(message); err != nil {
				return fmt.Errorf("save note: %w", err)
			}

			// Reset state to idle
			if err := cond.ResetState(ctx); err != nil {
				return fmt.Errorf("reset state: %w", err)
			}

			fmt.Println("Answer submitted. State reset to idle.")
			fmt.Println("\nNext steps:")
			fmt.Printf("  %s\n", kitdisplay.Cyan("mehr plan")+"      "+kitdisplay.Muted("- Continue planning with your answer"))
			fmt.Printf("  %s\n", kitdisplay.Cyan("mehr implement")+" "+kitdisplay.Muted("- Start implementation"))

			return nil
		}

		// Case 3: Regular note (not in waiting state)
		if err := saveNote(message); err != nil {
			return fmt.Errorf("save note: %w", err)
		}

		fmt.Println("Note saved.")

		return nil
	}

	// Interactive mode
	scanner := bufio.NewScanner(os.Stdin)

	status, _ := cond.Status(ctx)
	fmt.Printf("Task: %s (state: %s)\n", status.TaskID, status.State)

	// Track state for handling waiting without pending question
	hasPendingQuestion := ws.HasPendingQuestion(taskID)
	isWaitingState := status.State == string(workflow.StateWaiting)

	// Show appropriate prompt based on state
	if hasPendingQuestion {
		q, _ := ws.LoadPendingQuestion(taskID)
		fmt.Printf("\n⚠ Pending question from agent:\n  %s\n", q.Question)
		if len(q.Options) > 0 {
			fmt.Println("  Options:")
			for i, opt := range q.Options {
				fmt.Printf("    %d. %s\n", i+1, opt.Label)
			}
		}
		fmt.Println("\nType your answer below (or 'exit' to leave without answering):")
	} else if isWaitingState {
		fmt.Println("\n⚠ Task is in waiting state. Type your answer to continue.")
	} else {
		fmt.Println("\nEntering interactive mode. Type 'exit' or 'quit' to leave.")
	}
	fmt.Println()

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		message := strings.TrimSpace(scanner.Text())
		if message == "" {
			continue
		}

		if message == "exit" || message == "quit" {
			break
		}

		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Case 1: Pending question - use AnswerQuestion
		if hasPendingQuestion {
			if err := cond.AnswerQuestion(ctx, message); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)

				continue
			}

			fmt.Println("Answer submitted.")
			fmt.Println("\nNext steps:")
			fmt.Printf("  %s\n", kitdisplay.Cyan("mehr plan")+"      "+kitdisplay.Muted("- Continue planning with your answer"))
			fmt.Printf("  %s\n", kitdisplay.Cyan("mehr implement")+" "+kitdisplay.Muted("- Start implementation"))

			hasPendingQuestion = false
			isWaitingState = false
			fmt.Println("\nContinue adding notes, or type 'exit' to leave.")

			continue
		}

		// Case 2: Waiting state but no pending question - reset state
		if isWaitingState {
			if err := saveNote(message); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)

				continue
			}

			if err := cond.ResetState(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "Error resetting state: %v\n", err)

				continue
			}

			fmt.Println("Answer submitted. State reset to idle.")
			fmt.Println("\nNext steps:")
			fmt.Printf("  %s\n", kitdisplay.Cyan("mehr plan")+"      "+kitdisplay.Muted("- Continue planning with your answer"))
			fmt.Printf("  %s\n", kitdisplay.Cyan("mehr implement")+" "+kitdisplay.Muted("- Start implementation"))

			isWaitingState = false
			fmt.Println("\nContinue adding notes, or type 'exit' to leave.")

			continue
		}

		// Case 3: Regular note
		if err := saveNote(message); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)

			continue
		}

		fmt.Println("Note saved.")
		fmt.Println()
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	return nil
}

// noteContext provides the state context needed for handleNoteMessage.
// This allows testing without mocking workspace internals.
type noteContext struct {
	hasPendingQuestion bool
	isWaitingState     bool
}

// handleNoteMessage processes a note message based on the current state.
// This function is extracted from runNote to enable unit testing.
//
// Returns:
//   - nil on success
//   - error if the operation fails
func handleNoteMessage(ctx context.Context, cond ConductorAPI, message string, nc noteContext) error {
	// Case 1: Pending question - use AnswerQuestion for full handling
	if nc.hasPendingQuestion {
		if err := cond.AnswerQuestion(ctx, message); err != nil {
			return fmt.Errorf("answer question: %w", err)
		}

		return nil
	}

	// Case 2: Waiting state but no pending question - save note and reset
	if nc.isWaitingState {
		if err := cond.AddNote(ctx, message); err != nil {
			return fmt.Errorf("save note: %w", err)
		}

		if err := cond.ResetState(ctx); err != nil {
			return fmt.Errorf("reset state: %w", err)
		}

		return nil
	}

	// Case 3: Regular note (not in waiting state)
	if err := cond.AddNote(ctx, message); err != nil {
		return fmt.Errorf("save note: %w", err)
	}

	return nil
}

// addNoteToQueueTask adds a note to a queue task without starting it.
//
//nolint:unparam // ctx is kept for a consistent signature with other command functions
func addNoteToQueueTask(ctx context.Context, cond *conductor.Conductor, taskRef string, args []string) error {
	// Parse queue task reference
	queueID, taskID, err := conductor.ParseQueueTaskRef(taskRef)
	if err != nil {
		return err
	}

	ws := cond.GetWorkspace()

	// Get the message
	var message string
	if len(args) > 0 {
		message = strings.Join(args, " ")
	} else {
		// Interactive mode
		fmt.Printf("Adding notes to: %s/%s\n", queueID, taskID)
		fmt.Println("Enter your note below (empty line to finish, or Ctrl+C to cancel):")

		reader := bufio.NewReader(os.Stdin)
		var lines []string

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("read input: %w", err)
			}

			line = strings.TrimSuffix(line, "\n")
			if line == "" {
				break
			}
			lines = append(lines, line)
		}

		message = strings.Join(lines, "\n")
	}

	if strings.TrimSpace(message) == "" {
		return errors.New("note cannot be empty")
	}

	// Add note to the queue task
	if err := ws.AppendQueueNote(queueID, taskID, message); err != nil {
		return fmt.Errorf("save note: %w", err)
	}

	fmt.Printf("✓ Note saved to %s/%s\n", queueID, taskID)

	// Show next steps
	fmt.Println("\nNext steps:")
	fmt.Printf("  %s\n", kitdisplay.Cyan(fmt.Sprintf("mehr note --task=%s/%s \"more context\"", queueID, taskID)))
	fmt.Printf("  %s\n", kitdisplay.Cyan(fmt.Sprintf("mehr optimize --task=%s/%s", queueID, taskID)))

	return nil
}

// addNoteToRunningTask adds a note to a running parallel task.
func addNoteToRunningTask(ctx context.Context, runningID string, args []string) error {
	registry := GetParallelRegistry()
	if registry == nil {
		return errors.New("no parallel tasks are running")
	}

	// Check if the task exists
	task := registry.Get(runningID)
	if task == nil {
		return fmt.Errorf("running task %q not found\n\nUse 'mehr list --running' to see active parallel tasks", runningID)
	}

	// Get the message
	var message string
	if len(args) > 0 {
		message = strings.Join(args, " ")
	} else {
		// Interactive mode
		fmt.Printf("Adding note to running task: %s (%s)\n", runningID, task.Reference)
		fmt.Println("Enter your note below (empty line to finish, or Ctrl+C to cancel):")

		reader := bufio.NewReader(os.Stdin)
		var lines []string

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("read input: %w", err)
			}

			line = strings.TrimSuffix(line, "\n")
			if line == "" {
				break
			}
			lines = append(lines, line)
		}

		message = strings.Join(lines, "\n")
	}

	if strings.TrimSpace(message) == "" {
		return errors.New("note cannot be empty")
	}

	// Send note to a running task
	if err := registry.AddNote(ctx, runningID, message); err != nil {
		return fmt.Errorf("send note: %w", err)
	}

	fmt.Printf("✓ Note sent to running task %s\n", runningID)

	return nil
}

// runNoteList lists all notes for the active task.
func runNoteList(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	cond, err := initializeConductor(ctx)
	if err != nil {
		return err
	}

	if cond.GetActiveTask() == nil {
		fmt.Print(display.NoActiveTaskError())

		return errors.New("no active task")
	}

	taskID := cond.GetActiveTask().ID
	ws := cond.GetWorkspace()

	notes, err := ws.LoadNotes(taskID)
	if err != nil {
		return fmt.Errorf("load notes: %w", err)
	}

	if len(notes) == 0 {
		fmt.Println("No notes yet.")
		fmt.Println("\nAdd a note with:")
		fmt.Printf("  %s\n", kitdisplay.Cyan("mehr note \"your message\""))

		return nil
	}

	fmt.Printf("%s\n\n", kitdisplay.Bold(fmt.Sprintf("Notes (%d)", len(notes))))

	for _, note := range notes {
		stateInfo := ""
		if note.State != "" {
			stateInfo = kitdisplay.Muted(fmt.Sprintf(" [%s]", note.State))
		}

		// Truncate content for list view
		content := note.Content
		if len(content) > 60 {
			content = content[:57] + "..."
		}
		content = strings.ReplaceAll(content, "\n", " ")

		fmt.Printf("  %s %s%s\n",
			kitdisplay.Cyan(fmt.Sprintf("#%d", note.Number)),
			note.Timestamp.Format("2006-01-02 15:04"),
			stateInfo)
		fmt.Printf("     %s\n\n", kitdisplay.Muted(content))
	}

	fmt.Println("View full content with:")
	fmt.Printf("  %s\n", kitdisplay.Cyan("mehr note view")+"       "+kitdisplay.Muted("- View all notes"))
	fmt.Printf("  %s\n", kitdisplay.Cyan("mehr note view 1")+"     "+kitdisplay.Muted("- View note #1"))

	return nil
}

// runNoteView displays the full content of notes.
func runNoteView(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	cond, err := initializeConductor(ctx)
	if err != nil {
		return err
	}

	if cond.GetActiveTask() == nil {
		fmt.Print(display.NoActiveTaskError())

		return errors.New("no active task")
	}

	taskID := cond.GetActiveTask().ID
	ws := cond.GetWorkspace()

	notes, err := ws.LoadNotes(taskID)
	if err != nil {
		return fmt.Errorf("load notes: %w", err)
	}

	if len(notes) == 0 {
		fmt.Println("No notes yet.")
		fmt.Println("\nAdd a note with:")
		fmt.Printf("  %s\n", kitdisplay.Cyan("mehr note \"your message\""))

		return nil
	}

	// If a number is provided, show only that note
	if len(args) > 0 {
		number, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid note number: %w", err)
		}

		note := findNoteByNumber(notes, number)
		if note == nil {
			fmt.Printf("Note #%d not found. Available notes: 1-%d\n", number, len(notes))

			return fmt.Errorf("note #%d not found", number)
		}

		displayNote(note)

		return nil
	}

	// Show all notes
	for i, note := range notes {
		displayNote(&note)
		if i < len(notes)-1 {
			fmt.Println(strings.Repeat("─", 60))
			fmt.Println()
		}
	}

	return nil
}

// findNoteByNumber finds a note by its number.
func findNoteByNumber(notes []storage.Note, number int) *storage.Note {
	for _, note := range notes {
		if note.Number == number {
			return &note
		}
	}

	return nil
}

// displayNote formats and prints a single note.
func displayNote(note *storage.Note) {
	stateInfo := ""
	if note.State != "" {
		stateInfo = fmt.Sprintf(" [%s]", note.State)
	}

	fmt.Printf("%s %s%s\n\n",
		kitdisplay.Cyan(fmt.Sprintf("#%d", note.Number)),
		kitdisplay.Muted(note.Timestamp.Format("2006-01-02 15:04:05")),
		kitdisplay.Muted(stateInfo))
	fmt.Println(note.Content)
	fmt.Println()
}
