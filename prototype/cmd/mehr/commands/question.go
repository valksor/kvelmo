package commands

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/display"
	kitdisplay "github.com/valksor/go-toolkit/display"
)

var questionCmd = &cobra.Command{
	Use:     "question [message]",
	Aliases: []string{"ask", "q"},
	Short:   "Ask the active agent a question during implementation",
	Long: `Send a question to the currently executing agent and receive an answer.

This command allows you to have a conversation with the agent during planning,
implementation, or review without changing the workflow state.

The agent sees:
- Your question
- Current task title and state
- Latest specification content
- Recent conversation history

WHEN TO USE:
  • You want to understand why the agent made a decision
  • You need clarification on the implementation approach
  • You want to discuss alternatives before proceeding
  • You need help understanding the codebase

USE THIS COMMAND FOR:
  Asking questions and getting answers from the agent

RELATED COMMANDS:
  note      - Add context (agent won't respond)
  plan      - Create specifications (agent may ask questions)
  implement - Execute specifications
  guide     - Check next actions

Examples:
  mehr question                           # Enter interactive mode
  mehr question "Why did you use interface X?"
  mehr ask "What's the reason for this pattern?"
  mehr q "Explain this approach"`,
	RunE: runQuestion,
}

func init() {
	rootCmd.AddCommand(questionCmd)
}

func runQuestion(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Build conductor options
	opts := []conductor.Option{
		conductor.WithVerbose(verbose),
	}

	// Initialize conductor
	cond, err := initializeConductor(ctx, opts...)
	if err != nil {
		return err
	}

	// Set up verbose event handlers for agent output
	SetupVerboseEventHandlers(cond)

	// Check for active task
	if cond.GetActiveTask() == nil {
		fmt.Print(display.NoActiveTaskError())

		return errors.New("no active task")
	}

	taskID := cond.GetActiveTask().ID

	// Get current status
	status, _ := cond.Status(ctx)
	currentState := status.State

	// Check if questions are allowed in current state
	validStates := map[string]bool{
		"planning":     true,
		"implementing": true,
		"reviewing":    true,
	}
	if !validStates[currentState] {
		fmt.Printf("Cannot ask questions in state '%s'.\n", kitdisplay.Cyan(currentState))
		fmt.Println("\nQuestions are allowed during:")
		fmt.Println("  ", kitdisplay.Cyan("planning"), "     - Use 'mehr plan' first")
		fmt.Println("  ", kitdisplay.Cyan("implementing"), " - Use 'mehr implement' first")
		fmt.Println("  ", kitdisplay.Cyan("reviewing"), "    - Use 'mehr review' first")

		return nil
	}

	// Helper function to ask a question
	askQuestion := func(question string) error {
		if question == "" {
			return errors.New("question cannot be empty")
		}

		fmt.Printf("\n%s %s\n", kitdisplay.Muted("Question:"), question)
		fmt.Println()

		if err := cond.AskQuestion(ctx, question); err != nil {
			// Check if agent asked a back-question
			if errors.Is(err, conductor.ErrPendingQuestion) {
				fmt.Println()
				fmt.Println(kitdisplay.Warning("Agent has a follow-up question."))
				fmt.Println("Use 'mehr note' to respond.")

				return nil
			}

			return fmt.Errorf("ask question: %w", err)
		}

		return nil
	}

	// If message provided as argument, ask it and exit
	if len(args) > 0 {
		question := joinArgs(args)
		if err := askQuestion(question); err != nil {
			return err
		}

		fmt.Println()

		return nil
	}

	// Interactive mode
	fmt.Printf("Task: %s (state: %s)\n", taskID, kitdisplay.Cyan(currentState))
	fmt.Println()
	fmt.Println("Entering interactive mode. Type 'exit' or 'quit' to leave.")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		question := scanner.Text()
		if question == "" {
			continue
		}

		if question == "exit" || question == "quit" {
			break
		}

		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := askQuestion(question); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)

			continue
		}
		fmt.Println()
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	return nil
}

// joinArgs joins command line arguments into a single string.
func joinArgs(args []string) string {
	result := ""
	var resultSb182 strings.Builder
	for i, arg := range args {
		if i > 0 {
			resultSb182.WriteString(" ")
		}
		resultSb182.WriteString(arg)
	}
	result += resultSb182.String()

	return result
}
