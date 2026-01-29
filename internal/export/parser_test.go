package export

import (
	"testing"

	"github.com/valksor/go-mehrhof/internal/storage"
)

func TestParseProjectPlan(t *testing.T) {
	content := `## Tasks

### task-1: Implement authentication
- **Priority**: 1
- **Status**: ready
- **Labels**: backend, security
- **Description**: Set up JWT-based auth system with token refresh

### task-2: Create user profile page
- **Priority**: 2
- **Status**: blocked
- **Depends on**: task-1
- **Labels**: frontend
- **Assignee**: developer@example.com
- **Description**: Build profile page that uses auth

### task-3: Write integration tests
- **Priority**: 3
- **Status**: pending
- **Depends on**: task-1, task-2
- **Labels**: testing

## Questions
1. Should we use OAuth or just JWT?
2. What's the expected user load?

## Blockers
- Need API credentials for external service
- Waiting on design mockups
`

	plan := ParseProjectPlan(content)

	// Test tasks
	if len(plan.Tasks) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(plan.Tasks))
	}

	// Test first task
	if plan.Tasks[0].ID != "task-1" {
		t.Errorf("expected task-1, got %s", plan.Tasks[0].ID)
	}
	if plan.Tasks[0].Title != "Implement authentication" {
		t.Errorf("expected 'Implement authentication', got %s", plan.Tasks[0].Title)
	}
	if plan.Tasks[0].Priority != 1 {
		t.Errorf("expected priority 1, got %d", plan.Tasks[0].Priority)
	}
	if plan.Tasks[0].Status != storage.TaskStatusReady {
		t.Errorf("expected ready status, got %s", plan.Tasks[0].Status)
	}
	if len(plan.Tasks[0].Labels) != 2 {
		t.Errorf("expected 2 labels, got %d", len(plan.Tasks[0].Labels))
	}
	if plan.Tasks[0].Description != "Set up JWT-based auth system with token refresh" {
		t.Errorf("expected description, got %s", plan.Tasks[0].Description)
	}

	// Test second task
	if plan.Tasks[1].ID != "task-2" {
		t.Errorf("expected task-2, got %s", plan.Tasks[1].ID)
	}
	if plan.Tasks[1].Status != storage.TaskStatusBlocked {
		t.Errorf("expected blocked status, got %s", plan.Tasks[1].Status)
	}
	if len(plan.Tasks[1].DependsOn) != 1 {
		t.Errorf("expected 1 dependency, got %d", len(plan.Tasks[1].DependsOn))
	}
	if plan.Tasks[1].DependsOn[0] != "task-1" {
		t.Errorf("expected depends on task-1, got %s", plan.Tasks[1].DependsOn[0])
	}
	if plan.Tasks[1].Assignee != "developer@example.com" {
		t.Errorf("expected assignee, got %s", plan.Tasks[1].Assignee)
	}

	// Test third task with multiple dependencies
	if len(plan.Tasks[2].DependsOn) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(plan.Tasks[2].DependsOn))
	}

	// Test questions
	if len(plan.Questions) != 2 {
		t.Errorf("expected 2 questions, got %d", len(plan.Questions))
	}
	if plan.Questions[0] != "Should we use OAuth or just JWT?" {
		t.Errorf("expected first question, got %s", plan.Questions[0])
	}

	// Test blockers
	if len(plan.Blockers) != 2 {
		t.Errorf("expected 2 blockers, got %d", len(plan.Blockers))
	}
	if plan.Blockers[0] != "Need API credentials for external service" {
		t.Errorf("expected first blocker, got %s", plan.Blockers[0])
	}
}

func TestParseProjectPlanMinimal(t *testing.T) {
	content := `## Tasks

### task-1: Simple task
- **Priority**: 1
- **Status**: ready
`

	plan := ParseProjectPlan(content)

	if len(plan.Tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(plan.Tasks))
	}
	if plan.Tasks[0].ID != "task-1" {
		t.Errorf("expected task-1, got %s", plan.Tasks[0].ID)
	}
	if plan.Tasks[0].Title != "Simple task" {
		t.Errorf("expected 'Simple task', got %s", plan.Tasks[0].Title)
	}
}

func TestParseProjectPlanEmpty(t *testing.T) {
	content := `No tasks here, just some text.`

	plan := ParseProjectPlan(content)

	if len(plan.Tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(plan.Tasks))
	}
	if len(plan.Questions) != 0 {
		t.Errorf("expected 0 questions, got %d", len(plan.Questions))
	}
	if len(plan.Blockers) != 0 {
		t.Errorf("expected 0 blockers, got %d", len(plan.Blockers))
	}
}

func TestParseTaskContent(t *testing.T) {
	content := `
- **Priority**: 5
- **Status**: blocked
- **Labels**: api, backend, urgent
- **Depends on**: task-1, task-2, task-3
- **Assignee**: test@example.com
- **Description**: This is a detailed description
`

	task := parseTaskContent("task-10", "Test Task", content)

	if task.ID != "task-10" {
		t.Errorf("expected task-10, got %s", task.ID)
	}
	if task.Title != "Test Task" {
		t.Errorf("expected 'Test Task', got %s", task.Title)
	}
	if task.Priority != 5 {
		t.Errorf("expected priority 5, got %d", task.Priority)
	}
	if task.Status != storage.TaskStatusBlocked {
		t.Errorf("expected blocked status, got %s", task.Status)
	}
	if len(task.Labels) != 3 {
		t.Errorf("expected 3 labels, got %d", len(task.Labels))
	}
	if len(task.DependsOn) != 3 {
		t.Errorf("expected 3 dependencies, got %d", len(task.DependsOn))
	}
	if task.Assignee != "test@example.com" {
		t.Errorf("expected assignee, got %s", task.Assignee)
	}
	if task.Description != "This is a detailed description" {
		t.Errorf("expected description, got %s", task.Description)
	}
}

func TestExtractSection(t *testing.T) {
	content := `## Overview
This is the overview section.

## Tasks
Here are the tasks.

## Questions
1. First question
`

	tasks := extractSection(content, "Tasks")
	if tasks != "Here are the tasks." {
		t.Errorf("expected 'Here are the tasks.', got '%s'", tasks)
	}

	questions := extractSection(content, "Questions")
	if questions != "1. First question" {
		t.Errorf("expected '1. First question', got '%s'", questions)
	}

	missing := extractSection(content, "NonExistent")
	if missing != "" {
		t.Errorf("expected empty string, got '%s'", missing)
	}
}

func TestParseQuestions(t *testing.T) {
	content := `## Questions
1. First question?
2. Second question?
- Third question with bullet?
`

	questions := parseQuestions(content)

	if len(questions) != 3 {
		t.Errorf("expected 3 questions, got %d", len(questions))
	}
	if questions[0] != "First question?" {
		t.Errorf("expected 'First question?', got '%s'", questions[0])
	}
}

func TestParseBlockers(t *testing.T) {
	content := `## Blockers
- First blocker
- Second blocker
* Third blocker with asterisk
`

	blockers := parseBlockers(content)

	if len(blockers) != 3 {
		t.Errorf("expected 3 blockers, got %d", len(blockers))
	}
	if blockers[0] != "First blocker" {
		t.Errorf("expected 'First blocker', got '%s'", blockers[0])
	}
}

func TestParseTaskOrder(t *testing.T) {
	content := `## Recommended Order

1. task-3 - Setup database (no dependencies)
2. task-1 - Create API endpoints (depends on task-3)
3. task-2 - Implement auth (depends on task-1)
4. task-4 - Write tests

## Reasoning

Tasks were reordered based on dependency analysis. task-3 has no dependencies
and blocks other tasks, so it should be done first.
`

	order, reasoning, err := ParseTaskOrder(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(order) != 4 {
		t.Errorf("expected 4 tasks, got %d", len(order))
	}

	expected := []string{"task-3", "task-1", "task-2", "task-4"}
	for i, taskID := range expected {
		if order[i] != taskID {
			t.Errorf("expected order[%d] = %s, got %s", i, taskID, order[i])
		}
	}

	if reasoning == "" {
		t.Error("expected reasoning to be non-empty")
	}
}

func TestParseTaskOrder_NoOrderSection(t *testing.T) {
	content := `## Some Other Section

This content doesn't have a Recommended Order section.
`

	_, _, err := ParseTaskOrder(content)
	if err == nil {
		t.Error("expected error for missing order section")
	}
}

func TestParseTaskOrder_NoTasks(t *testing.T) {
	content := `## Recommended Order

No tasks listed here, just text.

## Reasoning

Nothing to reorder.
`

	_, _, err := ParseTaskOrder(content)
	if err == nil {
		t.Error("expected error for no tasks found")
	}
}

func TestParseTaskContentWithParent(t *testing.T) {
	content := `
- **Priority**: 2
- **Status**: ready
- **Parent**: task-1
- **Labels**: backend, subtask
- **Description**: This is a subtask of task-1
`

	task := parseTaskContent("task-2", "Subtask", content)

	if task.ParentID != "task-1" {
		t.Errorf("expected ParentID task-1, got %s", task.ParentID)
	}
	if task.Priority != 2 {
		t.Errorf("expected priority 2, got %d", task.Priority)
	}
	if len(task.Labels) != 2 {
		t.Errorf("expected 2 labels, got %d", len(task.Labels))
	}
}

func TestParseProjectPlanWithSubtasks(t *testing.T) {
	content := `## Tasks

### task-1: Setup infrastructure
- **Priority**: 1
- **Status**: ready
- **Labels**: infrastructure
- **Description**: Set up the base infrastructure

### task-2: Create database schema
- **Priority**: 2
- **Status**: ready
- **Parent**: task-1
- **Labels**: database
- **Description**: Design and create the database schema

### task-3: Implement API endpoints
- **Priority**: 2
- **Status**: blocked
- **Parent**: task-1
- **Depends on**: task-2
- **Labels**: api, backend
- **Description**: Implement REST API endpoints
`

	plan := ParseProjectPlan(content)

	if len(plan.Tasks) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(plan.Tasks))
	}

	// Verify task-1 has no parent
	if plan.Tasks[0].ParentID != "" {
		t.Errorf("task-1 should have no parent, got %s", plan.Tasks[0].ParentID)
	}

	// Verify task-2 has task-1 as parent
	if plan.Tasks[1].ParentID != "task-1" {
		t.Errorf("task-2 ParentID should be task-1, got %s", plan.Tasks[1].ParentID)
	}

	// Verify task-3 has BOTH parent AND depends_on (orthogonal concepts)
	if plan.Tasks[2].ParentID != "task-1" {
		t.Errorf("task-3 ParentID should be task-1, got %s", plan.Tasks[2].ParentID)
	}
	if len(plan.Tasks[2].DependsOn) != 1 || plan.Tasks[2].DependsOn[0] != "task-2" {
		t.Errorf("task-3 should depend on task-2, got %v", plan.Tasks[2].DependsOn)
	}
}
