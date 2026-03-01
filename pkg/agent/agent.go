// Package agent provides an interface and registry for AI agents.
// Based on flow_v2.md design: supports Claude, Codex, and custom agents
// with WebSocket (primary) and CLI (fallback) connection modes.
package agent

import (
	"context"
	"strings"
	"time"

	"github.com/valksor/kvelmo/pkg/agent/permission"
)

// Agent is the interface for AI agents (Claude, Codex, custom).
//
//nolint:interfacebloat // All methods are required for agent lifecycle management
type Agent interface {
	// Name returns the agent's identifier (e.g., "claude", "codex")
	Name() string

	// Available checks if the agent is available (binary exists, API reachable)
	Available() error

	// Connect establishes connection (WebSocket or process)
	Connect(ctx context.Context) error

	// Connected returns true if the agent is connected
	Connected() bool

	// SendPrompt sends a prompt and returns streaming events
	SendPrompt(ctx context.Context, prompt string) (<-chan Event, error)

	// HandlePermission responds to a permission request
	HandlePermission(requestID string, approved bool) error

	// Close closes the connection
	Close() error

	// WithEnv adds environment variable (returns new Agent for chaining)
	WithEnv(key, value string) Agent

	// WithArgs adds CLI arguments (returns new Agent for chaining)
	WithArgs(args ...string) Agent

	// WithWorkDir sets the working directory (returns new Agent for chaining)
	WithWorkDir(dir string) Agent

	// WithTimeout sets the execution timeout (returns new Agent for chaining)
	WithTimeout(d time.Duration) Agent
}

// EventType identifies the type of streaming event.
type EventType string

const (
	EventStream     EventType = "stream"      // Token-by-token output
	EventAssistant  EventType = "assistant"   // Full assistant message
	EventToolUse    EventType = "tool_use"    // Tool call initiated
	EventToolResult EventType = "tool_result" // Tool call completed
	EventPermission EventType = "permission"  // Permission request
	EventComplete   EventType = "complete"    // Job completed successfully
	EventError      EventType = "error"       // Error occurred
	EventInit       EventType = "init"        // Session initialized
	EventKeepAlive  EventType = "keep_alive"  // Heartbeat
	EventSubagent   EventType = "subagent"    // Subagent lifecycle event
)

// Event represents a streaming event from an agent.
type Event struct {
	Type      EventType      `json:"type"`
	Content   string         `json:"content,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
	Timestamp time.Time      `json:"timestamp,omitempty"`

	// For EventPermission
	PermissionRequest *PermissionRequest `json:"permission_request,omitempty"`

	// For EventError
	Error string `json:"error,omitempty"`

	// For EventSubagent
	Subagent *SubagentEvent `json:"subagent,omitempty"`
}

// SubagentStatus indicates the lifecycle state of a subagent.
type SubagentStatus string

const (
	SubagentStarted   SubagentStatus = "started"
	SubagentCompleted SubagentStatus = "completed"
	SubagentFailed    SubagentStatus = "failed"
)

// SubagentEvent represents a subagent lifecycle event.
type SubagentEvent struct {
	ID          string         `json:"id"`
	Type        string         `json:"type"`        // "Explore", "Plan", etc.
	Description string         `json:"description"` // Short description from agent
	Status      SubagentStatus `json:"status"`
	StartedAt   time.Time      `json:"started_at,omitempty"`
	CompletedAt time.Time      `json:"completed_at,omitempty"`
	Duration    int64          `json:"duration,omitempty"`    // milliseconds
	ExitReason  string         `json:"exit_reason,omitempty"` // For failed status
}

// PermissionRequest represents a tool permission request from the agent.
type PermissionRequest struct {
	ID     string         `json:"id"`
	Tool   string         `json:"tool"`
	Input  map[string]any `json:"input,omitempty"`
	Action string         `json:"action,omitempty"`

	// DangerLevel indicates how risky this operation is.
	// Set by danger detection when evaluating the request.
	DangerLevel permission.DangerLevel `json:"danger_level,omitempty"`
	// DangerReason explains why the operation is flagged.
	DangerReason string `json:"danger_reason,omitempty"`
}

// PermissionHandler evaluates permission requests.
// Returns true to approve, false to deny.
type PermissionHandler func(req PermissionRequest) bool

// PermissionResult holds the evaluation result with danger info.
type PermissionResult struct {
	Approved     bool
	DangerLevel  permission.DangerLevel
	DangerReason string
}

// EvaluatePermission evaluates a permission request with danger detection.
// Returns the result including danger level and reason.
func EvaluatePermission(req PermissionRequest) PermissionResult {
	// Check for dangerous operations first
	danger := permission.DetectDanger(req.Tool, req.Input)

	// Dangerous operations are always denied
	if danger.Level == permission.Dangerous {
		return PermissionResult{
			Approved:     false,
			DangerLevel:  danger.Level,
			DangerReason: danger.Reason,
		}
	}

	// Check if tool is in safe list (case-insensitive)
	approved := isSafeTool(req.Tool)

	return PermissionResult{
		Approved:     approved,
		DangerLevel:  danger.Level,
		DangerReason: danger.Reason,
	}
}

// safeTools are read-only tools that can be auto-approved.
// Keys are lowercase; use isSafeTool for case-insensitive lookup.
// Includes aliases for PascalCase tool names (e.g., "ReadFile" → "readfile").
var safeTools = map[string]bool{
	"read_file": true,
	"readfile":  true, // PascalCase alias
	"read":      true,
	"glob":      true,
	"grep":      true,
	"list_dir":  true,
	"listdir":   true, // PascalCase alias
	"ls":        true,
	"search":    true,
}

// isSafeTool checks if a tool is in the safe list (case-insensitive).
func isSafeTool(name string) bool {
	return safeTools[strings.ToLower(name)]
}

// DefaultPermissionHandler auto-approves safe read-only tools.
// Denies dangerous operations regardless of tool type.
func DefaultPermissionHandler(req PermissionRequest) bool {
	result := EvaluatePermission(req)

	return result.Approved
}

// ConnectionMode indicates how the agent is connected.
type ConnectionMode string

const (
	ModeWebSocket ConnectionMode = "websocket" // WebSocket server mode
	ModeCLI       ConnectionMode = "cli"       // CLI subprocess mode
	ModeAPI       ConnectionMode = "api"       // Direct API mode
)

// Config holds common agent configuration.
type Config struct {
	// Connection preferences
	PreferWebSocket bool              // Try WebSocket first (default: true)
	WebSocketPort   int               // Port for WebSocket server (default: 0 = auto)
	Command         []string          // CLI command (e.g., ["claude"], ["codex"])
	Args            []string          // Additional CLI arguments
	Environment     map[string]string // Environment variables

	// Execution settings
	WorkDir    string        // Working directory
	Timeout    time.Duration // Execution timeout (default: 30m)
	RetryCount int           // Retry attempts (default: 3)
	RetryDelay time.Duration // Delay between retries (default: 1s)

	// Permission handling
	PermissionHandler PermissionHandler // Custom permission handler
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		PreferWebSocket:   true,
		WebSocketPort:     0, // Auto-assign
		Timeout:           30 * time.Minute,
		RetryCount:        3,
		RetryDelay:        time.Second,
		Environment:       make(map[string]string),
		PermissionHandler: DefaultPermissionHandler,
	}
}

// Merge merges another config into this one (other takes precedence).
func (c Config) Merge(other Config) Config {
	if other.Command != nil {
		c.Command = other.Command
	}
	if other.Args != nil {
		c.Args = append(c.Args, other.Args...)
	}
	if other.WorkDir != "" {
		c.WorkDir = other.WorkDir
	}
	if other.Timeout > 0 {
		c.Timeout = other.Timeout
	}
	if other.Environment != nil {
		if c.Environment == nil {
			c.Environment = make(map[string]string)
		}
		for k, v := range other.Environment {
			c.Environment[k] = v
		}
	}
	if other.PermissionHandler != nil {
		c.PermissionHandler = other.PermissionHandler
	}

	return c
}
