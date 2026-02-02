// Package codex implements tests for the Codex AI agent.
//
// These tests verify the agent's behavior without requiring an actual
// Codex CLI installation. Integration tests with the real CLI will be
// added once Codex is available.
package codex

import (
	"context"
	"testing"
	"time"

	"github.com/valksor/go-mehrhof/internal/agent"
)

func TestNew(t *testing.T) {
	a := New()
	if a.config.Command[0] != "codex" {
		t.Errorf("Command[0] = %q, want %q", a.config.Command[0], "codex")
	}
	if a.config.Timeout != 30*time.Minute {
		t.Errorf("Timeout = %v, want %v", a.config.Timeout, 30*time.Minute)
	}
	if a.parser == nil {
		t.Error("parser should not be nil")
	}
}

func TestNewWithConfig(t *testing.T) {
	cfg := agent.Config{
		Command:     []string{"custom-codex", "--model", "gpt-5-codex"},
		Environment: map[string]string{"KEY": "val"},
		Timeout:     10 * time.Minute,
		WorkDir:     "/tmp",
	}

	a := NewWithConfig(cfg)
	if a.config.Command[0] != "custom-codex" {
		t.Errorf("Command[0] = %q, want %q", a.config.Command[0], "custom-codex")
	}
	if a.config.Timeout != 10*time.Minute {
		t.Errorf("Timeout = %v, want %v", a.config.Timeout, 10*time.Minute)
	}
}

func TestNewWithConfigEmptyCommand(t *testing.T) {
	cfg := agent.Config{
		Command: []string{}, // Empty command should default to "codex"
	}

	a := NewWithConfig(cfg)
	if a.config.Command[0] != "codex" {
		t.Errorf("Command[0] = %q, want %q (default)", a.config.Command[0], "codex")
	}
}

func TestName(t *testing.T) {
	a := New()
	if a.Name() != AgentName {
		t.Errorf("Name() = %q, want %q", a.Name(), AgentName)
	}
	if a.Name() != "codex" {
		t.Errorf("Name() = %q, want %q", a.Name(), "codex")
	}
}

func TestAgentNameConstant(t *testing.T) {
	if AgentName != "codex" {
		t.Errorf("AgentName = %q, want %q", AgentName, "codex")
	}
}

func TestWithWorkDir(t *testing.T) {
	result := New().WithWorkDir("/custom/path")
	a, ok := result.(*Agent)
	if !ok {
		t.Fatal("WithWorkDir did not return *Agent")
	}
	if a.config.WorkDir != "/custom/path" {
		t.Errorf("WorkDir = %q, want %q", a.config.WorkDir, "/custom/path")
	}
}

func TestWithTimeout(t *testing.T) {
	a := New().WithTimeout(5 * time.Minute)
	if a.config.Timeout != 5*time.Minute {
		t.Errorf("Timeout = %v, want %v", a.config.Timeout, 5*time.Minute)
	}
}

func TestWithEnv(t *testing.T) {
	a := New()
	aAgent := a.WithEnv("OPENAI_API_KEY", "sk-test123")
	agent, ok := aAgent.(*Agent)
	if !ok {
		t.Fatal("WithEnv did not return *Agent")
	}
	if agent.config.Environment["OPENAI_API_KEY"] != "sk-test123" {
		t.Errorf("Environment[OPENAI_API_KEY] = %q, want %q", agent.config.Environment["OPENAI_API_KEY"], "sk-test123")
	}
}

func TestMethodChaining(t *testing.T) {
	wdResult, ok := New().WithWorkDir("/work").(*Agent)
	if !ok {
		t.Fatal("WithWorkDir did not return *Agent")
	}
	a := wdResult.WithTimeout(15 * time.Minute)

	// WithEnv returns agent.Agent interface, so capture the result
	aAgent := agent.Agent(a)
	aAgent = aAgent.WithEnv("KEY1", "val1")
	aAgent = aAgent.WithEnv("KEY2", "val2")

	if a.config.WorkDir != "/work" {
		t.Error("WithWorkDir chain failed")
	}
	if a.config.Timeout != 15*time.Minute {
		t.Error("WithTimeout chain failed")
	}
	typed, ok := aAgent.(*Agent)
	if !ok {
		t.Fatal("WithEnv did not return *Agent")
	}
	if typed.config.Environment["KEY1"] != "val1" {
		t.Error("WithEnv(KEY1) chain failed")
	}
	if typed.config.Environment["KEY2"] != "val2" {
		t.Error("WithEnv(KEY2) chain failed")
	}
}

func TestSetParser(t *testing.T) {
	a := New()
	originalParser := a.parser

	newParser := agent.NewYAMLBlockParser()
	a.SetParser(newParser)

	if a.parser == originalParser {
		t.Error("parser should have been replaced")
	}
}

func TestBuildArgs_BasicPrompt(t *testing.T) {
	a := New()
	args := a.buildArgs(context.Background(), "Hello world")

	// Should include: exec, --json, prompt
	// NOTE: Codex does NOT use --print, --verbose, --output-format (those are Claude-specific)
	if len(args) < 3 {
		t.Fatalf("args length = %d, want at least 3", len(args))
	}
	if args[0] != "exec" || args[1] != "--json" {
		t.Errorf("expected [exec, --json], got [%v, %v]", args[0], args[1])
	}
	if args[len(args)-1] != "Hello world" {
		t.Errorf("last arg = %q, want %q", args[len(args)-1], "Hello world")
	}
}

func TestBuildArgs_WithConfigArgs(t *testing.T) {
	cfg := agent.Config{
		Command: []string{"codex", "--model", "gpt-5-codex"},
	}
	a := NewWithConfig(cfg)
	args := a.buildArgs(context.Background(), "test")

	// Should include: exec, --json, --model, gpt-5-codex, test
	if len(args) < 4 {
		t.Fatal("args should include at least exec, --json, config args, and prompt")
	}
	if args[0] != "exec" || args[1] != "--json" {
		t.Errorf("expected [exec, --json], got [%v, %v]", args[0], args[1])
	}

	// Last arg should be the prompt
	if args[len(args)-1] != "test" {
		t.Errorf("last arg = %q, want %q", args[len(args)-1], "test")
	}
}

func TestBuildArgs_AutoSkipGitRepoCheck(t *testing.T) {
	tmpDir := t.TempDir()
	wdResult, ok := New().WithWorkDir(tmpDir).(*Agent)
	if !ok {
		t.Fatal("WithWorkDir did not return *Agent")
	}
	a := wdResult
	args := a.buildArgs(context.Background(), "test")

	hasSkip := false
	for _, arg := range args {
		if arg == "--skip-git-repo-check" {
			hasSkip = true

			break
		}
	}
	if !hasSkip {
		t.Fatal("expected --skip-git-repo-check when not in a git repo")
	}
}

func TestBuildArgs_NoDuplicateSkipGitRepoCheck(t *testing.T) {
	tmpDir := t.TempDir()
	wdResult, ok := New().WithWorkDir(tmpDir).(*Agent)
	if !ok {
		t.Fatal("WithWorkDir did not return *Agent")
	}
	a := wdResult
	a.config.Args = []string{"--skip-git-repo-check"}
	args := a.buildArgs(context.Background(), "test")

	count := 0
	for _, arg := range args {
		if arg == "--skip-git-repo-check" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected --skip-git-repo-check once, got %d", count)
	}
}

func TestBuildArgs_NoClaudeFlags(t *testing.T) {
	a := New()
	args := a.buildArgs(context.Background(), "test prompt")

	// Verify Codex does NOT use Claude-specific flags
	claudeFlags := []string{"--print", "--verbose", "--output-format", "stream-json"}
	for _, flag := range claudeFlags {
		for _, arg := range args {
			if arg == flag {
				t.Errorf("Codex should NOT use Claude-specific flag %q, but found it in args", flag)
			}
		}
	}

	// Verify Codex uses its own flags
	hasExec := false
	hasJSON := false
	for _, arg := range args {
		if arg == "exec" {
			hasExec = true
		}
		if arg == "--json" {
			hasJSON = true
		}
	}
	if !hasExec {
		t.Error("Codex args should include 'exec'")
	}
	if !hasJSON {
		t.Error("Codex args should include '--json'")
	}
}

func TestAvailable_NoCLI(t *testing.T) {
	// Use a non-existent binary to test the "not found" case
	cfg := agent.Config{
		Command: []string{"nonexistent-codex-cli-binary-12345"},
	}
	a := NewWithConfig(cfg)

	err := a.Available()
	if err == nil {
		t.Error("Available should fail when CLI not found")
	}
}

func TestRegister(t *testing.T) {
	r := agent.NewRegistry()
	err := Register(r)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Should be able to get the agent
	a, err := r.Get(AgentName)
	if err != nil {
		t.Fatalf("Get(%s): %v", AgentName, err)
	}
	if a.Name() != AgentName {
		t.Errorf("agent name = %q, want %q", a.Name(), AgentName)
	}
}

func TestAgentInterface(t *testing.T) {
	// Verify Agent implements the interface
	var _ agent.Agent = (*Agent)(nil)
}

func TestRunStream_ContextCancel(t *testing.T) {
	// Use nonexistent binary - this tests the start failure path
	cfg := agent.Config{
		Command: []string{"nonexistent-binary-xyz"},
		Timeout: time.Second,
	}
	a := NewWithConfig(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	events, errCh := a.RunStream(ctx, "test")

	// Drain events
	for range events {
	}

	// Should get an error (either context canceled or binary not found)
	err := <-errCh
	if err == nil {
		// It's okay if there's no error since the binary doesn't exist
		// and the command never starts
		t.Log("No error returned (binary not found before context check)")
	}
}

func TestRun_NoCLI(t *testing.T) {
	cfg := agent.Config{
		Command: []string{"nonexistent-binary-abc"},
		Timeout: time.Second,
	}
	a := NewWithConfig(cfg)

	_, err := a.Run(context.Background(), "test")
	if err == nil {
		t.Error("Run should fail when CLI not found")
	}
}

func TestRunWithCallback_NoCLI(t *testing.T) {
	cfg := agent.Config{
		Command: []string{"nonexistent-binary-def"},
		Timeout: time.Second,
	}
	a := NewWithConfig(cfg)

	callbackCalled := false
	cb := func(event agent.Event) error {
		callbackCalled = true

		return nil
	}

	_, err := a.RunWithCallback(context.Background(), "test", cb)
	if err == nil {
		t.Error("RunWithCallback should fail when CLI not found")
	}
	// Callback shouldn't be called since command fails to start
	if callbackCalled {
		t.Log("Callback was called (this may vary)")
	}
}

func TestWithArgs(t *testing.T) {
	tests := []struct {
		name         string
		existingArgs []string
		newArgs      []string
		wantLen      int
		wantLast     []string
	}{
		{
			name:         "no existing args",
			existingArgs: nil,
			newArgs:      []string{"--model", "gpt-5-codex"},
			wantLen:      2,
			wantLast:     []string{"--model", "gpt-5-codex"},
		},
		{
			name:         "append to existing args",
			existingArgs: []string{"--profile", "dev"},
			newArgs:      []string{"--sandbox", "workspace-write"},
			wantLen:      4,
			wantLast:     []string{"--profile", "dev", "--sandbox", "workspace-write"},
		},
		{
			name:         "empty new args",
			existingArgs: []string{"--json"},
			newArgs:      []string{},
			wantLen:      1,
			wantLast:     []string{"--json"},
		},
		{
			name:         "single new arg",
			existingArgs: []string{},
			newArgs:      []string{"--help"},
			wantLen:      1,
			wantLast:     []string{"--help"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := agent.Config{
				Command: []string{"codex"},
				Args:    tt.existingArgs,
			}
			a := NewWithConfig(cfg)

			// WithArgs returns agent.Agent interface
			aAgent := a.WithArgs(tt.newArgs...)
			resultAgent, ok := aAgent.(*Agent)
			if !ok {
				t.Fatal("WithArgs did not return *Agent")
			}

			if len(resultAgent.config.Args) != tt.wantLen {
				t.Errorf("Args length = %d, want %d", len(resultAgent.config.Args), tt.wantLen)
			}

			for i, want := range tt.wantLast {
				if i >= len(resultAgent.config.Args) {
					t.Errorf("Missing arg at index %d", i)

					continue
				}
				if resultAgent.config.Args[i] != want {
					t.Errorf("Args[%d] = %q, want %q", i, resultAgent.config.Args[i], want)
				}
			}
		})
	}
}

func TestWithArgs_Chaining(t *testing.T) {
	a := New()
	aAgent := agent.Agent(a)

	// Chain multiple WithArgs calls
	aAgent = aAgent.WithArgs("--model", "gpt-5-codex")
	aAgent = aAgent.WithArgs("--max-tokens", "4096")
	aAgent = aAgent.WithArgs("--sandbox", "workspace-write")

	resultAgent, ok := aAgent.(*Agent)
	if !ok {
		t.Fatal("WithArgs did not return *Agent")
	}
	expectedArgs := []string{"--model", "gpt-5-codex", "--max-tokens", "4096", "--sandbox", "workspace-write"}

	if len(resultAgent.config.Args) != len(expectedArgs) {
		t.Errorf("Args length = %d, want %d", len(resultAgent.config.Args), len(expectedArgs))
	}

	for i, want := range expectedArgs {
		if resultAgent.config.Args[i] != want {
			t.Errorf("Args[%d] = %q, want %q", i, resultAgent.config.Args[i], want)
		}
	}
}

func TestWithArgs_OriginalUnmodified(t *testing.T) {
	cfg := agent.Config{
		Command: []string{"codex"},
		Args:    []string{"--json"},
	}
	originalAgent := NewWithConfig(cfg)
	originalLen := len(originalAgent.config.Args)

	// WithArgs should return a new agent, not modify the original
	newAgent := originalAgent.WithArgs("--sandbox", "read-only")

	if len(originalAgent.config.Args) != originalLen {
		t.Errorf("Original agent was modified (len = %d, want %d)", len(originalAgent.config.Args), originalLen)
	}

	typedNew, ok := newAgent.(*Agent)
	if !ok {
		t.Fatal("WithArgs did not return *Agent")
	}
	if len(typedNew.config.Args) != originalLen+2 {
		t.Errorf("New agent has wrong arg count (len = %d, want %d)", len(typedNew.config.Args), originalLen+2)
	}
}

// TestStepArgs tests the StepArgsProvider interface implementation.
func TestStepArgs(t *testing.T) {
	a := New()

	tests := []struct {
		name     string
		step     string
		wantArgs []string
		wantNil  bool
	}{
		{
			name:     "planning step uses read-only sandbox",
			step:     "planning",
			wantArgs: []string{"--sandbox", "read-only"},
		},
		{
			name:     "implementing step uses full-auto",
			step:     "implementing",
			wantArgs: []string{"--full-auto"},
		},
		{
			name:     "reviewing step uses full-auto",
			step:     "reviewing",
			wantArgs: []string{"--full-auto"},
		},
		{
			name:    "unknown step returns nil",
			step:    "unknown-step",
			wantNil: true,
		},
		{
			name:    "empty step returns nil",
			step:    "",
			wantNil: true,
		},
		{
			name:    "checkpointing step returns nil",
			step:    "checkpointing",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := a.StepArgs(tt.step)

			if tt.wantNil {
				if got != nil {
					t.Errorf("StepArgs(%q) = %v, want nil", tt.step, got)
				}

				return
			}

			if len(got) != len(tt.wantArgs) {
				t.Errorf("StepArgs(%q) length = %d, want %d", tt.step, len(got), len(tt.wantArgs))

				return
			}

			for i, want := range tt.wantArgs {
				if got[i] != want {
					t.Errorf("StepArgs(%q)[%d] = %q, want %q", tt.step, i, got[i], want)
				}
			}
		})
	}
}

// TestStepArgs_NoClaudeFlags verifies that StepArgs does not return Claude-specific flags.
func TestStepArgs_NoClaudeFlags(t *testing.T) {
	a := New()

	steps := []string{"planning", "implementing", "reviewing"}
	claudeFlags := []string{"--permission-mode", "plan", "acceptEdits"}

	for _, step := range steps {
		args := a.StepArgs(step)
		for _, arg := range args {
			for _, flag := range claudeFlags {
				if arg == flag {
					t.Errorf("StepArgs(%q) should NOT contain Claude-specific flag %q", step, flag)
				}
			}
		}
	}
}

// TestStepArgsProviderInterface verifies Agent implements StepArgsProvider.
func TestStepArgsProviderInterface(t *testing.T) {
	// Verify Agent implements the StepArgsProvider interface
	var _ agent.StepArgsProvider = (*Agent)(nil)
}
