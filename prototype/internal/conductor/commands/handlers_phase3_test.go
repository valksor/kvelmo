package commands

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestPhase3CommandsRegistered(t *testing.T) {
	commands := []string{
		"reset", "auto", "find", "simplify", "label",
		"memory", "library", "links", "question",
		"delete", "export", "optimize", "submit", "submit-source", "sync",
		"specifications", "sessions",
	}

	for _, name := range commands {
		if !IsKnownCommand(name) {
			t.Fatalf("command %q must be registered in router", name)
		}
	}
}

func TestPhase3CommandValidationPaths(t *testing.T) {
	cond := mustNewConductor(t)

	tests := []struct {
		name      string
		executeFn func() (*Result, error)
		wantErr   string
	}{
		{
			name: "find requires query",
			executeFn: func() (*Result, error) {
				return Execute(context.Background(), cond, "find", Invocation{})
			},
			wantErr: "find requires a query",
		},
		{
			name: "links needs workspace",
			executeFn: func() (*Result, error) {
				return Execute(context.Background(), cond, "links", Invocation{Args: []string{"list"}})
			},
			wantErr: "workspace not initialized",
		},
		{
			name: "delete requires queue task ref",
			executeFn: func() (*Result, error) {
				return Execute(context.Background(), cond, "delete", Invocation{})
			},
			wantErr: "delete requires a task reference",
		},
		{
			name: "export requires queue task ref",
			executeFn: func() (*Result, error) {
				return Execute(context.Background(), cond, "export", Invocation{})
			},
			wantErr: "export requires a task reference",
		},
		{
			name: "optimize requires queue task ref",
			executeFn: func() (*Result, error) {
				return Execute(context.Background(), cond, "optimize", Invocation{})
			},
			wantErr: "optimize requires a task reference",
		},
		{
			name: "submit requires queue task and provider",
			executeFn: func() (*Result, error) {
				return Execute(context.Background(), cond, "submit", Invocation{})
			},
			wantErr: "submit requires: submit",
		},
		{
			name: "submit-source requires source",
			executeFn: func() (*Result, error) {
				return Execute(context.Background(), cond, "submit-source", Invocation{})
			},
			wantErr: "source is required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.executeFn()
			if result != nil {
				t.Fatalf("expected nil result, got %#v", result)
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestPhase3DataCommandsDisabledFallback(t *testing.T) {
	cond := mustNewConductor(t)

	memoryResult, err := Execute(context.Background(), cond, "memory", Invocation{Args: []string{"search", "foo"}})
	if err != nil {
		t.Fatalf("memory search should not error when disabled: %v", err)
	}
	if memoryResult == nil {
		t.Fatal("expected memory result")
	}
	memoryData, ok := memoryResult.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected memory data map, got %T", memoryResult.Data)
	}
	if enabled, _ := memoryData["enabled"].(bool); enabled {
		t.Fatalf("expected memory enabled=false, got %v", memoryData["enabled"])
	}

	libraryResult, err := Execute(context.Background(), cond, "library", Invocation{Args: []string{"list"}})
	if err != nil {
		t.Fatalf("library list should not error when disabled: %v", err)
	}
	if libraryResult == nil {
		t.Fatal("expected library result")
	}
	libraryData, ok := libraryResult.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected library data map, got %T", libraryResult.Data)
	}
	if enabled, _ := libraryData["enabled"].(bool); enabled {
		t.Fatalf("expected library enabled=false, got %v", libraryData["enabled"])
	}
}

func TestPhase3TaskRequiredCommands(t *testing.T) {
	cond := mustNewConductor(t)

	tests := []struct {
		name string
		cmd  string
		args []string
	}{
		{name: "reset", cmd: "reset"},
		{name: "simplify", cmd: "simplify"},
		{name: "sync", cmd: "sync"},
		{name: "question", cmd: "question", args: []string{"why"}},
		{name: "label", cmd: "label", args: []string{"list"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Execute(context.Background(), cond, tc.cmd, Invocation{Args: tc.args})
			if result != nil {
				t.Fatalf("expected nil result, got %#v", result)
			}
			if !errors.Is(err, ErrNoActiveTask) {
				t.Fatalf("expected ErrNoActiveTask, got %v", err)
			}
		})
	}
}
