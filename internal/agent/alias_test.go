package agent

import (
	"context"
	"testing"
)

// aliasTestAgent is a mock agent for testing AliasAgent functionality.
type aliasTestAgent struct {
	name    string
	command string
	env     map[string]string
	args    []string
}

func (m *aliasTestAgent) Name() string { return m.name }

func (m *aliasTestAgent) Run(_ context.Context, _ string) (*Response, error) {
	return &Response{}, nil
}

func (m *aliasTestAgent) RunStream(_ context.Context, _ string) (<-chan Event, <-chan error) {
	return nil, nil
}

func (m *aliasTestAgent) RunWithCallback(_ context.Context, _ string, _ StreamCallback) (*Response, error) {
	return &Response{}, nil
}

func (m *aliasTestAgent) Available() error { return nil }

func (m *aliasTestAgent) WithEnv(key, value string) Agent {
	newEnv := make(map[string]string)
	for k, v := range m.env {
		newEnv[k] = v
	}
	newEnv[key] = value

	return &aliasTestAgent{name: m.name, command: m.command, env: newEnv, args: m.args}
}

func (m *aliasTestAgent) WithArgs(args ...string) Agent {
	newArgs := append([]string{}, m.args...)
	newArgs = append(newArgs, args...)

	return &aliasTestAgent{name: m.name, command: m.command, env: m.env, args: newArgs}
}

func (m *aliasTestAgent) WithCommand(command string) Agent {
	return &aliasTestAgent{name: m.name, command: command, env: m.env, args: m.args}
}

func (m *aliasTestAgent) WithRetries(_ int) Agent {
	return m
}

func TestNewAlias(t *testing.T) {
	base := &aliasTestAgent{name: "claude", command: "claude"}
	alias := NewAlias("custom", base, "", map[string]string{"KEY": "val"}, []string{"--arg"}, "Custom agent")

	if alias.Name() != "custom" {
		t.Errorf("Name() = %q, want %q", alias.Name(), "custom")
	}
	if alias.Description() != "Custom agent" {
		t.Errorf("Description() = %q, want %q", alias.Description(), "Custom agent")
	}
	if alias.BaseAgent() != base {
		t.Error("BaseAgent() should return the base agent")
	}
}

func TestNewAlias_WithBinaryPath(t *testing.T) {
	base := &aliasTestAgent{name: "claude", command: "claude"}
	alias := NewAlias("custom", base, "/custom/bin/claude", nil, nil, "")

	if alias.binaryPath != "/custom/bin/claude" {
		t.Errorf("binaryPath = %q, want %q", alias.binaryPath, "/custom/bin/claude")
	}
}

func TestAliasAgent_ConfiguredAppliesBinaryPath(t *testing.T) {
	base := &aliasTestAgent{name: "claude", command: "claude"}
	alias := NewAlias("custom", base, "/custom/bin/claude", nil, nil, "")

	configured := alias.configured()

	mock, ok := configured.(*aliasTestAgent)
	if !ok {
		t.Fatal("configured() should return *aliasTestAgent")
	}
	if mock.command != "/custom/bin/claude" {
		t.Errorf("command = %q, want %q", mock.command, "/custom/bin/claude")
	}
}

func TestAliasAgent_ConfiguredAppliesEnvAndArgs(t *testing.T) {
	base := &aliasTestAgent{name: "claude", command: "claude"}
	alias := NewAlias("custom", base, "", map[string]string{"KEY": "val"}, []string{"--arg"}, "")

	configured := alias.configured()

	mock, ok := configured.(*aliasTestAgent)
	if !ok {
		t.Fatal("configured() should return *aliasTestAgent")
	}
	if mock.env["KEY"] != "val" {
		t.Error("env should include KEY=val")
	}
	if len(mock.args) != 1 || mock.args[0] != "--arg" {
		t.Error("args should include --arg")
	}
}

func TestAliasAgent_ConfiguredAppliesAllInOrder(t *testing.T) {
	// Binary path should be applied first, then env, then args
	base := &aliasTestAgent{name: "claude", command: "claude"}
	alias := NewAlias("custom", base, "/custom/bin", map[string]string{"K": "V"}, []string{"--a"}, "")

	configured := alias.configured()

	mock, ok := configured.(*aliasTestAgent)
	if !ok {
		t.Fatal("configured() should return *aliasTestAgent")
	}
	if mock.command != "/custom/bin" {
		t.Errorf("command = %q, want %q", mock.command, "/custom/bin")
	}
	if mock.env["K"] != "V" {
		t.Error("env should include K=V")
	}
	if len(mock.args) != 1 || mock.args[0] != "--a" {
		t.Error("args should include --a")
	}
}

func TestAliasAgent_WithEnvPreservesBinaryPath(t *testing.T) {
	base := &aliasTestAgent{name: "claude", command: "claude"}
	alias := NewAlias("custom", base, "/custom/bin", nil, nil, "")

	newAlias := alias.WithEnv("KEY", "val")

	typed, ok := newAlias.(*AliasAgent)
	if !ok {
		t.Fatal("WithEnv should return *AliasAgent")
	}
	if typed.binaryPath != "/custom/bin" {
		t.Errorf("binaryPath = %q, want %q", typed.binaryPath, "/custom/bin")
	}
}

func TestAliasAgent_WithArgsPreservesBinaryPath(t *testing.T) {
	base := &aliasTestAgent{name: "claude", command: "claude"}
	alias := NewAlias("custom", base, "/custom/bin", nil, nil, "")

	newAlias := alias.WithArgs("--arg")

	typed, ok := newAlias.(*AliasAgent)
	if !ok {
		t.Fatal("WithArgs should return *AliasAgent")
	}
	if typed.binaryPath != "/custom/bin" {
		t.Errorf("binaryPath = %q, want %q", typed.binaryPath, "/custom/bin")
	}
}

func TestAliasAgent_EmptyBinaryPathSkipsCommand(t *testing.T) {
	base := &aliasTestAgent{name: "claude", command: "original"}
	alias := NewAlias("custom", base, "", nil, nil, "")

	configured := alias.configured()

	mock, ok := configured.(*aliasTestAgent)
	if !ok {
		t.Fatal("configured() should return *aliasTestAgent")
	}
	// Command should remain unchanged since binaryPath is empty
	if mock.command != "original" {
		t.Errorf("command = %q, want %q (unchanged)", mock.command, "original")
	}
}
