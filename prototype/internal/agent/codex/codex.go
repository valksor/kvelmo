// Package codex implements an AI agent using OpenAI's Codex CLI.
//
// WARNING: This agent implementation is based on Codex CLI documentation
// and has NOT been tested against an actual Codex CLI. The JSON output
// format, event structure, and tool use format are assumed to be similar
// to Claude but may differ significantly.
//
// Once Codex CLI is available, the following should be validated:
//   - JSON event structure from `codex exec --json`
//   - Tool call format (Read, Write, Edit, etc.)
//   - YAML block format for file operations
//   - Actual behavior of --sandbox and --full-auto flags
//
// Reference: https://developers.openai.com/codex/cli/reference/
package codex

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/valksor/go-mehrhof/internal/agent"
	"github.com/valksor/go-mehrhof/internal/sandbox"
	"github.com/valksor/go-mehrhof/internal/vcs"
)

const AgentName = "codex"

const (
	// scannerBufferSize is the size of the buffer used for scanning agent output.
	// 1MB allows handling large responses without excessive allocations.
	scannerBufferSize = 1024 * 1024
)

// scannerBufferPool reuses buffers for scanner operations.
// This significantly reduces GC pressure when making many agent calls.
// Using *[]byte avoids slice header allocations.
var scannerBufferPool = sync.Pool{
	New: func() any {
		b := make([]byte, scannerBufferSize)

		return &b
	},
}

// Agent wraps the Codex CLI.
type Agent struct {
	parser        agent.Parser
	config        agent.Config
	sandboxConfig *sandbox.Config
}

// New creates a Codex agent with default config.
func New() *Agent {
	return &Agent{
		config: agent.Config{
			Command:     []string{"codex"},
			Environment: make(map[string]string),
			Timeout:     30 * time.Minute,
			RetryCount:  3,
			RetryDelay:  time.Second,
		},
		parser: agent.NewJSONLineParser(),
	}
}

// NewWithConfig creates a Codex agent with custom config.
func NewWithConfig(cfg agent.Config) *Agent {
	if len(cfg.Command) == 0 {
		cfg.Command = []string{"codex"}
	}

	return &Agent{
		config: cfg,
		parser: agent.NewJSONLineParser(),
	}
}

// Name returns the agent identifier.
func (a *Agent) Name() string {
	return AgentName
}

// Available checks if the Codex CLI is installed and configured.
func (a *Agent) Available() error {
	binary := a.config.Command[0]
	path, err := exec.LookPath(binary)
	if err != nil {
		return fmt.Errorf("codex CLI not found: %w", err)
	}

	// Verify it runs
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, path, "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("codex CLI not working: %w", err)
	}

	return nil
}

// Run executes a prompt and returns the aggregated response.
func (a *Agent) Run(ctx context.Context, prompt string) (*agent.Response, error) {
	events, errCh := a.RunStream(ctx, prompt)

	var collected []agent.Event
	for event := range events {
		collected = append(collected, event)
	}

	// Check for streaming errors
	if err := <-errCh; err != nil {
		return nil, err
	}

	return a.parser.Parse(collected)
}

// RunStream executes a prompt and streams events.
func (a *Agent) RunStream(ctx context.Context, prompt string) (<-chan agent.Event, <-chan error) {
	eventCh := make(chan agent.Event, 100)
	errCh := make(chan error, 1)

	go func() {
		defer close(eventCh)
		defer close(errCh)

		err := a.executeStream(ctx, prompt, eventCh)
		if err != nil {
			errCh <- err
		}
	}()

	return eventCh, errCh
}

// RunWithCallback executes with a callback for each event.
func (a *Agent) RunWithCallback(ctx context.Context, prompt string, cb agent.StreamCallback) (*agent.Response, error) {
	events, errCh := a.RunStream(ctx, prompt)

	var collected []agent.Event
	for event := range events {
		if err := cb(event); err != nil {
			return nil, fmt.Errorf("callback error: %w", err)
		}
		collected = append(collected, event)
	}

	if err := <-errCh; err != nil {
		return nil, err
	}

	return a.parser.Parse(collected)
}

func (a *Agent) executeStream(ctx context.Context, prompt string, eventCh chan<- agent.Event) error {
	// Build command with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, a.config.Timeout)
	defer cancel()

	args := a.buildArgs(ctx, prompt)
	cmd := exec.CommandContext(timeoutCtx, a.config.Command[0], args...)

	// Set working directory
	if a.config.WorkDir != "" {
		cmd.Dir = a.config.WorkDir
	}

	// Set environment
	cmd.Env = os.Environ()
	for k, v := range a.config.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Get stdout pipe
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	// Get stderr pipe for error messages
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start command: %w", err)
	}

	// Read output line by line
	scanner := bufio.NewScanner(stdout)
	// Get buffer from pool for large responses, return when done
	buf, ok := scannerBufferPool.Get().(*[]byte)
	if !ok {
		return errors.New("scanner buffer pool returned wrong type")
	}
	defer scannerBufferPool.Put(buf)
	scanner.Buffer(*buf, scannerBufferSize)

	for scanner.Scan() {
		select {
		case <-timeoutCtx.Done():
			_ = cmd.Process.Kill()

			return timeoutCtx.Err()
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		event, err := a.parser.ParseEvent(line)
		if err != nil {
			// Log parse error but continue
			slog.Debug("codex parser error", "error", err, "line", string(line))

			continue
		}

		eventCh <- event

		// Check for completion
		if event.Type == agent.EventComplete {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	// Read any stderr output
	stderrBytes, err := bufio.NewReader(stderr).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		// Log but don't fail - stderr may not have content
		slog.Debug("error reading stderr", "error", err)
	}

	// Wait for command to finish
	if err := cmd.Wait(); err != nil {
		// Check if it's just a non-zero exit (which might be okay)
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if exitErr.ExitCode() != 0 {
				if stderrBytes != "" {
					return fmt.Errorf("codex exited with code %d: %s", exitErr.ExitCode(), stderrBytes)
				}

				return fmt.Errorf("codex exited with code %d", exitErr.ExitCode())
			}
		}

		return fmt.Errorf("wait error: %w", err)
	}

	return nil
}

// buildArgs constructs the command-line arguments for Codex exec.
//
// Codex CLI structure differs from Claude:
// - Uses "codex exec --json <prompt>" instead of "claude --print --verbose --output-format stream-json <prompt>"
// - Does NOT have --print, --verbose, or --output-format flags (those are Claude-specific).
func (a *Agent) buildArgs(ctx context.Context, prompt string) []string {
	args := []string{}

	// Start with 'exec' subcommand and '--json' flag (Codex-specific)
	args = append(args, "exec", "--json")

	// Add base arguments from config (if Command has more than just binary)
	if len(a.config.Command) > 1 {
		args = append(args, a.config.Command[1:]...)
	}

	// Add configured CLI arguments
	if len(a.config.Args) > 0 {
		args = append(args, a.config.Args...)
	}

	// Allow Codex to run outside a git repo automatically when needed.
	if shouldSkipGitRepoCheck(ctx, a.config.WorkDir, args) {
		args = append(args, "--skip-git-repo-check")
	}

	// Add --yolo flag when sandbox is enabled (sandbox provides isolation)
	// This prevents Codex from asking for confirmations that can't be answered
	if a.sandboxConfig != nil && a.sandboxConfig.Enabled {
		args = append(args, "--yolo")
	}

	// Add prompt as positional argument (last)
	args = append(args, prompt)

	return args
}

func shouldSkipGitRepoCheck(ctx context.Context, workDir string, args []string) bool {
	for _, arg := range args {
		if arg == "--skip-git-repo-check" {
			return false
		}
	}

	if workDir == "" {
		workDir = "."
	}

	return !vcs.IsRepo(ctx, workDir)
}

// SetParser allows overriding the default parser.
func (a *Agent) SetParser(p agent.Parser) {
	a.parser = p
}

// WithWorkDir sets the working directory.
// Returns a new Agent instance with the updated config to avoid data races.
func (a *Agent) WithWorkDir(dir string) agent.Agent {
	newConfig := a.config
	newConfig.WorkDir = dir

	return &Agent{
		config:        newConfig,
		parser:        a.parser,
		sandboxConfig: a.sandboxConfig,
	}
}

// WithTimeout sets execution timeout.
// Returns a new Agent instance with the updated config to avoid data races.
func (a *Agent) WithTimeout(d time.Duration) *Agent {
	newConfig := a.config
	newConfig.Timeout = d

	return &Agent{
		config:        newConfig,
		parser:        a.parser,
		sandboxConfig: a.sandboxConfig,
	}
}

// WithEnv adds an environment variable.
// Returns a new Agent instance with the updated config to avoid data races.
//
// Thread safety: This method is safe for concurrent use as it returns a new
// Agent instance rather than modifying the receiver. The returned Agent shares
// the same parser reference with the original; if the parser is not thread-safe,
// avoid calling Run/RunStream on multiple Agent instances concurrently.
func (a *Agent) WithEnv(key, value string) agent.Agent {
	newConfig := a.config
	newConfig.Environment = make(map[string]string, len(a.config.Environment)+1)
	for k, v := range a.config.Environment {
		newConfig.Environment[k] = v
	}
	newConfig.Environment[key] = value

	return &Agent{
		config:        newConfig,
		parser:        a.parser,
		sandboxConfig: a.sandboxConfig,
	}
}

// WithArgs adds CLI arguments to pass to the agent process.
// Returns a new Agent instance with the updated config to avoid data races.
func (a *Agent) WithArgs(args ...string) agent.Agent {
	newConfig := a.config
	newArgs := make([]string, len(a.config.Args), len(a.config.Args)+len(args))
	copy(newArgs, a.config.Args)
	newConfig.Args = append(newArgs, args...)

	return &Agent{
		config:        newConfig,
		parser:        a.parser,
		sandboxConfig: a.sandboxConfig,
	}
}

// WithCommand sets a custom binary path for the agent.
// Returns a new Agent instance with the updated config to avoid data races.
func (a *Agent) WithCommand(command string) agent.Agent {
	newConfig := a.config
	newConfig.Command = []string{command}

	return &Agent{
		config:        newConfig,
		parser:        a.parser,
		sandboxConfig: a.sandboxConfig,
	}
}

// WithRetries sets the retry count for the agent.
// Returns a new Agent instance with the updated config to avoid data races.
// Use 0 to disable retries entirely (single attempt).
func (a *Agent) WithRetries(n int) agent.Agent {
	newConfig := a.config
	newConfig.RetryCount = n

	return &Agent{
		config:        newConfig,
		parser:        a.parser,
		sandboxConfig: a.sandboxConfig,
	}
}

// WithSandbox sets the sandbox configuration for the Codex agent.
// When sandbox is enabled, the --yolo flag is added to skip confirmations.
func (a *Agent) WithSandbox(cfg *sandbox.Config) agent.Agent {
	newConfig := a.config

	return &Agent{
		config:        newConfig,
		parser:        a.parser,
		sandboxConfig: cfg,
	}
}

// Register adds the Codex agent to a registry.
func Register(r *agent.Registry) error {
	return r.Register(New())
}

// StepArgs returns step-specific CLI args for Codex.
//
// Codex uses different flags than Claude:
// - Claude uses --permission-mode (plan/acceptEdits)
// - Codex uses --sandbox (read-only/workspace-write) and --ask-for-approval flags
//
// The --full-auto flag is a convenience that sets:
// - --sandbox workspace-write (allows file writes within workspace)
// - --ask-for-approval on-failure (only prompt for approval on errors).
func (a *Agent) StepArgs(step string) []string {
	switch step {
	case "planning":
		// Use read-only sandbox for planning (no file modifications)
		// This ensures planning step doesn't accidentally modify files
		return []string{"--sandbox", "read-only"}
	case "implementing", "reviewing":
		// --full-auto enables unattended execution:
		// - Allows writing files within the workspace
		// - Only prompts for approval on failures/errors
		return []string{"--full-auto"}
	default:
		return nil
	}
}

// Ensure Agent implements agent.Agent.
var _ agent.Agent = (*Agent)(nil)

// Ensure Agent implements agent.StepArgsProvider.
var _ agent.StepArgsProvider = (*Agent)(nil)
