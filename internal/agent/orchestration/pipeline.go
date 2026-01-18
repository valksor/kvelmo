package orchestration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/valksor/go-mehrhof/internal/storage"
)

// Pipeline manages data flow between agent steps.
type Pipeline struct {
	storage   *storage.Workspace
	taskID    string
	artifacts map[string]string
	mu        sync.RWMutex // Protects artifacts map
}

// NewPipeline creates a new pipeline for a task.
func NewPipeline(storage *storage.Workspace, taskID string) *Pipeline {
	return &Pipeline{
		storage:   storage,
		taskID:    taskID,
		artifacts: make(map[string]string),
	}
}

// GetArtifact retrieves an artifact by name.
func (p *Pipeline) GetArtifact(name string) (string, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	artifact, ok := p.artifacts[name]

	return artifact, ok
}

// SetArtifact stores an artifact.
func (p *Pipeline) SetArtifact(name, content string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.artifacts[name] = content
}

// SaveArtifact saves an artifact to a file.
func (p *Pipeline) SaveArtifact(name, content string) error {
	workPath := p.storage.WorkPath(p.taskID)
	artifactDir := filepath.Join(workPath, ".artifacts")

	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		return fmt.Errorf("create artifact directory: %w", err)
	}

	artifactPath := filepath.Join(artifactDir, name)
	if err := os.WriteFile(artifactPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write artifact: %w", err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	p.artifacts[name] = artifactPath

	return nil
}

// LoadArtifact loads an artifact from a file.
func (p *Pipeline) LoadArtifact(name string) (string, error) {
	// First check with read lock for fast path
	p.mu.RLock()
	artifact, ok := p.artifacts[name]
	p.mu.RUnlock()

	if ok {
		return artifact, nil
	}

	// Acquire write lock before reading file to prevent multiple goroutines from reading the same file
	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check: another goroutine might have loaded it while we were acquiring the lock
	if artifact, ok := p.artifacts[name]; ok {
		return artifact, nil
	}

	// Load from file while holding lock
	workPath := p.storage.WorkPath(p.taskID)
	artifactPath := filepath.Join(workPath, ".artifacts", name)

	content, err := os.ReadFile(artifactPath)
	if err != nil {
		return "", fmt.Errorf("read artifact: %w", err)
	}

	artifactStr := string(content)
	p.artifacts[name] = artifactStr

	return artifactStr, nil
}

// ListArtifacts returns all artifact names.
func (p *Pipeline) ListArtifacts() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	names := make([]string, 0, len(p.artifacts))
	for name := range p.artifacts {
		names = append(names, name)
	}

	return names
}

// Clear removes all artifacts.
func (p *Pipeline) Clear() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.artifacts = make(map[string]string)
}

// BuildInput constructs input for an agent step from artifacts.
func (p *Pipeline) BuildInput(step AgentStep) (string, error) {
	if len(step.Input) == 0 {
		return "", nil
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	var sb strings.Builder

	// Add role/instructions
	if step.Role != "" {
		sb.WriteString(fmt.Sprintf("Role: %s\n\n", step.Role))
	}

	// Add inputs from artifacts
	for i, inputName := range step.Input {
		artifact, ok := p.artifacts[inputName]
		if !ok {
			return "", fmt.Errorf("input artifact not found: %s", inputName)
		}

		if i > 0 {
			sb.WriteString("\n\n---\n\n")
		}

		sb.WriteString(fmt.Sprintf("## Input from %s\n", inputName))
		sb.WriteString(artifact)
	}

	return sb.String(), nil
}

// ExecuteWithPipeline executes an agent step with pipeline context.
func (p *Pipeline) ExecuteWithPipeline(ctx context.Context, step AgentStep, registry AgentRegistry, task *storage.TaskWork) (*StepResult, error) {
	startTime := time.Now()

	// Build input from pipeline artifacts
	input, err := p.BuildInput(step)
	if err != nil {
		return nil, fmt.Errorf("build input: %w", err)
	}

	// Get agent
	agentInst, err := registry.Get(step.Agent)
	if err != nil {
		return nil, fmt.Errorf("get agent: %w", err)
	}

	// Apply step configuration
	if len(step.Env) > 0 {
		for k, v := range step.Env {
			agentInst = agentInst.WithEnv(k, v)
		}
	}

	if len(step.Args) > 0 {
		agentInst = agentInst.WithArgs(step.Args...)
	}

	// Prepare task context
	taskCtx := TaskContext{
		TaskID:  task.Metadata.ID,
		Title:   task.Metadata.Title,
		State:   "implementing",
		Inputs:  []string{input},
		WorkDir: p.storage.WorkPath(p.taskID),
	}

	// Combine role with input
	fullInput := step.Role
	if input != "" {
		fullInput = step.Role + "\n\n" + input
	}

	// Execute agent
	output, tokenUsage, cost, err := executeAgent(ctx, agentInst, &taskCtx, fullInput)

	result := &StepResult{
		StepName:   step.Name,
		AgentName:  step.Agent,
		Output:     output,
		Artifacts:  make(map[string]string),
		Duration:   time.Since(startTime),
		TokenUsage: tokenUsage,
		CostUSD:    cost,
	}

	if err != nil {
		result.Error = err

		return result, fmt.Errorf("agent execution failed: %w", err)
	}

	// Store output as artifact if specified
	if step.Output != "" {
		p.SetArtifact(step.Output, output)
		result.Artifacts[step.Output] = output

		// Also save to file
		if err := p.SaveArtifact(step.Output, output); err != nil {
			return result, fmt.Errorf("save artifact: %w", err)
		}
	}

	return result, nil
}
