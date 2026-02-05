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
	"github.com/valksor/go-mehrhof/internal/events"
	"github.com/valksor/go-mehrhof/internal/library"
	"github.com/valksor/go-mehrhof/internal/memory"
	"github.com/valksor/go-mehrhof/internal/storage"
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
		result, err := routercommands.Execute(ctx, s.cond, cmd, args)
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

			// Render the result for CLI display
			s.renderResult(result)

			// Update local state if result includes state
			if result.State != "" {
				s.state = workflow.State(result.State)
			}
		}

		return nil
	}

	// Handle CLI-specific commands that aren't in the router
	// These have special behavior like streaming, interactive prompts, or complex subcommands
	switch cmd {
	case "chat":
		return s.handleChat(ctx, strings.Join(args, " "))

	case "find":
		return s.handleFind(ctx, args)

	case "simplify":
		return s.handleSimplify(ctx, args)

	case "label":
		return s.handleLabel(ctx, args)

	case "memory":
		return s.handleMemory(ctx, args)

	case "library":
		return s.handleLibrary(ctx, args)

	default:
		// If no recognized command, treat as a chat message
		return s.handleChat(ctx, input)
	}
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

	// Handle if the agent asked a question
	if response != nil && response.Question != nil {
		return s.handleAgentQuestion(response.Question)
	}

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

// handleSimplify simplifies code based on the current workflow The handleSimplify function optimizes code according to the current workflow status.
//
//nolint:unparam // args are kept for consistent signature with other handlers
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

// handleLibrary manages the documentation library.
func (s *InteractiveSession) handleLibrary(ctx context.Context, args []string) error {
	lib := s.cond.GetLibrary()
	if lib == nil {
		// Check if there was an initialization error
		if initErr := s.cond.GetLibraryError(); initErr != nil {
			return initErr
		}

		return errors.New("library system is not enabled. Enable in .mehrhof/config.yaml under 'library:'")
	}

	// Default to list if no subcommand
	subcommand := "list"
	if len(args) > 0 {
		subcommand = args[0]
		args = args[1:]
	}

	switch subcommand {
	case "list", "ls":
		return s.handleLibraryList(ctx, lib)
	case "show":
		if len(args) == 0 {
			return errors.New("usage: library show <name>")
		}

		return s.handleLibraryShow(ctx, lib, args[0])
	case "search":
		if len(args) == 0 {
			return errors.New("usage: library search <query>")
		}

		return s.handleLibrarySearch(ctx, lib, strings.Join(args, " "))
	default:
		// Treat unknown subcommand as collection name for show
		return s.handleLibraryShow(ctx, lib, subcommand)
	}
}

// handleLibraryList lists all library collections.
func (s *InteractiveSession) handleLibraryList(ctx context.Context, lib *library.Manager) error {
	collections, err := lib.List(ctx, &library.ListOptions{})
	if err != nil {
		return fmt.Errorf("list collections: %w", err)
	}

	if len(collections) == 0 {
		s.printf(true, "No library collections. Use 'mehr library pull <source>' to add documentation.\n")

		return nil
	}

	s.printf(true, "\n%s\n", display.Bold(fmt.Sprintf("%d Collection(s):", len(collections))))
	for _, c := range collections {
		mode := string(c.IncludeMode)
		location := c.Location
		s.printf(true, "  %s [%s, %s]\n", display.Cyan(c.Name), mode, location)
		s.printf(true, "    Source: %s\n", display.Muted(c.Source))
		s.printf(true, "    Pages: %d  Size: %s\n", c.PageCount, formatSize(c.TotalSize))
	}
	s.printf(true, "\n")

	return nil
}

// handleLibraryShow shows details of a collection.
func (s *InteractiveSession) handleLibraryShow(ctx context.Context, lib *library.Manager, name string) error {
	coll, err := lib.Show(ctx, name)
	if err != nil {
		return fmt.Errorf("show collection: %w", err)
	}

	s.printf(true, "\n%s\n", display.Bold("Collection: "+coll.Name))
	s.printf(true, "  ID:          %s\n", coll.ID)
	s.printf(true, "  Source:      %s\n", coll.Source)
	s.printf(true, "  Type:        %s\n", coll.SourceType)
	s.printf(true, "  Mode:        %s\n", coll.IncludeMode)
	s.printf(true, "  Location:    %s\n", coll.Location)
	s.printf(true, "  Pages:       %d\n", coll.PageCount)
	s.printf(true, "  Total Size:  %s\n", formatSize(coll.TotalSize))

	if len(coll.Paths) > 0 {
		s.printf(true, "  Paths:       %s\n", strings.Join(coll.Paths, ", "))
	}
	if len(coll.Tags) > 0 {
		s.printf(true, "  Tags:        %s\n", strings.Join(coll.Tags, ", "))
	}

	// List pages
	pages, err := lib.ListPages(ctx, coll.ID)
	if err == nil && len(pages) > 0 {
		s.printf(true, "\n%s\n", display.Bold("Pages:"))
		limit := 10
		for i, page := range pages {
			if i >= limit {
				s.printf(true, "  ... and %d more\n", len(pages)-limit)

				break
			}
			s.printf(true, "  - %s\n", page)
		}
	}
	s.printf(true, "\n")

	return nil
}

// handleLibrarySearch searches library documentation.
func (s *InteractiveSession) handleLibrarySearch(ctx context.Context, lib *library.Manager, query string) error {
	s.printf(true, "Searching library for: %s\n", display.Cyan(query))

	// Use the library context search
	docCtx, err := lib.GetDocsForQuery(ctx, query, 10000)
	if err != nil {
		return fmt.Errorf("search library: %w", err)
	}

	if docCtx == nil || len(docCtx.Pages) == 0 {
		s.printf(true, "No matching documentation found.\n")

		return nil
	}

	// Extract unique collection names from pages
	collectionSet := make(map[string]bool)
	for _, p := range docCtx.Pages {
		collectionSet[p.CollectionName] = true
	}
	var collNames []string
	for name := range collectionSet {
		collNames = append(collNames, name)
	}

	s.printf(true, "\n%s\n", display.Bold(fmt.Sprintf("Found %d page(s) from %d collection(s):", len(docCtx.Pages), len(collNames))))
	for _, name := range collNames {
		s.printf(true, "  - %s\n", display.Cyan(name))
	}

	// Show preview of first page
	if len(docCtx.Pages) > 0 {
		page := docCtx.Pages[0]
		s.printf(true, "\n%s\n", display.Bold("First match: "+page.Title))
		preview := page.Content
		if len(preview) > 500 {
			preview = preview[:500] + "\n... (truncated)"
		}
		s.printf(true, "%s\n", display.Muted(preview))
	}
	s.printf(true, "\n")

	return nil
}

// formatSize formats bytes as human-readable string.
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// capitalizeFirst capitalizes the first letter of a string.
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}

	return strings.ToUpper(s[:1]) + s[1:]
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

	// Also publish to eventbus for other listeners
	s.cond.GetEventBus().PublishRaw(eventbus.Event{
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

// renderResult formats and displays a router result for CLI output.
func (s *InteractiveSession) renderResult(result *routercommands.Result) {
	switch result.Type {
	case routercommands.ResultMessage:
		s.printf(true, "%s %s\n", display.SuccessMsg("✓"), result.Message)

	case routercommands.ResultStatus:
		if data, ok := result.Data.(routercommands.StatusData); ok {
			s.printf(true, "\n%s\n", display.Bold("Task Status:"))
			if data.TaskID != "" {
				s.printf(true, "  ID:      %s\n", data.TaskID)
			}
			if data.Title != "" {
				s.printf(true, "  Title:   %s\n", data.Title)
			}
			s.printf(true, "  State:   %s\n", mehrhofdisplay.ColorState(data.State, data.State))
			if data.Branch != "" {
				s.printf(true, "  Branch:  %s\n", display.Cyan(data.Branch))
			}
			if data.SpecCount > 0 {
				s.printf(true, "  Specs:   %d\n", data.SpecCount)
			}
			s.printf(true, "\n")
		} else {
			s.printf(true, "%s\n", result.Message)
		}

	case routercommands.ResultCost:
		if data, ok := result.Data.(routercommands.CostData); ok {
			s.printf(true, "\n%s\n", display.Bold("Cost Summary:"))
			s.printf(true, "  Input tokens:   %d\n", data.InputTokens)
			s.printf(true, "  Output tokens:  %d\n", data.OutputTokens)
			s.printf(true, "  Cached tokens:  %d\n", data.CachedTokens)
			s.printf(true, "  Total tokens:   %d\n", data.TotalTokens)
			s.printf(true, "  Total cost:     $%.4f\n", data.TotalCostUSD)
			s.printf(true, "\n")
		} else {
			s.printf(true, "%s\n", result.Message)
		}

	case routercommands.ResultBudget:
		if data, ok := result.Data.(routercommands.BudgetData); ok {
			s.printf(true, "\n%s\n", display.Bold("Budget Status:"))
			s.printf(true, "  Type:       %s\n", data.Type)
			s.printf(true, "  Used:       %s\n", data.Used)
			s.printf(true, "  Max:        %s\n", data.Max)
			s.printf(true, "  Percentage: %.1f%%\n", data.Percentage)
			if data.Warned {
				s.printf(true, "  %s\n", display.WarningMsg("Warning threshold reached"))
			}
			s.printf(true, "\n")
		} else {
			s.printf(true, "%s\n", result.Message)
		}

	case routercommands.ResultList:
		s.printf(true, "\n%s\n", display.Bold(result.Message))
		switch items := result.Data.(type) {
		case []routercommands.TaskListItem:
			for _, item := range items {
				prefix := "  "
				if result.TaskID != "" && item.ID == result.TaskID {
					prefix = "* "
				}
				title := item.Title
				if len(title) > 40 {
					title = title[:37] + "..."
				}
				s.printf(true, "%s%s [%s] %s\n",
					prefix, display.Cyan(item.ID),
					mehrhofdisplay.ColorState(item.State, item.State), title)
			}
		case []routercommands.SpecificationItem:
			for _, item := range items {
				icon := mehrhofdisplay.GetSpecificationStatusIcon(item.Status)
				title := item.Title
				if title == "" {
					title = "(untitled)"
				}
				if len(title) > 50 {
					title = title[:47] + "..."
				}
				s.printf(true, "  %s spec-%d: %s [%s]\n",
					icon, item.Number, title, item.Status)
			}
		default:
			// Generic list handling
			s.printf(true, "  %v\n", result.Data)
		}
		s.printf(true, "\n")

	case routercommands.ResultHelp:
		// Help output is handled by printHelp, but router can provide command info
		if cmds, ok := result.Data.([]routercommands.CommandInfo); ok {
			s.printf(true, "\n%s\n", display.Bold("Available Commands:"))
			categories := make(map[string][]routercommands.CommandInfo)
			for _, cmd := range cmds {
				categories[cmd.Category] = append(categories[cmd.Category], cmd)
			}
			for category, commands := range categories {
				s.printf(true, "\n%s\n", display.Bold(capitalizeFirst(category)+":"))
				for _, cmd := range commands {
					aliases := ""
					if len(cmd.Aliases) > 0 {
						aliases = " (aliases: " + strings.Join(cmd.Aliases, ", ") + ")"
					}
					s.printf(true, "  %-15s %s%s\n", cmd.Name, cmd.Description, display.Muted(aliases))
				}
			}
			s.printf(true, "\n")
		} else {
			s.printHelp()
		}

	case routercommands.ResultError:
		s.printf(false, "%s %s\n", display.ErrorMsg("Error:"), result.Message)

	case routercommands.ResultChat:
		// Chat responses display the message directly
		s.printf(true, "%s\n", result.Message)

	case routercommands.ResultSpecifications:
		// Specifications are handled as a list
		s.printf(true, "\n%s\n", display.Bold(result.Message))
		if items, ok := result.Data.([]routercommands.SpecificationItem); ok {
			for _, item := range items {
				icon := mehrhofdisplay.GetSpecificationStatusIcon(item.Status)
				title := item.Title
				if title == "" {
					title = "(untitled)"
				}
				if len(title) > 50 {
					title = title[:47] + "..."
				}
				s.printf(true, "  %s spec-%d: %s [%s]\n",
					icon, item.Number, title, item.Status)
			}
		}
		s.printf(true, "\n")

	case routercommands.ResultQuestion:
		// Question results require user input
		s.printf(true, "\n%s\n", display.WarningMsg("Question:"))
		s.printf(true, "  %s\n", result.Message)
		s.printf(true, "\n")

	case routercommands.ResultExit:
		// Exit is handled before renderResult is called, but include for exhaustiveness
		// Nothing to display
	}
}

// handleEvent processes events from the event bus.
func (s *InteractiveSession) handleEvent(e eventbus.Event) {
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

	// Add the current task context
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

	// Use ColorState from the internal display package for consistent coloring
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
