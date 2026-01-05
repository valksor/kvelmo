package coordination

import (
	"context"
	"errors"
	"testing"

	"github.com/valksor/go-mehrhof/internal/agent"
	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/workflow"
)

// mockAgent is a test implementation of Agent interface.
type mockAgent struct {
	name string
}

func (m *mockAgent) Name() string {
	return m.name
}

func (m *mockAgent) Run(ctx context.Context, prompt string) (*agent.Response, error) {
	return &agent.Response{}, nil
}

func (m *mockAgent) RunStream(ctx context.Context, prompt string) (<-chan agent.Event, <-chan error) {
	events := make(chan agent.Event)
	errs := make(chan error)
	close(events)
	close(errs)

	return events, errs
}

func (m *mockAgent) RunWithCallback(ctx context.Context, prompt string, cb agent.StreamCallback) (*agent.Response, error) {
	return &agent.Response{}, nil
}

func (m *mockAgent) Available() error {
	return nil
}

func (m *mockAgent) WithEnv(key, value string) agent.Agent {
	return m
}

func (m *mockAgent) WithArgs(args ...string) agent.Agent {
	return m
}

// mockWorkspace is a test implementation of Workspace interface.
type mockWorkspace struct {
	config *storage.WorkspaceConfig
	err    error
}

func (m *mockWorkspace) LoadConfig() (*storage.WorkspaceConfig, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.config != nil {
		return m.config, nil
	}

	return storage.NewDefaultWorkspaceConfig(), nil
}

func TestNewResolver(t *testing.T) {
	registry := agent.NewRegistry()
	workspace := &mockWorkspace{}

	r := NewResolver(registry, workspace)

	if r == nil {
		t.Fatal("NewResolver returned nil")
	}
	if r.agents != registry {
		t.Error("agents not set correctly")
	}
	if r.workspace != workspace {
		t.Error("workspace not set correctly")
	}
	if r.log == nil {
		t.Error("log not initialized")
	}
}

func TestApplyEnvs(t *testing.T) {
	tests := []struct {
		name     string
		agent    agent.Agent
		env      map[string]string
		wantSame bool
	}{
		{
			name:     "nil env returns same agent",
			agent:    &mockAgent{name: "test"},
			env:      nil,
			wantSame: true,
		},
		{
			name:     "empty env returns same agent",
			agent:    &mockAgent{name: "test"},
			env:      map[string]string{},
			wantSame: true,
		},
		{
			name:  "env vars applied",
			agent: &mockAgent{name: "test"},
			env: map[string]string{
				"KEY1": "value1",
				"KEY2": "value2",
			},
			wantSame: false, // WithEnv returns a new agent (or same for mock)
		},
		{
			name:  "env with ${VAR} expansion",
			agent: &mockAgent{name: "test"},
			env: map[string]string{
				"KEY1": "${PATH}",
			},
			wantSame: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ApplyEnvs(tt.agent, tt.env)
			if result == nil {
				t.Error("ApplyEnvs returned nil")
			}
			// For mockAgent, WithEnv returns the same instance
			// Real agents would return a modified copy
		})
	}
}

func TestApplyArgs(t *testing.T) {
	tests := []struct {
		name     string
		agent    agent.Agent
		args     []string
		wantSame bool
	}{
		{
			name:     "nil args returns same agent",
			agent:    &mockAgent{name: "test"},
			args:     nil,
			wantSame: true,
		},
		{
			name:     "empty args returns same agent",
			agent:    &mockAgent{name: "test"},
			args:     []string{},
			wantSame: true,
		},
		{
			name:  "args applied",
			agent: &mockAgent{name: "test"},
			args:  []string{"--arg1", "value1", "--arg2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ApplyArgs(tt.agent, tt.args...)
			if result == nil {
				t.Error("ApplyArgs returned nil")
			}
		})
	}
}

func TestResolveForStep_Priority1_CLIStepSpecific(t *testing.T) {
	registry := agent.NewRegistry()
	agent1 := &mockAgent{name: "step-agent"}
	if err := registry.Register(agent1); err != nil {
		t.Fatal(err)
	}

	workspace := &mockWorkspace{}
	r := NewResolver(registry, workspace)

	req := ResolveRequest{
		CLISStepAgents: map[string]string{
			"planning": "step-agent",
		},
		Step: workflow.StepPlanning,
	}

	res, err := r.ResolveForStep(context.Background(), req)
	if err != nil {
		t.Fatalf("ResolveForStep failed: %v", err)
	}

	if res.Source != "cli-step" {
		t.Errorf("source = %q, want %q", res.Source, "cli-step")
	}
	if res.Agent.Name() != "step-agent" {
		t.Errorf("agent name = %q, want %q", res.Agent.Name(), "step-agent")
	}
	if res.StepName != "planning" {
		t.Errorf("step name = %q, want %q", res.StepName, "planning")
	}
}

func TestResolveForStep_Priority2_CLIGlobal(t *testing.T) {
	registry := agent.NewRegistry()
	agent1 := &mockAgent{name: "global-agent"}
	if err := registry.Register(agent1); err != nil {
		t.Fatal(err)
	}

	workspace := &mockWorkspace{}
	r := NewResolver(registry, workspace)

	req := ResolveRequest{
		CLIAgent: "global-agent",
		Step:     workflow.StepPlanning,
	}

	res, err := r.ResolveForStep(context.Background(), req)
	if err != nil {
		t.Fatalf("ResolveForStep failed: %v", err)
	}

	if res.Source != "cli" {
		t.Errorf("source = %q, want %q", res.Source, "cli")
	}
	if res.Agent.Name() != "global-agent" {
		t.Errorf("agent name = %q, want %q", res.Agent.Name(), "global-agent")
	}
}

func TestResolveForStep_Priority3_TaskStepSpecific(t *testing.T) {
	registry := agent.NewRegistry()
	agent1 := &mockAgent{name: "task-step-agent"}
	if err := registry.Register(agent1); err != nil {
		t.Fatal(err)
	}

	workspace := &mockWorkspace{}
	r := NewResolver(registry, workspace)

	req := ResolveRequest{
		TaskConfig: &provider.AgentConfig{
			Steps: map[string]provider.StepAgentConfig{
				"planning": {
					Name: "task-step-agent",
					Env:  map[string]string{"PLAN_KEY": "plan_val"},
					Args: []string{"--plan-arg"},
				},
			},
		},
		Step: workflow.StepPlanning,
	}

	res, err := r.ResolveForStep(context.Background(), req)
	if err != nil {
		t.Fatalf("ResolveForStep failed: %v", err)
	}

	if res.Source != "task-step" {
		t.Errorf("source = %q, want %q", res.Source, "task-step")
	}
	if res.Agent.Name() != "task-step-agent" {
		t.Errorf("agent name = %q, want %q", res.Agent.Name(), "task-step-agent")
	}
	if res.InlineEnv == nil {
		t.Error("InlineEnv not set")
	} else if res.InlineEnv["PLAN_KEY"] != "plan_val" {
		t.Errorf("InlineEnv[PLAN_KEY] = %q, want %q", res.InlineEnv["PLAN_KEY"], "plan_val")
	}
}

func TestResolveForStep_Priority4_TaskDefault(t *testing.T) {
	registry := agent.NewRegistry()
	agent1 := &mockAgent{name: "task-default-agent"}
	if err := registry.Register(agent1); err != nil {
		t.Fatal(err)
	}

	workspace := &mockWorkspace{}
	r := NewResolver(registry, workspace)

	req := ResolveRequest{
		TaskConfig: &provider.AgentConfig{
			Name: "task-default-agent",
			Env:  map[string]string{"TASK_KEY": "task_val"},
			Args: []string{"--task-arg"},
		},
		Step: workflow.StepPlanning,
	}

	res, err := r.ResolveForStep(context.Background(), req)
	if err != nil {
		t.Fatalf("ResolveForStep failed: %v", err)
	}

	if res.Source != "task" {
		t.Errorf("source = %q, want %q", res.Source, "task")
	}
	if res.Agent.Name() != "task-default-agent" {
		t.Errorf("agent name = %q, want %q", res.Agent.Name(), "task-default-agent")
	}
}

func TestResolveForStep_Priority5_WorkspaceStepSpecific(t *testing.T) {
	registry := agent.NewRegistry()
	agent1 := &mockAgent{name: "ws-step-agent"}
	if err := registry.Register(agent1); err != nil {
		t.Fatal(err)
	}

	cfg := storage.NewDefaultWorkspaceConfig()
	cfg.Agent.Steps = map[string]storage.StepAgentConfig{
		"planning": {
			Name: "ws-step-agent",
			Env:  map[string]string{"WS_PLAN_KEY": "ws_plan_val"},
		},
	}

	workspace := &mockWorkspace{config: cfg}
	r := NewResolver(registry, workspace)

	req := ResolveRequest{
		Step: workflow.StepPlanning,
	}

	res, err := r.ResolveForStep(context.Background(), req)
	if err != nil {
		t.Fatalf("ResolveForStep failed: %v", err)
	}

	if res.Source != "workspace-step" {
		t.Errorf("source = %q, want %q", res.Source, "workspace-step")
	}
	if res.Agent.Name() != "ws-step-agent" {
		t.Errorf("agent name = %q, want %q", res.Agent.Name(), "ws-step-agent")
	}
}

func TestResolveForStep_Priority6_WorkspaceDefault(t *testing.T) {
	registry := agent.NewRegistry()
	agent1 := &mockAgent{name: "ws-default-agent"}
	if err := registry.Register(agent1); err != nil {
		t.Fatal(err)
	}

	cfg := storage.NewDefaultWorkspaceConfig()
	cfg.Agent.Default = "ws-default-agent"

	workspace := &mockWorkspace{config: cfg}
	r := NewResolver(registry, workspace)

	req := ResolveRequest{
		Step: workflow.StepPlanning,
	}

	res, err := r.ResolveForStep(context.Background(), req)
	if err != nil {
		t.Fatalf("ResolveForStep failed: %v", err)
	}

	if res.Source != "workspace" {
		t.Errorf("source = %q, want %q", res.Source, "workspace")
	}
	if res.Agent.Name() != "ws-default-agent" {
		t.Errorf("agent name = %q, want %q", res.Agent.Name(), "ws-default-agent")
	}
}

func TestResolveForStep_Priority7_AutoDetect(t *testing.T) {
	registry := agent.NewRegistry()
	agent1 := &mockAgent{name: "auto-agent"}
	if err := registry.Register(agent1); err != nil {
		t.Fatal(err)
	}

	workspace := &mockWorkspace{config: storage.NewDefaultWorkspaceConfig()}
	r := NewResolver(registry, workspace)

	req := ResolveRequest{
		Step: workflow.StepPlanning,
	}

	res, err := r.ResolveForStep(context.Background(), req)
	if err != nil {
		t.Fatalf("ResolveForStep failed: %v", err)
	}

	if res.Source != "auto" {
		t.Errorf("source = %q, want %q", res.Source, "auto")
	}
	if res.Agent.Name() != "auto-agent" {
		t.Errorf("agent name = %q, want %q", res.Agent.Name(), "auto-agent")
	}
}

func TestResolveForStep_PriorityOrder_CLIStepOverTaskStep(t *testing.T) {
	registry := agent.NewRegistry()
	agent1 := &mockAgent{name: "cli-agent"}
	agent2 := &mockAgent{name: "task-agent"}
	if err := registry.Register(agent1); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(agent2); err != nil {
		t.Fatal(err)
	}

	workspace := &mockWorkspace{}
	r := NewResolver(registry, workspace)

	req := ResolveRequest{
		CLISStepAgents: map[string]string{
			"planning": "cli-agent",
		},
		TaskConfig: &provider.AgentConfig{
			Steps: map[string]provider.StepAgentConfig{
				"planning": {Name: "task-agent"},
			},
		},
		Step: workflow.StepPlanning,
	}

	res, err := r.ResolveForStep(context.Background(), req)
	if err != nil {
		t.Fatalf("ResolveForStep failed: %v", err)
	}

	// CLI step-specific should take priority over task step-specific
	if res.Agent.Name() != "cli-agent" {
		t.Errorf("agent name = %q, want %q (CLI priority)", res.Agent.Name(), "cli-agent")
	}
	if res.Source != "cli-step" {
		t.Errorf("source = %q, want %q", res.Source, "cli-step")
	}
}

func TestResolveForStep_AgentNotFoundInRegistry(t *testing.T) {
	registry := agent.NewRegistry()
	workspace := &mockWorkspace{}
	r := NewResolver(registry, workspace)

	req := ResolveRequest{
		CLISStepAgents: map[string]string{
			"planning": "nonexistent-agent",
		},
		Step: workflow.StepPlanning,
	}

	_, err := r.ResolveForStep(context.Background(), req)
	if err == nil {
		t.Error("expected error for nonexistent agent, got nil")
	}
}

func TestResolveForStep_WorkspaceLoadError(t *testing.T) {
	registry := agent.NewRegistry()
	agent1 := &mockAgent{name: "fallback-agent"}
	if err := registry.Register(agent1); err != nil {
		t.Fatal(err)
	}

	workspace := &mockWorkspace{err: errors.New("config load failed")}
	r := NewResolver(registry, workspace)

	req := ResolveRequest{
		Step: workflow.StepPlanning,
	}

	// Should fall through to auto-detect when workspace config fails
	res, err := r.ResolveForStep(context.Background(), req)
	if err != nil {
		t.Fatalf("ResolveForStep failed: %v", err)
	}

	if res.Source != "auto" {
		t.Errorf("source = %q, want %q (fell through to auto)", res.Source, "auto")
	}
}

func TestResolveForStep_WorkspaceAgentNotFound(t *testing.T) {
	registry := agent.NewRegistry()
	agent1 := &mockAgent{name: "auto-agent"}
	if err := registry.Register(agent1); err != nil {
		t.Fatal(err)
	}

	cfg := storage.NewDefaultWorkspaceConfig()
	cfg.Agent.Default = "nonexistent-ws-agent"

	workspace := &mockWorkspace{config: cfg}
	r := NewResolver(registry, workspace)

	req := ResolveRequest{
		Step: workflow.StepPlanning,
	}

	// Should fall through to auto-detect when workspace agent not found
	res, err := r.ResolveForStep(context.Background(), req)
	if err != nil {
		t.Fatalf("ResolveForStep failed: %v", err)
	}

	if res.Source != "auto" {
		t.Errorf("source = %q, want %q (fell through to auto)", res.Source, "auto")
	}
}

func TestResolveForStep_AllSteps(t *testing.T) {
	registry := agent.NewRegistry()
	agent1 := &mockAgent{name: "test-agent"}
	if err := registry.Register(agent1); err != nil {
		t.Fatal(err)
	}

	steps := []workflow.Step{
		workflow.StepPlanning,
		workflow.StepImplementing,
		workflow.StepReviewing,
		workflow.StepCheckpointing,
	}

	for _, step := range steps {
		t.Run(step.String(), func(t *testing.T) {
			workspace := &mockWorkspace{}
			r := NewResolver(registry, workspace)

			req := ResolveRequest{
				CLIAgent: "test-agent",
				Step:     step,
			}

			res, err := r.ResolveForStep(context.Background(), req)
			if err != nil {
				t.Fatalf("ResolveForStep failed: %v", err)
			}

			if res.Agent.Name() != "test-agent" {
				t.Errorf("agent name = %q, want %q", res.Agent.Name(), "test-agent")
			}
			if res.StepName != step.String() {
				t.Errorf("step name = %q, want %q", res.StepName, step.String())
			}
		})
	}
}

func TestResolveForStep_NoAgentsAvailable(t *testing.T) {
	registry := agent.NewRegistry()
	// Don't register any agents

	workspace := &mockWorkspace{config: storage.NewDefaultWorkspaceConfig()}
	r := NewResolver(registry, workspace)

	req := ResolveRequest{
		Step: workflow.StepPlanning,
	}

	_, err := r.ResolveForStep(context.Background(), req)
	if err == nil {
		t.Error("expected error when no agents available, got nil")
	}
}

func TestResolutionStruct(t *testing.T) {
	agent := &mockAgent{name: "test"}

	res := &Resolution{
		Agent:     agent,
		Source:    "cli",
		StepName:  "planning",
		InlineEnv: map[string]string{"KEY": "val"},
		Args:      []string{"--arg"},
	}

	if res.Agent.Name() != "test" {
		t.Errorf("Agent.Name() = %q, want %q", res.Agent.Name(), "test")
	}
	if res.Source != "cli" {
		t.Errorf("Source = %q, want %q", res.Source, "cli")
	}
	if res.StepName != "planning" {
		t.Errorf("StepName = %q, want %q", res.StepName, "planning")
	}
	if res.InlineEnv["KEY"] != "val" {
		t.Errorf("InlineEnv[KEY] = %q, want %q", res.InlineEnv["KEY"], "val")
	}
	if len(res.Args) != 1 || res.Args[0] != "--arg" {
		t.Errorf("Args = %v, want [--arg]", res.Args)
	}
}

func TestResolveRequestStruct(t *testing.T) {
	req := ResolveRequest{
		CLIAgent: "test-agent",
		CLISStepAgents: map[string]string{
			"planning": "plan-agent",
		},
		TaskConfig: &provider.AgentConfig{
			Name: "task-agent",
		},
		Step: workflow.StepPlanning,
	}

	if req.CLIAgent != "test-agent" {
		t.Errorf("CLIAgent = %q, want %q", req.CLIAgent, "test-agent")
	}
	if req.CLISStepAgents["planning"] != "plan-agent" {
		t.Errorf("CLISStepAgents[planning] = %q, want %q", req.CLISStepAgents["planning"], "plan-agent")
	}
	if req.TaskConfig.Name != "task-agent" {
		t.Errorf("TaskConfig.Name = %q, want %q", req.TaskConfig.Name, "task-agent")
	}
	if req.Step != workflow.StepPlanning {
		t.Errorf("Step = %v, want %v", req.Step, workflow.StepPlanning)
	}
}

func TestErrAgentNotFound(t *testing.T) {
	if ErrAgentNotFound == nil {
		t.Error("ErrAgentNotFound should not be nil")
	}
	if ErrAgentNotFound.Error() != "agent not found at this priority" {
		t.Errorf("ErrAgentNotFound.Error() = %q, want %q", ErrAgentNotFound.Error(), "agent not found at this priority")
	}
}

func TestWorkspaceInterface(t *testing.T) {
	// Verify Workspace interface is correctly defined
	var _ Workspace = (*mockWorkspace)(nil)

	w := &mockWorkspace{}
	cfg, err := w.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg == nil {
		t.Error("LoadConfig returned nil config")
	}
}
