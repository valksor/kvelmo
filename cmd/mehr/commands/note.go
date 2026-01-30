package commands

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/display"
	"github.com/valksor/go-mehrhof/internal/storage"
	kitdisplay "github.com/valksor/go-toolkit/display"
)

var (
	noteTask    string // Add note to queue task (format: <queue-id>/<task-id>)
	noteRunning string // Add note to running parallel task (running task ID)
)

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

	// If --running specified, add note to running parallel task
	if noteRunning != "" {
		return addNoteToRunningTask(ctx, noteRunning, args)
	}

	// If --task specified, add note to queue task
	if noteTask != "" {
		return addNoteToQueueTask(ctx, cond, noteTask, args)
	}

	// Check for active task
	if cond.GetActiveTask() == nil {
		fmt.Print(display.NoActiveTaskError())

		return errors.New("no active task")
	}

	taskID := cond.GetActiveTask().ID
	ws := cond.GetWorkspace()

	// Helper function to save a note
	saveNote := func(message string) error {
		if ws.HasPendingQuestion(taskID) {
			q, _ := ws.LoadPendingQuestion(taskID)
			note := fmt.Sprintf("**Q:** %s\n\n**A:** %s", q.Question, message)
			if err := ws.AppendNote(taskID, note, "answer"); err != nil {
				return fmt.Errorf("save answer: %w", err)
			}

			// Persist answer to latest session for context recovery
			latestSessionFile := ws.GetLatestSessionFile(taskID)
			if latestSessionFile != "" {
				session, err := ws.LoadSession(taskID, latestSessionFile)
				if err == nil && session != nil {
					now := time.Now()
					// Record the user's answer
					session.Exchanges = append(session.Exchanges, storage.Exchange{
						Role:      "user",
						Content:   "ANSWER: " + message,
						Timestamp: now,
					})
					_ = ws.SaveSession(taskID, latestSessionFile, session)
				}
			}

			// Archive full context to transcript if available
			if q.FullContext != "" {
				transcriptFile := time.Now().Format("2006-01-02T15-04-05") + "-answer.log"
				fullEntry := fmt.Sprintf("=== Question ===\n%s\n\n=== Answer ===\n%s\n\n=== Context ===\n%s",
					q.Question, message, q.FullContext)
				_ = ws.SaveTranscript(taskID, transcriptFile, fullEntry)
			}

			_ = ws.ClearPendingQuestion(taskID)

			return nil
		}

		return ws.AppendNote(taskID, message, cond.GetActiveTask().State)
	}

	// If message provided as argument, save it and exit
	if len(args) > 0 {
		message := strings.Join(args, " ")

		// Check if answering a question BEFORE saving (saveNote clears the question)
		hadPendingQuestion := ws.HasPendingQuestion(taskID)

		// Show the question being answered (so user knows what they're responding to)
		if hadPendingQuestion {
			q, _ := ws.LoadPendingQuestion(taskID)
			fmt.Printf("Answering: %s\n", q.Question)
		}

		if err := saveNote(message); err != nil {
			return fmt.Errorf("save note: %w", err)
		}

		// Context-aware success message
		if hadPendingQuestion {
			fmt.Println("Answer submitted.")
			fmt.Println("\nRun 'mehr plan' to continue with your answer.")
		} else {
			fmt.Println("Note saved.")
		}

		return nil
	}

	// Interactive mode
	scanner := bufio.NewScanner(os.Stdin)

	status, _ := cond.Status(ctx)
	fmt.Printf("Task: %s (state: %s)\n", status.TaskID, status.State)

	// Show pending question if any
	if ws.HasPendingQuestion(taskID) {
		q, _ := ws.LoadPendingQuestion(taskID)
		fmt.Printf("\n⚠ Pending question from agent:\n  %s\n", q.Question)
		if len(q.Options) > 0 {
			fmt.Println("  Options:")
			for i, opt := range q.Options {
				fmt.Printf("    %d. %s\n", i+1, opt.Label)
			}
		}
		fmt.Println("\nType your answer, then run 'mehr plan' to continue.")
	}

	fmt.Println("\nEntering interactive mode. Type 'exit' or 'quit' to leave.")
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

// addNoteToQueueTask adds a note to a queue task without starting it.
//
//nolint:unparam // ctx is kept for consistent signature with other command functions
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

	// Add note to queue task
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

	// Send note to running task
	if err := registry.AddNote(ctx, runningID, message); err != nil {
		return fmt.Errorf("send note: %w", err)
	}

	fmt.Printf("✓ Note sent to running task %s\n", runningID)

	return nil
}
