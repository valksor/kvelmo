package conductor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valksor/go-mehrhof/internal/agent"
	"github.com/valksor/go-mehrhof/internal/storage"
)

func TestNormalizeSourceRef(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "spec.md")
	if err := os.WriteFile(filePath, []byte("spec content"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	tests := []struct {
		name   string
		input  string
		want   string
		errMsg string
	}{
		{
			name:  "file path",
			input: filePath,
			want:  "file:" + filePath,
		},
		{
			name:  "dir path",
			input: tmpDir,
			want:  "research:" + tmpDir,
		},
		{
			name:  "already prefixed",
			input: "file:" + filePath,
			want:  "file:" + filePath,
		},
		{
			name:  "provider ref",
			input: "github:123",
			want:  "github:123",
		},
		{
			name:  "unknown path passthrough",
			input: "custom:ref",
			want:  "custom:ref",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeSourceRef(tt.input)
			if tt.errMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errMsg) {
					t.Fatalf("normalizeSourceRef() error = %v, want %q", err, tt.errMsg)
				}

				return
			}
			if err != nil {
				t.Fatalf("normalizeSourceRef() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("normalizeSourceRef() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseDraftTask(t *testing.T) {
	input := `---
title: Draft Title
priority: 1
labels: bug, urgent
description: |
  Line one
  Line two
---
`

	got, err := parseDraftTask(input)
	if err != nil {
		t.Fatalf("parseDraftTask() error = %v", err)
	}
	if got.Title != "Draft Title" {
		t.Fatalf("Title = %q, want %q", got.Title, "Draft Title")
	}
	if got.Priority != 1 {
		t.Fatalf("Priority = %d, want %d", got.Priority, 1)
	}
	if got.Description != "Line one\nLine two" {
		t.Fatalf("Description = %q, want %q", got.Description, "Line one\nLine two")
	}
	if len(got.Labels) != 2 || got.Labels[0] != "bug" || got.Labels[1] != "urgent" {
		t.Fatalf("Labels = %v, want [bug urgent]", got.Labels)
	}
}

func TestCreateQueueTaskFromSource(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "spec.md")
	if err := os.WriteFile(filePath, []byte("spec content"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	c, err := New(WithWorkDir(tmpDir), WithCreateBranch(false), WithAgent("mock"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	mock := &sourceTaskAgent{
		response: `---
title: Example Task
priority: 2
labels: api, backend
description: |
  Do the thing.
  Keep it concise.
---`,
	}
	if err := c.GetAgentRegistry().Register(mock); err != nil {
		t.Fatalf("Register mock agent: %v", err)
	}
	if err := c.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	result, err := c.CreateQueueTaskFromSource(context.Background(), filePath, SourceTaskOptions{
		Notes:    []string{"Focus on API impacts", "Use existing patterns"},
		Provider: "github",
		Labels:   []string{"triage"},
	})
	if err != nil {
		t.Fatalf("CreateQueueTaskFromSource() error = %v", err)
	}

	if result.QueueID == "" || result.TaskID == "" {
		t.Fatalf("expected queue/task IDs, got %q/%q", result.QueueID, result.TaskID)
	}

	queue, err := storage.LoadTaskQueue(c.GetWorkspace(), result.QueueID)
	if err != nil {
		t.Fatalf("LoadTaskQueue: %v", err)
	}
	task := queue.GetTask(result.TaskID)
	if task == nil {
		t.Fatalf("task not found: %s", result.TaskID)
	}
	if task.Title != "Example Task" {
		t.Fatalf("task title = %q, want %q", task.Title, "Example Task")
	}
	if task.Priority != 2 {
		t.Fatalf("task priority = %d, want %d", task.Priority, 2)
	}
	if task.Description == "" || !strings.Contains(task.Description, "Do the thing.") {
		t.Fatalf("task description not set: %q", task.Description)
	}
	if !containsAll(task.Labels, []string{"api", "backend", "triage"}) {
		t.Fatalf("task labels = %v, want to include api/backend/triage", task.Labels)
	}

	notes, err := c.GetWorkspace().LoadQueueNotes(result.QueueID, result.TaskID)
	if err != nil {
		t.Fatalf("LoadQueueNotes: %v", err)
	}
	noteText := storage.QueueNotesPlainText(notes)
	for _, expected := range []string{"Focus on API impacts", "Use existing patterns", "Target provider: github"} {
		if !strings.Contains(noteText, expected) {
			t.Fatalf("notes missing %q: %s", expected, noteText)
		}
	}
}

func containsAll(haystack []string, needles []string) bool {
	set := make(map[string]bool, len(haystack))
	for _, item := range haystack {
		set[item] = true
	}
	for _, needle := range needles {
		if !set[needle] {
			return false
		}
	}

	return true
}

type sourceTaskAgent struct {
	response string
}

func (a *sourceTaskAgent) Name() string {
	return "mock"
}

func (a *sourceTaskAgent) Run(ctx context.Context, prompt string) (*agent.Response, error) {
	return &agent.Response{Summary: a.response}, nil
}

func (a *sourceTaskAgent) RunStream(ctx context.Context, prompt string) (<-chan agent.Event, <-chan error) {
	return nil, nil
}

func (a *sourceTaskAgent) RunWithCallback(ctx context.Context, prompt string, cb agent.StreamCallback) (*agent.Response, error) {
	return a.Run(ctx, prompt)
}

func (a *sourceTaskAgent) Available() error {
	return nil
}

func (a *sourceTaskAgent) WithEnv(key, value string) agent.Agent {
	return a
}

func (a *sourceTaskAgent) WithArgs(args ...string) agent.Agent {
	return a
}
