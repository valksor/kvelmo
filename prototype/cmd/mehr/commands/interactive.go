package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chzyer/readline"
	"github.com/spf13/cobra"

	"github.com/valksor/go-mehrhof/internal/agent"
	"github.com/valksor/go-mehrhof/internal/conductor"
	mehrhofdisplay "github.com/valksor/go-mehrhof/internal/display"
	"github.com/valksor/go-mehrhof/internal/events"
	"github.com/valksor/go-mehrhof/internal/memory"
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
- Executing workflow commands (start, plan, implement, review, finish)
- Real-time streaming of agent responses
- Pausing/resuming operations
- Managing tasks without exiting the REPL

Commands:
  chat <msg>      Chat with agent (aliases: ask, c)
  start <ref>     Start a new task
  plan [prompt]   Enter planning phase
  implement       Execute specifications (alias: impl)
  review          Review code
  continue        Resume from waiting/paused (alias: cont)
  finish          Complete the task
  abandon         Discard the task
  status          Show task status (alias: st)
  answer <resp>   Answer agent's question (alias: a)
  note <msg>      Add a note
  quick <desc>    Create a quick task
  cost            Show token usage
  budget          Show token budget status
  list            List all tasks
  specification <n>  View specification (alias: spec)
  find <query>    AI-powered code search
  simplify [files] Simplify code based on state
  label add|rm|set|list  Manage labels
  memory <query>  Search semantic memory
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
	cancelMu   sync.Mutex         // Protects cancelFunc from concurrent access
	cancelFunc context.CancelFunc // Cancel function for current operation
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

	// Create cancellable context for this command
	opCtx, cancel := context.WithCancel(ctx)

	// Store cancel func with mutex protection for signal handler access
	s.cancelMu.Lock()
	s.cancelFunc = cancel
	s.cancelMu.Unlock()

	// Clean up cancel func when done
	defer func() {
		s.cancelMu.Lock()
		s.cancelFunc = nil
		s.cancelMu.Unlock()
		cancel() // Always cancel to free resources
	}()

	// Execute command with cancellable context
	err := s.executeCommand(opCtx, cmd, args, input)

	// Handle cancellation gracefully - return nil to stay in REPL
	if errors.Is(err, context.Canceled) {
		return nil
	}

	return err
}

// executeCommand runs the actual command logic.
func (s *InteractiveSession) executeCommand(ctx context.Context, cmd string, args []string, input string) error {
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

	case "finish":
		return s.handleFinish(ctx)

	case "abandon":
		return s.handleAbandon(ctx)

	case "note":
		return s.handleNote(ctx, strings.Join(args, " "))

	case "quick":
		return s.handleQuick(ctx, strings.Join(args, " "))

	case "cost":
		return s.handleCost(ctx)

	case "list":
		return s.handleList(ctx)

	case "specification", "spec":
		return s.handleSpecification(ctx, args)

	case "find":
		return s.handleFind(ctx, args)

	case "simplify":
		return s.handleSimplify(ctx, args)

	case "label":
		return s.handleLabel(ctx, args)

	case "memory":
		return s.handleMemory(ctx, args)

	case "budget":
		return s.handleBudget(ctx)

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
	s.printf(true, "  chat <message>      Chat with the agent (aliases: ask, c)\n")
	s.printf(true, "  answer <response>   Answer agent's question (alias: a)\n")
	s.printf(true, "  note <message>      Add a note to the current task\n")

	s.printf(true, "\n%s\n", display.Bold("Workflow:"))
	s.printf(true, "  start <reference>   Start a new task from reference\n")
	s.printf(true, "  plan [prompt]       Enter planning phase\n")
	s.printf(true, "  implement           Execute specifications (alias: impl)\n")
	s.printf(true, "  review              Review code\n")
	s.printf(true, "  continue            Resume from waiting/paused (alias: cont)\n")
	s.printf(true, "  finish              Complete the task\n")
	s.printf(true, "  abandon             Discard the task\n")

	s.printf(true, "\n%s\n", display.Bold("Control:"))
	s.printf(true, "  status              Show task status (alias: st)\n")
	s.printf(true, "  undo                Undo to previous checkpoint\n")
	s.printf(true, "  redo                Redo to next checkpoint\n")

	s.printf(true, "\n%s\n", display.Bold("Search:"))
	s.printf(true, "  find <query>        AI-powered code search\n")
	s.printf(true, "  memory <query>      Search semantic memory\n")

	s.printf(true, "\n%s\n", display.Bold("Task:"))
	s.printf(true, "  simplify [files]    Simplify code based on state\n")
	s.printf(true, "  label add <lbl...>  Add labels\n")
	s.printf(true, "  label rm <lbl...>   Remove labels\n")
	s.printf(true, "  label set <lbl...>  Set labels (replace)\n")
	s.printf(true, "  label clear         Clear all labels\n")
	s.printf(true, "  label list          List labels\n")

	s.printf(true, "\n%s\n", display.Bold("Info:"))
	s.printf(true, "  cost                Show token usage and costs\n")
	s.printf(true, "  budget              Show token budget status\n")
	s.printf(true, "  list                List all tasks\n")
	s.printf(true, "  specification <n>  View specification (alias: spec)\n")
	s.printf(true, "  quick <desc>        Create a quick task\n")

	s.printf(true, "\n%s\n", display.Bold("Session:"))
	s.printf(true, "  clear               Clear screen\n")
	s.printf(true, "  help                Show this help (alias: ?)\n")
	s.printf(true, "  exit                Exit interactive mode (aliases: quit, q)\n")

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

// handleFinish completes the current task.
func (s *InteractiveSession) handleFinish(ctx context.Context) error {
	if s.cond.GetActiveTask() == nil {
		return errors.New("no active task")
	}

	// Show confirmation
	status, err := s.cond.Status(ctx)
	if err != nil {
		return fmt.Errorf("get status: %w", err)
	}

	s.printf(true, "\n%s\n", display.Bold("About to finish task:"))
	s.printf(true, "  ID:      %s\n", status.TaskID)
	if status.Title != "" {
		s.printf(true, "  Title:   %s\n", status.Title)
	}
	if status.Branch != "" {
		s.printf(true, "  Branch:  %s\n", display.Cyan(status.Branch))
	}
	s.printf(true, "  State:   %s\n", mehrhofdisplay.ColorState(status.State, status.State))
	s.printf(true, "\n%s\n", display.Muted("Press Enter to confirm, Ctrl+C to cancel"))

	// Simple confirmation
	line, _ := s.rl.Readline()
	if line != "" && !strings.HasPrefix(strings.ToLower(line), "y") {
		s.printf(true, "Cancelled\n")

		return nil
	}

	opts := conductor.FinishOptions{}

	if err := s.cond.Finish(ctx, opts); err != nil {
		return fmt.Errorf("finish: %w", err)
	}

	s.state = workflow.StateDone
	s.printf(true, "\n%s Task completed\n", display.SuccessMsg("✓"))

	return nil
}

// handleAbandon discards the current task.
func (s *InteractiveSession) handleAbandon(ctx context.Context) error {
	if s.cond.GetActiveTask() == nil {
		return errors.New("no active task")
	}

	status, err := s.cond.Status(ctx)
	if err != nil {
		return fmt.Errorf("get status: %w", err)
	}

	s.printf(true, "\n%s\n", display.WarningMsg("About to abandon task:"))
	s.printf(true, "  ID:      %s\n", status.TaskID)
	if status.Title != "" {
		s.printf(true, "  Title:   %s\n", status.Title)
	}
	if status.Branch != "" {
		s.printf(true, "  Branch:  %s\n", display.Cyan(status.Branch))
	}
	s.printf(true, "  State:   %s\n", mehrhofdisplay.ColorState(status.State, status.State))
	if status.Branch != "" {
		s.printf(true, "\n%s This will delete the branch and work directory!\n",
			display.ErrorMsg("WARNING:"))
	}
	s.printf(true, "\n%s\n", display.Muted("Type 'yes' to confirm, Ctrl+C to cancel"))

	line, _ := s.rl.Readline()
	if strings.ToLower(line) != "yes" {
		s.printf(true, "Cancelled\n")

		return nil
	}

	opts := conductor.DeleteOptions{
		Force:      true,
		KeepBranch: false,
		DeleteWork: conductor.BoolPtr(true),
	}

	if err := s.cond.Delete(ctx, opts); err != nil {
		return fmt.Errorf("abandon: %w", err)
	}

	s.state = workflow.StateIdle
	s.printf(true, "\n%s Task abandoned\n", display.SuccessMsg("✓"))

	return nil
}

// handleNote adds a note to the current task.
//
//nolint:unparam // ctx is kept for consistent signature with other handlers
func (s *InteractiveSession) handleNote(ctx context.Context, message string) error {
	if s.cond.GetActiveTask() == nil {
		return errors.New("no active task")
	}

	if message == "" {
		return errors.New("usage: note <message>")
	}

	task := s.cond.GetActiveTask()
	ws := s.cond.GetWorkspace()

	// Handle pending question
	if ws.HasPendingQuestion(task.ID) {
		q, _ := ws.LoadPendingQuestion(task.ID)
		note := fmt.Sprintf("**Q:** %s\n\n**A:** %s", q.Question, message)
		if err := ws.AppendNote(task.ID, note, "answer"); err != nil {
			return fmt.Errorf("save answer: %w", err)
		}
		_ = ws.ClearPendingQuestion(task.ID)
		s.printf(true, "%s Answer saved\n", display.SuccessMsg("✓"))

		return nil
	}

	if err := ws.AppendNote(task.ID, message, task.State); err != nil {
		return fmt.Errorf("save note: %w", err)
	}

	s.printf(true, "%s Note saved\n", display.SuccessMsg("✓"))

	return nil
}

// handleQuick creates a quick task.
func (s *InteractiveSession) handleQuick(ctx context.Context, description string) error {
	if description == "" {
		return errors.New("usage: quick <description>")
	}

	s.printf(true, "Creating quick task...\n")

	result, err := s.cond.CreateQuickTask(ctx, conductor.QuickTaskOptions{
		Description: description,
		QueueID:     "quick-tasks",
	})
	if err != nil {
		return fmt.Errorf("create quick task: %w", err)
	}

	s.printf(true, "%s Quick task created: %s\n",
		display.SuccessMsg("✓"), display.Cyan(result.TaskID))
	s.printf(true, "Queue: %s\n", result.QueueID)
	s.printf(true, "\nNext steps:\n")
	s.printf(true, "  start %s  - Start working on it\n", result.TaskID)

	return nil
}

// handleCost shows token usage and costs.
//
//nolint:unparam // ctx is kept for consistent signature with other handlers
func (s *InteractiveSession) handleCost(ctx context.Context) error {
	task := s.cond.GetActiveTask()
	if task == nil {
		return errors.New("no active task")
	}

	work := s.cond.GetTaskWork()
	if work == nil {
		return errors.New("unable to load task work")
	}

	costs := work.Costs

	s.printf(true, "\n%s\n", display.Bold("Cost Summary:"))
	s.printf(true, "  Input tokens:   %d\n", costs.TotalInputTokens)
	s.printf(true, "  Output tokens:  %d\n", costs.TotalOutputTokens)
	s.printf(true, "  Cached tokens:  %d\n", costs.TotalCachedTokens)
	s.printf(true, "  Total tokens:   %d\n", costs.TotalInputTokens+costs.TotalOutputTokens)
	s.printf(true, "  Total cost:     $%.4f\n", costs.TotalCostUSD)

	// Show by-step breakdown if available
	if len(costs.ByStep) > 0 {
		s.printf(true, "\n%s\n", display.Bold("By Step:"))
		for step, stepCost := range costs.ByStep {
			s.printf(true, "  %s: %d tokens ($%.4f)\n",
				step, stepCost.InputTokens+stepCost.OutputTokens, stepCost.CostUSD)
		}
	}
	s.printf(true, "\n")

	return nil
}

// handleList lists all tasks in the workspace.
//
//nolint:unparam // ctx is kept for consistent signature with other handlers
func (s *InteractiveSession) handleList(ctx context.Context) error {
	ws := s.cond.GetWorkspace()
	taskIDs, err := ws.ListWorks()
	if err != nil {
		return fmt.Errorf("list tasks: %w", err)
	}

	if len(taskIDs) == 0 {
		s.printf(true, "No tasks in workspace\n")

		return nil
	}

	s.printf(true, "\n%s\n", display.Bold("Tasks:"))

	// Get active task ID for highlighting
	activeID := ""
	var activeTask *storage.ActiveTask
	if ws.HasActiveTask() {
		activeTask, _ = ws.LoadActiveTask()
		if activeTask != nil {
			activeID = activeTask.ID
		}
	}

	for _, id := range taskIDs {
		work, _ := ws.LoadWork(id)
		prefix := "  "
		if id == activeID {
			prefix = "* "
		}
		title := "(no title)"
		if work != nil && work.Metadata.Title != "" {
			title = work.Metadata.Title
		}
		if len(title) > 40 {
			title = title[:37] + "..."
		}
		state := "idle"
		if id == activeID && activeTask != nil {
			state = mehrhofdisplay.FormatStateString(activeTask.State)
		}
		s.printf(true, "%s%s [%s] %s\n",
			prefix, display.Cyan(id), mehrhofdisplay.ColorState(state, state), title)
	}
	s.printf(true, "\n")

	return nil
}

// handleSpecification views or lists specifications.
//
//nolint:unparam // ctx is kept for consistent signature with other handlers
func (s *InteractiveSession) handleSpecification(ctx context.Context, args []string) error {
	task := s.cond.GetActiveTask()
	if task == nil {
		return errors.New("no active task")
	}

	ws := s.cond.GetWorkspace()

	// If no args, list all specs
	if len(args) == 0 {
		specs, err := ws.ListSpecificationsWithStatus(task.ID)
		if err != nil {
			return fmt.Errorf("list specifications: %w", err)
		}

		if len(specs) == 0 {
			s.printf(true, "\nNo specifications yet. Use 'plan' to create them.\n\n")

			return nil
		}

		s.printf(true, "\n%s\n", display.Bold("Specifications:"))
		for _, spec := range specs {
			icon := mehrhofdisplay.GetSpecificationStatusIcon(spec.Status)
			title := spec.Title
			if title == "" {
				title = "(untitled)"
			}
			if len(title) > 50 {
				title = title[:47] + "..."
			}
			s.printf(true, "  %s spec-%d: %s [%s]\n",
				icon, spec.Number, title, spec.Status)
		}
		s.printf(true, "\n")
		s.printf(true, "Use 'specification <number>' to view a specific specification\n\n")

		return nil
	}

	// View specific spec
	num, err := strconv.Atoi(args[0])
	if err != nil {
		return errors.New("usage: specification <number>")
	}

	content, err := ws.LoadSpecification(task.ID, num)
	if err != nil {
		return fmt.Errorf("load specification: %w", err)
	}

	// Also load the spec metadata for title/status
	specs, _ := ws.ListSpecificationsWithStatus(task.ID)
	var title, status string
	for _, spec := range specs {
		if spec.Number == num {
			title = spec.Title
			status = spec.Status

			break
		}
	}

	s.printf(true, "\n%s\n", display.Bold(fmt.Sprintf("Specification %d:", num)))
	if title != "" {
		s.printf(true, "Title: %s\n", title)
	}
	if status != "" {
		s.printf(true, "Status: %s\n", status)
	}
	s.printf(true, "\n%s\n\n", content)

	return nil
}

// handleFind performs AI-powered code search.
func (s *InteractiveSession) handleFind(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: find <query>")
	}

	query := strings.Join(args, " ")
	s.printf(true, "Searching for: %s\n", display.Cyan(query))

	findOpts := conductor.FindOptions{
		Query:     query,
		Path:      "",
		Pattern:   "",
		Context:   3,
		Workspace: s.cond.GetWorkspace(),
	}

	resultChan, err := s.cond.Find(ctx, findOpts)
	if err != nil {
		return fmt.Errorf("find: %w", err)
	}

	var results []conductor.FindResult
	for result := range resultChan {
		if result.File == "__error__" {
			return fmt.Errorf("search error: %s", result.Snippet)
		}
		results = append(results, result)
	}

	if len(results) == 0 {
		s.printf(true, "No matches found.\n")

		return nil
	}

	s.printf(true, "\n%s\n", display.Bold(fmt.Sprintf("Found %d match(es):", len(results))))
	for i, r := range results {
		s.printf(true, "%d. %s:%d\n", i+1, r.File, r.Line)
		if r.Snippet != "" {
			s.printf(true, "   %s\n", display.Muted(r.Snippet))
		}
		if r.Reason != "" {
			s.printf(true, "   %s %s\n", display.Cyan("->"), r.Reason)
		}
	}
	s.printf(true, "\n")

	return nil
}

// handleSimplify simplifies code based on current workflow state.
//
//nolint:unparam // args is kept for consistent signature with other handlers
func (s *InteractiveSession) handleSimplify(ctx context.Context, args []string) error {
	if s.cond.GetActiveTask() == nil {
		return errors.New("no active task")
	}

	s.printf(true, "Simplifying...\n")

	if err := s.cond.Simplify(ctx, "", true); err != nil {
		return fmt.Errorf("simplify: %w", err)
	}

	s.printf(true, "%s Simplification complete\n", display.SuccessMsg("OK"))

	return nil
}

// handleLabel manages task labels.
func (s *InteractiveSession) handleLabel(ctx context.Context, args []string) error {
	if len(args) == 0 {
		s.listLabels(ctx)

		return nil
	}

	subcommand := args[0]
	subArgs := args[1:]

	taskID := ""
	if s.cond.GetActiveTask() != nil {
		taskID = s.cond.GetActiveTask().ID
	}

	switch subcommand {
	case "add":
		return s.handleLabelAdd(ctx, taskID, subArgs)
	case "remove", "rm":
		return s.handleLabelRemove(ctx, taskID, subArgs)
	case "set":
		return s.handleLabelSet(ctx, taskID, subArgs)
	case "clear":
		return s.handleLabelSet(ctx, taskID, []string{})
	case "list", "ls":
		s.listLabels(ctx)

		return nil
	default:
		return s.handleLabelAdd(ctx, taskID, args)
	}
}

// handleLabelAdd adds labels to the active task.
//
//nolint:unparam // ctx is kept for consistent signature with other handlers
func (s *InteractiveSession) handleLabelAdd(ctx context.Context, taskID string, labels []string) error {
	if taskID == "" {
		return errors.New("no active task")
	}
	if len(labels) == 0 {
		return errors.New("usage: label add <label...>")
	}
	ws := s.cond.GetWorkspace()
	for _, label := range labels {
		if err := ws.AddLabel(taskID, label); err != nil {
			return fmt.Errorf("add label %q: %w", label, err)
		}
	}
	s.printf(true, "%s Added %d label(s)\n", display.SuccessMsg("OK"), len(labels))

	return nil
}

// handleLabelRemove removes labels from the active task.
//
//nolint:unparam // ctx is kept for consistent signature with other handlers
func (s *InteractiveSession) handleLabelRemove(ctx context.Context, taskID string, labels []string) error {
	if taskID == "" {
		return errors.New("no active task")
	}
	if len(labels) == 0 {
		return errors.New("usage: label remove <label...>")
	}
	ws := s.cond.GetWorkspace()
	for _, label := range labels {
		if err := ws.RemoveLabel(taskID, label); err != nil {
			return fmt.Errorf("remove label %q: %w", label, err)
		}
	}
	s.printf(true, "%s Removed %d label(s)\n", display.SuccessMsg("OK"), len(labels))

	return nil
}

// handleLabelSet sets labels on the active task.
//
//nolint:unparam // ctx is kept for consistent signature with other handlers
func (s *InteractiveSession) handleLabelSet(ctx context.Context, taskID string, labels []string) error {
	if taskID == "" {
		return errors.New("no active task")
	}
	ws := s.cond.GetWorkspace()
	if err := ws.SetLabels(taskID, labels); err != nil {
		return fmt.Errorf("set labels: %w", err)
	}
	if len(labels) == 0 {
		s.printf(true, "%s Cleared all labels\n", display.SuccessMsg("OK"))
	} else {
		s.printf(true, "%s Set %d label(s)\n", display.SuccessMsg("OK"), len(labels))
	}

	return nil
}

// listLabels lists labels for the active task.
func (s *InteractiveSession) listLabels(context.Context) {
	task := s.cond.GetActiveTask()
	if task == nil {
		s.printf(true, "No active task\n")

		return
	}
	ws := s.cond.GetWorkspace()
	labels, err := ws.GetLabels(task.ID)
	if err != nil {
		s.printf(false, "%s %v\n", display.ErrorMsg("Error:"), err)

		return
	}
	s.printf(true, "\n%s\n", display.Bold("Labels:"))
	if len(labels) == 0 {
		s.printf(true, "  (no labels)\n")

		return
	}
	for _, label := range labels {
		s.printf(true, "  - %s\n", display.Cyan(label))
	}
	s.printf(true, "\n")
}

// handleMemory searches semantic memory.
func (s *InteractiveSession) handleMemory(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: memory <query>")
	}

	mem := s.cond.GetMemory()
	if mem == nil {
		return errors.New("memory system is not enabled")
	}

	query := strings.Join(args, " ")

	s.printf(true, "Searching memory for: %s\n", display.Cyan(query))

	results, err := mem.Search(ctx, query, memory.SearchOptions{
		Limit:    5,
		MinScore: 0.65,
	})
	if err != nil {
		return fmt.Errorf("memory search: %w", err)
	}

	if len(results) == 0 {
		s.printf(true, "No similar tasks found.\n")

		return nil
	}

	s.printf(true, "\n%s\n", display.Bold(fmt.Sprintf("Found %d similar task(s):", len(results))))
	for i, result := range results {
		doc := result.Document
		s.printf(true, "%d. Task %s (%.0f%% similar)\n", i+1, doc.TaskID, result.Score*100)
		s.printf(true, "   Type: %s\n", doc.Type)
		preview := doc.Content
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		s.printf(true, "   %s\n\n", display.Muted(preview))
	}

	return nil
}

// handleBudget displays budget status.
func (s *InteractiveSession) handleBudget(context.Context) error {
	task := s.cond.GetActiveTask()
	if task == nil {
		return errors.New("no active task")
	}

	ws := s.cond.GetWorkspace()
	work, err := ws.LoadWork(task.ID)
	if err != nil {
		return fmt.Errorf("load task: %w", err)
	}

	cfg, err := ws.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	taskBudget := cfg.Budget.PerTask
	if work.Budget != nil {
		taskBudget = *work.Budget
	}

	costs := work.Costs
	totalTokens := costs.TotalInputTokens + costs.TotalOutputTokens

	s.printf(true, "\n%s\n", display.Bold("Budget Status:"))
	s.printf(true, "  Task:    %s\n", task.ID)
	s.printf(true, "  Tokens:  %d / %s\n",
		totalTokens,
		formatLimit(taskBudget.MaxTokens))
	s.printf(true, "  Cost:    %s / %s\n",
		formatCost(costs.TotalCostUSD),
		formatCost(taskBudget.MaxCost))
	s.printf(true, "\n")

	return nil
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
		readline.PcItem("finish"),
		readline.PcItem("abandon"),
		readline.PcItem("note"),
		readline.PcItem("quick"),
		readline.PcItem("cost"),
		readline.PcItem("list"),
		readline.PcItem("specification"),
		readline.PcItem("spec"),
		readline.PcItem("find"),
		readline.PcItem("simplify"),
		readline.PcItem("label"),
		readline.PcItem("memory"),
		readline.PcItem("budget"),
	)
}

// cancelCurrentOperation cancels the current operation.
func (s *InteractiveSession) cancelCurrentOperation() {
	s.cancelMu.Lock()
	defer s.cancelMu.Unlock()
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
