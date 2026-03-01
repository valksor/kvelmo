package codex

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/valksor/kvelmo/pkg/agent"
)

// CLIConnection manages Codex via CLI subprocess.
// This is the fallback mode when WebSocket is unavailable.
type CLIConnection struct {
	config Config

	cmd       *exec.Cmd
	cmdMu     sync.Mutex
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	stderr    io.ReadCloser
	connected atomic.Bool
	closed    atomic.Bool

	// Event channel for current prompt
	events chan agent.Event

	// Subagent tracker for detecting Task tool calls
	subagents *agent.SubagentTracker
}

// cliMessage represents JSON output from Codex CLI.
type cliMessage struct {
	Type string `json:"type"`

	// For content
	Content string `json:"content,omitempty"`
	Text    string `json:"text,omitempty"`

	// For tool_use
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`

	// For tool_result
	ToolUseID string `json:"tool_use_id,omitempty"`
	IsError   bool   `json:"is_error,omitempty"`

	// For result
	Result  string `json:"result,omitempty"`
	Success bool   `json:"success,omitempty"`
	Error   string `json:"error,omitempty"`

	// For message
	Role    string `json:"role,omitempty"`
	Message string `json:"message,omitempty"`
}

// NewCLIConnection creates a new CLI connection for Codex.
func NewCLIConnection(cfg Config) *CLIConnection {
	events := make(chan agent.Event, 100)

	return &CLIConnection{
		config:    cfg,
		events:    events,
		subagents: agent.NewSubagentTracker(events),
	}
}

// Connect starts the Codex CLI process.
func (c *CLIConnection) Connect(ctx context.Context) error {
	c.cmdMu.Lock()
	defer c.cmdMu.Unlock()

	if c.connected.Load() {
		return nil
	}

	args := c.buildArgs()
	c.cmd = exec.CommandContext(ctx, c.config.Command[0], args...)

	if c.config.WorkDir != "" {
		c.cmd.Dir = c.config.WorkDir
	}

	// Set environment
	for k, v := range c.config.Environment {
		c.cmd.Env = append(c.cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Get stdin for sending prompts
	stdin, err := c.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}
	c.stdin = stdin

	// Get stdout for reading responses
	stdout, err := c.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	c.stdout = stdout

	// Get stderr for errors
	stderr, err := c.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}
	c.stderr = stderr

	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("start codex: %w", err)
	}

	c.connected.Store(true)

	// Start output reader
	go c.readOutput()

	// Start error reader
	go c.readErrors()

	// Wait for process in background
	go func() {
		_ = c.cmd.Wait()
		c.connected.Store(false)
	}()

	return nil
}

// buildArgs constructs CLI arguments for Codex.
func (c *CLIConnection) buildArgs() []string {
	args := []string{
		"exec",
		"--json",
	}

	// Add configured arguments
	args = append(args, c.config.Args...)

	// Add model if specified
	if c.config.Model != "" {
		args = append(args, "--model", c.config.Model)
	}

	return args
}

// readOutput reads JSON from Codex's stdout.
func (c *CLIConnection) readOutput() {
	scanner := bufio.NewScanner(c.stdout)
	// Increase buffer for large outputs
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var msg cliMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}

		c.handleMessage(msg)
	}

	// Scanner done - close events
	if !c.closed.Load() {
		close(c.events)
	}
}

// readErrors reads stderr for error messages.
func (c *CLIConnection) readErrors() {
	scanner := bufio.NewScanner(c.stderr)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			c.events <- agent.Event{
				Type:      agent.EventError,
				Content:   line,
				Timestamp: time.Now(),
			}
		}
	}
}

// handleMessage processes a CLI message.
func (c *CLIConnection) handleMessage(msg cliMessage) {
	switch msg.Type {
	case "text", "content", "assistant":
		content := msg.Content
		if content == "" {
			content = msg.Text
		}
		if content != "" {
			c.events <- agent.Event{
				Type:      agent.EventStream,
				Content:   content,
				Timestamp: time.Now(),
			}
		}

	case "tool_use":
		var input map[string]any
		if msg.Input != nil {
			_ = json.Unmarshal(msg.Input, &input)
		}

		// Check for subagent spawn (Task tool)
		toolCallID := msg.ID
		if toolCallID == "" {
			toolCallID = uuid.NewString()
		}
		c.subagents.OnToolUse(toolCallID, msg.Name, input)

		c.events <- agent.Event{
			Type:      agent.EventToolUse,
			Content:   msg.Name,
			Data:      input,
			Timestamp: time.Now(),
		}

	case "tool_result":
		// Check for subagent completion
		toolCallID := msg.ToolUseID
		if toolCallID != "" {
			c.subagents.OnToolResult(toolCallID, !msg.IsError, msg.Error)
		}

		c.events <- agent.Event{
			Type:      agent.EventToolResult,
			Content:   msg.Result,
			Timestamp: time.Now(),
		}

	case "result":
		if msg.Success {
			c.events <- agent.Event{
				Type:      agent.EventComplete,
				Timestamp: time.Now(),
			}
		} else {
			c.events <- agent.Event{
				Type:      agent.EventError,
				Error:     msg.Error,
				Timestamp: time.Now(),
			}
		}

	case "error":
		c.events <- agent.Event{
			Type:      agent.EventError,
			Error:     msg.Error,
			Content:   msg.Message,
			Timestamp: time.Now(),
		}

	case "done", "complete":
		c.events <- agent.Event{
			Type:      agent.EventComplete,
			Timestamp: time.Now(),
		}
	}
}

// Connected returns true if the CLI process is running.
func (c *CLIConnection) Connected() bool {
	return c.connected.Load()
}

// SendPrompt sends a prompt to Codex CLI.
func (c *CLIConnection) SendPrompt(ctx context.Context, prompt string) (<-chan agent.Event, error) {
	if !c.connected.Load() {
		return nil, errors.New("not connected")
	}

	// Create new event channel for this prompt
	c.events = make(chan agent.Event, 100)
	c.subagents.SetEventChannel(c.events)

	// Write prompt to stdin
	c.cmdMu.Lock()
	if c.stdin != nil {
		// Codex expects plain text or JSON input
		msg := map[string]any{
			"prompt": prompt,
		}
		data, err := json.Marshal(msg)
		if err == nil {
			data = append(data, '\n')
			_, _ = c.stdin.Write(data)
		}
	}
	c.cmdMu.Unlock()

	// Return filtered event channel
	filtered := make(chan agent.Event, 100)
	go func() {
		defer close(filtered)
		for {
			select {
			case event, ok := <-c.events:
				if !ok {
					return
				}
				filtered <- event
				if event.Type == agent.EventComplete || event.Type == agent.EventError {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return filtered, nil
}

// HandlePermission is a no-op in CLI mode.
func (c *CLIConnection) HandlePermission(requestID string, approved bool) error {
	return nil
}

// Close stops the CLI process.
func (c *CLIConnection) Close() error {
	if c.closed.Swap(true) {
		return nil // Already closed
	}

	c.connected.Store(false)

	c.cmdMu.Lock()
	defer c.cmdMu.Unlock()

	// Close stdin to signal EOF
	if c.stdin != nil {
		_ = c.stdin.Close()
	}

	// Kill process if still running
	if c.cmd != nil && c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
	}

	return nil
}
