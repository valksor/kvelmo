package noop

import (
	"context"
	"testing"

	"github.com/valksor/go-mehrhof/internal/agent"
)

func TestAgent_Name(t *testing.T) {
	a := New()
	if got := a.Name(); got != "noop" {
		t.Errorf("Name() = %q, want %q", got, "noop")
	}
}

func TestAgent_Available(t *testing.T) {
	a := New()
	if err := a.Available(); err != nil {
		t.Errorf("Available() = %v, want nil", err)
	}
}

func TestAgent_Run(t *testing.T) {
	a := New()
	resp, err := a.Run(context.Background(), "test prompt")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if resp == nil {
		t.Fatal("Run() returned nil response")
	}
	if resp.Summary == "" {
		t.Error("Run() returned empty summary")
	}
}

func TestAgent_RunStream(t *testing.T) {
	a := New()
	eventCh, errCh := a.RunStream(context.Background(), "test prompt")

	var events []agent.Event
	for event := range eventCh {
		events = append(events, event)
	}

	// Check for errors
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("RunStream() error = %v", err)
		}
	default:
	}

	if len(events) == 0 {
		t.Error("RunStream() returned no events")
	}
	if events[0].Type != agent.EventComplete {
		t.Errorf("RunStream() event type = %v, want %v", events[0].Type, agent.EventComplete)
	}
}

func TestAgent_RunWithCallback(t *testing.T) {
	a := New()

	var called bool
	cb := func(event agent.Event) error {
		called = true

		return nil
	}

	resp, err := a.RunWithCallback(context.Background(), "test prompt", cb)
	if err != nil {
		t.Fatalf("RunWithCallback() error = %v", err)
	}
	if !called {
		t.Error("RunWithCallback() callback not called")
	}
	if resp == nil {
		t.Fatal("RunWithCallback() returned nil response")
	}
}

func TestAgent_WithEnv(t *testing.T) {
	a := New()
	a2 := a.WithEnv("KEY", "VALUE")

	// Should return a new agent
	if a2 == a {
		t.Error("WithEnv() should return a new agent")
	}

	// Original should be unchanged
	if len(a.env) != 0 {
		t.Error("WithEnv() modified original agent")
	}
}

func TestAgent_WithArgs(t *testing.T) {
	a := New()
	a2 := a.WithArgs("--flag", "value")

	// Should return a new agent
	if a2 == a {
		t.Error("WithArgs() should return a new agent")
	}

	// Original should be unchanged
	if len(a.args) != 0 {
		t.Error("WithArgs() modified original agent")
	}
}

func TestRegister(t *testing.T) {
	registry := agent.NewRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Should be able to get the agent
	a, err := registry.Get("noop")
	if err != nil {
		t.Fatalf("registry.Get() error = %v", err)
	}
	if a.Name() != "noop" {
		t.Errorf("registry.Get() name = %q, want %q", a.Name(), "noop")
	}
}
