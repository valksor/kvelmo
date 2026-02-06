package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/workflow"
)

// executeInteractiveTaskCommand handles task management interactive commands.
// Commands: cost, budget, list, note, quick, specification, spec, simplify, label,
// delete, export, optimize, submit, sync, answer, question.
func (s *Server) executeInteractiveTaskCommand(ctx context.Context, command string, args []string) (string, error) {
	cond := s.config.Conductor

	switch command {
	case "cost":
		task := cond.GetActiveTask()
		if task == nil {
			return "", errors.New("no active task")
		}
		ws := cond.GetWorkspace()
		work, loadErr := ws.LoadWork(task.ID)
		if loadErr != nil {
			return "", loadErr
		}
		costs := work.Costs

		return fmt.Sprintf("Input: %d tokens\nOutput: %d tokens\nTotal: $%.4f",
			costs.TotalInputTokens, costs.TotalOutputTokens, costs.TotalCostUSD), nil

	case "list":
		ws := cond.GetWorkspace()
		taskIDs, listErr := ws.ListWorks()
		if listErr != nil {
			return "", listErr
		}
		if len(taskIDs) == 0 {
			return "No tasks found", nil
		}
		var lines []string
		for _, id := range taskIDs {
			work, loadErr := ws.LoadWork(id)
			if loadErr != nil {
				continue
			}
			shortID := id
			if len(id) > 8 {
				shortID = id[:8]
			}
			title := work.Metadata.Title
			if title == "" {
				title = work.Source.Ref
			}
			lines = append(lines, fmt.Sprintf("• %s: %s (%s)", shortID, title, work.Metadata.State))
		}

		return strings.Join(lines, "\n"), nil

	case "note":
		if len(args) == 0 {
			return "", errors.New("note requires a message")
		}
		task := cond.GetActiveTask()
		if task == nil {
			return "", errors.New("no active task")
		}
		ws := cond.GetWorkspace()
		noteMsg := strings.Join(args, " ")
		if noteErr := ws.AppendNote(task.ID, noteMsg, task.State); noteErr != nil {
			return "", noteErr
		}

		return "Note saved", nil

	case "quick":
		if len(args) == 0 {
			return "", errors.New("quick requires a description")
		}
		quickResult, quickErr := cond.CreateQuickTask(ctx, conductor.QuickTaskOptions{
			Description: strings.Join(args, " "),
			QueueID:     "quick-tasks",
		})
		if quickErr != nil {
			return "", quickErr
		}

		return "Quick task created: " + quickResult.TaskID, nil

	case "specification", "spec":
		task := cond.GetActiveTask()
		if task == nil {
			return "", errors.New("no active task")
		}
		ws := cond.GetWorkspace()
		if len(args) == 0 {
			specs, specErr := ws.ListSpecificationsWithStatus(task.ID)
			if specErr != nil {
				return "", specErr
			}

			return fmt.Sprintf("Found %d specifications", len(specs)), nil
		}
		num, parseErr := strconv.Atoi(args[0])
		if parseErr != nil {
			return "", errors.New("specification number must be an integer")
		}
		spec, loadErr := ws.LoadSpecification(task.ID, num)
		if loadErr != nil {
			return "", loadErr
		}
		// Spec is raw markdown content, show first 500 chars
		preview := spec
		if len(preview) > 500 {
			preview = preview[:500] + "..."
		}

		return fmt.Sprintf("Specification %d:\n%s", num, preview), nil

	case "simplify":
		task := cond.GetActiveTask()
		if task == nil {
			return "", errors.New("no active task")
		}
		if err := cond.Simplify(ctx, "", true); err != nil {
			return "", err
		}

		return "Simplification complete", nil

	case "label":
		task := cond.GetActiveTask()
		if task == nil {
			return "", errors.New("no active task")
		}
		ws := cond.GetWorkspace()
		if len(args) == 0 {
			labels, _ := ws.GetLabels(task.ID)
			if len(labels) == 0 {
				return "No labels", nil
			}

			return "Labels: " + strings.Join(labels, ", "), nil
		}
		subCmd := args[0]
		subArgs := args[1:]
		switch subCmd {
		case "add":
			for _, label := range subArgs {
				_ = ws.AddLabel(task.ID, label)
			}

			return fmt.Sprintf("Added %d label(s)", len(subArgs)), nil
		case "remove", "rm":
			for _, label := range subArgs {
				_ = ws.RemoveLabel(task.ID, label)
			}

			return fmt.Sprintf("Removed %d label(s)", len(subArgs)), nil
		case "clear":
			_ = ws.SetLabels(task.ID, []string{})

			return "Labels cleared", nil
		case "list", "ls":
			labels, _ := ws.GetLabels(task.ID)
			if len(labels) == 0 {
				return "No labels", nil
			}

			return "Labels: " + strings.Join(labels, ", "), nil
		default:
			// Treat as adding labels directly
			for _, label := range args {
				_ = ws.AddLabel(task.ID, label)
			}

			return fmt.Sprintf("Added %d label(s)", len(args)), nil
		}

	case "budget":
		task := cond.GetActiveTask()
		if task == nil {
			return "", errors.New("no active task")
		}
		ws := cond.GetWorkspace()
		work, loadErr := ws.LoadWork(task.ID)
		if loadErr != nil {
			return "", loadErr
		}
		cfg, cfgErr := ws.LoadConfig()
		if cfgErr != nil {
			return "", cfgErr
		}
		taskBudget := cfg.Budget.PerTask
		if work.Budget != nil {
			taskBudget = *work.Budget
		}
		costs := work.Costs
		totalTokens := costs.TotalInputTokens + costs.TotalOutputTokens

		return fmt.Sprintf("Tokens: %d\nCost: $%.4f / $%.2f budget",
			totalTokens, costs.TotalCostUSD, taskBudget.MaxCost), nil

	case "delete":
		// Delete a queue task: delete <queue>/<task-id>
		if len(args) == 0 {
			return "", errors.New("delete requires a task reference (e.g., quick-tasks/task-1)")
		}
		queueID, taskID, parseErr := conductor.ParseQueueTaskRef(args[0])
		if parseErr != nil {
			return "", parseErr
		}
		ws := cond.GetWorkspace()
		queue, loadErr := storage.LoadTaskQueue(ws, queueID)
		if loadErr != nil {
			return "", fmt.Errorf("queue not found: %s", queueID)
		}
		if !queue.RemoveTask(taskID) {
			return "", fmt.Errorf("task not found: %s/%s", queueID, taskID)
		}
		if saveErr := queue.Save(); saveErr != nil {
			return "", fmt.Errorf("save queue: %w", saveErr)
		}
		// Delete notes file
		notesPath := ws.QueueNotePath(queueID, taskID)
		_ = ws.DeleteFile(notesPath)

		return fmt.Sprintf("Deleted task %s from %s", taskID, queueID), nil

	case "export":
		// Export a queue task to markdown: export <queue>/<task-id>
		if len(args) == 0 {
			return "", errors.New("export requires a task reference (e.g., quick-tasks/task-1)")
		}
		queueID, taskID, parseErr := conductor.ParseQueueTaskRef(args[0])
		if parseErr != nil {
			return "", parseErr
		}
		markdown, exportErr := cond.ExportQueueTask(queueID, taskID)
		if exportErr != nil {
			return "", exportErr
		}
		// Return the markdown content (truncated for display)
		preview := markdown
		if len(preview) > 1000 {
			preview = preview[:1000] + "\n... (truncated)"
		}

		return fmt.Sprintf("Exported %s/%s:\n%s", queueID, taskID, preview), nil

	case "optimize":
		// AI optimize a queue task: optimize <queue>/<task-id>
		if len(args) == 0 {
			return "", errors.New("optimize requires a task reference (e.g., quick-tasks/task-1)")
		}
		queueID, taskID, parseErr := conductor.ParseQueueTaskRef(args[0])
		if parseErr != nil {
			return "", parseErr
		}
		optimized, optimizeErr := cond.OptimizeQueueTask(ctx, queueID, taskID)
		if optimizeErr != nil {
			return "", optimizeErr
		}
		var changes []string
		if optimized.OriginalTitle != optimized.OptimizedTitle {
			changes = append(changes, fmt.Sprintf("Title: %s → %s", optimized.OriginalTitle, optimized.OptimizedTitle))
		}
		if len(optimized.AddedLabels) > 0 {
			changes = append(changes, "Added labels: "+strings.Join(optimized.AddedLabels, ", "))
		}
		if len(optimized.ImprovementNotes) > 0 {
			changes = append(changes, "Improvements: "+strings.Join(optimized.ImprovementNotes, "; "))
		}
		if len(changes) == 0 {
			return "Task optimized (no major changes)", nil
		}

		return "Task optimized:\n• " + strings.Join(changes, "\n• "), nil

	case "submit":
		// Submit a queue task to provider: submit <queue>/<task-id> <provider>
		if len(args) < 2 {
			return "", errors.New("submit requires: submit <queue>/<task-id> <provider>")
		}
		queueID, taskID, parseErr := conductor.ParseQueueTaskRef(args[0])
		if parseErr != nil {
			return "", parseErr
		}
		providerName := args[1]
		submitResult, submitErr := cond.SubmitQueueTask(ctx, queueID, taskID, conductor.SubmitOptions{
			Provider: providerName,
			TaskIDs:  []string{taskID},
		})
		if submitErr != nil {
			return "", submitErr
		}
		if len(submitResult.Tasks) == 0 {
			return "No tasks submitted", nil
		}
		r := submitResult.Tasks[0]

		return fmt.Sprintf("Submitted to %s: %s\nURL: %s", providerName, r.ExternalID, r.ExternalURL), nil

	case "sync":
		// Sync task from provider: sync <task-id>
		task := cond.GetActiveTask()
		if task == nil {
			return "", errors.New("no active task to sync")
		}
		// For now, indicate that sync is available - full implementation would require
		// provider fetch and delta spec generation
		return fmt.Sprintf("Sync requested for task %s. Use 'mehr sync %s' from CLI for full provider sync.", task.ID, task.ID), nil

	case "answer":
		if len(args) == 0 {
			return "", errors.New("answer requires a response")
		}
		task := cond.GetActiveTask()
		if task == nil {
			return "", errors.New("no active task")
		}
		ws := cond.GetWorkspace()
		// Clear pending question
		if clearErr := ws.ClearPendingQuestion(task.ID); clearErr != nil {
			slog.Warn("clear pending question", "error", clearErr)
		}
		// Save answer as note
		response := strings.Join(args, " ")
		if noteErr := ws.AppendNote(task.ID, response, task.State); noteErr != nil {
			return "", noteErr
		}
		result := "Answer saved, resuming..."
		// Resume workflow based on state with timeout context
		state := workflow.State(task.State)
		const resumeTimeout = 5 * time.Minute
		switch state {
		case workflow.StatePlanning:
			go func(ctx context.Context) {
				resumeCtx, cancel := context.WithTimeout(ctx, resumeTimeout)
				defer cancel()
				if resumeErr := cond.Plan(resumeCtx); resumeErr != nil {
					slog.Error("workflow resume failed", "step", "plan", "error", resumeErr)
				}
			}(ctx)
		case workflow.StateImplementing:
			go func(ctx context.Context) {
				resumeCtx, cancel := context.WithTimeout(ctx, resumeTimeout)
				defer cancel()
				if resumeErr := cond.Implement(resumeCtx); resumeErr != nil {
					slog.Error("workflow resume failed", "step", "implement", "error", resumeErr)
				}
			}(ctx)
		case workflow.StateReviewing:
			go func(ctx context.Context) {
				resumeCtx, cancel := context.WithTimeout(ctx, resumeTimeout)
				defer cancel()
				if resumeErr := cond.Review(resumeCtx); resumeErr != nil {
					slog.Error("workflow resume failed", "step", "review", "error", resumeErr)
				}
			}(ctx)
		case workflow.StateIdle, workflow.StateDone, workflow.StateFailed,
			workflow.StateWaiting, workflow.StatePaused,
			workflow.StateCheckpointing, workflow.StateReverting, workflow.StateRestoring:
			// These states are not resumable - do nothing
		}

		return result, nil

	case "question":
		if len(args) == 0 {
			return "", errors.New("question requires a message")
		}
		question := strings.Join(args, " ")
		if questionErr := cond.AskQuestion(ctx, question); questionErr != nil {
			return "", questionErr
		}

		return "Question sent to agent", nil

	default:
		return "", fmt.Errorf("unknown task command: %s", command)
	}
}
