package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/valksor/go-mehrhof/internal/conductor"
)

// ErrUnknownCommand is returned when a command is not found.
var ErrUnknownCommand = errors.New("unknown command")

// ErrNoActiveTask is returned when a command requires an active task but none exists.
var ErrNoActiveTask = errors.New("no active task")

// ErrBadRequest is returned when a command receives invalid input.
// The HTTP adapter maps this to 400 Bad Request.
var ErrBadRequest = errors.New("bad request")

// InvocationSource identifies where a command call originated.
type InvocationSource string

const (
	SourceCLI         InvocationSource = "cli"
	SourceREPL        InvocationSource = "repl"
	SourceAPI         InvocationSource = "api"
	SourceInteractive InvocationSource = "interactive"
)

// Invocation is a structured command invocation.
type Invocation struct {
	Args    []string       `json:"args,omitempty"`
	Options map[string]any `json:"options,omitempty"`
	Source  InvocationSource
}

// HandlerFunc is the signature for all command handlers.
// It receives the conductor, parsed arguments, and returns a Result.
type HandlerFunc func(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error)

// Command represents a registered command with its handler and metadata.
type Command struct {
	Info    CommandInfo
	Handler HandlerFunc
}

// registry holds all registered commands by their canonical name.
var registry = make(map[string]Command)

// aliases maps command aliases to their canonical names.
var aliases = make(map[string]string)

// Register adds a command to the registry.
func Register(cmd Command) {
	registry[cmd.Info.Name] = cmd
	for _, alias := range cmd.Info.Aliases {
		aliases[alias] = cmd.Info.Name
	}
}

// Execute runs a command by name with the given arguments.
// It handles alias resolution, task requirement checks, and error wrapping.
func Execute(ctx context.Context, cond *conductor.Conductor, name string, inv Invocation) (*Result, error) {
	// Resolve alias to canonical name
	canonical := resolveAlias(name)

	// Look up command
	cmd, ok := registry[canonical]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownCommand, name)
	}

	// Check if command requires an active task
	if cmd.Info.RequiresTask && (cond == nil || cond.GetActiveTask() == nil) {
		return nil, ErrNoActiveTask
	}

	// Execute the handler
	result, err := cmd.Handler(ctx, cond, inv)
	if err != nil {
		return nil, err
	}

	// Add current state to result if not already set
	if result != nil && result.State == "" && cond != nil {
		if task := cond.GetActiveTask(); task != nil {
			result.State = task.State
			result.TaskID = task.ID
		}
	}

	return result, nil
}

// ExecuteWithRun executes the command and runs its executor when provided.
func ExecuteWithRun(ctx context.Context, cond *conductor.Conductor, name string, inv Invocation) (*Result, error) {
	result, err := Execute(ctx, cond, name, inv)
	if err != nil {
		if errors.Is(err, ErrUnknownCommand) || errors.Is(err, ErrNoActiveTask) || errors.Is(err, ErrBadRequest) {
			return nil, err
		}

		return ClassifyError(result, err), nil
	}
	if result != nil && result.Executor != nil {
		if execErr := result.Executor(ctx); execErr != nil {
			return ClassifyError(result, execErr), nil
		}
	}

	return result, nil
}

// resolveAlias returns the canonical command name for an alias.
func resolveAlias(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	if canonical, ok := aliases[name]; ok {
		return canonical
	}

	return name
}

// IsKnownCommand checks if a command name or alias is registered.
func IsKnownCommand(name string) bool {
	canonical := resolveAlias(name)
	_, ok := registry[canonical]

	return ok
}

// ParseInput splits a raw input string into command and arguments.
// Returns the command name and slice of arguments.
func ParseInput(input string) (string, []string) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", nil
	}

	parts := strings.Fields(input)
	if len(parts) == 0 {
		return "", nil
	}

	return strings.ToLower(parts[0]), parts[1:]
}

// GetCurrentState returns the current workflow state from the conductor.
func GetCurrentState(cond *conductor.Conductor) string {
	if cond != nil {
		if task := cond.GetActiveTask(); task != nil {
			return task.State
		}
	}

	return "idle"
}
