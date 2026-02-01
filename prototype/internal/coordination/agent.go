// Package coordination provides agent resolution and coordination services.
// It separates the complexity of agent selection from the conductor.
package coordination

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/valksor/go-mehrhof/internal/agent"
	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/workflow"
	"github.com/valksor/go-toolkit/env"
)

// ErrAgentNotFound indicates no agent was found at a priority level.
var ErrAgentNotFound = errors.New("agent not found at this priority")

// Resolver handles agent resolution with multiple priority sources.
type Resolver struct {
	agents    *agent.Registry
	workspace Workspace
	log       *slog.Logger
}

// Workspace provides access to configuration.
type Workspace interface {
	LoadConfig() (*storage.WorkspaceConfig, error)
}

// NewResolver creates a new agent resolver.
func NewResolver(agents *agent.Registry, workspace Workspace) *Resolver {
	return &Resolver{
		agents:    agents,
		workspace: workspace,
		log:       slog.Default(),
	}
}

// Resolution holds the result of agent resolution.
type Resolution struct {
	Agent     agent.Agent
	Source    string            // Where it was resolved from: cli-step, cli, task-step, task, workspace-step, workspace, auto
	StepName  string            // Which step this is for
	InlineEnv map[string]string // Resolved inline env vars
	Args      []string          // CLI args for this step
}

// ResolveRequest contains all inputs for agent resolution.
type ResolveRequest struct {
	// CLI-provided values
	CLIAgent       string                   // --agent flag
	CLISStepAgents map[string]string        // --agent-plan, --agent-implement, etc.
	WorkspaceCfg   *storage.WorkspaceConfig // Workspace config (optional, can be loaded if nil)

	// Task frontmatter values (may be nil)
	TaskConfig *provider.AgentConfig

	// Step to resolve for
	Step workflow.Step
}

// ResolveForStep resolves an agent for a specific workflow step.
// Priority order:
// 1. CLI step-specific flag (--agent-plan)
// 2. CLI global flag (--agent)
// 3. Task frontmatter step-specific (agent_steps.planning.agent)
// 4. Task frontmatter default (agent)
// 5. Workspace config step-specific (agent.steps.planning.name)
// 6. Workspace config default (agent.default)
// 7. Auto-detect.
func (r *Resolver) ResolveForStep(ctx context.Context, req ResolveRequest) (*Resolution, error) {
	stepStr := req.Step.String()

	// Try each priority level
	for priority := 1; priority <= 7; priority++ {
		resolution, err := r.tryResolveAtPriority(ctx, req, priority, stepStr)
		if errors.Is(err, ErrAgentNotFound) {
			// No agent at this priority, try next one
			continue
		}
		if err != nil {
			return nil, err
		}
		if resolution != nil {
			return resolution, nil
		}
	}

	// Should never reach here since priority 7 (auto-detect) should always succeed
	return nil, fmt.Errorf("failed to resolve agent for step %s", req.Step)
}

// tryResolveAtPriority attempts resolution at a specific priority level.
// Returns the resolution if successful, nil if no agent found at this level (continue to next),
// or an error if resolution failed unexpectedly.
func (r *Resolver) tryResolveAtPriority(_ context.Context, req ResolveRequest, priority int, stepStr string) (*Resolution, error) {
	switch priority {
	case 1: // CLI step-specific
		if name, ok := req.CLISStepAgents[stepStr]; ok && name != "" {
			agentInst, err := r.agents.Get(name)
			if err != nil {
				return nil, fmt.Errorf("get agent %s: %w", name, err)
			}

			return &Resolution{
				Agent:    agentInst,
				Source:   "cli-step",
				StepName: stepStr,
			}, nil
		}

	case 2: // CLI global
		if req.CLIAgent != "" {
			agentInst, err := r.agents.Get(req.CLIAgent)
			if err != nil {
				return nil, fmt.Errorf("get agent %s: %w", req.CLIAgent, err)
			}

			return &Resolution{
				Agent:    agentInst,
				Source:   "cli",
				StepName: stepStr,
			}, nil
		}

	case 3: // Task frontmatter step-specific
		if req.TaskConfig != nil && req.TaskConfig.Steps != nil {
			if stepCfg, ok := req.TaskConfig.Steps[stepStr]; ok && stepCfg.Name != "" {
				agentInst, err := r.agents.Get(stepCfg.Name)
				if err != nil {
					return nil, fmt.Errorf("get agent %s: %w", stepCfg.Name, err)
				}

				return &Resolution{
					Agent:     agentInst,
					Source:    "task-step",
					StepName:  stepStr,
					InlineEnv: stepCfg.Env,
					Args:      stepCfg.Args,
				}, nil
			}
		}

	case 4: // Task frontmatter default
		if req.TaskConfig != nil && req.TaskConfig.Name != "" {
			agentInst, err := r.agents.Get(req.TaskConfig.Name)
			if err != nil {
				return nil, fmt.Errorf("get agent %s: %w", req.TaskConfig.Name, err)
			}

			return &Resolution{
				Agent:     agentInst,
				Source:    "task",
				StepName:  stepStr,
				InlineEnv: req.TaskConfig.Env,
				Args:      req.TaskConfig.Args,
			}, nil
		}

	case 5, 6: // Workspace config
		cfg := req.WorkspaceCfg
		if cfg == nil {
			var err error
			cfg, err = r.workspace.LoadConfig()
			if err != nil {
				// No workspace config available, continue to next priority
				return nil, ErrAgentNotFound
			}
		}
		if cfg != nil {
			if priority == 5 {
				// Workspace step-specific
				if stepCfg, ok := cfg.Agent.Steps[stepStr]; ok && stepCfg.Name != "" {
					agentInst, err := r.agents.Get(stepCfg.Name)
					if err != nil {
						// Workspace config agent not found - log warning but continue
						r.log.Warn("Workspace step agent not found", "step", stepStr, "agent", stepCfg.Name, "error", err)

						return nil, ErrAgentNotFound
					}

					return &Resolution{
						Agent:     agentInst,
						Source:    "workspace-step",
						StepName:  stepStr,
						InlineEnv: stepCfg.Env,
						Args:      stepCfg.Args,
					}, nil
				}
			} else {
				// Workspace default
				if cfg.Agent.Default != "" {
					agentInst, err := r.agents.Get(cfg.Agent.Default)
					if err != nil {
						// Workspace default agent not found - log warning but continue
						r.log.Warn("Workspace default agent not found", "agent", cfg.Agent.Default, "error", err)

						return nil, ErrAgentNotFound
					}

					return &Resolution{
						Agent:    agentInst,
						Source:   "workspace",
						StepName: stepStr,
					}, nil
				}
			}
		}

	case 7: // Auto-detect
		agentInst, err := r.agents.Detect()
		if err != nil {
			return nil, fmt.Errorf("detect agent: %w", err)
		}

		return &Resolution{
			Agent:    agentInst,
			Source:   "auto",
			StepName: stepStr,
		}, nil
	}

	return nil, ErrAgentNotFound
}

// ApplyEnvs applies environment variables to an agent instance.
func ApplyEnvs(agentInst agent.Agent, envMap map[string]string) agent.Agent {
	if len(envMap) == 0 {
		return agentInst
	}
	resolvedEnv := env.ExpandEnvInMap(envMap)
	for k, v := range resolvedEnv {
		agentInst = agentInst.WithEnv(k, v)
	}

	return agentInst
}

// ApplyArgs applies CLI arguments to an agent instance.
func ApplyArgs(agentInst agent.Agent, args ...string) agent.Agent {
	if len(args) == 0 {
		return agentInst
	}

	return agentInst.WithArgs(args...)
}

// ResolutionStep explains one step in the resolution process.
type ResolutionStep struct {
	Priority int    // Priority level (1-7)
	Source   string // Source name (cli-step, cli, task-step, task, workspace-step, workspace, auto)
	Agent    string // Agent name at this level (empty if not set)
	Skipped  bool   // True if this level was skipped (agent not found or not configured)
}

// ResolutionExplanation provides a detailed explanation of agent resolution.
type ResolutionExplanation struct {
	Step      string           // Workflow step being resolved
	Effective string           // The agent that was selected
	Source    string           // Where the effective agent came from
	AllSteps  []ResolutionStep // All 7 priority levels with their status
}

// ExplainAgentResolution explains how an agent would be resolved for a given step.
// Returns a detailed explanation showing all 7 priority levels and which one won.
func (r *Resolver) ExplainAgentResolution(ctx context.Context, req ResolveRequest) (*ResolutionExplanation, error) {
	stepStr := req.Step.String()
	steps := make([]ResolutionStep, 7)

	var effectiveAgent string
	var effectiveSource string
	found := false

	// Check each priority level
	for priority := 1; priority <= 7; priority++ {
		source, agentName, skipped := r.checkAgentAtPriority(ctx, req, priority, stepStr)

		// Mark as skipped if we already found an agent at a previous priority
		if found {
			skipped = true
		}

		steps[priority-1] = ResolutionStep{
			Priority: priority,
			Source:   source,
			Agent:    agentName,
			Skipped:  skipped,
		}

		// The first non-skipped level is the effective one
		if !skipped && !found {
			effectiveAgent = agentName
			effectiveSource = source
			found = true
		}
	}

	return &ResolutionExplanation{
		Step:      stepStr,
		Effective: effectiveAgent,
		Source:    effectiveSource,
		AllSteps:  steps,
	}, nil
}

// checkAgentAtPriority checks what agent (if any) is configured at a given priority level.
// Returns (source, agentName, skipped).
func (r *Resolver) checkAgentAtPriority(_ context.Context, req ResolveRequest, priority int, stepStr string) (string, string, bool) {
	switch priority {
	case 1: // CLI step-specific
		if name, ok := req.CLISStepAgents[stepStr]; ok && name != "" {
			if _, err := r.agents.Get(name); err == nil {
				return "cli-step (CLI --agent-" + stepStr + ")", name, false
			}
		}

		return "cli-step", "", true

	case 2: // CLI global
		if req.CLIAgent != "" {
			if _, err := r.agents.Get(req.CLIAgent); err == nil {
				return "cli (CLI --agent)", req.CLIAgent, false
			}
		}

		return "cli", "", true

	case 3: // Task frontmatter step-specific
		if req.TaskConfig != nil && req.TaskConfig.Steps != nil {
			if stepCfg, ok := req.TaskConfig.Steps[stepStr]; ok && stepCfg.Name != "" {
				if _, err := r.agents.Get(stepCfg.Name); err == nil {
					return "task-step (task frontmatter agent_steps." + stepStr + ".agent)", stepCfg.Name, false
				}
			}
		}

		return "task-step", "", true

	case 4: // Task frontmatter default
		if req.TaskConfig != nil && req.TaskConfig.Name != "" {
			if _, err := r.agents.Get(req.TaskConfig.Name); err == nil {
				return "task (task frontmatter agent)", req.TaskConfig.Name, false
			}
		}

		return "task", "", true

	case 5: // Workspace step-specific
		cfg := req.WorkspaceCfg
		if cfg == nil {
			if _, err := r.workspace.LoadConfig(); err != nil {
				return "workspace-step", "", true
			}
		}
		if cfg != nil {
			if stepCfg, ok := cfg.Agent.Steps[stepStr]; ok && stepCfg.Name != "" {
				if _, err := r.agents.Get(stepCfg.Name); err == nil {
					return "workspace-step (config agent.steps." + stepStr + ".name)", stepCfg.Name, false
				}
			}
		}

		return "workspace-step", "", true

	case 6: // Workspace default
		cfg := req.WorkspaceCfg
		if cfg == nil {
			if _, err := r.workspace.LoadConfig(); err != nil {
				return "workspace (config agent.default)", "", true
			}
		}
		if cfg != nil && cfg.Agent.Default != "" {
			if _, err := r.agents.Get(cfg.Agent.Default); err == nil {
				return "workspace (config agent.default)", cfg.Agent.Default, false
			}
		}

		return "workspace", "", true

	case 7: // Auto-detect
		if agentInst, err := r.agents.Detect(); err == nil {
			return "auto (first available agent)", agentInst.Name(), false
		}

		return "auto", "", true
	}

	return "", "", true
}
