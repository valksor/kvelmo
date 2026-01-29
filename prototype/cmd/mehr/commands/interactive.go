package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/spf13/cobra"

	"github.com/valksor/go-mehrhof/internal/agent"
	"github.com/valksor/go-mehrhof/internal/conductor"
	mehrhofdisplay "github.com/valksor/go-mehrhof/internal/display"
	"github.com/valksor/go-mehrhof/internal/events"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/workflow"
	"github.com/valksor/go-toolkit/display"
)

var (
	interactiveCmd = &cobra.Command{
		Use:     "interactive",
		Aliases: []string{"i", "repl"},
		Short:   "Enter interactive mode with real-time agent chat",
		Long: `Start a REPL session for continuous interaction with the AI agent.

Interactive mode provides a command-line interface for:
- Chatting directly with the AI agent
- Executing workflow commands (start, plan, implement, review)
- Real-time streaming of agent responses
- Pausing/resuming operations

Commands:
  chat <msg>      Chat with agent (aliases: ask, c)
  start <ref>     Start a new task
  plan [prompt]   Enter planning phase
  implement       Execute specifications (alias: impl)
  review          Review code
  continue        Resume from waiting/paused (alias: cont)
  status          Show task status (alias: st)
  answer <resp>   Answer agent's question (alias: a)
  undo            Undo to previous checkpoint
  redo            Redo to next checkpoint
  clear           Clear screen
  help            Show available commands (alias: ?)
  exit            Exit interactive mode (alias: quit, q)

Press Ctrl+C to stop the current operation.
Type 'exit' or 'quit' to leave interactive mode.`,
		RunE:    runInteractive,
		GroupID: "workflow",
	}

	// Interactive mode flags.
	interactiveNoHistory bool
)

func init() {
	rootCmd.AddCommand(interactiveCmd)

	interactiveCmd.Flags().BoolVar(&interactiveNoHistory, "no-history", false, "Disable command history")
}

// runInteractive starts the interactive REPL session.
func runInteractive(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Initialize conductor
	opts := BuildConductorOptions(CommandOptions{
		Verbose: verbose,
		Sandbox: sandbox,
	})
	cond, err := initializeConductor(ctx, opts...)
	if err != nil {
		return err
	}

	// Create interactive session
	session := newInteractiveSession(cond)
	if err := session.Initialize(ctx); err != nil {
		return err
	}

	// Run the REPL
	return session.Run(ctx)
}

// InteractiveSession manages an interactive REPL session.
type InteractiveSession struct {
	cond       *conductor.Conductor
	rl         *readline.Instance
	subID      string
	state      workflow.State
	history    []string
	transcript *strings.Builder
	sessionID  string
	cancelFunc context.CancelFunc
}

// newInteractiveSession creates a new interactive session.
func newInteractiveSession(cond *conductor.Conductor) *InteractiveSession {
	sessionID := time.Now().Format("20060102-150405")

	return &InteractiveSession{
		cond:       cond,
		sessionID:  sessionID,
		transcript: &strings.Builder{},
		state:      workflow.StateIdle,
	}
}

// Initialize sets up the interactive session.
func (s *InteractiveSession) Initialize(ctx context.Context) error {
	// Get current state
	if task := s.cond.GetActiveTask(); task != nil {
		s.state = workflow.State(task.State)
		s.printf(true, "Active task: %s\n", display.Bold(task.ID))
		s.printf(true, "State: %s\n", mehrhofdisplay.ColorState(string(s.state), string(s.state)))
	}

	// Setup readline
	historyFile := ""
	if !interactiveNoHistory {
		historyFile = filepath.Join(os.Getenv("HOME"), ".mehr_history")
	}

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          s.getPrompt(),
		HistoryFile:     historyFile,
		HistoryLimit:    1000,
		AutoComplete:    s.getCompleter(),
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		return fmt.Errorf("initialize readline: %w", err)
	}
	s.rl = rl

	// Subscribe to state change events for real-time updates
	s.subID = s.cond.GetEventBus().SubscribeAll(func(e events.Event) {
		s.handleEvent(e)
	})

	return nil
}

// Run starts the REPL loop.
func (s *InteractiveSession) Run(ctx context.Context) error {
	defer s.cleanup()

	// Setup signal handling for canceling operations
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	defer signal.Stop(sigCh)

	go func() {
		for range sigCh {
			// Handle Ctrl+C - cancel current operation but stay in REPL
			s.cancelCurrentOperation()
			s.printf(true, "\nOperation stopped. %s\n", display.Muted("Type 'exit' to quit."))
		}
	}()

	s.printf(true, "\n%s\n", display.Bold("Mehrhof Interactive Mode"))
	s.printf(true, "Type %s for help, %s to exit\n\n", display.Cyan("help"), display.Cyan("exit"))

	for {
		// Update prompt based on current state
		s.rl.Config.Prompt = s.getPrompt()

		line, err := s.rl.Readline()
		if err != nil {
			if errors.Is(err, readline.ErrInterrupt) {
				continue
			}
			if errors.Is(err, io.EOF) {
				s.printf(true, "\n")

				return nil
			}

			return fmt.Errorf("read input: %w", err)
		}

		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}

		// Add to history
		s.history = append(s.history, input)

		// Execute command
		if err := s.handleCommand(ctx, input); err != nil {
			if errors.Is(err, io.EOF) {
				s.printf(true, "\n")

				return nil
			}
			s.printf(false, "%s %s\n", display.ErrorMsg("Error:"), err)
		}
	}
}

// handleCommand processes a user command.
func (s *InteractiveSession) handleCommand(ctx context.Context, input string) error {
	// Parse command and arguments
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return nil
	}

	cmd := parts[0]
	args := parts[1:]

	// Handle aliases
	switch cmd {
	case "c", "ask":
		cmd = "chat"
	case "impl":
		cmd = "implement"
	case "cont":
		cmd = "continue"
	case "st":
		cmd = "status"
	case "a":
		cmd = "answer"
	case "?":
		cmd = "help"
	case "q", "quit":
		cmd = "exit"
	}

	// Execute command
	switch cmd {
	case "exit":
		return io.EOF // Signal to exit the REPL

	case "help":
		s.printHelp()

	case "chat":
		return s.handleChat(ctx, strings.Join(args, " "))

	case "start":
		return s.handleStart(ctx, args)

	case "plan":
		return s.handlePlan(ctx, strings.Join(args, " "))

	case "implement":
		return s.handleImplement(ctx)

	case "review":
		return s.handleReview(ctx)

	case "continue":
		return s.handleContinue(ctx)

	case "status":
		return s.handleStatus(ctx)

	case "answer":
		return s.handleAnswer(ctx, strings.Join(args, " "))

	case "undo":
		return s.handleUndo(ctx)

	case "redo":
		return s.handleRedo(ctx)

	case "clear":
		s.handleClear()

	default:
		// If no recognized command, treat as chat message
		return s.handleChat(ctx, input)
	}

	return nil
}

// printHelp displays available commands.
func (s *InteractiveSession) printHelp() {
	s.printf(true, "\n%s\n", display.Bold("Available Commands:"))

	s.printf(true, "\n%s\n", display.Bold("Chat:"))
	s.printf(true, "  chat <message>      Chat with the agent\n")
	s.printf(true, "  answer <response>   Answer agent's question\n")

	s.printf(true, "\n%s\n", display.Bold("Workflow:"))
	s.printf(true, "  start <reference>   Start a new task from reference\n")
	s.printf(true, "  plan [prompt]       Enter planning phase\n")
	s.printf(true, "  implement           Execute specifications\n")
	s.printf(true, "  review              Review code\n")
	s.printf(true, "  continue            Resume from waiting/paused\n")

	s.printf(true, "\n%s\n", display.Bold("Control:"))
	s.printf(true, "  status              Show task status\n")
	s.printf(true, "  undo                Undo to previous checkpoint\n")
	s.printf(true, "  redo                Redo to next checkpoint\n")

	s.printf(true, "\n%s\n", display.Bold("Session:"))
	s.printf(true, "  clear               Clear screen\n")
	s.printf(true, "  help                Show this help\n")
	s.printf(true, "  exit                Exit interactive mode\n")

	s.printf(true, "\n%s\n", display.Muted("Press Ctrl+C to stop current operation"))
	s.printf(true, "%s\n", display.Muted("Any unrecognized input will be sent to the agent as a chat message"))
	s.printf(true, "\n")
}

// handleChat sends a chat message to the agent.
func (s *InteractiveSession) handleChat(ctx context.Context, message string) error {
	if message == "" {
		return errors.New("message cannot be empty")
	}

	// Check if we have an active agent
	activeAgent := s.cond.GetActiveAgent()
	if activeAgent == nil {
		return errors.New("no agent available")
	}

	s.printf(true, "\n%s %s\n", display.Bold("You:"), message)
	s.printf(true, "%s\n", display.Bold("Agent:"))

	// Build prompt with context
	prompt := s.buildChatPrompt(message)

	// Run agent with streaming
	response, err := activeAgent.RunWithCallback(ctx, prompt, func(event agent.Event) error {
		return s.handleAgentEvent(event)
	})
	if err != nil {
		return fmt.Errorf("agent error: %w", err)
	}

	fmt.Println() // New line after response

	// Handle if agent asked a question
	if response != nil && response.Question != nil {
		return s.handleAgentQuestion(response.Question)
	}

	return nil
}

// handleStart starts a new task.
func (s *InteractiveSession) handleStart(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: start <reference>")
	}

	reference := args[0]

	s.printf(true, "Starting task from: %s\n", display.Cyan(reference))

	if err := s.cond.Start(ctx, reference); err != nil {
		return err
	}

	// Update state
	s.state = workflow.StateIdle
	s.printf(true, "%s Task started successfully\n", display.SuccessMsg("✓"))
	s.printf(true, "Next: Use %s to enter planning phase\n", display.Cyan("plan"))

	return nil
}

// handlePlan enters the planning phase.
func (s *InteractiveSession) handlePlan(ctx context.Context, _ string) error {
	if s.cond.GetActiveTask() == nil {
		return errors.New("no active task - use 'start <reference>' first")
	}

	s.printf(true, "Entering planning phase...\n")

	if err := s.cond.Plan(ctx); err != nil {
		return err
	}

	s.state = workflow.StatePlanning
	s.printf(true, "%s Planning phase started\n", display.SuccessMsg("✓"))

	return nil
}

// handleImplement enters the implementation phase.
func (s *InteractiveSession) handleImplement(ctx context.Context) error {
	if s.cond.GetActiveTask() == nil {
		return errors.New("no active task")
	}

	s.printf(true, "Entering implementation phase...\n")

	if err := s.cond.Implement(ctx); err != nil {
		return err
	}

	s.state = workflow.StateImplementing
	s.printf(true, "%s Implementation phase started\n", display.SuccessMsg("✓"))

	return nil
}

// handleReview enters the review phase.
func (s *InteractiveSession) handleReview(ctx context.Context) error {
	if s.cond.GetActiveTask() == nil {
		return errors.New("no active task")
	}

	s.printf(true, "Entering review phase...\n")

	if err := s.cond.Review(ctx); err != nil {
		return err
	}

	s.state = workflow.StateReviewing
	s.printf(true, "%s Review phase started\n", display.SuccessMsg("✓"))

	return nil
}

// handleContinue resumes from waiting state.
func (s *InteractiveSession) handleContinue(ctx context.Context) error {
	if s.cond.GetActiveTask() == nil {
		return errors.New("no active task")
	}

	s.printf(true, "Continuing...\n")

	// Check if there's a pending question
	task := s.cond.GetActiveTask()
	question, err := s.cond.GetWorkspace().LoadPendingQuestion(task.ID)
	if err == nil && question != nil {
		// Need to answer the question first
		return errors.New("agent has a pending question - use 'answer <response>'")
	}

	// Resume workflow
	if err := s.cond.ResumePaused(ctx); err != nil {
		return err
	}

	s.printf(true, "%s Resumed\n", display.SuccessMsg("✓"))

	return nil
}

// handleStatus displays task status.
func (s *InteractiveSession) handleStatus(ctx context.Context) error {
	status, err := s.cond.Status(ctx)
	if err != nil {
		return err
	}

	s.printf(true, "\n%s\n", display.Bold("Task Status:"))
	s.printf(true, "  ID:      %s\n", status.TaskID)
	s.printf(true, "  Title:   %s\n", status.Title)
	s.printf(true, "  State:   %s\n", mehrhofdisplay.ColorState(status.State, status.State))
	if status.Branch != "" {
		s.printf(true, "  Branch:  %s\n", display.Cyan(status.Branch))
	}
	s.printf(true, "  Specs:   %d\n", status.Specifications)
	s.printf(true, "  Checkpoints: %d\n", status.Checkpoints)
	s.printf(true, "\n")

	return nil
}

// handleAnswer responds to an agent question.
func (s *InteractiveSession) handleAnswer(ctx context.Context, response string) error {
	if response == "" {
		return errors.New("response cannot be empty")
	}

	task := s.cond.GetActiveTask()
	if task == nil {
		return errors.New("no active task")
	}

	s.printf(true, "Answering agent question...\n")

	// Clear the pending question
	if err := s.cond.GetWorkspace().ClearPendingQuestion(task.ID); err != nil {
		return fmt.Errorf("clear pending question: %w", err)
	}

	// Add answer as a note
	if err := s.cond.GetWorkspace().AppendNote(task.ID, string(s.state), response); err != nil {
		return fmt.Errorf("save answer: %w", err)
	}

	// Resume workflow based on current state
	switch s.state {
	case workflow.StatePlanning:
		if err := s.cond.Plan(ctx); err != nil {
			return err
		}
	case workflow.StateImplementing:
		if err := s.cond.Implement(ctx); err != nil {
			return err
		}
	case workflow.StateReviewing:
		if err := s.cond.Review(ctx); err != nil {
			return err
		}
	case workflow.StateIdle, workflow.StateDone, workflow.StateFailed,
		workflow.StateWaiting, workflow.StatePaused, workflow.StateCheckpointing,
		workflow.StateReverting, workflow.StateRestoring:
		// Cannot resume from these states
		return fmt.Errorf("cannot resume from state: %s", s.state)
	}

	s.printf(true, "%s Answer sent, resuming...\n", display.SuccessMsg("✓"))

	return nil
}

// handleUndo undoes to the previous checkpoint.
func (s *InteractiveSession) handleUndo(ctx context.Context) error {
	s.printf(true, "Undoing to previous checkpoint...\n")

	if err := s.cond.Undo(ctx); err != nil {
		return err
	}

	s.printf(true, "%s Undo complete\n", display.SuccessMsg("✓"))

	return nil
}

// handleRedo redoes to the next checkpoint.
func (s *InteractiveSession) handleRedo(ctx context.Context) error {
	s.printf(true, "Redoing to next checkpoint...\n")

	if err := s.cond.Redo(ctx); err != nil {
		return err
	}

	s.printf(true, "%s Redo complete\n", display.SuccessMsg("✓"))

	return nil
}

// handleClear clears the screen.
func (s *InteractiveSession) handleClear() {
	// ANSI escape code to clear screen
	fmt.Print("\033[H\033[2J")
}

// handleAgentEvent processes an agent streaming event.
func (s *InteractiveSession) handleAgentEvent(event agent.Event) error {
	switch event.Type {
	case agent.EventText:
		fmt.Print(event.Text)
		s.transcript.WriteString(event.Text)

	case agent.EventToolUse:
		if event.ToolCall != nil {
			s.printf(false, "\n→ %s\n", display.Muted(event.ToolCall.Name))
		}

	case agent.EventToolResult, agent.EventFile, agent.EventError, agent.EventUsage, agent.EventComplete:
		// Ignore other event types for display purposes
	}

	// Also publish to event bus for other listeners
	s.cond.GetEventBus().PublishRaw(events.Event{
		Type: events.TypeAgentMessage,
		Data: map[string]any{"event": event},
	})

	return nil
}

// handleAgentQuestion handles when the agent asks a question.
func (s *InteractiveSession) handleAgentQuestion(q *agent.Question) error {
	s.state = workflow.StateWaiting

	// Save the pending question
	task := s.cond.GetActiveTask()
	if task != nil {
		pendingQuestion := &storage.PendingQuestion{
			Question: q.Text,
		}
		for _, opt := range q.Options {
			pendingQuestion.Options = append(pendingQuestion.Options, storage.QuestionOption{
				Label:       opt.Label,
				Description: opt.Description,
			})
		}
		_ = s.cond.GetWorkspace().SavePendingQuestion(task.ID, pendingQuestion)
	}

	fmt.Println()
	s.printf(true, "\n%s\n", display.WarningMsg("Agent has a question:"))
	s.printf(true, "  %s\n\n", display.Bold(q.Text))

	if len(q.Options) > 0 {
		s.printf(true, "%s\n", display.Muted("Options:"))
		for i, opt := range q.Options {
			s.printf(true, "  %s %s", display.Info(fmt.Sprintf("%d.", i+1)), opt.Label)
			if opt.Description != "" {
				s.printf(true, " %s", display.Muted("- "+opt.Description))
			}
			fmt.Println()
		}
		fmt.Println()
	}

	s.printf(true, "%s\n", display.Muted("Answer with: answer <response> or answer <number>"))

	return nil
}

// handleEvent processes events from the event bus.
func (s *InteractiveSession) handleEvent(e events.Event) {
	switch e.Type {
	case events.TypeStateChanged:
		if to, ok := e.Data["to"].(string); ok {
			s.state = workflow.State(to)
			s.printf(true, "\n[%s]\n", mehrhofdisplay.ColorState(to, to))
		}

	case events.TypeProgress:
		if msg, ok := e.Data["message"].(string); ok {
			s.printf(false, "  %s\n", display.Muted(msg))
		}

	case events.TypeFileChanged:
		if path, ok := e.Data["path"].(string); ok {
			if op, ok := e.Data["operation"].(string); ok {
				s.printf(false, "  [%s] %s\n", display.Muted(op), display.Cyan(path))
			}
		}

	case events.TypeError:
		if errMsg, ok := e.Data["error"].(string); ok {
			s.printf(false, "%s %s\n", display.ErrorMsg("Error:"), errMsg)
		}
	}
}

// buildChatPrompt builds a prompt for chat with context.
func (s *InteractiveSession) buildChatPrompt(message string) string {
	var builder strings.Builder

	builder.WriteString("You are an AI assistant helping with a software development task.\n\n")

	// Add current task context
	task := s.cond.GetActiveTask()
	if task != nil {
		if work := s.cond.GetTaskWork(); work != nil {
			builder.WriteString(fmt.Sprintf("Task: %s\n", work.Metadata.Title))
			builder.WriteString(fmt.Sprintf("Current State: %s\n\n", s.state))
		}
	}

	builder.WriteString("User message: ")
	builder.WriteString(message)

	return builder.String()
}

// getPrompt returns the current prompt string.
func (s *InteractiveSession) getPrompt() string {
	stateStr := string(s.state)

	// Use ColorState from internal display package for consistent coloring
	return fmt.Sprintf("mehrhof (%s) > ", mehrhofdisplay.ColorState(stateStr, stateStr))
}

// getCompleter returns the readline completer.
func (s *InteractiveSession) getCompleter() *readline.PrefixCompleter {
	return readline.NewPrefixCompleter(
		readline.PcItem("chat"),
		readline.PcItem("ask"),
		readline.PcItem("start"),
		readline.PcItem("plan"),
		readline.PcItem("implement"),
		readline.PcItem("impl"),
		readline.PcItem("review"),
		readline.PcItem("continue"),
		readline.PcItem("cont"),
		readline.PcItem("status"),
		readline.PcItem("st"),
		readline.PcItem("answer"),
		readline.PcItem("a"),
		readline.PcItem("undo"),
		readline.PcItem("redo"),
		readline.PcItem("clear"),
		readline.PcItem("help"),
		readline.PcItem("?"),
		readline.PcItem("exit"),
		readline.PcItem("quit"),
		readline.PcItem("q"),
	)
}

// cancelCurrentOperation cancels the current operation.
func (s *InteractiveSession) cancelCurrentOperation() {
	if s.cancelFunc != nil {
		s.cancelFunc()
		s.cancelFunc = nil
	}
}

// cleanup cleans up resources.
func (s *InteractiveSession) cleanup() {
	if s.subID != "" {
		s.cond.GetEventBus().Unsubscribe(s.subID)
	}
	if s.rl != nil {
		_ = s.rl.Close() // Error from closing readline is not critical
	}
}

// printf prints formatted output.
func (s *InteractiveSession) printf(force bool, format string, args ...any) {
	w := os.Stdout
	if !force {
		w = os.Stderr
	}
	_, _ = fmt.Fprintf(w, format, args...) // Ignore write errors to stdout/stderr
}
