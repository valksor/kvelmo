package conductor

import (
	"context"
	"strings"
	"testing"
)

func TestPlanNoActiveTask(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	err = c.Plan(context.Background())
	if err == nil {
		t.Fatal("Plan should return error without active task")
	}
	if !strings.Contains(err.Error(), "no active task") {
		t.Errorf("expected 'no active task' error, got %v", err)
	}
}

func TestImplementNoActiveTask(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	err = c.Implement(context.Background())
	if err == nil {
		t.Fatal("Implement should return error without active task")
	}
	if !strings.Contains(err.Error(), "no active task") {
		t.Errorf("expected 'no active task' error, got %v", err)
	}
}

func TestReviewNoActiveTask(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	err = c.Review(context.Background())
	if err == nil {
		t.Fatal("Review should return error without active task")
	}
	if !strings.Contains(err.Error(), "no active task") {
		t.Errorf("expected 'no active task' error, got %v", err)
	}
}

func TestFinishNoActiveTask(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	err = c.Finish(context.Background(), FinishOptions{})
	if err == nil {
		t.Fatal("Finish should return error without active task")
	}
	if !strings.Contains(err.Error(), "no active task") {
		t.Errorf("expected 'no active task' error, got %v", err)
	}
}

func TestUndoNoActiveTask(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	err = c.Undo(context.Background())
	if err == nil {
		t.Fatal("Undo should return error without active task")
	}
	if !strings.Contains(err.Error(), "no active task") && !strings.Contains(err.Error(), "undo") {
		t.Errorf("expected task/undo related error, got %v", err)
	}
}

func TestRedoNoActiveTask(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	err = c.Redo(context.Background())
	if err == nil {
		t.Fatal("Redo should return error without active task")
	}
	if !strings.Contains(err.Error(), "no active task") && !strings.Contains(err.Error(), "redo") {
		t.Errorf("expected task/redo related error, got %v", err)
	}
}

func TestSimplifyNoActiveTask(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	err = c.Simplify(context.Background(), "", true)
	if err == nil {
		t.Fatal("Simplify should return error without active task")
	}
	if !strings.Contains(err.Error(), "no active task") {
		t.Errorf("expected 'no active task' error, got %v", err)
	}
}

func TestResumePausedNoActiveTask(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	err = c.ResumePaused(context.Background())
	if err == nil {
		t.Fatal("ResumePaused should return error without active task")
	}
}

func TestAskQuestionNoActiveTask(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	err = c.AskQuestion(context.Background(), "test question")
	if err == nil {
		t.Fatal("AskQuestion should return error without active task")
	}
}

func TestAnswerQuestionNoActiveTask(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	err = c.AnswerQuestion(context.Background(), "test answer")
	if err == nil {
		t.Fatal("AnswerQuestion should return error without active task")
	}
}

func TestResetStateNoActiveTask(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	err = c.ResetState(context.Background())
	if err == nil {
		t.Fatal("ResetState should return error without active task")
	}
	if !strings.Contains(err.Error(), "no active task") {
		t.Errorf("expected 'no active task' error, got %v", err)
	}
}

func TestImplementReviewNoActiveTask(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	err = c.ImplementReview(context.Background(), 1)
	if err == nil {
		t.Fatal("ImplementReview should return error without active task")
	}
	if !strings.Contains(err.Error(), "no active task") {
		t.Errorf("expected 'no active task' error, got %v", err)
	}
}

func TestImplementReviewInvalidNumber(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Even with no active task, we should validate review number first
	err = c.ImplementReview(context.Background(), 0)
	if err == nil {
		t.Fatal("ImplementReview should return error for review number 0")
	}
	// Could be "no active task" or "invalid review number" - both are acceptable
}

func TestGetTaskID_Empty(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	taskID := c.GetTaskID()
	if taskID != "" {
		t.Errorf("GetTaskID = %q, want empty when no active task", taskID)
	}
}

func TestGetActiveTask_Nil(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	task := c.GetActiveTask()
	if task != nil {
		t.Error("GetActiveTask should return nil before task is started")
	}
}

func TestGetWorkspace_Nil(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ws := c.GetWorkspace()
	if ws != nil {
		t.Error("GetWorkspace should return nil before initialization")
	}
}

func TestGetGit_Nil(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	git := c.GetGit()
	if git != nil {
		t.Error("GetGit should return nil before initialization")
	}
}

func TestStatus_NoWorkspace(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	status, err := c.Status(context.Background())
	if err == nil {
		t.Fatal("Status should return error without workspace")
	}
	if status != nil {
		t.Errorf("status = %v, want nil on error", status)
	}
}

func TestAddNote_NoWorkspace(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	err = c.AddNote(context.Background(), "test note")
	if err == nil {
		t.Fatal("AddNote should return error without workspace")
	}
}
