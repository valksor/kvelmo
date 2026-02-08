package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/valksor/go-mehrhof/internal/conductor/commands"
	"github.com/valksor/go-mehrhof/internal/events"
	"github.com/valksor/go-toolkit/eventbus"
)

// handleInteractiveCommand executes a workflow command.
// POST /api/v1/interactive/command.
func (s *Server) handleInteractiveCommand(w http.ResponseWriter, r *http.Request) {
	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")

		return
	}

	req, err := parseCommandRequest(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

		return
	}

	slog.Debug("interactive command received", "command", req.Command, "args", req.Args)

	opCtx, cancel := context.WithCancel(r.Context())
	defer cancel()

	sessionID := s.getSessionID(r)
	s.registerOperation(sessionID, cancel, req.Command)
	defer s.unregisterOperation(sessionID)

	if !commands.IsKnownCommand(req.Command) {
		s.writeError(w, http.StatusBadRequest, "unknown command: "+req.Command)

		return
	}

	result, err := commands.Execute(opCtx, s.config.Conductor, req.Command, commands.Invocation{
		Args:   req.Args,
		Source: commands.SourceInteractive,
	})
	if err != nil {
		switch {
		case errors.Is(err, commands.ErrNoActiveTask):
			s.writeError(w, http.StatusBadRequest, "no active task")
		case errors.Is(err, commands.ErrUnknownCommand):
			s.writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, context.Canceled):
			state := ""
			if task := s.config.Conductor.GetActiveTask(); task != nil {
				state = task.State
			}
			s.writeJSON(w, http.StatusOK, commandResponse{
				Success: true,
				Message: req.Command + " cancelled",
				State:   state,
			})
		default:
			s.writeError(w, http.StatusInternalServerError, err.Error())
		}

		return
	}

	if result == nil {
		s.writeJSON(w, http.StatusOK, commandResponse{Success: true, Message: "OK"})

		return
	}

	if result.Type == commands.ResultExit {
		s.writeJSON(w, http.StatusOK, commandResponse{
			Success: true,
			Message: "exit",
			State:   result.State,
		})

		return
	}

	response := s.routerResultToJSON(result)
	s.writeJSON(w, http.StatusOK, response)

	if result.TaskID != "" && result.State != "" {
		s.config.EventBus.PublishRaw(eventbus.Event{
			Type: events.TypeStateChanged,
			Data: map[string]any{
				"task_id": result.TaskID,
				"state":   result.State,
			},
		})
	}
}
