package agent

import (
	"context"
	"maps"
	"slices"
)

// AliasAgent wraps an existing agent with pre-configured environment variables
// and CLI arguments. This allows users to define custom agent configurations in
// their workspace config without modifying the core agent code.
type AliasAgent struct {
	name        string
	description string
	base        Agent
	binaryPath  string // Custom binary path (empty = use base agent's default)
	env         map[string]string
	args        []string
}

// NewAlias creates a new alias agent that wraps an existing agent with
// pre-configured environment variables, CLI arguments, and optionally a custom binary path.
func NewAlias(name string, base Agent, binaryPath string, env map[string]string, args []string, description string) *AliasAgent {
	return &AliasAgent{
		name:        name,
		description: description,
		base:        base,
		binaryPath:  binaryPath,
		env:         env,
		args:        args,
	}
}

// Name returns the alias name.
func (a *AliasAgent) Name() string {
	return a.name
}

// Description returns the human-readable description.
func (a *AliasAgent) Description() string {
	return a.description
}

// BaseAgent returns the underlying agent being wrapped.
func (a *AliasAgent) BaseAgent() Agent {
	return a.base
}

// CommandConfigurable is implemented by agents that support custom binary paths.
type CommandConfigurable interface {
	WithCommand(command string) Agent
}

// configured returns the base agent with binary path, env vars and args applied.
func (a *AliasAgent) configured() Agent {
	agent := a.base

	// Apply custom binary path first (if specified and base supports it)
	if a.binaryPath != "" {
		if configurable, ok := agent.(CommandConfigurable); ok {
			agent = configurable.WithCommand(a.binaryPath)
		}
	}

	for k, v := range a.env {
		agent = agent.WithEnv(k, v)
	}
	if len(a.args) > 0 {
		agent = agent.WithArgs(a.args...)
	}

	return agent
}

// Run executes the prompt by delegating to the base agent with env vars and args applied.
func (a *AliasAgent) Run(ctx context.Context, prompt string) (*Response, error) {
	return a.configured().Run(ctx, prompt)
}

// RunStream executes the prompt and streams events.
func (a *AliasAgent) RunStream(ctx context.Context, prompt string) (<-chan Event, <-chan error) {
	return a.configured().RunStream(ctx, prompt)
}

// RunWithCallback executes with a callback for each event.
func (a *AliasAgent) RunWithCallback(ctx context.Context, prompt string, cb StreamCallback) (*Response, error) {
	return a.configured().RunWithCallback(ctx, prompt, cb)
}

// Available checks if the configured agent is available.
// If a custom binary path is set, it checks if that binary exists.
func (a *AliasAgent) Available() error {
	return a.configured().Available()
}

// WithEnv adds an additional environment variable to the alias.
// This creates a new AliasAgent with the combined environment variables.
func (a *AliasAgent) WithEnv(key, value string) Agent {
	newEnv := maps.Clone(a.env)
	if newEnv == nil {
		newEnv = make(map[string]string)
	}
	newEnv[key] = value

	return &AliasAgent{
		name:        a.name,
		description: a.description,
		base:        a.base,
		binaryPath:  a.binaryPath,
		env:         newEnv,
		args:        a.args,
	}
}

// WithArgs adds additional CLI arguments to the alias.
// This creates a new AliasAgent with the combined arguments.
func (a *AliasAgent) WithArgs(args ...string) Agent {
	newArgs := slices.Concat(a.args, args)

	return &AliasAgent{
		name:        a.name,
		description: a.description,
		base:        a.base,
		binaryPath:  a.binaryPath,
		env:         a.env,
		args:        newArgs,
	}
}

// WithRetries delegates to the base agent.
// This allows retry configuration to be applied to the underlying agent.
func (a *AliasAgent) WithRetries(n int) Agent {
	return &AliasAgent{
		name:        a.name,
		description: a.description,
		base:        a.base.WithRetries(n),
		binaryPath:  a.binaryPath,
		env:         a.env,
		args:        a.args,
	}
}

// StepArgs delegates to the base agent if it implements StepArgsProvider.
// This ensures aliases like "glm" (which extends "claude") get the same
// step-specific args as their base agent.
func (a *AliasAgent) StepArgs(step string) []string {
	if provider, ok := a.base.(StepArgsProvider); ok {
		return provider.StepArgs(step)
	}

	return nil
}

// Ensure AliasAgent implements Agent interface.
var _ Agent = (*AliasAgent)(nil)

// Ensure AliasAgent implements StepArgsProvider interface.
var _ StepArgsProvider = (*AliasAgent)(nil)
