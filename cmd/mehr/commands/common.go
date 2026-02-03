package commands

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/valksor/go-mehrhof/internal/agent"
	"github.com/valksor/go-mehrhof/internal/conductor"
	mehrhofdisplay "github.com/valksor/go-mehrhof/internal/display"
	"github.com/valksor/go-mehrhof/internal/events"
	"github.com/valksor/go-mehrhof/internal/registration"
	"github.com/valksor/go-mehrhof/internal/vcs"
	"github.com/valksor/go-toolkit/display"
	"github.com/valksor/go-toolkit/eventbus"
	"github.com/valksor/go-toolkit/output"
)

var (
	dedupStdout sync.Once
	dedupWriter *output.DeduplicatingWriter
)

// getDeduplicatingStdout returns a deduplicating writer that wraps os.Stdout.
// The writer suppresses consecutive identical lines.
// Uses sync.Once to ensure thread-safe initialization.
func getDeduplicatingStdout() io.Writer {
	dedupStdout.Do(func() {
		dedupWriter = output.NewDeduplicatingWriter(os.Stdout)
	})

	return dedupWriter
}

// initializeConductor creates and initializes a conductor with the standard
// providers (file, directory) and agents (claude) registered.
//
// This is the common initialization sequence used by most commands.
// The caller should build options to customize behavior per command.
func initializeConductor(ctx context.Context, opts ...conductor.Option) (*conductor.Conductor, error) {
	// Create a conductor with provided options
	cond, err := conductor.New(opts...)
	if err != nil {
		return nil, errors.New(mehrhofdisplay.ConductorError("create", err))
	}

	// Register standard providers
	registration.RegisterStandardProviders(cond)

	// Register standard agents
	if err := registration.RegisterStandardAgents(cond); err != nil {
		return nil, errors.New(mehrhofdisplay.ConductorError("register", err))
	}

	// Initialize the conductor (loads workspace, detects agent, etc.)
	if err := cond.Initialize(ctx); err != nil {
		return nil, errors.New(mehrhofdisplay.ConductorError("initialize", err))
	}

	return cond, nil
}

// confirmAction prompts the user for confirmation unless skipConfirm is true.
// Returns true if the action should proceed, false if canceled.
// The prompt parameter should describe what will happen (e.g., "delete this task").
func confirmAction(prompt string, skipConfirm bool) (bool, error) {
	if skipConfirm {
		return true, nil
	}

	fmt.Printf("%s\nAre you sure? [y/N]: ", prompt)

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("read response: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))

	return response == "y" || response == "yes", nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Helper functions for reducing command duplication
// ─────────────────────────────────────────────────────────────────────────────

// CommandOptions holds common options for command execution.
type CommandOptions struct {
	Verbose         bool
	Quiet           bool
	DryRun          bool
	StepAgent       string // Per-step agent override (e.g., "planning", "implementing")
	FullContext     bool
	OptimizePrompts bool // Optimize prompts before sending to agents
	Sandbox         bool // Enable sandboxing for agent execution
	LibraryInclude  bool // Auto-include library documentation
}

// IsQuiet returns true if quiet mode is enabled.
func IsQuiet() bool {
	return quiet
}

// IsSandbox returns true if sandbox mode is enabled.
func IsSandbox() bool {
	return sandbox
}

// BuildConductorOptions creates conductor options from command options.
// This centralizes the common pattern of building options.
func BuildConductorOptions(cmdOpts CommandOptions) []conductor.Option {
	opts := []conductor.Option{
		conductor.WithVerbose(cmdOpts.Verbose),
	}

	if cmdOpts.DryRun {
		opts = append(opts, conductor.WithDryRun(true))
	}

	if cmdOpts.FullContext {
		opts = append(opts, conductor.WithIncludeFullContext(true))
	}

	if cmdOpts.OptimizePrompts {
		opts = append(opts, conductor.WithOptimizePrompts(true))
	}

	if cmdOpts.Sandbox {
		opts = append(opts, conductor.WithSandbox(true))
	}

	if cmdOpts.LibraryInclude {
		opts = append(opts, conductor.WithLibraryAutoInclude(true))
	}

	if cmdOpts.StepAgent != "" {
		// Derive the step name from the step agent variable name
		// e.g., "planAgentPlanning" -> "planning"
		stepName := deriveStepName(cmdOpts.StepAgent)
		if stepName != "" {
			opts = append(opts, conductor.WithStepAgent(stepName, cmdOpts.StepAgent))
		}
	}

	// Use deduplicating stdout in verbose mode
	if cmdOpts.Verbose {
		opts = append(opts, conductor.WithStdout(getDeduplicatingStdout()))
	}

	return opts
}

// deriveStepName converts a step agent variable suffix to a step name.
// For example, "planning" -> "planning", "implementing" -> "implementing"
// This is used when the agent name is passed directly without a step prefix.
func deriveStepName(agentVar string) string {
	// Map common variable names to step names
	stepMap := map[string]string{
		"planning":      "planning",
		"implementing":  "implementing",
		"implement":     "implementing",
		"reviewing":     "reviewing",
		"review":        "reviewing",
		"checkpointing": "checkpointing",
	}

	if step, ok := stepMap[strings.ToLower(agentVar)]; ok {
		return step
	}

	return ""
}

// RequireActiveTask checks for an active task and prints an error if none exists.
// Returns true if an active task exists, false otherwise.
func RequireActiveTask(cond *conductor.Conductor) bool {
	if cond.GetActiveTask() == nil {
		fmt.Print(mehrhofdisplay.NoActiveTaskError())

		return false
	}

	return true
}

// SetupVerboseEventHandlers subscribes to common events for verbose output.
// This centralizes the verbose event subscription pattern.
// Quiet mode suppresses progress and file change events but keeps errors.
func SetupVerboseEventHandlers(cond *conductor.Conductor) {
	w := cond.GetStdout()
	cond.GetEventBus().SubscribeAll(func(e eventbus.Event) {
		// Suppress non-essential output in quiet mode
		if IsQuiet() {
			switch e.Type {
			case events.TypeProgress, events.TypeFileChanged, events.TypeCheckpoint, events.TypeBrowserAction, events.TypeBrowserTabOpened, events.TypeBrowserScreenshot:
				return
			case events.TypeStateChanged, events.TypeError, events.TypeAgentMessage, events.TypeBlueprintReady, events.TypeBranchCreated, events.TypePlanCompleted, events.TypeImplementDone, events.TypePRCreated:
				// Let other events through
			}
		}

		switch e.Type {
		case events.TypeProgress:
			if msg, ok := e.Data["message"].(string); ok {
				_, err := fmt.Fprintf(w, "  %s\n", msg)
				if err != nil {
					slog.Debug("write progress", "error", err)
				}
			}
		case events.TypeFileChanged:
			if path, ok := e.Data["path"].(string); ok {
				op, _ := e.Data["operation"].(string)
				_, err := fmt.Fprintf(w, "  [%s] %s\n", op, path)
				if err != nil {
					slog.Debug("write file change", "error", err)
				}
			}
		case events.TypeCheckpoint:
			if num, ok := e.Data["checkpoint"].(int); ok {
				_, err := fmt.Fprintf(w, "  Checkpoint #%d created\n", num)
				if err != nil {
					slog.Debug("write checkpoint", "error", err)
				}
			}
		case events.TypeAgentMessage:
			if agentEvent, ok := e.Data["event"].(agent.Event); ok {
				printAgentEventTo(w, agentEvent)
			}
		case events.TypeBrowserAction:
			action, _ := e.Data["action"].(string)
			tabID, _ := e.Data["tab_id"].(string)
			if selector, ok := e.Data["selector"].(string); ok {
				_, err := fmt.Fprintf(w, "  🌐 Browser [%s]: %s on %s\n", tabID, action, selector)
				if err != nil {
					slog.Debug("write browser action", "error", err)
				}
			} else {
				_, err := fmt.Fprintf(w, "  🌐 Browser [%s]: %s\n", tabID, action)
				if err != nil {
					slog.Debug("write browser action", "error", err)
				}
			}
		case events.TypeBrowserTabOpened:
			tabID, _ := e.Data["tab_id"].(string)
			url, _ := e.Data["url"].(string)
			title, _ := e.Data["title"].(string)
			if title != "" {
				_, err := fmt.Fprintf(w, "  🌐 Browser tab opened: %s - %s [%s]\n", title, url, tabID)
				if err != nil {
					slog.Debug("write browser tab opened", "error", err)
				}
			} else {
				_, err := fmt.Fprintf(w, "  🌐 Browser tab opened: %s [%s]\n", url, tabID)
				if err != nil {
					slog.Debug("write browser tab opened", "error", err)
				}
			}
		case events.TypeBrowserScreenshot:
			tabID, _ := e.Data["tab_id"].(string)
			format, _ := e.Data["format"].(string)
			size, _ := e.Data["size"].(int)
			_, err := fmt.Fprintf(w, "  📸 Screenshot captured [%s]: %s (%d bytes)\n", tabID, format, size)
			if err != nil {
				slog.Debug("write browser screenshot", "error", err)
			}
		case events.TypeStateChanged, events.TypeError, events.TypeBlueprintReady, events.TypeBranchCreated, events.TypePlanCompleted, events.TypeImplementDone, events.TypePRCreated:
			// Ignore other event types
		}
	})
}

// PrintNextSteps prints common next steps after a command completes.
// Respects quiet mode - suppresses output if enabled.
func PrintNextSteps(steps ...string) {
	if IsQuiet() {
		return
	}
	fmt.Println()
	fmt.Println(display.Muted("Next steps:"))
	for _, step := range steps {
		fmt.Printf("  %s\n", display.Cyan(step))
	}
	fmt.Println()
}

// WorkspaceResolution holds the result of resolving the workspace root and git context.
type WorkspaceResolution struct {
	Root       string   // Workspace root directory (main repo path if in worktree)
	Git        *vcs.Git // Git instance (nil if not in a git repository)
	IsWorktree bool     // True if currently in a git worktree
}

// ResolveWorkspaceRoot resolves the workspace root directory and git context.
// This function centralizes the common pattern of finding the workspace root
// while handling git worktrees correctly.
//
// If in a git worktree, it returns the main repository path as the root.
// If not in git, it returns the current working directory.
// The git instance is only non-nil if successfully created.
func ResolveWorkspaceRoot(ctx context.Context) (WorkspaceResolution, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return WorkspaceResolution{}, fmt.Errorf("get working directory: %w", err)
	}

	git, err := vcs.New(ctx, cwd)
	if err != nil {
		// Not in a git repository, use cwd as root
		//nolint:nilerr // Intentional: non-git repos return valid resolution without error
		return WorkspaceResolution{
			Root: cwd,
			Git:  nil,
		}, nil
	}

	// In a git repository - check if we're in a worktree
	if git.IsWorktree() {
		mainRepo, err := git.GetMainWorktreePath(ctx)
		if err != nil {
			return WorkspaceResolution{}, fmt.Errorf("get main repo from worktree: %w", err)
		}

		return WorkspaceResolution{
			Root:       mainRepo,
			Git:        git,
			IsWorktree: true,
		}, nil
	}

	// In the main git repository
	return WorkspaceResolution{
		Root: git.Root(),
		Git:  git,
	}, nil
}

// printAgentEventTo prints meaningful content from agent events to the specified writer.
// This is exported so it can be used by other commands.
func printAgentEventTo(w io.Writer, e agent.Event) {
	// Print text content if available
	if e.Text != "" {
		_, err := fmt.Fprint(w, e.Text)
		if err != nil {
			slog.Debug("write agent text", "error", err)
		}
	}

	// Print tool call if available
	if e.ToolCall != nil {
		printToolCallTo(w, e.ToolCall)
	}

	// Also check tool_calls array for multiple tools
	if toolCalls, ok := e.Data["tool_calls"].([]*agent.ToolCall); ok {
		for _, tc := range toolCalls {
			printToolCallTo(w, tc)
		}
	}

	// Fallback: check for a result in data
	if e.Text == "" && e.ToolCall == nil {
		if result, ok := e.Data["result"].(string); ok {
			_, err := fmt.Fprint(w, result)
			if err != nil {
				slog.Debug("write agent result", "error", err)
			}
		}
	}
}

// printToolCallTo prints a formatted tool call to the specified writer.
func printToolCallTo(w io.Writer, tc *agent.ToolCall) {
	if tc == nil {
		return
	}

	if tc.Description != "" {
		_, err := fmt.Fprintf(w, "→ %s: %s\n", tc.Name, tc.Description)
		if err != nil {
			slog.Debug("write tool call", "error", err)
		}
	} else {
		_, err := fmt.Fprintf(w, "→ %s\n", tc.Name)
		if err != nil {
			slog.Debug("write tool call", "error", err)
		}
	}
}

// RunWithSpinner runs a function with either verbose output or a spinner.
// This consolidates the spinner/verbose pattern used across commands.
// If verbose is true, runs the function directly and returns its error.
// If verbose is false, shows a spinner with the given message during execution.
func RunWithSpinner(verbose bool, spinnerMsg string, fn func() error) error {
	if verbose {
		return fn()
	}

	spinner := display.NewSpinner(spinnerMsg)
	spinner.Start()
	err := fn()

	// Handle spinner result
	if err != nil && !errors.Is(err, conductor.ErrPendingQuestion) {
		spinner.StopWithError("Failed")
	} else if errors.Is(err, conductor.ErrPendingQuestion) {
		spinner.Stop()
	} else {
		spinner.StopWithSuccess("Complete")
	}

	return err
}

// DisplayPendingQuestion displays a pending question from the agent.
// Returns true if a question was displayed.
func DisplayPendingQuestion(cond *conductor.Conductor) bool {
	activeTask := cond.GetActiveTask()
	if activeTask == nil {
		return false
	}

	q, err := cond.GetWorkspace().LoadPendingQuestion(activeTask.ID)
	if err != nil || q == nil {
		return false
	}

	fmt.Println()
	fmt.Println(display.WarningMsg("Agent has a question:"))
	fmt.Println()
	fmt.Printf("  %s\n\n", display.Bold(q.Question))

	if len(q.Options) > 0 {
		fmt.Println(display.Muted("  Options:"))
		for i, opt := range q.Options {
			fmt.Printf("    %s %s", display.Info(fmt.Sprintf("%d.", i+1)), opt.Label)
			if opt.Description != "" {
				fmt.Printf(" %s", display.Muted("- "+opt.Description))
			}
			fmt.Println()
		}
		fmt.Println()
	}

	fmt.Println(display.Muted("Answer with:"))
	fmt.Printf("  %s\n", display.Cyan("mehr answer \"your response\""))

	return true
}
