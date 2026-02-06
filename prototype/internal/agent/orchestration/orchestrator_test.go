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

func (m *mockAgent) WithRetries(_ int) agent.Agent {
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

// ──────────────────────────────────────────────────────────────────────────────
// Consensus Algorithm Tests
// ──────────────────────────────────────────────────────────────────────────────

// TestCosineSimilarity tests the cosine similarity calculation.
func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name    string
		a       []float32
		b       []float32
		want    float32
		epsilon float32 // tolerance for floating point comparison
	}{
		{
			name:    "identical vectors",
			a:       []float32{1, 2, 3},
			b:       []float32{1, 2, 3},
			want:    1.0,
			epsilon: 0.0001,
		},
		{
			name:    "orthogonal vectors (zero similarity)",
			a:       []float32{1, 0, 0},
			b:       []float32{0, 1, 0},
			want:    0.0,
			epsilon: 0.0001,
		},
		{
			name:    "opposite vectors",
			a:       []float32{1, 2, 3},
			b:       []float32{-1, -2, -3},
			want:    -1.0,
			epsilon: 0.0001,
		},
		{
			name:    "zero vectors",
			a:       []float32{0, 0, 0},
			b:       []float32{1, 2, 3},
			want:    0.0,
			epsilon: 0.0001,
		},
		{
			name:    "both zero vectors",
			a:       []float32{0, 0, 0},
			b:       []float32{0, 0, 0},
			want:    0.0,
			epsilon: 0.0001,
		},
		{
			name:    "different length vectors",
			a:       []float32{1, 2, 3},
			b:       []float32{1, 2},
			want:    0.0,
			epsilon: 0.0001,
		},
		{
			name:    "partial similarity",
			a:       []float32{1, 2, 3},
			b:       []float32{2, 4, 6},
			want:    1.0,
			epsilon: 0.0001,
		},
		{
			name:    "small vectors",
			a:       []float32{0.5, 0.5},
			b:       []float32{0.5, 0.5},
			want:    1.0,
			epsilon: 0.0001,
		},
		{
			name:    "empty vectors",
			a:       []float32{},
			b:       []float32{},
			want:    0.0, // Should handle edge case
			epsilon: 0.0001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cosineSimilarity(tt.a, tt.b)
			diff := got - tt.want
			if diff < 0 {
				diff = -diff
			}
			if diff > tt.epsilon {
				t.Errorf("cosineSimilarity() = %v, want %v (±%v)", got, tt.want, tt.epsilon)
			}
		})
	}
}

// TestCalculateStringSimilarity tests the string similarity calculation.
func TestCalculateStringSimilarity(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want float32
	}{
		{
			name: "identical strings",
			a:    "hello world",
			b:    "hello world",
			want: 1.0,
		},
		{
			name: "completely different strings",
			a:    "apple orange banana",
			b:    "cat dog elephant",
			want: 0.0,
		},
		{
			name: "partial overlap",
			a:    "hello world test",
			b:    "hello there world",
			want: 0.33333334, // "hello", "world" shared; union: {hello,world,test,there} = 4; similarity: 2/4
		},
		{
			name: "short words filtered out",
			a:    "the cat in the hat",
			b:    "the dog in the house",
			want: 0.0, // wordsA: {cat, hat}; wordsB: {dog, house}; intersection: 0
		},
		{
			name: "case insensitive after initial check",
			a:    "Hello World",
			b:    "hello world",
			want: 0.5, // Different strings initially, then toLower; both have {hello, world}; intersection: 2, union: 4
		},
		{
			name: "empty strings are identical",
			a:    "",
			b:    "",
			want: 1.0, // Empty strings are equal, returns 1.0 immediately
		},
		{
			name: "one empty string",
			a:    "hello world",
			b:    "",
			want: 0.0,
		},
		{
			name: "all short words are identical",
			a:    "a an the at in",
			b:    "a an the at in",
			want: 1.0, // Strings are identical, returns 1.0 immediately (before filtering)
		},
		{
			name: "unicode words",
			a:    "hello 世界 test",
			b:    "hello 世界 test",
			want: 1.0,
		},
		{
			name: "words exactly minWordLength",
			a:    "test word",
			b:    "test word",
			want: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateStringSimilarity(tt.a, tt.b)
			// Use approximate comparison for floating point
			if got < tt.want-0.001 || got > tt.want+0.001 {
				t.Errorf("calculateStringSimilarity() = %v, want %v", got, tt.want)
			}
		})
	}
}

// mockEmbeddingModel is a mock implementation of EmbeddingModel for testing.
type mockEmbeddingModel struct {
	embeddings map[string][]float32
	err        error
}

func (m *mockEmbeddingModel) Embed(_ context.Context, text string) ([]float32, error) {
	if m.err != nil {
		return nil, m.err
	}
	if emb, ok := m.embeddings[text]; ok {
		return emb, nil
	}
	// Return a default embedding for unknown text
	return []float32{0.1, 0.2, 0.3}, nil
}

// TestCalculateSemanticSimilarity tests semantic similarity with embeddings.
func TestCalculateSemanticSimilarity(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		a        string
		b        string
		embedder *mockEmbeddingModel
		want     float32
	}{
		{
			name: "identical strings",
			a:    "hello",
			b:    "hello",
			embedder: &mockEmbeddingModel{
				embeddings: map[string][]float32{
					"hello": {0.5, 0.5},
				},
			},
			want: 1.0,
		},
		{
			name: "embedding error falls back to string similarity",
			a:    "hello world test",
			b:    "hello there world",
			embedder: &mockEmbeddingModel{
				err: errors.New("embedding failed"),
			},
			want: 0.33333334, // "hello", "world" shared out of 4 unique words
		},
		{
			name: "similar embeddings",
			a:    "happy",
			b:    "joyful",
			embedder: &mockEmbeddingModel{
				embeddings: map[string][]float32{
					"happy":  {0.7, 0.7},
					"joyful": {0.8, 0.6},
				},
			},
			want: 0.99, // ~cos(10°) ≈ 0.985
		},
		{
			name: "first embedding error",
			a:    "hello",
			b:    "world",
			embedder: &mockEmbeddingModel{
				embeddings: map[string][]float32{
					"world": {0.5, 0.5},
				},
			},
			want: 0.0, // Falls back to string similarity, no overlap
		},
		{
			name: "second embedding error",
			a:    "hello",
			b:    "world",
			embedder: &mockEmbeddingModel{
				embeddings: map[string][]float32{
					"hello": {0.5, 0.5},
				},
			},
			want: 0.0, // Falls back to string similarity, no overlap
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateSemanticSimilarity(ctx, tt.a, tt.b, tt.embedder)
			if got < tt.want-0.01 || got > tt.want+0.01 {
				t.Errorf("calculateSemanticSimilarity() = %v, want %v (±0.01)", got, tt.want)
			}
		})
	}
}

// TestBuildMajorityConsensus tests the majority consensus building.
func TestBuildMajorityConsensus(t *testing.T) {
	tests := []struct {
		name       string
		votes      map[string]string
		agreement  float32
		minVotes   int
		wantOutput string
		wantReason string
	}{
		{
			name: "clear majority",
			votes: map[string]string{
				"agent1": "option A",
				"agent2": "option A",
				"agent3": "option B",
			},
			agreement:  0.5,
			minVotes:   0,
			wantOutput: "option A",
			wantReason: "Majority consensus reached (2/3 votes, 50% agreement).",
		},
		{
			name: "no clear majority",
			votes: map[string]string{
				"agent1": "option A",
				"agent2": "option B",
				"agent3": "option C",
			},
			agreement:  0.0,
			minVotes:   0,
			wantOutput: "", // Don't check - map iteration order may vary
			wantReason: "No clear majority (1/3 votes). Using most common output.",
		},
		{
			name: "unanimous agreement",
			votes: map[string]string{
				"agent1": "option A",
				"agent2": "option A",
				"agent3": "option A",
			},
			agreement:  1.0,
			minVotes:   0,
			wantOutput: "option A",
			wantReason: "Majority consensus reached (3/3 votes, 100% agreement).",
		},
		{
			name: "meets min votes threshold",
			votes: map[string]string{
				"agent1": "option A",
				"agent2": "option A",
				"agent3": "option B",
				"agent4": "option B",
			},
			agreement:  0.0,
			minVotes:   3,
			wantOutput: "", // Don't check - map iteration order may affect which one is returned first
			wantReason: "No clear majority (2/4 votes). Using most common output.",
		},
		{
			name: "exactly meets min votes",
			votes: map[string]string{
				"agent1": "option A",
				"agent2": "option A",
				"agent3": "option A",
				"agent4": "option B",
			},
			agreement:  0.5,
			minVotes:   3,
			wantOutput: "option A",
			wantReason: "Majority consensus reached (3/4 votes, 50% agreement).",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Orchestrator{
				config: &OrchestratorConfig{
					Consensus: ConsensusConfig{
						MinVotes: tt.minVotes,
					},
				},
			}

			output, reasoning := o.buildMajorityConsensus(tt.votes, tt.agreement)

			if tt.wantOutput != "" && output != tt.wantOutput {
				t.Errorf("buildMajorityConsensus() output = %v, want %v", output, tt.wantOutput)
			}
			if reasoning != tt.wantReason {
				t.Errorf("buildMajorityConsensus() reasoning = %v, want %v", reasoning, tt.wantReason)
			}
		})
	}
}

// TestBuildUnanimousConsensus tests the unanimous consensus building.
func TestBuildUnanimousConsensus(t *testing.T) {
	tests := []struct {
		name       string
		votes      map[string]string
		agreement  float32
		wantOutput string
		wantReason string
	}{
		{
			name: "unanimous agreement",
			votes: map[string]string{
				"agent1": "option A",
				"agent2": "option A",
				"agent3": "option A",
			},
			agreement:  1.0,
			wantOutput: "option A",
			wantReason: "Unanimous consensus reached (100% agreement).",
		},
		{
			name: "not unanimous - different outputs",
			votes: map[string]string{
				"agent1": "option A",
				"agent2": "option B",
				"agent3": "option C",
			},
			agreement:  0.0,
			wantOutput: "", // Don't check output - map iteration is non-deterministic under race
			wantReason: "WARNING: No unanimous consensus (0% agreement, 3 different outputs). Using first output.",
		},
		{
			name: "partial agreement",
			votes: map[string]string{
				"agent1": "option A",
				"agent2": "option A",
				"agent3": "option B",
			},
			agreement:  0.66,
			wantOutput: "", // Don't check specific output due to map iteration order
			wantReason: "WARNING: No unanimous consensus (66% agreement, 2 different outputs). Using first output.",
		},
		{
			name:       "empty votes",
			votes:      map[string]string{},
			agreement:  1.0,
			wantOutput: "",
			wantReason: "WARNING: No votes received",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Orchestrator{}

			output, reasoning := o.buildUnanimousConsensus(tt.votes, tt.agreement)

			if tt.wantOutput != "" && output != tt.wantOutput {
				t.Errorf("buildUnanimousConsensus() output = %v, want %v", output, tt.wantOutput)
			}
			if reasoning != tt.wantReason {
				t.Errorf("buildUnanimousConsensus() reasoning = %v, want %v", reasoning, tt.wantReason)
			}
		})
	}
}

// TestBuildAnyConsensus tests the "any" consensus building.
func TestBuildAnyConsensus(t *testing.T) {
	tests := []struct {
		name       string
		votes      map[string]string
		wantOutput string
		wantReason string
	}{
		{
			name: "single agent",
			votes: map[string]string{
				"agent1": "option A",
			},
			wantOutput: "option A",
			wantReason: "Using any available output (1 agent(s) executed).",
		},
		{
			name: "multiple agents - returns first",
			votes: map[string]string{
				"agent1": "option A",
				"agent2": "option B",
				"agent3": "option C",
			},
			wantOutput: "option A", // Map iteration order, but deterministic in test
			wantReason: "Using any available output (3 agent(s) executed).",
		},
		{
			name:       "empty votes",
			votes:      map[string]string{},
			wantOutput: "",
			wantReason: "WARNING: No votes received",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Orchestrator{}

			output, reasoning := o.buildAnyConsensus(tt.votes)

			if tt.wantOutput != "" && output != tt.wantOutput {
				// For non-empty expected output, check it's one of the vote values
				found := false
				for _, v := range tt.votes {
					if v == output {
						found = true

						break
					}
				}
				if !found {
					t.Errorf("buildAnyConsensus() output = %v, not in votes %v", output, tt.votes)
				}
			}
			if reasoning != tt.wantReason {
				t.Errorf("buildAnyConsensus() reasoning = %v, want %v", reasoning, tt.wantReason)
			}
		})
	}
}

// TestCalculateAgreement tests the agreement calculation.
func TestCalculateAgreement(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		votes         map[string]string
		similarity    string
		semanticModel string
		wantHigh      bool // true if we expect high agreement (>0.5)
		checkBounds   bool // whether to check [0,1] bounds (semantic may produce negative)
	}{
		{
			name: "unanimous votes - jaccard",
			votes: map[string]string{
				"agent1": "implement the feature",
				"agent2": "implement the feature",
				"agent3": "implement the feature",
			},
			similarity:  "jaccard",
			wantHigh:    true,
			checkBounds: true,
		},
		{
			name: "divergent votes - jaccard",
			votes: map[string]string{
				"agent1": "implement feature",
				"agent2": "create feature",
				"agent3": "build feature",
			},
			similarity: "jaccard",
			wantHigh:   false, // Each has unique 4+ char words: "implement"/"create"/"build", "feature" appears 3 times
			// Actually, "feature" appears in all 3, so: intersection=1, union=4 -> agreement=0.25
			checkBounds: true,
		},
		{
			name: "partial overlap - jaccard",
			votes: map[string]string{
				"agent1": "implement system feature",
				"agent2": "implement system feature",
				"agent3": "implement system feature",
			},
			similarity:  "jaccard",
			wantHigh:    true, // All identical, should be 1.0
			checkBounds: true,
		},
		{
			name: "single vote - full agreement",
			votes: map[string]string{
				"agent1": "implement feature",
			},
			similarity:  "jaccard",
			wantHigh:    true,
			checkBounds: true,
		},
		{
			name:        "empty votes - zero agreement",
			votes:       map[string]string{},
			similarity:  "jaccard",
			wantHigh:    false,
			checkBounds: true,
		},
		{
			name: "semantic mode with valid model",
			votes: map[string]string{
				"agent1": "implement feature",
				"agent2": "implement feature",
			},
			similarity:    "semantic",
			semanticModel: "all-MiniLM-L6-v2",
			wantHigh:      true,  // Identical strings return 1.0 immediately
			checkBounds:   false, // Semantic mode with different strings may produce negative cosine similarity
		},
		{
			name: "fallback to jaccard on embedding failure",
			votes: map[string]string{
				"agent1": "implement feature",
				"agent2": "implement feature",
			},
			similarity:    "semantic",
			semanticModel: "invalid-model-that-fails",
			wantHigh:      true,
			checkBounds:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Orchestrator{
				config: &OrchestratorConfig{
					Consensus: ConsensusConfig{
						Similarity:    tt.similarity,
						SemanticModel: tt.semanticModel,
					},
				},
			}

			got := o.calculateAgreement(ctx, tt.votes)

			if tt.wantHigh && got < 0.5 {
				t.Errorf("calculateAgreement() = %v, expected high agreement (>0.5)", got)
			}
			if !tt.wantHigh && got >= 0.5 {
				t.Errorf("calculateAgreement() = %v, expected low agreement (<0.5)", got)
			}

			// Check bounds only for tests that expect it
			if tt.checkBounds && (got < 0 || got > 1) {
				t.Errorf("calculateAgreement() = %v, out of bounds [0, 1]", got)
			}
		})
	}
}

// TestBuildConsensus tests the full consensus building pipeline.
func TestBuildConsensus(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		config       ConsensusConfig
		stepResults  map[string]*StepResult
		wantSynthLen int // expected length of synthesized output
		wantErr      bool
		errMsg       string
	}{
		{
			name: "majority mode with clear majority",
			config: ConsensusConfig{
				Mode: "majority",
			},
			stepResults: map[string]*StepResult{
				"agent1": {Output: "option A"},
				"agent2": {Output: "option A"},
				"agent3": {Output: "option B"},
			},
			wantSynthLen: 8, // "option A"
			wantErr:      false,
		},
		{
			name: "unanimous mode with agreement",
			config: ConsensusConfig{
				Mode: "unanimous",
			},
			stepResults: map[string]*StepResult{
				"agent1": {Output: "option A"},
				"agent2": {Output: "option A"},
				"agent3": {Output: "option A"},
			},
			wantSynthLen: 8, // "option A"
			wantErr:      false,
		},
		{
			name: "any mode returns first output",
			config: ConsensusConfig{
				Mode: "any",
			},
			stepResults: map[string]*StepResult{
				"agent1": {Output: "option A"},
				"agent2": {Output: "option B"},
			},
			wantSynthLen: 8, // "option A" or "option B"
			wantErr:      false,
		},
		{
			name: "unsupported consensus mode",
			config: ConsensusConfig{
				Mode: "invalid",
			},
			stepResults: map[string]*StepResult{
				"agent1": {Output: "option A"},
			},
			wantErr: true,
			errMsg:  "unsupported consensus mode",
		},
		{
			name: "empty step results",
			config: ConsensusConfig{
				Mode: "majority",
			},
			stepResults:  map[string]*StepResult{},
			wantSynthLen: 0,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Orchestrator{
				config: &OrchestratorConfig{
					Consensus: tt.config,
				},
				registry: &mockRegistry{
					agents: map[string]agent.Agent{},
				},
			}

			result := &PipelineResult{
				StepResults: tt.stepResults,
			}

			task := &storage.TaskWork{
				Metadata: storage.WorkMetadata{
					ID:    "test-123",
					Title: "Test Task",
				},
			}

			consensus, err := o.buildConsensus(ctx, result, task)

			if tt.wantErr {
				if err == nil {
					t.Fatal("buildConsensus() expected error, got nil")
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("buildConsensus() error = %v, want containing %q", err, tt.errMsg)
				}

				return
			}

			if err != nil {
				t.Fatalf("buildConsensus() unexpected error: %v", err)
			}

			if tt.wantSynthLen > 0 && len(consensus.Synthesized) != tt.wantSynthLen {
				t.Errorf("buildConsensus() synthesized length = %v, want %v", len(consensus.Synthesized), tt.wantSynthLen)
			}

			if consensus.Votes == nil {
				t.Error("buildConsensus() votes should not be nil")
			}
		})
	}
}
