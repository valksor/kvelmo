package commands

import (
	"context"
	"errors"
	"testing"

	"github.com/valksor/go-mehrhof/internal/conductor"
)

func mustNewConductor(t *testing.T) *conductor.Conductor {
	t.Helper()

	cond, err := conductor.New()
	if err != nil {
		t.Fatalf("conductor.New() failed: %v", err)
	}

	return cond
}

func TestExecuteUnknownCommand(t *testing.T) {
	cond := mustNewConductor(t)

	result, err := Execute(context.Background(), cond, "definitely-unknown-command", Invocation{})
	if result != nil {
		t.Fatalf("expected nil result, got %#v", result)
	}
	if !errors.Is(err, ErrUnknownCommand) {
		t.Fatalf("expected ErrUnknownCommand, got %v", err)
	}
}

func TestExecuteRequiresTask(t *testing.T) {
	cond := mustNewConductor(t)
	handlerCalled := false

	Register(Command{
		Info: CommandInfo{
			Name:         "requires-task-test",
			Aliases:      []string{"rtt"},
			Description:  "test command requiring task",
			Category:     "test",
			RequiresTask: true,
		},
		Handler: func(_ context.Context, _ *conductor.Conductor, _ Invocation) (*Result, error) {
			handlerCalled = true

			return NewResult("ok"), nil
		},
	})

	result, err := Execute(context.Background(), cond, "rtt", Invocation{})
	if result != nil {
		t.Fatalf("expected nil result, got %#v", result)
	}
	if !errors.Is(err, ErrNoActiveTask) {
		t.Fatalf("expected ErrNoActiveTask, got %v", err)
	}
	if handlerCalled {
		t.Fatalf("handler should not be called when no active task exists")
	}
}

func TestExecuteAliasAndHandlerError(t *testing.T) {
	cond := mustNewConductor(t)
	expectedErr := errors.New("handler failed")
	receivedArgs := []string(nil)

	Register(Command{
		Info: CommandInfo{
			Name:         "execute-alias-test",
			Aliases:      []string{"eat"},
			Description:  "test command using alias",
			Category:     "test",
			RequiresTask: false,
		},
		Handler: func(_ context.Context, _ *conductor.Conductor, inv Invocation) (*Result, error) {
			receivedArgs = append(receivedArgs, inv.Args...)

			return nil, expectedErr
		},
	})

	result, err := Execute(context.Background(), cond, "eat", Invocation{Args: []string{"one", "two"}})
	if result != nil {
		t.Fatalf("expected nil result, got %#v", result)
	}
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected handler error, got %v", err)
	}
	if len(receivedArgs) != 2 || receivedArgs[0] != "one" || receivedArgs[1] != "two" {
		t.Fatalf("received args = %#v", receivedArgs)
	}
}

func TestExecuteReturnsResult(t *testing.T) {
	cond := mustNewConductor(t)

	Register(Command{
		Info: CommandInfo{
			Name:         "execute-ok-test",
			Aliases:      []string{"eot"},
			Description:  "test command returning result",
			Category:     "test",
			RequiresTask: false,
		},
		Handler: func(_ context.Context, _ *conductor.Conductor, _ Invocation) (*Result, error) {
			return &Result{
				Type:    ResultMessage,
				Message: "ok",
			}, nil
		},
	})

	result, err := Execute(context.Background(), cond, "eot", Invocation{})
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}
	if result == nil || result.Message != "ok" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if result.State != "" || result.TaskID != "" {
		t.Fatalf("state/task should remain empty without active task, got %#v", result)
	}
}

func TestParseInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantCmd  string
		wantArgs []string
	}{
		{
			name:     "empty input",
			input:    "   ",
			wantCmd:  "",
			wantArgs: nil,
		},
		{
			name:     "command only",
			input:    "STATUS",
			wantCmd:  "status",
			wantArgs: []string{},
		},
		{
			name:     "command with args",
			input:    "review view 3",
			wantCmd:  "review",
			wantArgs: []string{"view", "3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCmd, gotArgs := ParseInput(tt.input)
			if gotCmd != tt.wantCmd {
				t.Fatalf("ParseInput(%q) command = %q, want %q", tt.input, gotCmd, tt.wantCmd)
			}
			if len(gotArgs) != len(tt.wantArgs) {
				t.Fatalf("ParseInput(%q) args len = %d, want %d", tt.input, len(gotArgs), len(tt.wantArgs))
			}
			for i := range gotArgs {
				if gotArgs[i] != tt.wantArgs[i] {
					t.Fatalf("ParseInput(%q) args[%d] = %q, want %q", tt.input, i, gotArgs[i], tt.wantArgs[i])
				}
			}
		})
	}
}

func TestMetadataCategoriesAndLookup(t *testing.T) {
	Register(Command{
		Info: CommandInfo{
			Name:         "metadata-test-command",
			Aliases:      []string{"mtc"},
			Description:  "metadata command for testing",
			Category:     "metadata-test",
			RequiresTask: false,
		},
		Handler: func(_ context.Context, _ *conductor.Conductor, _ Invocation) (*Result, error) {
			return NewResult("ok"), nil
		},
	})

	all := Metadata()
	found := false
	for _, info := range all {
		if info.Name == "metadata-test-command" {
			found = true

			break
		}
	}
	if !found {
		t.Fatalf("Metadata() missing test command")
	}

	categories := Categories()
	group, ok := categories["metadata-test"]
	if !ok || len(group) == 0 {
		t.Fatalf("Categories() missing metadata-test category")
	}

	info, ok := GetCommandInfo("mtc")
	if !ok {
		t.Fatalf("GetCommandInfo(alias) expected true")
	}
	if info.Name != "metadata-test-command" {
		t.Fatalf("GetCommandInfo(alias).Name = %q", info.Name)
	}
}

func TestIsKnownCommandAndCurrentState(t *testing.T) {
	if !IsKnownCommand("status") {
		t.Fatalf("expected status to be a known command")
	}
	if IsKnownCommand("unknown-never-registered") {
		t.Fatalf("unexpected known command for unknown name")
	}

	cond := mustNewConductor(t)
	if state := GetCurrentState(cond); state != "idle" {
		t.Fatalf("GetCurrentState() = %q, want idle", state)
	}
}

func TestResultBuilders(t *testing.T) {
	msg := NewResult("hello")
	if msg.Type != ResultMessage || msg.Message != "hello" {
		t.Fatalf("unexpected NewResult: %#v", msg)
	}

	errResult := NewErrorResult(errors.New("boom"))
	if errResult.Type != ResultError || errResult.Message != "boom" {
		t.Fatalf("unexpected NewErrorResult: %#v", errResult)
	}

	status := NewStatusResult("ok", "planning", "task-1", map[string]string{"x": "y"})
	if status.Type != ResultStatus || status.State != "planning" || status.TaskID != "task-1" {
		t.Fatalf("unexpected NewStatusResult: %#v", status)
	}

	updated := NewResult("base").WithState("reviewing").WithTaskID("task-2").WithData(123)
	if updated.State != "reviewing" || updated.TaskID != "task-2" || updated.Data != 123 {
		t.Fatalf("unexpected chained result mutation: %#v", updated)
	}

	if ExitResult.Type != ResultExit {
		t.Fatalf("ExitResult type = %q, want %q", ExitResult.Type, ResultExit)
	}
}

func TestExecuteWithRun(t *testing.T) {
	cond := mustNewConductor(t)
	executed := false

	Register(Command{
		Info: CommandInfo{
			Name:        "execute-with-run-test",
			Description: "test execute with run",
			Category:    "test",
		},
		Handler: func(_ context.Context, _ *conductor.Conductor, _ Invocation) (*Result, error) {
			return &Result{
				Type:    ResultMessage,
				Message: "ok",
				Executor: func(context.Context) error {
					executed = true

					return nil
				},
			}, nil
		},
	})

	_, err := Execute(context.Background(), cond, "execute-with-run-test", Invocation{})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if executed {
		t.Fatalf("Execute should not run executor")
	}

	_, err = ExecuteWithRun(context.Background(), cond, "execute-with-run-test", Invocation{})
	if err != nil {
		t.Fatalf("ExecuteWithRun returned error: %v", err)
	}
	if !executed {
		t.Fatalf("ExecuteWithRun should run executor")
	}
}

func TestExecuteWithRunClassifiesExecutorError(t *testing.T) {
	cond := mustNewConductor(t)

	Register(Command{
		Info: CommandInfo{
			Name:        "execute-classify-test",
			Description: "test classify",
			Category:    "test",
		},
		Handler: func(_ context.Context, _ *conductor.Conductor, _ Invocation) (*Result, error) {
			return &Result{
				Type:    ResultMessage,
				Message: "ok",
				State:   "planning",
				TaskID:  "task-1",
				Executor: func(context.Context) error {
					return conductor.ErrBudgetPaused
				},
			}, nil
		},
	})

	result, err := ExecuteWithRun(context.Background(), cond, "execute-classify-test", Invocation{})
	if err != nil {
		t.Fatalf("ExecuteWithRun returned error: %v", err)
	}
	if result == nil || result.Type != ResultPaused {
		t.Fatalf("expected paused result, got %#v", result)
	}
}

func TestExecuteWithRunClassifiesExecuteError(t *testing.T) {
	cond := mustNewConductor(t)

	Register(Command{
		Info: CommandInfo{
			Name:        "execute-classify-stage-test",
			Description: "test classify execute-stage errors",
			Category:    "test",
		},
		Handler: func(_ context.Context, _ *conductor.Conductor, _ Invocation) (*Result, error) {
			return nil, conductor.ErrPendingQuestion
		},
	})

	result, err := ExecuteWithRun(context.Background(), cond, "execute-classify-stage-test", Invocation{})
	if err != nil {
		t.Fatalf("ExecuteWithRun returned error: %v", err)
	}
	if result == nil || result.Type != ResultWaiting {
		t.Fatalf("expected waiting result, got %#v", result)
	}
}

func TestExecuteWithRunPreservesRouterErrors(t *testing.T) {
	cond := mustNewConductor(t)

	result, err := ExecuteWithRun(context.Background(), cond, "definitely-unknown-command", Invocation{})
	if result != nil {
		t.Fatalf("expected nil result, got %#v", result)
	}
	if !errors.Is(err, ErrUnknownCommand) {
		t.Fatalf("expected ErrUnknownCommand, got %v", err)
	}

	Register(Command{
		Info: CommandInfo{
			Name:         "execute-with-run-no-task-test",
			Description:  "test no active task pass-through",
			Category:     "test",
			RequiresTask: true,
		},
		Handler: func(_ context.Context, _ *conductor.Conductor, _ Invocation) (*Result, error) {
			return NewResult("ok"), nil
		},
	})

	result, err = ExecuteWithRun(context.Background(), cond, "execute-with-run-no-task-test", Invocation{})
	if result != nil {
		t.Fatalf("expected nil result, got %#v", result)
	}
	if !errors.Is(err, ErrNoActiveTask) {
		t.Fatalf("expected ErrNoActiveTask, got %v", err)
	}
}
