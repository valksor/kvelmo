package commands

import (
	"context"
	"errors"

	"github.com/valksor/go-mehrhof/internal/conductor"
)

// ClassifyError converts an error into an appropriate Result type.
// It handles domain-specific errors (budget paused/stopped, pending question, context canceled)
// and falls back to a generic error result for unknown errors.
func ClassifyError(result *Result, err error) *Result {
	state := ""
	taskID := ""
	if result != nil {
		state = result.State
		taskID = result.TaskID
	}

	switch {
	case errors.Is(err, conductor.ErrPendingQuestion):
		return &Result{
			Type:    ResultWaiting,
			Message: "Agent has a question",
			State:   state,
			TaskID:  taskID,
		}
	case errors.Is(err, conductor.ErrBudgetPaused):
		return &Result{
			Type:    ResultPaused,
			Message: "Task paused due to budget limit",
			State:   state,
			TaskID:  taskID,
		}
	case errors.Is(err, conductor.ErrBudgetStopped):
		return &Result{
			Type:    ResultStopped,
			Message: "Task stopped due to budget limit",
			State:   state,
			TaskID:  taskID,
		}
	case errors.Is(err, context.Canceled):
		return &Result{
			Type:    ResultMessage,
			Message: "cancelled",
			State:   state,
			TaskID:  taskID,
		}
	default:
		return &Result{
			Type:    ResultError,
			Message: err.Error(),
			State:   state,
			TaskID:  taskID,
		}
	}
}

// EnrichWaitingResult loads pending question details into result.Data.
func EnrichWaitingResult(result *Result, cond *conductor.Conductor) {
	if result == nil || cond == nil || result.Type != ResultWaiting {
		return
	}

	task := cond.GetActiveTask()
	if task == nil {
		return
	}
	if result.TaskID == "" {
		result.TaskID = task.ID
	}
	if result.State == "" {
		result.State = task.State
	}

	ws := cond.GetWorkspace()
	if ws == nil {
		return
	}
	q, err := ws.LoadPendingQuestion(task.ID)
	if err != nil || q == nil {
		return
	}

	options := make([]QuestionOption, 0, len(q.Options))
	for _, opt := range q.Options {
		options = append(options, QuestionOption{
			Label:       opt.Label,
			Value:       opt.Value,
			Description: opt.Description,
		})
	}

	phase := q.Phase
	if phase == "" {
		phase = result.State
	}

	result.Data = WaitingData{
		Question: q.Question,
		Options:  options,
		Phase:    phase,
	}
}
