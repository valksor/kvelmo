// Package noop provides a no-operation agent for testing.
// This agent is automatically registered when MEHR_TEST_MODE=1 is set.
package noop

import (
	"context"
	"time"

	"github.com/valksor/go-mehrhof/internal/agent"
)

const agentName = "noop"

// Agent is a no-operation agent for testing environments.
// It always reports as available but performs no actual AI operations.
type Agent struct {
	env  map[string]string
	args []string
}

// New creates a new noop agent.
func New() *Agent {
	return &Agent{
		env: make(map[string]string),
	}
}

// Name returns the agent identifier.
func (a *Agent) Name() string {
	return agentName
}

// Available always returns nil (always available).
func (a *Agent) Available() error {
	return nil
}

// Run returns an empty response. Noop agent doesn't execute anything.
func (a *Agent) Run(_ context.Context, _ string) (*agent.Response, error) {
	return &agent.Response{
		Summary:  "noop agent: no operation performed (test mode)",
		Messages: []string{"noop agent active - test mode"},
	}, nil
}

// RunStream returns a channel with a single completion event.
func (a *Agent) RunStream(ctx context.Context, _ string) (<-chan agent.Event, <-chan error) {
	eventCh := make(chan agent.Event, 1)
	errCh := make(chan error, 1)

	go func() {
		defer close(eventCh)
		defer close(errCh)

		select {
		case eventCh <- agent.Event{
			Type:      agent.EventComplete,
			Timestamp: time.Now(),
			Text:      "noop agent: no operation performed (test mode)",
		}:
		case <-ctx.Done():
			errCh <- ctx.Err()
		}
	}()

	return eventCh, errCh
}

// RunWithCallback executes with a callback, sending a single completion event.
func (a *Agent) RunWithCallback(ctx context.Context, _ string, cb agent.StreamCallback) (*agent.Response, error) {
	if err := cb(agent.Event{
		Type:      agent.EventComplete,
		Timestamp: time.Now(),
		Text:      "noop agent: no operation performed (test mode)",
	}); err != nil {
		return nil, err
	}

	return &agent.Response{
		Summary:  "noop agent: no operation performed (test mode)",
		Messages: []string{"noop agent active - test mode"},
	}, nil
}

// WithEnv returns a new agent with the environment variable set.
func (a *Agent) WithEnv(key, value string) agent.Agent {
	newEnv := make(map[string]string, len(a.env)+1)
	for k, v := range a.env {
		newEnv[k] = v
	}
	newEnv[key] = value

	return &Agent{
		env:  newEnv,
		args: a.args,
	}
}

// WithArgs returns a new agent with additional CLI arguments.
func (a *Agent) WithArgs(args ...string) agent.Agent {
	newArgs := make([]string, len(a.args), len(a.args)+len(args))
	copy(newArgs, a.args)
	newArgs = append(newArgs, args...)

	return &Agent{
		env:  a.env,
		args: newArgs,
	}
}

// WithRetries returns the agent unchanged (noop doesn't retry).
func (a *Agent) WithRetries(_ int) agent.Agent {
	return a
}

// Register registers the noop agent with the given registry.
func Register(registry *agent.Registry) error {
	return registry.Register(New())
}
