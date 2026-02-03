package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-toolkit/paths"
)

func TestWorkspaceGetSpecifications(t *testing.T) {
	tests := []struct {
		name        string
		setupTask   bool
		setupActive bool
		setupCount  int
		args        map[string]interface{}
		wantError   bool
		wantErrText string
		checkResult func(t *testing.T, text string)
	}{
		{
			name:       "explicit task_id with specifications",
			setupTask:  true,
			setupCount: 2,
			args:       map[string]interface{}{"task_id": "test-task"},
			checkResult: func(t *testing.T, text string) {
				t.Helper()
				var result map[string]interface{}
				if err := json.Unmarshal([]byte(text), &result); err != nil {
					t.Fatalf("Failed to parse JSON: %v", err)
				}
				items, ok := result["specifications"].([]interface{})
				if !ok {
					t.Fatal("Expected specifications array in result")
				}
				if len(items) != 2 {
					t.Errorf("Expected 2 specifications, got %d", len(items))
				}
			},
		},
		{
			name:        "active task fallback",
			setupTask:   true,
			setupActive: true,
			setupCount:  1,
			args:        map[string]interface{}{},
			checkResult: func(t *testing.T, text string) {
				t.Helper()
				var result map[string]interface{}
				if err := json.Unmarshal([]byte(text), &result); err != nil {
					t.Fatalf("Failed to parse JSON: %v", err)
				}
				if result["task_id"] != "test-task" {
					t.Errorf("Expected task_id=test-task, got %v", result["task_id"])
				}
			},
		},
		{
			name:       "summary_only omits content",
			setupTask:  true,
			setupCount: 1,
			args:       map[string]interface{}{"task_id": "test-task", "summary_only": true},
			checkResult: func(t *testing.T, text string) {
				t.Helper()
				// summary_only=true should not include "content" key per specification
				if strings.Contains(text, `"content"`) {
					t.Error("summary_only=true should not include content field")
				}
			},
		},
		{
			name:        "no task and no active task",
			setupTask:   false,
			args:        map[string]interface{}{},
			wantError:   true,
			wantErrText: "no task specified",
		},
		{
			name:       "task with no specifications",
			setupTask:  true,
			setupCount: 0,
			args:       map[string]interface{}{"task_id": "test-task"},
			checkResult: func(t *testing.T, text string) {
				t.Helper()
				var result map[string]interface{}
				if err := json.Unmarshal([]byte(text), &result); err != nil {
					t.Fatalf("Failed to parse JSON: %v", err)
				}
				items, ok := result["specifications"].([]interface{})
				if !ok {
					t.Fatal("Expected specifications array")
				}
				if len(items) != 0 {
					t.Errorf("Expected 0 specifications, got %d", len(items))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			homeDir := t.TempDir()
			ctx := context.Background()

			// Set global home dir override so MCP tools find the same workspace
			t.Cleanup(paths.SetHomeDirForTesting(homeDir))

			wsCfg := storage.NewDefaultWorkspaceConfig()
			wsCfg.Storage.HomeDir = homeDir
			ws, err := storage.OpenWorkspace(ctx, tmpDir, wsCfg)
			if err != nil {
				t.Fatalf("Failed to open workspace: %v", err)
			}

			if tt.setupTask {
				if _, err := ws.CreateWork("test-task", storage.SourceInfo{Type: "file", Ref: "test.md"}); err != nil {
					t.Fatalf("CreateWork failed: %v", err)
				}
				for i := range tt.setupCount {
					spec := &storage.Specification{
						Number:  i + 1,
						Title:   "Specification " + string(rune('A'+i)),
						Status:  storage.SpecificationStatusReady,
						Content: "# Content for specification " + string(rune('A'+i)),
					}
					if err := ws.SaveSpecificationWithMeta("test-task", spec); err != nil {
						t.Fatalf("SaveSpecificationWithMeta failed: %v", err)
					}
				}
			}

			if tt.setupActive {
				if err := ws.SaveActiveTask(&storage.ActiveTask{
					ID:    "test-task",
					Ref:   "file:test.md",
					State: "implementing",
				}); err != nil {
					t.Fatalf("SaveActiveTask failed: %v", err)
				}
			}

			registry := NewToolRegistry(nil)
			RegisterWorkspaceTools(registry)
			t.Chdir(tmpDir)

			result, err := registry.CallTool(ctx, "workspace_get_specs", tt.args)
			if err != nil {
				t.Fatalf("CallTool failed: %v", err)
			}

			if tt.wantError {
				if !result.IsError {
					t.Fatalf("Expected error, got success: %s", result.Content[0].Text)
				}
				if !strings.Contains(result.Content[0].Text, tt.wantErrText) {
					t.Errorf("Expected error containing %q, got %q", tt.wantErrText, result.Content[0].Text)
				}

				return
			}

			if result.IsError {
				t.Fatalf("Tool returned error: %s", result.Content[0].Text)
			}

			if tt.checkResult != nil {
				tt.checkResult(t, result.Content[0].Text)
			}
		})
	}
}

func TestWorkspaceGetSessions(t *testing.T) {
	tests := []struct {
		name        string
		setupTask   bool
		setupActive bool
		sessions    []*storage.Session
		args        map[string]interface{}
		wantError   bool
		wantErrText string
		checkResult func(t *testing.T, text string)
	}{
		{
			name:      "explicit task_id with sessions",
			setupTask: true,
			sessions: []*storage.Session{
				{
					Version:  "1",
					Kind:     "planning",
					Metadata: storage.SessionMetadata{StartedAt: time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC), Agent: "claude"},
				},
				{
					Version:  "1",
					Kind:     "implementing",
					Metadata: storage.SessionMetadata{StartedAt: time.Date(2025, 1, 15, 11, 0, 0, 0, time.UTC), Agent: "claude"},
				},
			},
			args: map[string]interface{}{"task_id": "test-task"},
			checkResult: func(t *testing.T, text string) {
				t.Helper()
				var result map[string]interface{}
				if err := json.Unmarshal([]byte(text), &result); err != nil {
					t.Fatalf("Failed to parse JSON: %v", err)
				}
				items, ok := result["sessions"].([]interface{})
				if !ok {
					t.Fatal("Expected sessions array")
				}
				if len(items) != 2 {
					t.Errorf("Expected 2 sessions, got %d", len(items))
				}
			},
		},
		{
			name:        "active task fallback",
			setupTask:   true,
			setupActive: true,
			sessions: []*storage.Session{
				{
					Version:  "1",
					Kind:     "planning",
					Metadata: storage.SessionMetadata{StartedAt: time.Now(), Agent: "claude"},
				},
			},
			args: map[string]interface{}{},
			checkResult: func(t *testing.T, text string) {
				t.Helper()
				var result map[string]interface{}
				if err := json.Unmarshal([]byte(text), &result); err != nil {
					t.Fatalf("Failed to parse JSON: %v", err)
				}
				if result["task_id"] != "test-task" {
					t.Errorf("Expected task_id=test-task, got %v", result["task_id"])
				}
			},
		},
		{
			name:        "no task and no active task",
			setupTask:   false,
			args:        map[string]interface{}{},
			wantError:   true,
			wantErrText: "no task specified",
		},
		{
			name:      "session with usage data",
			setupTask: true,
			sessions: []*storage.Session{
				{
					Version:  "1",
					Kind:     "planning",
					Metadata: storage.SessionMetadata{StartedAt: time.Now(), Agent: "claude"},
					Usage:    &storage.UsageInfo{InputTokens: 1000, OutputTokens: 500},
				},
			},
			args: map[string]interface{}{"task_id": "test-task"},
			checkResult: func(t *testing.T, text string) {
				t.Helper()
				if !strings.Contains(text, "input_tokens") {
					t.Error("Expected input_tokens in response with usage data")
				}
				if !strings.Contains(text, "output_tokens") {
					t.Error("Expected output_tokens in response with usage data")
				}
			},
		},
		{
			name:      "session without usage data",
			setupTask: true,
			sessions: []*storage.Session{
				{
					Version:  "1",
					Kind:     "reviewing",
					Metadata: storage.SessionMetadata{StartedAt: time.Now(), Agent: "claude"},
					Usage:    nil,
				},
			},
			args: map[string]interface{}{"task_id": "test-task"},
			checkResult: func(t *testing.T, text string) {
				t.Helper()
				if strings.Contains(text, "input_tokens") {
					t.Error("Expected no input_tokens when usage is nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			homeDir := t.TempDir()
			ctx := context.Background()

			// Set global home dir override so MCP tools find the same workspace
			t.Cleanup(paths.SetHomeDirForTesting(homeDir))

			wsCfg := storage.NewDefaultWorkspaceConfig()
			wsCfg.Storage.HomeDir = homeDir
			ws, err := storage.OpenWorkspace(ctx, tmpDir, wsCfg)
			if err != nil {
				t.Fatalf("Failed to open workspace: %v", err)
			}

			if tt.setupTask {
				if _, err := ws.CreateWork("test-task", storage.SourceInfo{Type: "file", Ref: "test.md"}); err != nil {
					t.Fatalf("CreateWork failed: %v", err)
				}
				for i, session := range tt.sessions {
					filename := fmt.Sprintf("session-%03d.yaml", i+1)
					if err := ws.SaveSession("test-task", filename, session); err != nil {
						t.Fatalf("SaveSession failed: %v", err)
					}
				}
			}

			if tt.setupActive {
				if err := ws.SaveActiveTask(&storage.ActiveTask{
					ID:    "test-task",
					Ref:   "file:test.md",
					State: "planning",
				}); err != nil {
					t.Fatalf("SaveActiveTask failed: %v", err)
				}
			}

			registry := NewToolRegistry(nil)
			RegisterWorkspaceTools(registry)
			t.Chdir(tmpDir)

			result, err := registry.CallTool(ctx, "workspace_get_sessions", tt.args)
			if err != nil {
				t.Fatalf("CallTool failed: %v", err)
			}

			if tt.wantError {
				if !result.IsError {
					t.Fatalf("Expected error, got success: %s", result.Content[0].Text)
				}
				if !strings.Contains(result.Content[0].Text, tt.wantErrText) {
					t.Errorf("Expected error containing %q, got %q", tt.wantErrText, result.Content[0].Text)
				}

				return
			}

			if result.IsError {
				t.Fatalf("Tool returned error: %s", result.Content[0].Text)
			}

			if tt.checkResult != nil {
				tt.checkResult(t, result.Content[0].Text)
			}
		})
	}
}

func TestWorkspaceGetActiveTaskWithData(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := t.TempDir()
	ctx := context.Background()

	// Set global home dir override so MCP tools find the same workspace
	t.Cleanup(paths.SetHomeDirForTesting(homeDir))

	wsCfg := storage.NewDefaultWorkspaceConfig()
	wsCfg.Storage.HomeDir = homeDir
	ws, err := storage.OpenWorkspace(ctx, tmpDir, wsCfg)
	if err != nil {
		t.Fatalf("Failed to open workspace: %v", err)
	}

	if _, err := ws.CreateWork("feature-123", storage.SourceInfo{Type: "file", Ref: "feature.md"}); err != nil {
		t.Fatalf("CreateWork failed: %v", err)
	}

	spec := &storage.Specification{Number: 1, Title: "Add auth", Status: storage.SpecificationStatusReady, Content: "# Auth"}
	if err := ws.SaveSpecificationWithMeta("feature-123", spec); err != nil {
		t.Fatalf("SaveSpecificationWithMeta failed: %v", err)
	}

	if err := ws.SaveActiveTask(&storage.ActiveTask{
		ID: "feature-123", Ref: "file:feature.md", State: "implementing",
	}); err != nil {
		t.Fatalf("SaveActiveTask failed: %v", err)
	}

	registry := NewToolRegistry(nil)
	RegisterWorkspaceTools(registry)
	t.Chdir(tmpDir)

	result, err := registry.CallTool(ctx, "workspace_get_active_task", map[string]interface{}{})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("Tool returned error: %s", result.Content[0].Text)
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &data); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}
	if data["task_id"] != "feature-123" {
		t.Errorf("task_id = %v, want feature-123", data["task_id"])
	}
	if data["state"] != "implementing" {
		t.Errorf("state = %v, want implementing", data["state"])
	}
}

func TestWorkspaceListTasksWithData(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := t.TempDir()
	ctx := context.Background()

	// Set global home dir override so MCP tools find the same workspace
	t.Cleanup(paths.SetHomeDirForTesting(homeDir))

	wsCfg := storage.NewDefaultWorkspaceConfig()
	wsCfg.Storage.HomeDir = homeDir
	ws, err := storage.OpenWorkspace(ctx, tmpDir, wsCfg)
	if err != nil {
		t.Fatalf("Failed to open workspace: %v", err)
	}

	taskIDs := []string{"task-a", "task-b", "task-c"}
	for _, id := range taskIDs {
		if _, err := ws.CreateWork(id, storage.SourceInfo{Type: "file", Ref: id + ".md"}); err != nil {
			t.Fatalf("CreateWork(%s) failed: %v", id, err)
		}
	}

	if err := ws.SaveActiveTask(&storage.ActiveTask{
		ID: "task-b", Ref: "file:task-b.md", State: "reviewing",
	}); err != nil {
		t.Fatalf("SaveActiveTask failed: %v", err)
	}

	registry := NewToolRegistry(nil)
	RegisterWorkspaceTools(registry)
	t.Chdir(tmpDir)

	result, err := registry.CallTool(ctx, "workspace_list_tasks", map[string]interface{}{})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("Tool returned error: %s", result.Content[0].Text)
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &data); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}
	tasks, ok := data["tasks"].([]interface{})
	if !ok {
		t.Fatal("Expected tasks array")
	}
	if len(tasks) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(tasks))
	}
}
