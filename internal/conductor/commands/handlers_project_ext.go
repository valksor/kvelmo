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
			Name:        "project",
			Description: "Project workflow commands",
			Category:    "project",
			Subcommands: []string{
				"plan", "queues", "queue", "queue-delete",
				"tasks", "task-edit", "reorder", "submit",
				"start", "sync", "upload", "source",
			},
			MutatesState: true,
		},
		Handler: handleProject,
	})
}

func handleProject(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	if len(inv.Args) == 0 {
		return nil, errors.New("project requires a subcommand: plan, queues, queue, queue-delete, tasks, task-edit, reorder, submit, start, sync, upload, source")
	}

	subcommand := inv.Args[0]
	subInv := Invocation{
		Args:    inv.Args[1:],
		Options: inv.Options,
		Source:  inv.Source,
	}

	switch subcommand {
	case "plan":
		return handleProjectPlanCmd(ctx, cond, subInv)
	case "queues":
		return handleProjectQueuesCmd(ctx, cond, subInv)
	case "queue":
		return handleProjectQueueCmd(ctx, cond, subInv)
	case "queue-delete":
		return handleProjectQueueDeleteCmd(ctx, cond, subInv)
	case "tasks":
		return handleProjectTasksCmd(ctx, cond, subInv)
	case "task-edit":
		return handleProjectTaskEditCmd(ctx, cond, subInv)
	case "reorder":
		return handleProjectReorderCmd(ctx, cond, subInv)
	case "submit":
		return handleProjectSubmitCmd(ctx, cond, subInv)
	case "start":
		return handleProjectStartCmd(ctx, cond, subInv)
	case "sync":
		return handleProjectSyncCmd(ctx, cond, subInv)
	case "upload":
		return handleProjectUploadCmd(ctx, cond, subInv)
	case "source":
		return handleProjectSourceCmd(ctx, cond, subInv)
	default:
		return nil, fmt.Errorf("unknown project subcommand: %s", subcommand)
	}
}

// handleProjectPlanCmd creates a project plan from a source.
func handleProjectPlanCmd(ctx context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	source := strings.TrimSpace(GetString(inv.Options, "source"))
	if source == "" && len(inv.Args) > 0 {
		source = strings.TrimSpace(inv.Args[0])
	}
	if source == "" {
		return nil, errors.New("project plan requires a source")
	}

	opts := conductor.ProjectPlanOptions{
		Title:              strings.TrimSpace(GetString(inv.Options, "title")),
		CustomInstructions: strings.TrimSpace(GetString(inv.Options, "instructions")),
		UseSchema:          GetBool(inv.Options, "use_schema"),
	}

	result, err := cond.CreateProjectPlan(ctx, source, opts)
	if err != nil {
		return nil, fmt.Errorf("create plan failed: %w", err)
	}

	tasks := make([]map[string]any, 0, len(result.Tasks))
	for _, task := range result.Tasks {
		tasks = append(tasks, convertQueuedTaskMap(task))
	}

	return NewResult("Project plan created").WithData(map[string]any{
		"queue_id":  result.Queue.ID,
		"title":     result.Queue.Title,
		"source":    result.Queue.Source,
		"tasks":     tasks,
		"questions": result.Questions,
		"blockers":  result.Blockers,
	}), nil
}

// handleProjectQueuesCmd lists all project queues.
func handleProjectQueuesCmd(_ context.Context, cond *conductor.Conductor, _ Invocation) (*Result, error) {
	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("workspace not initialized")
	}

	queueIDs, err := ws.ListQueues()
	if err != nil {
		return nil, fmt.Errorf("list queues failed: %w", err)
	}

	queues := make([]map[string]any, 0, len(queueIDs))
	for _, id := range queueIDs {
		queue, loadErr := storage.LoadTaskQueue(ws, id)
		if loadErr != nil {
			continue // Skip broken queues
		}
		queues = append(queues, map[string]any{
			"id":         queue.ID,
			"title":      queue.Title,
			"source":     queue.Source,
			"status":     string(queue.Status),
			"task_count": len(queue.Tasks),
		})
	}

	return NewListResult(fmt.Sprintf("Found %d queue(s)", len(queues)), map[string]any{
		"queues": queues,
	}), nil
}

// handleProjectQueueCmd gets a specific queue by ID.
func handleProjectQueueCmd(_ context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("workspace not initialized")
	}

	queueID := strings.TrimSpace(GetString(inv.Options, "queue_id"))
	if queueID == "" && len(inv.Args) > 0 {
		queueID = strings.TrimSpace(inv.Args[0])
	}
	if queueID == "" {
		return nil, errors.New("queue ID is required")
	}

	queue, err := storage.LoadTaskQueue(ws, queueID)
	if err != nil {
		return nil, fmt.Errorf("queue not found: %w", err)
	}

	tasks := make([]map[string]any, 0, len(queue.Tasks))
	for _, task := range queue.Tasks {
		tasks = append(tasks, convertQueuedTaskMap(task))
	}

	return NewResult("Queue: " + queue.Title).WithData(map[string]any{
		"queue_id":  queue.ID,
		"title":     queue.Title,
		"source":    queue.Source,
		"status":    string(queue.Status),
		"tasks":     tasks,
		"questions": queue.Questions,
		"blockers":  queue.Blockers,
	}), nil
}

// handleProjectQueueDeleteCmd deletes a queue.
func handleProjectQueueDeleteCmd(_ context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("workspace not initialized")
	}

	queueID := strings.TrimSpace(GetString(inv.Options, "queue_id"))
	if queueID == "" && len(inv.Args) > 0 {
		queueID = strings.TrimSpace(inv.Args[0])
	}
	if queueID == "" {
		return nil, errors.New("queue ID is required")
	}

	if err := ws.DeleteQueue(queueID); err != nil {
		return nil, fmt.Errorf("delete queue failed: %w", err)
	}

	return NewResult(fmt.Sprintf("Queue %s deleted", queueID)).WithData(map[string]any{
		"success":  true,
		"queue_id": queueID,
		"status":   "deleted",
	}), nil
}

// handleProjectTasksCmd lists tasks with optional filtering.
func handleProjectTasksCmd(_ context.Context, cond *conductor.Conductor, inv Invocation) (*Result, error) {
	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("workspace not initialized")
	}

	queueID := strings.TrimSpace(GetString(inv.Options, "queue_id"))
	statusFilter := strings.TrimSpace(GetString(inv.Options, "status"))

	// If no queue specified, use most recent
	if queueID == "" {
		queueID = mostRecentQueue(ws)
		if queueID == "" {
			return nil, errors.New("no queues found")
		}
	}

	queue, err := storage.LoadTaskQueue(ws, queueID)
	if err != nil {
		return nil, fmt.Errorf("queue not found: %w", err)
	}

	tasks := make([]map[string]any, 0, len(queue.Tasks))
	for _, task := range queue.Tasks {
		if statusFilter != "" && string(task.Status) != statusFilter {
			continue
		}
		tasks = append(tasks, convertQueuedTaskMap(task))
	}

	return NewListResult(fmt.Sprintf("Found %d task(s)", len(tasks)), map[string]any{
		"queue_id":  queue.ID,
		"tasks":     tasks,
		"questions": queue.Questions,
		"blockers":  queue.Blockers,
	}), nil
}

// mostRecentQueue returns the most recent queue ID, or empty if none exist.
func mostRecentQueue(ws *storage.Workspace) string {
	queueIDs, err := ws.ListQueues()
	if err != nil || len(queueIDs) == 0 {
		return ""
	}

	return queueIDs[len(queueIDs)-1]
}

// convertQueuedTaskMap converts a QueuedTask to a map for response data.
func convertQueuedTaskMap(task *storage.QueuedTask) map[string]any {
	if task == nil {
		return nil
	}

	return map[string]any{
		"id":           task.ID,
		"title":        task.Title,
		"description":  task.Description,
		"status":       string(task.Status),
		"priority":     task.Priority,
		"parent_id":    task.ParentID,
		"subtasks":     task.Subtasks,
		"depends_on":   task.DependsOn,
		"blocks":       task.Blocks,
		"labels":       task.Labels,
		"assignee":     task.Assignee,
		"external_id":  task.ExternalID,
		"external_url": task.ExternalURL,
	}
}
