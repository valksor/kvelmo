package commands

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestSimpleControlHandlers(t *testing.T) {
	tests := []struct {
		name    string
		handler func(context.Context, *any, Invocation) (*Result, error)
		wantMsg string
		wantTyp ResultType
	}{
		{
			name: "exit returns ExitResult",
			handler: func(ctx context.Context, _ *any, inv Invocation) (*Result, error) {
				return handleExit(ctx, nil, inv)
			},
			wantMsg: "", // ExitResult has no message
			wantTyp: ResultExit,
		},
		{
			name: "clear returns clear message",
			handler: func(ctx context.Context, _ *any, inv Invocation) (*Result, error) {
				return handleClear(ctx, nil, inv)
			},
			wantMsg: "clear",
			wantTyp: ResultMessage,
		},
		{
			name: "help returns help result",
			handler: func(ctx context.Context, _ *any, inv Invocation) (*Result, error) {
				return handleHelp(ctx, nil, inv)
			},
			wantMsg: "",
			wantTyp: ResultHelp,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.handler(context.Background(), nil, Invocation{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result")
			}
			if result.Type != tt.wantTyp {
				t.Errorf("Type = %v, want %v", result.Type, tt.wantTyp)
			}
			if tt.wantMsg != "" && result.Message != tt.wantMsg {
				t.Errorf("Message = %q, want %q", result.Message, tt.wantMsg)
			}
		})
	}
}

func TestHandleAnswerValidation(t *testing.T) {
	cond := mustNewConductor(t)

	result, err := handleAnswer(context.Background(), cond, Invocation{})
	if result != nil {
		t.Fatalf("expected nil result, got %#v", result)
	}
	if err == nil || !strings.Contains(err.Error(), "answer requires a response") {
		t.Fatalf("expected 'answer requires a response' error, got %v", err)
	}
}

func TestHandleAnswerNoWorkspace(t *testing.T) {
	cond := mustNewConductor(t)

	// With args but no workspace/task
	result, err := handleAnswer(context.Background(), cond, Invocation{Args: []string{"my", "response"}})
	if result != nil {
		t.Fatalf("expected nil result, got %#v", result)
	}
	// Should fail because no task/workspace
	if err == nil {
		t.Fatal("expected error without workspace")
	}
}

func TestHandleNoteValidation(t *testing.T) {
	cond := mustNewConductor(t)

	tests := []struct {
		name   string
		inv    Invocation
		errSub string
	}{
		{
			name:   "empty message",
			inv:    Invocation{},
			errSub: "note requires a message",
		},
		{
			name:   "message but no workspace",
			inv:    Invocation{Args: []string{"my note"}},
			errSub: "workspace not initialized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := handleNote(context.Background(), cond, tt.inv)
			if result != nil {
				t.Fatalf("expected nil result, got %#v", result)
			}
			if err == nil || !strings.Contains(err.Error(), tt.errSub) {
				t.Fatalf("expected error containing %q, got %v", tt.errSub, err)
			}
		})
	}
}

func TestHandleQuickValidation(t *testing.T) {
	cond := mustNewConductor(t)

	result, err := handleQuick(context.Background(), cond, Invocation{})
	if result != nil {
		t.Fatalf("expected nil result, got %#v", result)
	}
	if err == nil || !strings.Contains(err.Error(), "quick requires a description") {
		t.Fatalf("expected 'quick requires a description' error, got %v", err)
	}
}

func TestHandleQuestionValidation(t *testing.T) {
	cond := mustNewConductor(t)

	result, err := handleQuestion(context.Background(), cond, Invocation{})
	if result != nil {
		t.Fatalf("expected nil result, got %#v", result)
	}
	if err == nil || !strings.Contains(err.Error(), "question requires a message") {
		t.Fatalf("expected 'question requires a message' error, got %v", err)
	}
}

func TestHandleUndoRedoNotInitialized(t *testing.T) {
	cond := mustNewConductor(t)

	tests := []struct {
		name   string
		call   func() (*Result, error)
		errSub string
	}{
		{
			name: "undo fails",
			call: func() (*Result, error) {
				return handleUndo(context.Background(), cond, Invocation{})
			},
			errSub: "undo failed",
		},
		{
			name: "redo fails",
			call: func() (*Result, error) {
				return handleRedo(context.Background(), cond, Invocation{})
			},
			errSub: "redo failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.call()
			if result != nil {
				t.Fatalf("expected nil result, got %#v", result)
			}
			if err == nil || !strings.Contains(err.Error(), tt.errSub) {
				t.Fatalf("expected error containing %q, got %v", tt.errSub, err)
			}
		})
	}
}

func TestHandleResetNotInitialized(t *testing.T) {
	cond := mustNewConductor(t)

	result, err := handleReset(context.Background(), cond, Invocation{})
	if result != nil {
		t.Fatalf("expected nil result, got %#v", result)
	}
	if err == nil || !strings.Contains(err.Error(), "reset state") {
		t.Fatalf("expected 'reset state' error, got %v", err)
	}
}

func TestHandleSimplifyNotInitialized(t *testing.T) {
	cond := mustNewConductor(t)

	result, err := handleSimplify(context.Background(), cond, Invocation{})
	if result != nil {
		t.Fatalf("expected nil result, got %#v", result)
	}
	if err == nil || !strings.Contains(err.Error(), "simplification failed") {
		t.Fatalf("expected 'simplification failed' error, got %v", err)
	}
}

func TestHandleSyncNoTask(t *testing.T) {
	cond := mustNewConductor(t)

	// Without task ID in args or options, should require active task
	result, err := handleSync(context.Background(), cond, Invocation{})
	if result != nil {
		t.Fatalf("expected nil result, got %#v", result)
	}
	if !errors.Is(err, ErrNoActiveTask) {
		t.Fatalf("expected ErrNoActiveTask, got %v", err)
	}
}

func TestHandleAutoNoTask(t *testing.T) {
	cond := mustNewConductor(t)

	// Without ref, auto needs an active task
	result, err := handleAuto(context.Background(), cond, Invocation{})
	if result != nil {
		t.Fatalf("expected nil result, got %#v", result)
	}
	if !errors.Is(err, ErrNoActiveTask) {
		t.Fatalf("expected ErrNoActiveTask, got %v", err)
	}
}

func TestAutoOptionsDecoding(t *testing.T) {
	tests := []struct {
		name              string
		options           map[string]any
		wantRef           string
		wantMaxRetries    int
		wantNoPush        bool
		wantNoDelete      bool
		wantNoSquash      bool
		wantTargetBranch  string
		wantQualityTarget string
		wantNoQuality     bool
	}{
		{
			name:    "empty options",
			options: map[string]any{},
		},
		{
			name:           "ref option",
			options:        map[string]any{"ref": "github:123"},
			wantRef:        "github:123",
			wantMaxRetries: 0,
		},
		{
			name:           "max retries",
			options:        map[string]any{"max_retries": 5},
			wantMaxRetries: 5,
		},
		{
			name:       "no push flag",
			options:    map[string]any{"no_push": true},
			wantNoPush: true,
		},
		{
			name:         "no delete flag",
			options:      map[string]any{"no_delete": true},
			wantNoDelete: true,
		},
		{
			name:         "no squash flag",
			options:      map[string]any{"no_squash": true},
			wantNoSquash: true,
		},
		{
			name:             "target branch",
			options:          map[string]any{"target_branch": "develop"},
			wantTargetBranch: "develop",
		},
		{
			name:              "quality target",
			options:           map[string]any{"quality_target": "lint"},
			wantQualityTarget: "lint",
		},
		{
			name:          "no quality flag",
			options:       map[string]any{"no_quality": true},
			wantNoQuality: true,
		},
		{
			name: "all options",
			options: map[string]any{
				"ref":            "file:task.md",
				"max_retries":    10,
				"no_push":        true,
				"no_delete":      true,
				"no_squash":      true,
				"target_branch":  "main",
				"quality_target": "test",
				"no_quality":     false,
			},
			wantRef:           "file:task.md",
			wantMaxRetries:    10,
			wantNoPush:        true,
			wantNoDelete:      true,
			wantNoSquash:      true,
			wantTargetBranch:  "main",
			wantQualityTarget: "test",
			wantNoQuality:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv := Invocation{Options: tt.options}
			opts, err := DecodeOptions[autoOptions](inv)
			if err != nil {
				t.Fatalf("DecodeOptions failed: %v", err)
			}
			if opts.Ref != tt.wantRef {
				t.Errorf("Ref = %q, want %q", opts.Ref, tt.wantRef)
			}
			if opts.MaxRetries != tt.wantMaxRetries {
				t.Errorf("MaxRetries = %d, want %d", opts.MaxRetries, tt.wantMaxRetries)
			}
			if opts.NoPush != tt.wantNoPush {
				t.Errorf("NoPush = %v, want %v", opts.NoPush, tt.wantNoPush)
			}
			if opts.NoDelete != tt.wantNoDelete {
				t.Errorf("NoDelete = %v, want %v", opts.NoDelete, tt.wantNoDelete)
			}
			if opts.NoSquash != tt.wantNoSquash {
				t.Errorf("NoSquash = %v, want %v", opts.NoSquash, tt.wantNoSquash)
			}
			if opts.TargetBranch != tt.wantTargetBranch {
				t.Errorf("TargetBranch = %q, want %q", opts.TargetBranch, tt.wantTargetBranch)
			}
			if opts.QualityTarget != tt.wantQualityTarget {
				t.Errorf("QualityTarget = %q, want %q", opts.QualityTarget, tt.wantQualityTarget)
			}
			if opts.NoQuality != tt.wantNoQuality {
				t.Errorf("NoQuality = %v, want %v", opts.NoQuality, tt.wantNoQuality)
			}
		})
	}
}

func TestSimplifyOptionsDecoding(t *testing.T) {
	tests := []struct {
		name             string
		options          map[string]any
		wantNoCheckpoint bool
		wantAgent        string
	}{
		{
			name:    "empty options",
			options: map[string]any{},
		},
		{
			name:             "no checkpoint flag",
			options:          map[string]any{"no_checkpoint": true},
			wantNoCheckpoint: true,
		},
		{
			name:      "agent option",
			options:   map[string]any{"agent": "claude-sonnet"},
			wantAgent: "claude-sonnet",
		},
		{
			name:             "both options",
			options:          map[string]any{"no_checkpoint": true, "agent": "opus"},
			wantNoCheckpoint: true,
			wantAgent:        "opus",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv := Invocation{Options: tt.options}
			opts, err := DecodeOptions[simplifyOptions](inv)
			if err != nil {
				t.Fatalf("DecodeOptions failed: %v", err)
			}
			if opts.NoCheckpoint != tt.wantNoCheckpoint {
				t.Errorf("NoCheckpoint = %v, want %v", opts.NoCheckpoint, tt.wantNoCheckpoint)
			}
			if opts.Agent != tt.wantAgent {
				t.Errorf("Agent = %q, want %q", opts.Agent, tt.wantAgent)
			}
		})
	}
}

func TestQuickOptionsDecoding(t *testing.T) {
	tests := []struct {
		name         string
		options      map[string]any
		wantQueueID  string
		wantTitle    string
		wantPriority int
		wantLabels   []string
	}{
		{
			name:    "empty options",
			options: map[string]any{},
		},
		{
			name:        "queue id",
			options:     map[string]any{"queue_id": "q1"},
			wantQueueID: "q1",
		},
		{
			name:      "title",
			options:   map[string]any{"title": "My Task"},
			wantTitle: "My Task",
		},
		{
			name:         "priority",
			options:      map[string]any{"priority": 3},
			wantPriority: 3,
		},
		{
			name:       "labels",
			options:    map[string]any{"labels": []string{"bug", "urgent"}},
			wantLabels: []string{"bug", "urgent"},
		},
		{
			name: "all options",
			options: map[string]any{
				"queue_id": "backlog",
				"title":    "Fix bug",
				"priority": 1,
				"labels":   []string{"p0"},
			},
			wantQueueID:  "backlog",
			wantTitle:    "Fix bug",
			wantPriority: 1,
			wantLabels:   []string{"p0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv := Invocation{Options: tt.options}
			opts, err := DecodeOptions[quickOptions](inv)
			if err != nil {
				t.Fatalf("DecodeOptions failed: %v", err)
			}
			if opts.QueueID != tt.wantQueueID {
				t.Errorf("QueueID = %q, want %q", opts.QueueID, tt.wantQueueID)
			}
			if opts.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", opts.Title, tt.wantTitle)
			}
			if opts.Priority != tt.wantPriority {
				t.Errorf("Priority = %d, want %d", opts.Priority, tt.wantPriority)
			}
			if len(opts.Labels) != len(tt.wantLabels) {
				t.Errorf("Labels = %v, want %v", opts.Labels, tt.wantLabels)
			}
			for i, label := range tt.wantLabels {
				if i < len(opts.Labels) && opts.Labels[i] != label {
					t.Errorf("Labels[%d] = %q, want %q", i, opts.Labels[i], label)
				}
			}
		})
	}
}

func TestHandleSyncWithTaskID(t *testing.T) {
	cond := mustNewConductor(t)

	tests := []struct {
		name   string
		inv    Invocation
		errSub string
	}{
		{
			name:   "task id from options",
			inv:    Invocation{Options: map[string]any{"task_id": "task-123"}},
			errSub: "sync failed",
		},
		{
			name:   "task id from args",
			inv:    Invocation{Args: []string{"task-456"}},
			errSub: "sync failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := handleSync(context.Background(), cond, tt.inv)
			if result != nil {
				t.Fatalf("expected nil result, got %#v", result)
			}
			// Should fail with sync error (not ErrNoActiveTask)
			if err == nil || !strings.Contains(err.Error(), tt.errSub) {
				t.Fatalf("expected error containing %q, got %v", tt.errSub, err)
			}
		})
	}
}

func TestHandleAutoRefExtraction(t *testing.T) {
	// Test that ref is extracted from options or args
	tests := []struct {
		name    string
		inv     Invocation
		wantRef string
	}{
		{
			name:    "ref from options",
			inv:     Invocation{Options: map[string]any{"ref": "github:123"}},
			wantRef: "github:123",
		},
		{
			name:    "ref from args",
			inv:     Invocation{Args: []string{"file:task.md"}},
			wantRef: "file:task.md",
		},
		{
			name:    "options ref takes precedence",
			inv:     Invocation{Options: map[string]any{"ref": "github:1"}, Args: []string{"file:x"}},
			wantRef: "github:1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := DecodeOptions[autoOptions](tt.inv)
			if err != nil {
				t.Fatalf("DecodeOptions failed: %v", err)
			}

			ref := strings.TrimSpace(opts.Ref)
			if ref == "" && len(tt.inv.Args) > 0 {
				ref = strings.TrimSpace(tt.inv.Args[0])
			}

			if ref != tt.wantRef {
				t.Errorf("ref = %q, want %q", ref, tt.wantRef)
			}
		})
	}
}
