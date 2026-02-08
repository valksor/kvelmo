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
	"sync"
	"time"

	"github.com/chzyer/readline"
	"github.com/spf13/cobra"

	"github.com/valksor/go-mehrhof/internal/agent"
	"github.com/valksor/go-mehrhof/internal/conductor"
	routercommands "github.com/valksor/go-mehrhof/internal/conductor/commands"
	mehrhofdisplay "github.com/valksor/go-mehrhof/internal/display"
	"github.com/valksor/go-mehrhof/internal/workflow"
	"github.com/valksor/go-toolkit/display"
	"github.com/valksor/go-toolkit/eventbus"
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
  library [cmd]   Manage documentation library
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

	// Create an interactive session
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
	cancelFunc context.CancelFunc // Cancel function for the current operation
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
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			historyFile = filepath.Join(home, ".mehr_history")
		}
		// If home dir unavailable, historyFile stays empty → history disabled gracefully
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
	s.subID = s.cond.GetEventBus().SubscribeAll(func(e eventbus.Event) {
		s.handleEvent(e)
	})

	return nil
}

// Run starts the REPL loop.
func (s *InteractiveSession) Run(ctx context.Context) error {
	defer s.cleanup()

	// Set up signal handling for canceling operations
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	defer signal.Stop(sigCh)

	go func() {
		for range sigCh {
			// Handle Ctrl+C - cancel the current operation but stay in REPL
			s.cancelCurrentOperation()
			s.printf(true, "\nOperation stopped. %s\n", display.Muted("Type 'exit' to quit."))
		}
	}()

	s.printf(true, "\n%s\n", display.Bold("Mehrhof Interactive Mode"))
	s.printf(true, "Type %s for help, %s to exit\n\n", display.Cyan("help"), display.Cyan("exit"))

	for {
		// Update prompt based on the current state
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

	// Create a cancellable context for this command
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
	// First, check if the command is handled by the unified router
	if routercommands.IsKnownCommand(cmd) {
		// Build invocation with streaming callback for chat commands
		inv := routercommands.Invocation{
			Args:   args,
			Source: routercommands.SourceREPL,
		}

		// Chat commands need streaming callback for real-time output
		if cmd == "chat" || cmd == "c" || cmd == "ask" {
			s.printf(true, "\n%s %s\n", display.Bold("You:"), strings.Join(args, " "))
			s.printf(true, "%s\n", display.Bold("Agent:"))
			inv.StreamCB = func(event agent.Event) error {
				return s.handleAgentEvent(event)
			}
		}

		result, err := routercommands.Execute(ctx, s.cond, cmd, inv)
		if err != nil {
			// Check for specific error types
			if errors.Is(err, routercommands.ErrNoActiveTask) {
				return errors.New("no active task - use 'start <reference>' first")
			}

			return err
		}

		// Handle special result types
		if result != nil {
			// Handle exit signal
			if result.Type == routercommands.ResultExit {
				return io.EOF
			}

			// Render the result for CLI display (skip for chat, already streamed)
			if cmd != "chat" && cmd != "c" && cmd != "ask" {
				s.renderResult(result)
			} else {
				fmt.Println() // New line after streamed response
			}

			// Update local state if result includes state
			if result.State != "" {
				s.state = workflow.State(result.State)
			}
		}

		return nil
	}

	// Unknown commands are treated as chat messages - route through router
	return s.executeCommand(ctx, "chat", []string{input}, input)
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
	s.printf(true, "  implement review <n> Fix issues from review\n")
	s.printf(true, "  review              Run code review\n")
	s.printf(true, "  review <n>          View review content\n")
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
	s.printf(true, "  library [cmd]       Manage documentation library\n")

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
	s.printf(true, "  specification <n>   View specification (alias: spec)\n")
	s.printf(true, "  quick <desc>        Create a quick task\n")

	s.printf(true, "\n%s\n", display.Bold("Session:"))
	s.printf(true, "  clear               Clear screen\n")
	s.printf(true, "  help                Show this help (alias: ?)\n")
	s.printf(true, "  exit                Exit interactive mode (aliases: quit, q)\n")

	s.printf(true, "\n%s\n", display.Muted("Press Ctrl+C to stop current operation"))
	s.printf(true, "%s\n", display.Muted("Any unrecognized input will be sent to the agent as a chat message"))
	s.printf(true, "\n")
}

// getPrompt returns the current prompt string.
func (s *InteractiveSession) getPrompt() string {
	stateStr := string(s.state)

	// Use ColorState from the internal display package for consistent coloring
	return fmt.Sprintf("mehrhof (%s) > ", mehrhofdisplay.ColorState(stateStr, stateStr))
}

// getCompleter returns the readline completer.
func (s *InteractiveSession) getCompleter() *readline.PrefixCompleter {
	return readline.NewPrefixCompleter(
		readline.PcItem("chat"),
		readline.PcItem("c"),
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
		readline.PcItem("library",
			readline.PcItem("list"),
			readline.PcItem("show"),
			readline.PcItem("search"),
		),
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
