package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewTaskQueue(t *testing.T) {
	queue := NewTaskQueue("test-queue", "Test Queue", "dir:/tmp/test")

	if queue.ID != "test-queue" {
		t.Errorf("ID = %q, want %q", queue.ID, "test-queue")
	}
	if queue.Title != "Test Queue" {
		t.Errorf("Title = %q, want %q", queue.Title, "Test Queue")
	}
	if queue.Source != "dir:/tmp/test" {
		t.Errorf("Source = %q, want %q", queue.Source, "dir:/tmp/test")
	}
	if queue.Status != QueueStatusDraft {
		t.Errorf("Status = %q, want %q", queue.Status, QueueStatusDraft)
	}
	if queue.Version != TaskQueueVersion {
		t.Errorf("Version = %q, want %q", queue.Version, TaskQueueVersion)
	}
	if len(queue.Tasks) != 0 {
		t.Errorf("Tasks length = %d, want 0", len(queue.Tasks))
	}
}

func TestTaskQueue_AddTask(t *testing.T) {
	queue := NewTaskQueue("test-queue", "Test Queue", "")

	task := &QueuedTask{
		ID:          "task-1",
		Title:       "First task",
		Description: "Task description",
		Status:      TaskStatusReady,
		Priority:    1,
	}

	queue.AddTask(task)

	if len(queue.Tasks) != 1 {
		t.Fatalf("Tasks length = %d, want 1", len(queue.Tasks))
	}
	if queue.Tasks[0].ID != "task-1" {
		t.Errorf("Task ID = %q, want %q", queue.Tasks[0].ID, "task-1")
	}
}

func TestTaskQueue_GetTask(t *testing.T) {
	queue := NewTaskQueue("test-queue", "Test Queue", "")
	queue.AddTask(&QueuedTask{ID: "task-1", Title: "First"})
	queue.AddTask(&QueuedTask{ID: "task-2", Title: "Second"})

	task := queue.GetTask("task-1")
	if task == nil {
		t.Fatal("GetTask returned nil")
	}
	if task.Title != "First" {
		t.Errorf("Title = %q, want %q", task.Title, "First")
	}

	task = queue.GetTask("nonexistent")
	if task != nil {
		t.Errorf("GetTask returned non-nil for nonexistent task")
	}
}

func TestTaskQueue_UpdateTask(t *testing.T) {
	queue := NewTaskQueue("test-queue", "Test Queue", "")
	queue.AddTask(&QueuedTask{ID: "task-1", Title: "Original", Priority: 1})

	err := queue.UpdateTask("task-1", func(task *QueuedTask) {
		task.Title = "Updated"
		task.Priority = 2
	})
	if err != nil {
		t.Fatalf("UpdateTask error: %v", err)
	}

	task := queue.GetTask("task-1")
	if task.Title != "Updated" {
		t.Errorf("Title = %q, want %q", task.Title, "Updated")
	}
	if task.Priority != 2 {
		t.Errorf("Priority = %d, want %d", task.Priority, 2)
	}

	err = queue.UpdateTask("nonexistent", func(task *QueuedTask) {})
	if err == nil {
		t.Error("UpdateTask should return error for nonexistent task")
	}
}

func TestTaskQueue_RemoveTask(t *testing.T) {
	queue := NewTaskQueue("test-queue", "Test Queue", "")
	queue.AddTask(&QueuedTask{ID: "task-1"})
	queue.AddTask(&QueuedTask{ID: "task-2"})

	removed := queue.RemoveTask("task-1")
	if !removed {
		t.Error("RemoveTask should return true")
	}
	if len(queue.Tasks) != 1 {
		t.Errorf("Tasks length = %d, want 1", len(queue.Tasks))
	}
	if queue.Tasks[0].ID != "task-2" {
		t.Errorf("Remaining task ID = %q, want %q", queue.Tasks[0].ID, "task-2")
	}

	removed = queue.RemoveTask("nonexistent")
	if removed {
		t.Error("RemoveTask should return false for nonexistent task")
	}
}

func TestTaskQueue_ReorderTask(t *testing.T) {
	queue := NewTaskQueue("test-queue", "Test Queue", "")
	queue.AddTask(&QueuedTask{ID: "task-1"})
	queue.AddTask(&QueuedTask{ID: "task-2"})
	queue.AddTask(&QueuedTask{ID: "task-3"})

	// Move task-3 to the beginning
	err := queue.ReorderTask("task-3", 0)
	if err != nil {
		t.Fatalf("ReorderTask error: %v", err)
	}

	if queue.Tasks[0].ID != "task-3" {
		t.Errorf("First task = %q, want %q", queue.Tasks[0].ID, "task-3")
	}
	if queue.Tasks[1].ID != "task-1" {
		t.Errorf("Second task = %q, want %q", queue.Tasks[1].ID, "task-1")
	}
	if queue.Tasks[2].ID != "task-2" {
		t.Errorf("Third task = %q, want %q", queue.Tasks[2].ID, "task-2")
	}

	// Error cases
	err = queue.ReorderTask("nonexistent", 0)
	if err == nil {
		t.Error("ReorderTask should return error for nonexistent task")
	}

	err = queue.ReorderTask("task-1", -1)
	if err == nil {
		t.Error("ReorderTask should return error for invalid index")
	}

	err = queue.ReorderTask("task-1", 100)
	if err == nil {
		t.Error("ReorderTask should return error for out-of-bounds index")
	}
}

func TestTaskQueue_ComputeBlocksRelations(t *testing.T) {
	queue := NewTaskQueue("test-queue", "Test Queue", "")
	queue.AddTask(&QueuedTask{ID: "task-1"})
	queue.AddTask(&QueuedTask{ID: "task-2", DependsOn: []string{"task-1"}})
	queue.AddTask(&QueuedTask{ID: "task-3", DependsOn: []string{"task-1", "task-2"}})

	queue.ComputeBlocksRelations()

	// task-1 should block task-2 and task-3
	task1 := queue.GetTask("task-1")
	if len(task1.Blocks) != 2 {
		t.Errorf("task-1 blocks count = %d, want 2", len(task1.Blocks))
	}

	// task-2 should block task-3
	task2 := queue.GetTask("task-2")
	if len(task2.Blocks) != 1 || task2.Blocks[0] != "task-3" {
		t.Errorf("task-2 blocks = %v, want [task-3]", task2.Blocks)
	}

	// task-3 should block nothing
	task3 := queue.GetTask("task-3")
	if len(task3.Blocks) != 0 {
		t.Errorf("task-3 blocks count = %d, want 0", len(task3.Blocks))
	}
}

func TestTaskQueue_ComputeTaskStatuses(t *testing.T) {
	queue := NewTaskQueue("test-queue", "Test Queue", "")
	queue.AddTask(&QueuedTask{ID: "task-1", Status: TaskStatusReady})
	queue.AddTask(&QueuedTask{ID: "task-2", Status: TaskStatusReady, DependsOn: []string{"task-1"}})
	queue.AddTask(&QueuedTask{ID: "task-3", Status: TaskStatusReady, DependsOn: []string{"task-2"}})

	queue.ComputeTaskStatuses()

	// task-1 has no dependencies, should stay ready
	if queue.GetTask("task-1").Status != TaskStatusReady {
		t.Errorf("task-1 status = %q, want %q", queue.GetTask("task-1").Status, TaskStatusReady)
	}

	// task-2 depends on task-1 (not submitted), should be blocked
	if queue.GetTask("task-2").Status != TaskStatusBlocked {
		t.Errorf("task-2 status = %q, want %q", queue.GetTask("task-2").Status, TaskStatusBlocked)
	}

	// Mark task-1 as submitted
	_ = queue.UpdateTask("task-1", func(task *QueuedTask) {
		task.Status = TaskStatusSubmitted
	})
	queue.ComputeTaskStatuses()

	// Now task-2 should be ready
	if queue.GetTask("task-2").Status != TaskStatusReady {
		t.Errorf("task-2 status = %q, want %q", queue.GetTask("task-2").Status, TaskStatusReady)
	}
}

func TestTaskQueue_GetReadyTasks(t *testing.T) {
	queue := NewTaskQueue("test-queue", "Test Queue", "")
	queue.AddTask(&QueuedTask{ID: "task-1", Status: TaskStatusReady})
	queue.AddTask(&QueuedTask{ID: "task-2", Status: TaskStatusBlocked})
	queue.AddTask(&QueuedTask{ID: "task-3", Status: TaskStatusReady})
	queue.AddTask(&QueuedTask{ID: "task-4", Status: TaskStatusSubmitted})

	ready := queue.GetReadyTasks()
	if len(ready) != 2 {
		t.Errorf("Ready tasks count = %d, want 2", len(ready))
	}
}

func TestTaskQueue_NextTaskID(t *testing.T) {
	queue := NewTaskQueue("test-queue", "Test Queue", "")

	// First task should be task-1
	id := queue.NextTaskID()
	if id != "task-1" {
		t.Errorf("NextTaskID = %q, want %q", id, "task-1")
	}

	queue.AddTask(&QueuedTask{ID: "task-1"})
	queue.AddTask(&QueuedTask{ID: "task-5"})

	// Next should be task-6 (max+1)
	id = queue.NextTaskID()
	if id != "task-6" {
		t.Errorf("NextTaskID = %q, want %q", id, "task-6")
	}
}

func TestTaskQueue_SaveAndLoad(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Create a workspace
	ws := &Workspace{workspaceRoot: tmpDir}

	// Create and save queue
	queue := NewTaskQueue("test-queue", "Test Queue", "dir:/tmp/source")
	queue.path = ws.QueuePath("test-queue")
	queue.AddTask(&QueuedTask{
		ID:          "task-1",
		Title:       "First task",
		Description: "Description",
		Status:      TaskStatusReady,
		Priority:    1,
		Labels:      []string{"backend", "urgent"},
		DependsOn:   []string{},
	})
	queue.AddTask(&QueuedTask{
		ID:        "task-2",
		Title:     "Second task",
		Status:    TaskStatusBlocked,
		Priority:  2,
		DependsOn: []string{"task-1"},
	})
	queue.Questions = []string{"What API to use?"}
	queue.Blockers = []string{"Need credentials"}

	if err := queue.Save(); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(queue.path); os.IsNotExist(err) {
		t.Fatal("Queue file was not created")
	}

	// Load the queue
	loaded, err := LoadTaskQueue(ws, "test-queue")
	if err != nil {
		t.Fatalf("LoadTaskQueue error: %v", err)
	}

	if loaded.ID != queue.ID {
		t.Errorf("ID = %q, want %q", loaded.ID, queue.ID)
	}
	if loaded.Title != queue.Title {
		t.Errorf("Title = %q, want %q", loaded.Title, queue.Title)
	}
	if loaded.Source != queue.Source {
		t.Errorf("Source = %q, want %q", loaded.Source, queue.Source)
	}
	if len(loaded.Tasks) != 2 {
		t.Errorf("Tasks count = %d, want 2", len(loaded.Tasks))
	}
	if loaded.Tasks[0].Title != "First task" {
		t.Errorf("First task title = %q, want %q", loaded.Tasks[0].Title, "First task")
	}
	if len(loaded.Tasks[0].Labels) != 2 {
		t.Errorf("First task labels count = %d, want 2", len(loaded.Tasks[0].Labels))
	}
	if len(loaded.Questions) != 1 {
		t.Errorf("Questions count = %d, want 1", len(loaded.Questions))
	}
	if len(loaded.Blockers) != 1 {
		t.Errorf("Blockers count = %d, want 1", len(loaded.Blockers))
	}
}

func TestWorkspace_ListQueues(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	ws := &Workspace{workspaceRoot: tmpDir}

	// Empty workspace should return empty list
	queues, err := ws.ListQueues()
	if err != nil {
		t.Fatalf("ListQueues error: %v", err)
	}
	if len(queues) != 0 {
		t.Errorf("Queues count = %d, want 0", len(queues))
	}

	// Create some queues
	for _, id := range []string{"queue-b", "queue-a", "queue-c"} {
		queue := NewTaskQueue(id, "Test", "")
		queue.path = ws.QueuePath(id)
		if err := queue.Save(); err != nil {
			t.Fatalf("Save queue %s error: %v", id, err)
		}
	}

	queues, err = ws.ListQueues()
	if err != nil {
		t.Fatalf("ListQueues error: %v", err)
	}
	if len(queues) != 3 {
		t.Errorf("Queues count = %d, want 3", len(queues))
	}

	// Should be sorted
	if queues[0] != "queue-a" || queues[1] != "queue-b" || queues[2] != "queue-c" {
		t.Errorf("Queues not sorted: %v", queues)
	}
}

func TestWorkspace_DeleteQueue(t *testing.T) {
	tmpDir := t.TempDir()

	ws := &Workspace{workspaceRoot: tmpDir}

	// Create a queue
	queue := NewTaskQueue("test-queue", "Test", "")
	queue.path = ws.QueuePath("test-queue")
	if err := queue.Save(); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	if !ws.QueueExists("test-queue") {
		t.Error("Queue should exist after save")
	}

	// Delete it
	if err := ws.DeleteQueue("test-queue"); err != nil {
		t.Fatalf("DeleteQueue error: %v", err)
	}

	if ws.QueueExists("test-queue") {
		t.Error("Queue should not exist after delete")
	}
}

func TestTaskQueue_ConcurrentAccess(t *testing.T) {
	queue := NewTaskQueue("test-queue", "Test Queue", "")

	// Add initial tasks
	for i := 1; i <= 10; i++ {
		queue.AddTask(&QueuedTask{ID: queue.NextTaskID(), Title: "Task"})
	}

	done := make(chan bool)

	// Concurrent reads
	for range 5 {
		go func() {
			for range 100 {
				_ = queue.GetTask("task-1")
				_ = queue.GetReadyTasks()
				_ = queue.TaskCount()
			}
			done <- true
		}()
	}

	// Concurrent writes
	for range 5 {
		go func() {
			for j := range 100 {
				queue.AddTask(&QueuedTask{ID: queue.NextTaskID(), Title: "New"})
				_ = queue.UpdateTask("task-1", func(task *QueuedTask) {
					task.Priority = j
				})
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for range 10 {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for goroutines")
		}
	}
}

func TestLoadTaskQueue_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	ws := &Workspace{workspaceRoot: tmpDir}

	_, err := LoadTaskQueue(ws, "nonexistent")
	if err == nil {
		t.Error("LoadTaskQueue should return error for nonexistent queue")
	}
}

func TestWorkspace_QueuePath(t *testing.T) {
	ws := &Workspace{workspaceRoot: "/home/user/.valksor/mehrhof/workspaces/project123"}

	path := ws.QueuePath("my-queue")
	expected := filepath.Join("/home/user/.valksor/mehrhof/workspaces/project123", QueuesDir, "my-queue", "queue.yaml")

	if path != expected {
		t.Errorf("QueuePath = %q, want %q", path, expected)
	}
}

func TestTaskQueue_ComputeSubtaskRelations(t *testing.T) {
	queue := NewTaskQueue("test-queue", "Test Queue", "")
	queue.AddTask(&QueuedTask{ID: "task-1"})
	queue.AddTask(&QueuedTask{ID: "task-2", ParentID: "task-1"})
	queue.AddTask(&QueuedTask{ID: "task-3", ParentID: "task-1"})
	queue.AddTask(&QueuedTask{ID: "task-4", ParentID: "task-2"}) // Nested subtask

	queue.ComputeSubtaskRelations()

	// task-1 should have task-2 and task-3 as subtasks
	task1 := queue.GetTask("task-1")
	if len(task1.Subtasks) != 2 {
		t.Errorf("task-1 subtasks count = %d, want 2", len(task1.Subtasks))
	}

	// task-2 should have task-4 as subtask
	task2 := queue.GetTask("task-2")
	if len(task2.Subtasks) != 1 || task2.Subtasks[0] != "task-4" {
		t.Errorf("task-2 subtasks = %v, want [task-4]", task2.Subtasks)
	}

	// task-3 and task-4 should have no subtasks
	task3 := queue.GetTask("task-3")
	if len(task3.Subtasks) != 0 {
		t.Errorf("task-3 subtasks count = %d, want 0", len(task3.Subtasks))
	}

	task4 := queue.GetTask("task-4")
	if len(task4.Subtasks) != 0 {
		t.Errorf("task-4 subtasks count = %d, want 0", len(task4.Subtasks))
	}
}

func TestWorkspace_FindQueueTaskByExternalID(t *testing.T) {
	tests := []struct {
		name       string
		externalID string
		setup      func(ws *Workspace)
		wantTitle  string
		wantNil    bool
		wantErr    bool
	}{
		{
			name:       "empty external ID returns nil",
			externalID: "",
			setup:      func(ws *Workspace) {},
			wantNil:    true,
		},
		{
			name:       "no queues returns nil",
			externalID: "wrike-123",
			setup:      func(ws *Workspace) {},
			wantNil:    true,
		},
		{
			name:       "found in first queue",
			externalID: "wrike-456",
			setup: func(ws *Workspace) {
				q := NewTaskQueue("queue-1", "Queue 1", "")
				q.path = ws.QueuePath("queue-1")
				q.AddTask(&QueuedTask{ID: "task-1", Title: "Match", ExternalID: "wrike-456"})
				_ = q.Save()
			},
			wantTitle: "Match",
		},
		{
			name:       "found in second queue",
			externalID: "gh-789",
			setup: func(ws *Workspace) {
				q1 := NewTaskQueue("queue-1", "Queue 1", "")
				q1.path = ws.QueuePath("queue-1")
				q1.AddTask(&QueuedTask{ID: "task-1", Title: "No match", ExternalID: "other-id"})
				_ = q1.Save()

				q2 := NewTaskQueue("queue-2", "Queue 2", "")
				q2.path = ws.QueuePath("queue-2")
				q2.AddTask(&QueuedTask{ID: "task-2", Title: "Found it", ExternalID: "gh-789"})
				_ = q2.Save()
			},
			wantTitle: "Found it",
		},
		{
			name:       "not found in any queue",
			externalID: "nonexistent-999",
			setup: func(ws *Workspace) {
				q := NewTaskQueue("queue-1", "Queue 1", "")
				q.path = ws.QueuePath("queue-1")
				q.AddTask(&QueuedTask{ID: "task-1", Title: "Other task", ExternalID: "other-id"})
				_ = q.Save()
			},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := &Workspace{workspaceRoot: t.TempDir()}
			tt.setup(ws)

			task, err := ws.FindQueueTaskByExternalID(tt.externalID)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}

				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantNil {
				if task != nil {
					t.Errorf("expected nil, got task %q", task.Title)
				}

				return
			}

			if task == nil {
				t.Fatal("expected task, got nil")
			}
			if task.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", task.Title, tt.wantTitle)
			}
		})
	}
}

func TestQueuedTask_MetadataPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	ws := &Workspace{workspaceRoot: tmpDir}

	// Create queue with metadata-rich tasks
	queue := NewTaskQueue("meta-test", "Metadata Test", "")
	queue.path = ws.QueuePath("meta-test")
	queue.AddTask(&QueuedTask{
		ID:         "task-1",
		Title:      "Task with metadata",
		Status:     TaskStatusReady,
		SourcePath: "/projects/tasks/feature-auth.md",
		Metadata: map[string]any{
			"code_example":   "func Login() error { ... }",
			"reference_file": "internal/auth/handler.go",
			"priority_score": 42,
		},
	})
	queue.AddTask(&QueuedTask{
		ID:     "task-2",
		Title:  "Task without metadata",
		Status: TaskStatusReady,
	})

	if err := queue.Save(); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	// Load and verify
	loaded, err := LoadTaskQueue(ws, "meta-test")
	if err != nil {
		t.Fatalf("LoadTaskQueue error: %v", err)
	}

	task1 := loaded.GetTask("task-1")
	if task1.SourcePath != "/projects/tasks/feature-auth.md" {
		t.Errorf("SourcePath = %q, want %q", task1.SourcePath, "/projects/tasks/feature-auth.md")
	}
	if task1.Metadata == nil {
		t.Fatal("Metadata is nil after load")
	}
	if task1.Metadata["code_example"] != "func Login() error { ... }" {
		t.Errorf("Metadata[code_example] = %v", task1.Metadata["code_example"])
	}
	if task1.Metadata["reference_file"] != "internal/auth/handler.go" {
		t.Errorf("Metadata[reference_file] = %v", task1.Metadata["reference_file"])
	}
	// YAML deserializes numbers as int
	if task1.Metadata["priority_score"] != 42 {
		t.Errorf("Metadata[priority_score] = %v (type %T)", task1.Metadata["priority_score"], task1.Metadata["priority_score"])
	}

	// task-2 should have nil metadata
	task2 := loaded.GetTask("task-2")
	if task2.SourcePath != "" {
		t.Errorf("SourcePath = %q, want empty", task2.SourcePath)
	}
	if task2.Metadata != nil {
		t.Errorf("Metadata = %v, want nil", task2.Metadata)
	}
}

func TestTaskQueue_ParentIDPersistence(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	ws := &Workspace{workspaceRoot: tmpDir}

	// Create and save queue with parent relationships
	queue := NewTaskQueue("test-queue", "Test Queue", "")
	queue.path = ws.QueuePath("test-queue")
	queue.AddTask(&QueuedTask{ID: "task-1", Title: "Parent task"})
	queue.AddTask(&QueuedTask{ID: "task-2", Title: "Subtask", ParentID: "task-1"})

	if err := queue.Save(); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	// Load the queue and verify ParentID is preserved
	loaded, err := LoadTaskQueue(ws, "test-queue")
	if err != nil {
		t.Fatalf("LoadTaskQueue error: %v", err)
	}

	task2 := loaded.GetTask("task-2")
	if task2.ParentID != "task-1" {
		t.Errorf("ParentID = %q, want %q", task2.ParentID, "task-1")
	}
}
