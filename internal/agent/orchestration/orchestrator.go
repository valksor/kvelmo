package orchestration

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/valksor/go-mehrhof/internal/agent"
	"github.com/valksor/go-mehrhof/internal/storage"
)

// NewOrchestrator creates a new multi-agent orchestrator.
func NewOrchestrator(config *OrchestratorConfig, workspace *storage.Workspace, registry AgentRegistry) (*Orchestrator, error) {
	// Validate config
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &Orchestrator{
		config:   config,
		storage:  workspace,
		registry: registry,
	}, nil
}

// validateConfig validates the orchestrator configuration.
func validateConfig(config *OrchestratorConfig) error {
	if config == nil {
		return errors.New("config is required")
	}

	// Validate mode
	switch config.Mode {
	case ModeSingle, ModeSequential, ModeParallel, ModeConsensus:
		// Valid modes
	default:
		return fmt.Errorf("invalid mode: %s", config.Mode)
	}

	// For multi-agent modes, validate agents
	if config.Mode != ModeSingle {
		if len(config.Agents) == 0 {
			return fmt.Errorf("at least one agent is required for mode: %s", config.Mode)
		}

		// Check for duplicate step names
		names := make(map[string]bool)
		for _, step := range config.Agents {
			if step.Name == "" {
				return errors.New("agent step must have a name")
			}
			if names[step.Name] {
				return fmt.Errorf("duplicate step name: %s", step.Name)
			}
			names[step.Name] = true
		}

		// Validate consensus config for consensus mode
		if config.Mode == ModeConsensus {
			if config.Consensus.Mode == "" {
				return errors.New("consensus mode is required for consensus orchestration")
			}
		}
	}

	return nil
}

// Run executes the orchestration based on the configured mode.
func (o *Orchestrator) Run(ctx context.Context, task *storage.TaskWork) (*PipelineResult, error) {
	startTime := time.Now()
	result := &PipelineResult{
		StepResults: make(map[string]*StepResult),
	}

	var err error
	switch o.config.Mode {
	case ModeSingle:
		// Single agent mode - use the first agent
		err = o.runSingle(ctx, task, result)
	case ModeSequential:
		// Execute agents sequentially, passing outputs
		err = o.runSequential(ctx, task, result)
	case ModeParallel:
		// Execute agents in parallel
		err = o.runParallel(ctx, task, result)
	case ModeConsensus:
		// Execute agents in parallel and build consensus
		err = o.runConsensus(ctx, task, result)
	default:
		return nil, fmt.Errorf("unsupported orchestration mode: %s", o.config.Mode)
	}

	result.Duration = time.Since(startTime)

	if err != nil {
		result.Errors = append(result.Errors, err)

		return result, err
	}

	return result, nil
}

// runSingle executes a single agent (standard behavior).
func (o *Orchestrator) runSingle(ctx context.Context, task *storage.TaskWork, result *PipelineResult) error {
	if len(o.config.Agents) == 0 {
		return errors.New("no agent configured")
	}

	step := o.config.Agents[0]
	stepResult, err := o.executeStep(ctx, step, task, nil)
	if err != nil {
		return err
	}

	result.StepResults[step.Name] = stepResult
	result.FinalOutput = stepResult.Output

	return nil
}

// runSequential executes agents one after another, passing outputs.
func (o *Orchestrator) runSequential(ctx context.Context, task *storage.TaskWork, result *PipelineResult) error {
	// Build dependency graph and validate execution order
	if err := o.validateDependencies(); err != nil {
		return err
	}

	// Execute steps in order
	executed := make(map[string]bool)
	artifacts := make(map[string]string)

	for _, step := range o.config.Agents {
		// Check if dependencies are met
		for _, dep := range step.Depends {
			if !executed[dep] {
				return fmt.Errorf("dependency not met: %s depends on %s", step.Name, dep)
			}
		}

		// Gather input artifacts
		stepInputs := o.gatherInputs(step, artifacts)

		// Execute step
		stepResult, err := o.executeStep(ctx, step, task, stepInputs)
		if err != nil {
			return fmt.Errorf("step %s failed: %w", step.Name, err)
		}

		result.StepResults[step.Name] = stepResult

		// Store output artifact
		if step.Output != "" {
			artifacts[step.Output] = stepResult.Output
		}

		executed[step.Name] = true
	}

	// Set final output from last step
	if len(o.config.Agents) > 0 {
		lastStep := o.config.Agents[len(o.config.Agents)-1]
		if lastResult, ok := result.StepResults[lastStep.Name]; ok {
			result.FinalOutput = lastResult.Output
		}
	}

	return nil
}

// runParallel executes agents concurrently.
func (o *Orchestrator) runParallel(ctx context.Context, task *storage.TaskWork, result *PipelineResult) error {
	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := make([]error, len(o.config.Agents))

	for i, step := range o.config.Agents {
		wg.Add(1)
		go func(idx int, s AgentStep) {
			defer wg.Done()

			// Check for context cancellation before starting
			select {
			case <-ctx.Done():
				mu.Lock()
				errors[idx] = fmt.Errorf("step %s cancelled: %w", s.Name, ctx.Err())
				mu.Unlock()

				return
			default:
			}

			stepResult, err := o.executeStep(ctx, s, task, nil)
			mu.Lock()
			if err != nil {
				errors[idx] = fmt.Errorf("step %s failed: %w", s.Name, err)
				mu.Unlock()

				return
			}

			result.StepResults[s.Name] = stepResult
			mu.Unlock()
		}(i, step)
	}

	wg.Wait()

	// Check for errors - mu no longer needed here as all goroutines have completed
	for _, err := range errors {
		if err != nil {
			result.Errors = append(result.Errors, err)
		}
	}

	if len(result.Errors) > 0 {
		return fmt.Errorf("parallel execution completed with %d error(s)", len(result.Errors))
	}

	return nil
}

// runConsensus executes agents in parallel and builds consensus.
func (o *Orchestrator) runConsensus(ctx context.Context, task *storage.TaskWork, result *PipelineResult) error {
	// First run all agents in parallel
	if err := o.runParallel(ctx, task, result); err != nil {
		return err
	}

	// Build consensus from results
	consensus, err := o.buildConsensus(ctx, result, task)
	if err != nil {
		return fmt.Errorf("build consensus: %w", err)
	}

	result.Consensus = consensus.Agreement
	result.FinalOutput = consensus.Synthesized

	return nil
}

// validateDependencies validates that all dependencies exist.
func (o *Orchestrator) validateDependencies() error {
	stepNames := make(map[string]bool)
	for _, step := range o.config.Agents {
		stepNames[step.Name] = true
	}

	for _, step := range o.config.Agents {
		for _, dep := range step.Depends {
			if !stepNames[dep] {
				return fmt.Errorf("step %s depends on non-existent step: %s", step.Name, dep)
			}
		}
	}

	return nil
}

// gatherInputs collects input artifacts for a step.
func (o *Orchestrator) gatherInputs(step AgentStep, artifacts map[string]string) []string {
	var inputs []string

	for _, inputName := range step.Input {
		if artifact, ok := artifacts[inputName]; ok {
			inputs = append(inputs, artifact)
		}
	}

	return inputs
}

// executeStep executes a single agent step.
func (o *Orchestrator) executeStep(ctx context.Context, step AgentStep, task *storage.TaskWork, inputs []string) (*StepResult, error) {
	startTime := time.Now()

	// Get agent from registry
	agentInst, err := o.registry.Get(step.Agent)
	if err != nil {
		return nil, fmt.Errorf("get agent %s: %w", step.Agent, err)
	}

	// Apply step-specific configuration
	if len(step.Env) > 0 {
		for k, v := range step.Env {
			agentInst = agentInst.WithEnv(k, v)
		}
	}

	if len(step.Args) > 0 {
		agentInst = agentInst.WithArgs(step.Args...)
	}

	// Apply timeout if specified
	if step.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(step.Timeout)*time.Second)
		defer cancel()
	}

	// Prepare task context
	taskCtx := TaskContext{
		TaskID:  task.Metadata.ID,
		Title:   task.Metadata.Title,
		State:   "implementing",
		Inputs:  inputs,
		WorkDir: o.storage.WorkPath(task.Metadata.ID),
	}

	// Execute agent
	output, tokenUsage, cost, err := executeAgent(ctx, agentInst, &taskCtx, step.Role)

	// Create result
	result := &StepResult{
		StepName:   step.Name,
		AgentName:  step.Agent,
		Output:     output,
		Artifacts:  make(map[string]string),
		Duration:   time.Since(startTime),
		TokenUsage: tokenUsage,
		CostUSD:    cost,
	}

	// Handle error - set in result and return
	if err != nil {
		result.Error = err

		return result, err
	}

	// Store output artifact if named
	if step.Output != "" {
		result.Artifacts[step.Output] = output
	}

	return result, nil
}

// executeAgent executes an agent with the given context.
func executeAgent(ctx context.Context, agentInst agent.Agent, taskCtx *TaskContext, role string) (string, int, float64, error) {
	// Build prompt from task context
	prompt := buildAgentPrompt(taskCtx, role)

	// Execute the agent
	response, err := agentInst.Run(ctx, prompt)
	if err != nil {
		return "", 0, 0, fmt.Errorf("agent run: %w", err)
	}

	// Extract output from response
	output := response.Summary
	if output == "" && len(response.Messages) > 0 {
		output = response.Messages[0]
	}

	// Extract token usage and cost
	tokenUsage := 0
	var costUSD float64
	if response.Usage != nil {
		tokenUsage = response.Usage.InputTokens + response.Usage.OutputTokens + response.Usage.CachedTokens
		costUSD = response.Usage.CostUSD
	}

	return output, tokenUsage, costUSD, nil
}

// buildAgentPrompt builds a prompt for the agent from the task context.
func buildAgentPrompt(taskCtx *TaskContext, role string) string {
	var prompt strings.Builder

	prompt.WriteString(fmt.Sprintf("# Task: %s\n\n", taskCtx.Title))

	if role != "" {
		prompt.WriteString(fmt.Sprintf("Your role: %s\n\n", role))
	}

	if taskCtx.State != "" {
		prompt.WriteString(fmt.Sprintf("Current state: %s\n\n", taskCtx.State))
	}

	if len(taskCtx.Inputs) > 0 {
		prompt.WriteString("## Context from previous steps:\n\n")
		for i, input := range taskCtx.Inputs {
			prompt.WriteString(fmt.Sprintf("### Input %d\n%s\n\n", i+1, input))
		}
	}

	prompt.WriteString("Please complete your task based on the context above.")

	return prompt.String()
}
