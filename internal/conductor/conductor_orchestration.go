package conductor

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/valksor/go-mehrhof/internal/agent/orchestration"
	"github.com/valksor/go-mehrhof/internal/storage"
)

// getOrchestratorConfig returns the orchestration configuration for a step.
// Returns ad-hoc config from ParallelCount option if set, otherwise uses YAML config.
func (c *Conductor) getOrchestratorConfig(step string) (*orchestration.OrchestratorConfig, bool) {
	// Check for ad-hoc parallel execution from ParallelCount option
	if c.opts.ParallelCount != "" {
		if adHocConfig := c.createAdHocParallelConfig(step, c.opts.ParallelCount); adHocConfig != nil {
			return adHocConfig, true
		}
	}

	cfg, err := c.workspace.LoadConfig()
	if err != nil {
		return nil, false
	}

	if cfg.Orchestration == nil || !cfg.Orchestration.Enabled {
		return nil, false
	}

	stepConfig, ok := cfg.Orchestration.Steps[step]
	if !ok {
		return nil, false
	}

	// Convert storage config to orchestration config
	return convertOrchestratorConfig(&stepConfig, step), true
}

// createAdHocParallelConfig creates an ad-hoc orchestration config from ParallelCount option.
// ParallelCount can be:
//   - A number like "2", "3", "4" - runs N agents in parallel using the default agent
//   - Comma-separated agent names like "claude,gemini" - runs specific agents in parallel
func (c *Conductor) createAdHocParallelConfig(_ string, parallelCount string) *orchestration.OrchestratorConfig {
	var agentNames []string
	mode := orchestration.ModeParallel

	// Parse parallel count - could be a number or comma-separated agents
	if strings.Contains(parallelCount, ",") {
		// Comma-separated agent names
		agentNames = strings.Split(parallelCount, ",")
		for i, name := range agentNames {
			agentNames[i] = strings.TrimSpace(name)
		}
	} else {
		// Numeric count - use default agent that many times
		count, err := strconv.Atoi(parallelCount)
		if err != nil || count < 2 {
			// Invalid or less than 2, no point in parallel
			return nil
		}

		// Get default agent name from coordination or use first available
		defaultAgent := c.getDefaultAgentName()
		if defaultAgent == "" {
			return nil
		}

		for range count {
			agentNames = append(agentNames, defaultAgent)
		}
	}

	// Create agent steps for parallel execution
	config := &orchestration.OrchestratorConfig{
		Mode:   mode,
		Agents: make([]orchestration.AgentStep, 0, len(agentNames)),
	}

	for i, agentName := range agentNames {
		config.Agents = append(config.Agents, orchestration.AgentStep{
			Name:  fmt.Sprintf("parallel-%d", i+1),
			Agent: agentName,
			Role:  fmt.Sprintf("Parallel agent %d", i+1),
		})
	}

	return config
}

// getDefaultAgentName returns the default agent name for parallel execution.
// Tries to get the step-specific agent, then default agent from config.
func (c *Conductor) getDefaultAgentName() string {
	// Handle nil workspace (e.g., in tests)
	if c.workspace == nil {
		return "claude"
	}

	// Try to get agent from coordination
	cfg, err := c.workspace.LoadConfig()
	if err != nil {
		return "claude"
	}

	// Check for step-specific agent
	if cfg.Agent.Steps != nil {
		if stepConfig, ok := cfg.Agent.Steps["implementing"]; ok && stepConfig.Name != "" {
			return stepConfig.Name
		}
	}

	// Check for default agent
	if cfg.Agent.Default != "" {
		return cfg.Agent.Default
	}

	// Fallback to "claude" as the default
	return "claude"
}

// convertOrchestratorConfig converts storage orchestration config to internal format.
func convertOrchestratorConfig(cfg *storage.StepOrchestratorConfig, _ string) *orchestration.OrchestratorConfig {
	config := &orchestration.OrchestratorConfig{
		Mode: orchestration.OrchestratorMode(cfg.Mode),
	}

	// Convert agents
	for _, agentCfg := range cfg.Agents {
		config.Agents = append(config.Agents, orchestration.AgentStep{
			Name:    agentCfg.Name,
			Agent:   agentCfg.Agent,
			Model:   agentCfg.Model,
			Role:    agentCfg.Role,
			Input:   agentCfg.Input,
			Output:  agentCfg.Output,
			Depends: agentCfg.Depends,
			Env:     agentCfg.Env,
			Args:    agentCfg.Args,
			Timeout: agentCfg.Timeout,
		})
	}

	// Convert consensus config
	if cfg.Consensus.Mode != "" {
		config.Consensus = orchestration.ConsensusConfig{
			Mode:        cfg.Consensus.Mode,
			MinVotes:    cfg.Consensus.MinVotes,
			Synthesizer: cfg.Consensus.Synthesizer,
		}
	}

	return config
}

// ShouldUseOrchestration checks if orchestration should be used for a step.
func (c *Conductor) ShouldUseOrchestration(step string) bool {
	_, ok := c.getOrchestratorConfig(step)

	return ok
}

// isOrchestrationEnabledForPhase checks if orchestration is configured for a phase.
func (c *Conductor) isOrchestrationEnabledForPhase(phase string) bool {
	return c.ShouldUseOrchestration(phase)
}

// runOrchestratedStep executes a workflow step using multi-agent orchestration.
func (c *Conductor) runOrchestratedStep(ctx context.Context, step string, work *storage.TaskWork) (*orchestration.PipelineResult, error) {
	// Get orchestration config
	orchestratorConfig, ok := c.getOrchestratorConfig(step)
	if !ok {
		return nil, fmt.Errorf("no orchestration configuration for step: %s", step)
	}

	c.publishProgress(fmt.Sprintf("Running %s with multi-agent orchestration (mode: %s)", step, orchestratorConfig.Mode), 0)

	// Create orchestrator
	orchestrator, err := orchestration.NewOrchestrator(orchestratorConfig, c.workspace, c.agents)
	if err != nil {
		return nil, fmt.Errorf("create orchestrator: %w", err)
	}

	// Run orchestration
	result, err := orchestrator.Run(ctx, work)
	if err != nil {
		return nil, fmt.Errorf("run orchestration: %w", err)
	}

	// Log results
	c.logOrchestrationResult(step, result)

	c.publishProgress(fmt.Sprintf("Orchestration completed with %d agent steps", len(result.StepResults)), 100)

	return result, nil
}

// logOrchestrationResult logs the results of orchestration execution.
func (c *Conductor) logOrchestrationResult(step string, result *orchestration.PipelineResult) {
	c.logVerbosef("=== Orchestration Results for %s ===", step)
	c.logVerbosef("Mode: Orchestrated multi-agent execution")
	c.logVerbosef("Duration: %s", result.Duration)

	if result.Consensus > 0 {
		c.logVerbosef("Consensus: %.0f%%", result.Consensus*100)
	}

	for stepName, stepResult := range result.StepResults {
		c.logVerbosef("  Step: %s (Agent: %s, Duration: %s, Tokens: %d)",
			stepName, stepResult.AgentName, stepResult.Duration, stepResult.TokenUsage)

		if stepResult.Error != nil {
			c.logVerbosef("    Error: %v", stepResult.Error)
		}
	}

	if len(result.Errors) > 0 {
		c.logVerbosef("Errors: %d", len(result.Errors))
		for _, err := range result.Errors {
			c.logVerbosef("  - %v", err)
		}
	}
}

// extractFinalOutput extracts the final output from an orchestration result.
// For consensus mode: uses the synthesizer's final output.
// For sequential mode: uses the last step's output.
// For parallel mode: uses the last step's output (or could merge).
func (c *Conductor) extractFinalOutput(result *orchestration.PipelineResult) string {
	// If there's a final output from consensus synthesis, use it
	if result.FinalOutput != "" {
		return result.FinalOutput
	}

	// Otherwise, get the last step's output
	// Sort steps to get the last one
	var lastStep *orchestration.StepResult
	for _, step := range result.StepResults {
		if lastStep == nil || step.StepName > lastStep.StepName {
			lastStep = step
		}
	}

	if lastStep != nil {
		return lastStep.Output
	}

	return ""
}
