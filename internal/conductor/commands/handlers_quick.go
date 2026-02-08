package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/storage"
)

func init() {
	Register(Command{
		Info: CommandInfo{
			Name:        "quick-list",
			Description: "List quick tasks",
			Category:    "quick",
		},
		Handler: handleQuickList,
	})

	Register(Command{
		Info: CommandInfo{
			Name:        "quick-get",
			Description: "Get a quick task with notes",
			Category:    "quick",
		},
		Handler: handleQuickGet,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "quick-note",
			Description:  "Add a note to a quick task",
			Category:     "quick",
			MutatesState: true,
		},
		Handler: handleQuickNote,
	})
}

func handleQuickList(_ context.Context, cond *conductor.Conductor, _ Invocation) (*Result, error) {
	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("workspace not initialized")
	}

	queueID := "quick-tasks"
	queue, err := storage.LoadTaskQueue(ws, queueID)
	if err != nil {
		// Queue doesn't exist yet, return empty list.
		return NewResult("0 quick task(s)").WithData(map[string]any{ //nolint:nilerr // Graceful: no queue = empty list
			"tasks": []map[string]any{},
			"count": 0,
		}), nil
	}

	type quickTaskItem struct {
		ID        string   `json:"id"`
		Title     string   `json:"title"`
		Priority  int      `json:"priority"`
		Labels    []string `json:"labels"`
		Status    string   `json:"status"`
		NoteCount int      `json:"note_count"`
	}

	items := make([]quickTaskItem, 0, len(queue.Tasks))
	for _, task := range queue.Tasks {
		notes, _ := ws.LoadQueueNotes(queueID, task.ID)

		items = append(items, quickTaskItem{
			ID:        task.ID,
			Title:     task.Title,
			Priority:  task.Priority,
			Labels:    append([]string{}, task.Labels...),
			Status:    string(task.Status),
			NoteCount: len(notes),
		})
	}

	return NewResult(fmt.Sprintf("%d quick task(s)", len(items))).WithData(map[string]any{
		"tasks": items,
		"count": len(items),
	}), nil
}

func handleQuickGet(_ context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("workspace not initialized")
	}

	taskID := strings.TrimSpace(GetString(inv.Options, "task_id"))
	if taskID == "" && len(inv.Args) > 0 {
		taskID = strings.TrimSpace(inv.Args[0])
	}
	if taskID == "" {
		return nil, errors.New("task_id is required")
	}

	queueID := "quick-tasks"
	queue, err := storage.LoadTaskQueue(ws, queueID)
	if err != nil {
		return nil, fmt.Errorf("quick-tasks queue not found: %w", err)
	}

	task := queue.GetTask(taskID)
	if task == nil {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	notes, _ := ws.LoadQueueNotes(queueID, taskID)

	notesList := make([]map[string]any, 0, len(notes))
	for _, note := range notes {
		notesList = append(notesList, map[string]any{
			"timestamp": note.Timestamp,
			"content":   note.Content,
		})
	}

	return NewResult("Quick task: " + task.Title).WithData(map[string]any{
		"id":          task.ID,
		"title":       task.Title,
		"description": task.Description,
		"priority":    task.Priority,
		"labels":      task.Labels,
		"status":      string(task.Status),
		"notes":       notesList,
	}), nil
}

func handleQuickNote(_ context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("workspace not initialized")
	}

	taskID := strings.TrimSpace(GetString(inv.Options, "task_id"))
	if taskID == "" && len(inv.Args) > 0 {
		taskID = strings.TrimSpace(inv.Args[0])
	}
	if taskID == "" {
		return nil, errors.New("task_id is required")
	}

	note := strings.TrimSpace(GetString(inv.Options, "note"))
	if note == "" && len(inv.Args) > 1 {
		note = strings.TrimSpace(strings.Join(inv.Args[1:], " "))
	}
	if note == "" {
		return nil, errors.New("note content is required")
	}

	queueID := "quick-tasks"
	if err := ws.AppendQueueNote(queueID, taskID, note); err != nil {
		return nil, fmt.Errorf("failed to save note: %w", err)
	}

	return NewResult("Note added").WithData(map[string]any{
		"success": true,
		"message": "note saved",
	}), nil
}
