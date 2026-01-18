package orchestration

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/valksor/go-mehrhof/internal/agent"
	"github.com/valksor/go-mehrhof/internal/storage"
)

// mockAgent is a mock agent implementation for testing.
type mockAgent struct {
	name     string
	response *agent.Response
	runErr   error
}

func (m *mockAgent) Name() string {
	return m.name
}

func (m *mockAgent) Run(ctx context.Context, prompt string) (*agent.Response, error) {
	if m.runErr != nil {
		return nil, m.runErr
	}

	return m.response, nil
}

func (m *mockAgent) RunStream(ctx context.Context, prompt string) (<-chan agent.Event, <-chan error) {
	return nil, nil
}

func (m *mockAgent) RunWithCallback(ctx context.Context, prompt string, cb agent.StreamCallback) (*agent.Response, error) {
	return m.response, m.runErr
}

func (m *mockAgent) WithEnv(key, value string) agent.Agent {
	return m
}

func (m *mockAgent) WithArgs(args ...string) agent.Agent {
	return m
}

func (m *mockAgent) Configure(agent.Config) error {
	return nil
}

func (m *mockAgent) Available() error {
	return nil
}

// TestExecuteAgent_Success tests successful agent execution.
func TestExecuteAgent_Success(t *testing.T) {
	ctx := context.Background()
	mockResp := &agent.Response{
		Summary:  "Test output",
		Messages: []string{"Test output"},
		Usage: &agent.UsageStats{
			InputTokens:  10,
			OutputTokens: 20,
			CostUSD:      0.001,
		},
	}

	agentInst := &mockAgent{
		name:     "test-agent",
		response: mockResp,
	}

	taskCtx := &TaskContext{
		TaskID: "test-123",
		Title:  "Test Task",
		State:  "implementing",
	}

	output, tokens, cost, err := executeAgent(ctx, agentInst, taskCtx, "test role")
	if err != nil {
		t.Fatalf("executeAgent failed: %v", err)
	}

	if output != "Test output" {
		t.Errorf("expected output 'Test output', got '%s'", output)
	}

	if tokens != 30 { // 10 input + 20 output
		t.Errorf("expected 30 tokens, got %d", tokens)
	}

	if cost != 0.001 {
		t.Errorf("expected cost 0.001, got %f", cost)
	}
}

// TestExecuteAgent_Error tests agent execution with error.
func TestExecuteAgent_Error(t *testing.T) {
	ctx := context.Background()
	agentInst := &mockAgent{
		name:   "test-agent",
		runErr: errors.New("agent failed"),
	}

	taskCtx := &TaskContext{
		TaskID: "test-123",
		Title:  "Test Task",
	}

	// We only care about the error here
	//nolint:dogsled
	_, _, _, err := executeAgent(ctx, agentInst, taskCtx, "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err.Error() != "agent run: agent failed" {
		t.Errorf("expected error 'agent run: agent failed', got '%v'", err)
	}
}

// TestExecuteAgent_EmptySummary tests agent execution with empty summary.
func TestExecuteAgent_EmptySummary(t *testing.T) {
	ctx := context.Background()
	mockResp := &agent.Response{
		Summary:  "",
		Messages: []string{"Message 1", "Message 2"},
		Usage:    &agent.UsageStats{},
	}

	agentInst := &mockAgent{
		name:     "test-agent",
		response: mockResp,
	}

	taskCtx := &TaskContext{
		TaskID: "test-123",
		Title:  "Test Task",
	}

	output, _, _, err := executeAgent(ctx, agentInst, taskCtx, "")
	if err != nil {
		t.Fatalf("executeAgent failed: %v", err)
	}

	if output != "Message 1" {
		t.Errorf("expected output 'Message 1', got '%s'", output)
	}
}

// TestExecuteAgent_NoUsage tests agent execution with no usage stats.
func TestExecuteAgent_NoUsage(t *testing.T) {
	ctx := context.Background()
	mockResp := &agent.Response{
		Summary: "Test output",
		Usage:   nil, // No usage stats
	}

	agentInst := &mockAgent{
		name:     "test-agent",
		response: mockResp,
	}

	taskCtx := &TaskContext{
		TaskID: "test-123",
		Title:  "Test Task",
	}

	output, _, _, err := executeAgent(ctx, agentInst, taskCtx, "")
	if err != nil {
		t.Fatalf("executeAgent failed: %v", err)
	}

	if output != "Test output" {
		t.Errorf("expected output 'Test output', got '%s'", output)
	}
}

// TestBuildAgentPrompt tests prompt building for agent execution.
func TestBuildAgentPrompt(t *testing.T) {
	taskCtx := &TaskContext{
		TaskID:  "task-123",
		Title:   "Fix authentication bug",
		State:   "implementing",
		Inputs:  []string{"Previous output: Login fails"},
		WorkDir: "/tmp/work/task-123",
	}

	prompt := buildAgentPrompt(taskCtx, "developer")

	// Check for key elements in prompt
	required := []string{
		"# Task: Fix authentication bug",
		"Your role: developer",
		"Current state: implementing",
		"Previous output: Login fails",
	}

	for _, req := range required {
		if !contains(prompt, req) {
			t.Errorf("prompt missing required element: %q\nPrompt:\n%s", req, prompt)
		}
	}
}

// contains is a helper function to check if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || indexOfString(s, substr) >= 0)
}

// indexOfString finds the index of a substring.
func indexOfString(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}

	return -1
}

// TestPipeline_ConcurrentGetArtifact tests that concurrent artifact access is thread-safe.
// This test verifies the fix for the race condition in LoadArtifact.
func TestPipeline_ConcurrentGetArtifact(t *testing.T) {
	// Create a simple pipeline for testing
	pipeline := &Pipeline{
		taskID:    "test-task",
		artifacts: make(map[string]string),
	}

	// Set some initial artifacts
	pipeline.SetArtifact("artifact1", "content1")
	pipeline.SetArtifact("artifact2", "content2")
	pipeline.SetArtifact("artifact3", "content3")

	// Launch many goroutines reading artifacts concurrently
	numGoroutines := 100
	var wg sync.WaitGroup
	errChan := make(chan error, numGoroutines*3)

	for range numGoroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Try to get all artifacts
			for _, name := range []string{"artifact1", "artifact2", "artifact3"} {
				content, ok := pipeline.GetArtifact(name)
				if !ok {
					errChan <- errors.New("artifact not found")

					return
				}
				if content == "" {
					errChan <- errors.New("artifact content is empty")

					return
				}
			}
		}()
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		t.Errorf("concurrent access error: %v", err)
	}
}

// TestPipeline_ConcurrentSetArtifact tests that concurrent artifact writes are thread-safe.
func TestPipeline_ConcurrentSetArtifact(t *testing.T) {
	pipeline := &Pipeline{
		taskID:    "test-task",
		artifacts: make(map[string]string),
	}

	// Launch many goroutines setting artifacts concurrently
	numGoroutines := 100
	var wg sync.WaitGroup

	for i := range numGoroutines {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			name := fmt.Sprintf("artifact-%d", id)
			content := fmt.Sprintf("content-%d", id)
			pipeline.SetArtifact(name, content)
		}(i)
	}

	wg.Wait()

	// Verify all artifacts were set
	names := pipeline.ListArtifacts()
	if len(names) != numGoroutines {
		t.Errorf("expected %d artifacts, got %d", numGoroutines, len(names))
	}
}

// TestPipeline_ConcurrentMixedOperations tests concurrent reads and writes.
func TestPipeline_ConcurrentMixedOperations(t *testing.T) {
	pipeline := &Pipeline{
		taskID:    "test-task",
		artifacts: make(map[string]string),
	}

	// Set initial artifacts
	pipeline.SetArtifact("initial", "initial-value")

	numGoroutines := 50
	var wg sync.WaitGroup
	errChan := make(chan error, numGoroutines*2)

	// Start writers
	for i := range numGoroutines {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			name := fmt.Sprintf("artifact-%d", id)
			content := fmt.Sprintf("content-%d", id)
			pipeline.SetArtifact(name, content)
		}(i)
	}

	// Start readers
	for range numGoroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Try to get the initial artifact
			content, ok := pipeline.GetArtifact("initial")
			if !ok {
				errChan <- errors.New("initial artifact not found")

				return
			}
			if content != "initial-value" {
				errChan <- errors.New("initial artifact value changed unexpectedly")
			}

			// List artifacts
			names := pipeline.ListArtifacts()
			if len(names) == 0 {
				errChan <- errors.New("no artifacts found")
			}
		}()
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		t.Errorf("concurrent mixed operations error: %v", err)
	}
}

// TestValidateConfig tests configuration validation.
func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *OrchestratorConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
			errMsg:  "config is required",
		},
		{
			name: "valid single mode",
			config: &OrchestratorConfig{
				Mode: ModeSingle,
			},
			wantErr: false,
		},
		{
			name: "valid parallel mode",
			config: &OrchestratorConfig{
				Mode: ModeParallel,
				Agents: []AgentStep{
					{Name: "step1", Agent: "agent1"},
					{Name: "step2", Agent: "agent2"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid mode",
			config: &OrchestratorConfig{
				Mode: "invalid",
			},
			wantErr: true,
			errMsg:  "invalid mode",
		},
		{
			name: "multi-agent mode without agents",
			config: &OrchestratorConfig{
				Mode:   ModeParallel,
				Agents: []AgentStep{},
			},
			wantErr: true,
			errMsg:  "at least one agent",
		},
		{
			name: "duplicate step names",
			config: &OrchestratorConfig{
				Mode: ModeSequential,
				Agents: []AgentStep{
					{Name: "step1", Agent: "agent1"},
					{Name: "step1", Agent: "agent2"},
				},
			},
			wantErr: true,
			errMsg:  "duplicate step name",
		},
		{
			name: "empty step name",
			config: &OrchestratorConfig{
				Mode: ModeSequential,
				Agents: []AgentStep{
					{Name: "", Agent: "agent1"},
				},
			},
			wantErr: true,
			errMsg:  "must have a name",
		},
		{
			name: "consensus mode without consensus config",
			config: &OrchestratorConfig{
				Mode: ModeConsensus,
				Agents: []AgentStep{
					{Name: "step1", Agent: "agent1"},
				},
			},
			wantErr: true,
			errMsg:  "consensus mode is required",
		},
		{
			name: "valid consensus mode",
			config: &OrchestratorConfig{
				Mode: ModeConsensus,
				Agents: []AgentStep{
					{Name: "step1", Agent: "agent1"},
				},
				Consensus: ConsensusConfig{
					Mode: "majority",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validateConfig() expected error, got nil")

					return
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("validateConfig() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateConfig() unexpected error: %v", err)
				}
			}
		})
	}
}

// TestValidateDependencies tests dependency validation.
func TestValidateDependencies(t *testing.T) {
	tests := []struct {
		name    string
		agents  []AgentStep
		wantErr bool
		errMsg  string
	}{
		{
			name: "no dependencies",
			agents: []AgentStep{
				{Name: "step1", Agent: "agent1"},
				{Name: "step2", Agent: "agent2"},
			},
			wantErr: false,
		},
		{
			name: "valid dependencies",
			agents: []AgentStep{
				{Name: "step1", Agent: "agent1"},
				{Name: "step2", Agent: "agent2", Depends: []string{"step1"}},
				{Name: "step3", Agent: "agent3", Depends: []string{"step2"}},
			},
			wantErr: false,
		},
		{
			name: "non-existent dependency",
			agents: []AgentStep{
				{Name: "step1", Agent: "agent1"},
				{Name: "step2", Agent: "agent2", Depends: []string{"nonexistent"}},
			},
			wantErr: true,
			errMsg:  "depends on non-existent step",
		},
		{
			name: "circular dependency",
			agents: []AgentStep{
				{Name: "step1", Agent: "agent1", Depends: []string{"step2"}},
				{Name: "step2", Agent: "agent2", Depends: []string{"step1"}},
			},
			wantErr: false, // validateDependencies doesn't check for cycles
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Orchestrator{
				config: &OrchestratorConfig{
					Agents: tt.agents,
				},
			}
			err := o.validateDependencies()
			if tt.wantErr {
				if err == nil {
					t.Errorf("validateDependencies() expected error, got nil")

					return
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("validateDependencies() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateDependencies() unexpected error: %v", err)
				}
			}
		})
	}
}

// TestOrchestrator_ContextCancellation tests context cancellation handling in runParallel.
func TestOrchestrator_ContextCancellation(t *testing.T) {
	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Create an orchestrator
	registry := &mockRegistry{
		agents: map[string]agent.Agent{
			"test-agent": &mockAgent{
				name: "test-agent",
				response: &agent.Response{
					Summary: "test response",
				},
			},
		},
	}

	config := &OrchestratorConfig{
		Mode: ModeParallel,
		Agents: []AgentStep{
			{Name: "step1", Agent: "test-agent"},
			{Name: "step2", Agent: "test-agent"},
		},
	}

	o := &Orchestrator{
		config:   config,
		registry: registry,
	}

	// Cancel immediately
	cancel()

	// Create a minimal task - use empty but valid struct
	task := &storage.TaskWork{}

	// Run should handle cancellation gracefully
	result, _ := o.Run(ctx, task)
	if result == nil {
		t.Error("expected result even on cancellation, got nil")
	}
	// Context cancellation is not an error in this implementation
	// The errors are collected in result.Errors
}

// mockRegistry is a mock agent registry for testing.
type mockRegistry struct {
	agents map[string]agent.Agent
}

func (m *mockRegistry) Get(name string) (agent.Agent, error) {
	if a, ok := m.agents[name]; ok {
		return a, nil
	}

	return nil, fmt.Errorf("agent not found: %s", name)
}

func (m *mockRegistry) List() []string {
	names := make([]string, 0, len(m.agents))
	for name := range m.agents {
		names = append(names, name)
	}

	return names
}
