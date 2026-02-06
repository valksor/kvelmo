package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/conductor/commands"
	"github.com/valksor/go-mehrhof/internal/events"
	"github.com/valksor/go-mehrhof/internal/library"
	"github.com/valksor/go-mehrhof/internal/memory"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/workflow"
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

	// Parse request (supports both JSON and form-encoded from HTMX)
	req, err := parseCommandRequest(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

		return
	}

	slog.Debug("interactive command received", "command", req.Command, "args", req.Args)

	// Create cancellable context for ALL commands
	opCtx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Register operation for cancellation
	sessionID := s.getSessionID(r)
	s.registerOperation(sessionID, cancel, req.Command)
	defer s.unregisterOperation(sessionID)

	cond := s.config.Conductor

	// First, check if the command is handled by the unified router
	if commands.IsKnownCommand(req.Command) {
		result, err := commands.Execute(opCtx, cond, req.Command, req.Args)
		if err != nil {
			// Map specific errors to HTTP status codes
			if errors.Is(err, commands.ErrNoActiveTask) {
				s.writeError(w, http.StatusBadRequest, "no active task")

				return
			}
			if errors.Is(err, commands.ErrUnknownCommand) {
				s.writeError(w, http.StatusBadRequest, err.Error())

				return
			}
			s.writeError(w, http.StatusInternalServerError, err.Error())

			return
		}

		// Handle special result types
		if result != nil {
			// Handle exit signal (shouldn't happen in Web, but handle for completeness)
			if result.Type == commands.ResultExit {
				s.writeJSON(w, http.StatusOK, commandResponse{
					Success: true,
					Message: "exit",
					State:   result.State,
				})

				return
			}

			// Convert router result to JSON response
			response := s.routerResultToJSON(result)
			s.writeJSON(w, http.StatusOK, response)

			// Publish state update if applicable
			if result.TaskID != "" && result.State != "" {
				s.config.EventBus.PublishRaw(eventbus.Event{
					Type: events.TypeStateChanged,
					Data: map[string]any{
						"task_id": result.TaskID,
						"state":   result.State,
					},
				})
			}

			return
		}

		// Fallback response
		s.writeJSON(w, http.StatusOK, commandResponse{Success: true, Message: "OK"})

		return
	}

	// Handle Web-specific commands that aren't in the router
	var message string

	switch req.Command {
	case "reset":
		err = cond.ResetState(opCtx)
		message = "Workflow reset to idle"

	case "auto":
		// Auto-execute the next workflow step based on current state
		task := cond.GetActiveTask()
		if task == nil {
			s.writeError(w, http.StatusBadRequest, "no active task")

			return
		}
		switch task.State {
		case "idle":
			s.writeError(w, http.StatusBadRequest, "no active task, use 'start' first")

			return
		case "planning":
			err = cond.Plan(opCtx)
			message = "Planning started"
		case "implementing":
			err = cond.Implement(opCtx)
			message = "Implementation started"
		case "reviewing":
			err = cond.Review(opCtx)
			message = "Review started"
		case "waiting":
			s.writeError(w, http.StatusBadRequest, "task is waiting for user input")

			return
		case "done", "failed":
			s.writeError(w, http.StatusBadRequest, "task is already completed")

			return
		default:
			s.writeError(w, http.StatusBadRequest, "cannot auto-execute in state: "+task.State)

			return
		}

	case "find":
		if len(req.Args) == 0 {
			s.writeError(w, http.StatusBadRequest, "find requires a query")

			return
		}
		findOpts := conductor.FindOptions{
			Query:     strings.Join(req.Args, " "),
			Path:      "",
			Pattern:   "",
			Context:   3,
			Workspace: cond.GetWorkspace(),
		}
		resultChan, err := cond.Find(r.Context(), findOpts)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "find: "+err.Error())

			return
		}
		var results []conductor.FindResult
		for result := range resultChan {
			if result.File != "__error__" {
				results = append(results, result)
			}
		}
		message = fmt.Sprintf("Found %d match(es)", len(results))

	case "simplify":
		if cond.GetActiveTask() == nil {
			s.writeError(w, http.StatusBadRequest, "no active task")

			return
		}
		err = cond.Simplify(opCtx, "", true)
		message = "Simplification complete"

	case "label":
		task := cond.GetActiveTask()
		if task == nil {
			s.writeError(w, http.StatusBadRequest, "no active task")

			return
		}
		ws := cond.GetWorkspace()
		if len(req.Args) == 0 {
			labels, _ := ws.GetLabels(task.ID)
			message = fmt.Sprintf("Labels: %v", labels)
		} else {
			subCmd := req.Args[0]
			subArgs := req.Args[1:]
			switch subCmd {
			case "add":
				for _, label := range subArgs {
					_ = ws.AddLabel(task.ID, label)
				}
				message = fmt.Sprintf("Added %d label(s)", len(subArgs))
			case "remove", "rm":
				for _, label := range subArgs {
					_ = ws.RemoveLabel(task.ID, label)
				}
				message = fmt.Sprintf("Removed %d label(s)", len(subArgs))
			case "clear":
				_ = ws.SetLabels(task.ID, []string{})
				message = "Labels cleared"
			case "list", "ls":
				labels, _ := ws.GetLabels(task.ID)
				message = fmt.Sprintf("Labels: %v", labels)
			default:
				for _, label := range req.Args {
					_ = ws.AddLabel(task.ID, label)
				}
				message = fmt.Sprintf("Added %d label(s)", len(req.Args))
			}
		}

	case "memory":
		mem := cond.GetMemory()
		if mem == nil {
			s.writeError(w, http.StatusServiceUnavailable, "memory system is not enabled")

			return
		}
		if len(req.Args) == 0 {
			s.writeError(w, http.StatusBadRequest, "memory requires a subcommand: search <query>, index <task-id>, stats")

			return
		}
		subcommand := req.Args[0]
		subArgs := req.Args[1:]
		switch subcommand {
		case "search":
			if len(subArgs) == 0 {
				s.writeError(w, http.StatusBadRequest, "memory search requires a query")

				return
			}
			query := strings.Join(subArgs, " ")
			results, searchErr := mem.Search(r.Context(), query, memory.SearchOptions{
				Limit:    5,
				MinScore: 0.65,
			})
			if searchErr != nil {
				s.writeError(w, http.StatusInternalServerError, "memory search: "+searchErr.Error())

				return
			}
			message = fmt.Sprintf("Found %d similar task(s)", len(results))
		case "index":
			if len(subArgs) == 0 {
				s.writeError(w, http.StatusBadRequest, "memory index requires a task ID")

				return
			}
			ws := cond.GetWorkspace()
			if ws == nil {
				s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

				return
			}
			taskID := subArgs[0]
			if _, loadErr := ws.LoadWork(taskID); loadErr != nil {
				s.writeError(w, http.StatusNotFound, "task not found: "+loadErr.Error())

				return
			}
			indexer := memory.NewIndexer(mem, ws, nil)
			if indexErr := indexer.IndexTask(r.Context(), taskID); indexErr != nil {
				s.writeError(w, http.StatusInternalServerError, "failed to index task: "+indexErr.Error())

				return
			}
			message = fmt.Sprintf("Task %s indexed successfully", taskID)
		case "stats":
			ws := cond.GetWorkspace()
			if ws == nil {
				s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

				return
			}
			indexer := memory.NewIndexer(mem, ws, nil)
			stats, statsErr := indexer.GetStats(r.Context())
			if statsErr != nil {
				s.writeError(w, http.StatusInternalServerError, "failed to get stats: "+statsErr.Error())

				return
			}
			message = fmt.Sprintf("Total documents: %d", stats.TotalDocuments)
		default:
			// Backwards compatibility: treat unknown subcommand as search query
			query := strings.Join(req.Args, " ")
			results, searchErr := mem.Search(r.Context(), query, memory.SearchOptions{
				Limit:    5,
				MinScore: 0.65,
			})
			if searchErr != nil {
				s.writeError(w, http.StatusInternalServerError, "memory search: "+searchErr.Error())

				return
			}
			message = fmt.Sprintf("Found %d similar task(s)", len(results))
		}

	case "library":
		lib := cond.GetLibrary()
		if lib == nil {
			// Check if there was an initialization error
			errMsg := "library system is not enabled. Use the Library panel or enable in .mehrhof/config.yaml under 'library:'"
			if initErr := cond.GetLibraryError(); initErr != nil {
				errMsg = initErr.Error()
			}
			s.writeError(w, http.StatusServiceUnavailable, errMsg)

			return
		}
		// Default to list if no subcommand
		subcommand := "list"
		subArgs := req.Args
		if len(subArgs) > 0 {
			subcommand = subArgs[0]
			subArgs = subArgs[1:]
		}
		switch subcommand {
		case "list", "ls":
			collections, listErr := lib.List(r.Context(), &library.ListOptions{})
			if listErr != nil {
				s.writeError(w, http.StatusInternalServerError, "list collections: "+listErr.Error())

				return
			}
			message = fmt.Sprintf("Found %d collection(s)", len(collections))
		case "show":
			if len(subArgs) == 0 {
				s.writeError(w, http.StatusBadRequest, "usage: library show <name>")

				return
			}
			coll, showErr := lib.Show(r.Context(), subArgs[0])
			if showErr != nil {
				s.writeError(w, http.StatusInternalServerError, "show collection: "+showErr.Error())

				return
			}
			message = fmt.Sprintf("Collection: %s (%d pages)", coll.Name, coll.PageCount)
		case "search":
			if len(subArgs) == 0 {
				s.writeError(w, http.StatusBadRequest, "usage: library search <query>")

				return
			}
			searchQuery := strings.Join(subArgs, " ")
			docCtx, searchErr := lib.GetDocsForQuery(r.Context(), searchQuery, 10000)
			if searchErr != nil {
				s.writeError(w, http.StatusInternalServerError, "search library: "+searchErr.Error())

				return
			}
			if docCtx == nil || len(docCtx.Pages) == 0 {
				message = "No matching documentation found"
			} else {
				// Count unique collections
				collectionSet := make(map[string]bool)
				for _, p := range docCtx.Pages {
					collectionSet[p.CollectionName] = true
				}
				message = fmt.Sprintf("Found %d page(s) from %d collection(s)", len(docCtx.Pages), len(collectionSet))
			}
		case "pull":
			if len(subArgs) == 0 {
				s.writeError(w, http.StatusBadRequest, "usage: library pull <source>")

				return
			}
			source := subArgs[0]
			opts := &library.PullOptions{}
			// Parse simple flags
			for i := 1; i < len(subArgs); i++ {
				if subArgs[i] == "--name" && i+1 < len(subArgs) {
					opts.Name = subArgs[i+1]
					i++
				} else if subArgs[i] == "--shared" {
					opts.Shared = true
				}
			}
			pullResult, pullErr := lib.Pull(r.Context(), source, opts)
			if pullErr != nil {
				s.writeError(w, http.StatusInternalServerError, "pull library: "+pullErr.Error())

				return
			}
			message = fmt.Sprintf("Pulled collection: %s (%d pages)", pullResult.Collection.Name, pullResult.Collection.PageCount)
		case "remove", "rm":
			if len(subArgs) == 0 {
				s.writeError(w, http.StatusBadRequest, "usage: library remove <name>")

				return
			}
			if removeErr := lib.Remove(r.Context(), subArgs[0], false); removeErr != nil {
				s.writeError(w, http.StatusInternalServerError, "remove collection: "+removeErr.Error())

				return
			}
			message = fmt.Sprintf("Collection '%s' removed", subArgs[0])
		case "stats":
			collections, listErr := lib.List(r.Context(), &library.ListOptions{})
			if listErr != nil {
				s.writeError(w, http.StatusInternalServerError, "list collections: "+listErr.Error())

				return
			}
			var totalPages int
			for _, c := range collections {
				totalPages += c.PageCount
			}
			message = fmt.Sprintf("%d collections, %d total pages", len(collections), totalPages)
		default:
			// Treat as collection name for show
			coll, showErr := lib.Show(r.Context(), subcommand)
			if showErr != nil {
				s.writeError(w, http.StatusInternalServerError, "show collection: "+showErr.Error())

				return
			}
			message = fmt.Sprintf("Collection: %s (%d pages)", coll.Name, coll.PageCount)
		}

	case "links":
		ws := cond.GetWorkspace()
		if ws == nil {
			s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

			return
		}
		linkMgr := storage.GetLinkManager(r.Context(), ws)
		if linkMgr == nil {
			s.writeError(w, http.StatusServiceUnavailable, "links system is not available")

			return
		}
		subcommand := "list"
		subArgs := req.Args
		if len(subArgs) > 0 {
			subcommand = subArgs[0]
			subArgs = subArgs[1:]
		}
		switch subcommand {
		case "list", "ls":
			linkIndex := linkMgr.GetIndex()
			var totalLinks int
			for _, forwardLinks := range linkIndex.Forward {
				totalLinks += len(forwardLinks)
			}
			message = fmt.Sprintf("%d links from %d sources", totalLinks, len(linkIndex.Forward))
		case "backlinks":
			if len(subArgs) == 0 {
				s.writeError(w, http.StatusBadRequest, "usage: links backlinks <entity-id>")

				return
			}
			incoming := linkMgr.GetIncoming(subArgs[0])
			message = fmt.Sprintf("%d backlinks to %s", len(incoming), subArgs[0])
		case "search":
			if len(subArgs) == 0 {
				s.writeError(w, http.StatusBadRequest, "usage: links search <query>")

				return
			}
			query := strings.Join(subArgs, " ")
			queryLower := strings.ToLower(query)
			names := linkMgr.GetNames()
			var matchCount int
			for name := range names.Specs {
				if strings.Contains(strings.ToLower(name), queryLower) {
					matchCount++
				}
			}
			for name := range names.Decisions {
				if strings.Contains(strings.ToLower(name), queryLower) {
					matchCount++
				}
			}
			message = fmt.Sprintf("Found %d matching entities", matchCount)
		case "stats":
			stats := linkMgr.GetStats()
			if stats == nil {
				s.writeError(w, http.StatusInternalServerError, "failed to get link stats")

				return
			}
			message = fmt.Sprintf("%d links, %d sources, %d targets", stats.TotalLinks, stats.TotalSources, stats.TotalTargets)
		case "rebuild":
			if rebuildErr := linkMgr.Rebuild(); rebuildErr != nil {
				s.writeError(w, http.StatusInternalServerError, "rebuild failed: "+rebuildErr.Error())

				return
			}
			stats := linkMgr.GetStats()
			message = fmt.Sprintf("Index rebuilt: %d links", stats.TotalLinks)
		default:
			// Treat as entity ID
			outgoing := linkMgr.GetOutgoing(subcommand)
			incoming := linkMgr.GetIncoming(subcommand)
			message = fmt.Sprintf("%d outgoing, %d incoming links for %s", len(outgoing), len(incoming), subcommand)
		}

	case "question":
		if len(req.Args) == 0 {
			s.writeError(w, http.StatusBadRequest, "question requires a message")

			return
		}
		question := strings.Join(req.Args, " ")
		err = cond.AskQuestion(opCtx, question)
		message = "Question sent to agent"

	case "answer":
		if len(req.Args) == 0 {
			s.writeError(w, http.StatusBadRequest, "answer requires a response")

			return
		}
		task := cond.GetActiveTask()
		if task == nil {
			s.writeError(w, http.StatusBadRequest, "no active task")

			return
		}
		ws := cond.GetWorkspace()
		// Clear pending question
		if clearErr := ws.ClearPendingQuestion(task.ID); clearErr != nil {
			slog.Warn("clear pending question", "error", clearErr)
		}
		// Save answer as note
		response := strings.Join(req.Args, " ")
		if noteErr := ws.AppendNote(task.ID, response, task.State); noteErr != nil {
			s.writeError(w, http.StatusInternalServerError, "save answer: "+noteErr.Error())

			return
		}
		message = "Answer saved"
		// Resume workflow based on state
		state := workflow.State(task.State)
		switch state {
		case workflow.StatePlanning:
			go func(ctx context.Context) { _ = cond.Plan(ctx) }(opCtx)
		case workflow.StateImplementing:
			go func(ctx context.Context) { _ = cond.Implement(ctx) }(opCtx)
		case workflow.StateReviewing:
			go func(ctx context.Context) { _ = cond.Review(ctx) }(opCtx)
		case workflow.StateIdle, workflow.StateDone, workflow.StateFailed,
			workflow.StateWaiting, workflow.StatePaused,
			workflow.StateCheckpointing, workflow.StateReverting, workflow.StateRestoring:
			// These states are not resumable - do nothing
		}

	case "delete":
		// Delete a queue task: delete <queue>/<task-id>
		if len(req.Args) == 0 {
			s.writeError(w, http.StatusBadRequest, "delete requires a task reference (e.g., quick-tasks/task-1)")

			return
		}
		queueID, taskID, parseErr := conductor.ParseQueueTaskRef(req.Args[0])
		if parseErr != nil {
			s.writeError(w, http.StatusBadRequest, parseErr.Error())

			return
		}
		ws := cond.GetWorkspace()
		queue, loadErr := storage.LoadTaskQueue(ws, queueID)
		if loadErr != nil {
			s.writeError(w, http.StatusNotFound, "queue not found: "+queueID)

			return
		}
		if !queue.RemoveTask(taskID) {
			s.writeError(w, http.StatusNotFound, fmt.Sprintf("task not found: %s/%s", queueID, taskID))

			return
		}
		if saveErr := queue.Save(); saveErr != nil {
			s.writeError(w, http.StatusInternalServerError, "save queue: "+saveErr.Error())

			return
		}
		// Delete notes file
		notesPath := ws.QueueNotePath(queueID, taskID)
		_ = ws.DeleteFile(notesPath)
		message = fmt.Sprintf("Deleted task %s from %s", taskID, queueID)

	case "export":
		// Export a queue task to markdown: export <queue>/<task-id>
		if len(req.Args) == 0 {
			s.writeError(w, http.StatusBadRequest, "export requires a task reference (e.g., quick-tasks/task-1)")

			return
		}
		queueID, taskID, parseErr := conductor.ParseQueueTaskRef(req.Args[0])
		if parseErr != nil {
			s.writeError(w, http.StatusBadRequest, parseErr.Error())

			return
		}
		markdown, exportErr := cond.ExportQueueTask(queueID, taskID)
		if exportErr != nil {
			s.writeError(w, http.StatusInternalServerError, "export task: "+exportErr.Error())

			return
		}
		// Return the full markdown content
		s.writeJSON(w, http.StatusOK, map[string]any{
			"success":  true,
			"message":  fmt.Sprintf("Exported %s/%s", queueID, taskID),
			"markdown": markdown,
		})

		return

	case "optimize":
		// AI optimize a queue task: optimize <queue>/<task-id>
		if len(req.Args) == 0 {
			s.writeError(w, http.StatusBadRequest, "optimize requires a task reference (e.g., quick-tasks/task-1)")

			return
		}
		queueID, taskID, parseErr := conductor.ParseQueueTaskRef(req.Args[0])
		if parseErr != nil {
			s.writeError(w, http.StatusBadRequest, parseErr.Error())

			return
		}
		optimized, optimizeErr := cond.OptimizeQueueTask(opCtx, queueID, taskID)
		if optimizeErr != nil {
			s.writeError(w, http.StatusInternalServerError, "optimize task: "+optimizeErr.Error())

			return
		}
		s.writeJSON(w, http.StatusOK, map[string]any{
			"success":           true,
			"message":           "Task optimized",
			"original_title":    optimized.OriginalTitle,
			"optimized_title":   optimized.OptimizedTitle,
			"added_labels":      optimized.AddedLabels,
			"improvement_notes": optimized.ImprovementNotes,
		})

		return

	case "submit":
		// Submit a queue task to provider: submit <queue>/<task-id> <provider>
		if len(req.Args) < 2 {
			s.writeError(w, http.StatusBadRequest, "submit requires: submit <queue>/<task-id> <provider>")

			return
		}
		queueID, taskID, parseErr := conductor.ParseQueueTaskRef(req.Args[0])
		if parseErr != nil {
			s.writeError(w, http.StatusBadRequest, parseErr.Error())

			return
		}
		providerName := req.Args[1]
		submitResult, submitErr := cond.SubmitQueueTask(opCtx, queueID, taskID, conductor.SubmitOptions{
			Provider: providerName,
			TaskIDs:  []string{taskID},
		})
		if submitErr != nil {
			s.writeError(w, http.StatusInternalServerError, "submit task: "+submitErr.Error())

			return
		}
		if len(submitResult.Tasks) == 0 {
			s.writeError(w, http.StatusInternalServerError, "no tasks submitted")

			return
		}
		submittedTask := submitResult.Tasks[0]
		s.writeJSON(w, http.StatusOK, map[string]any{
			"success":     true,
			"message":     "Submitted to " + providerName,
			"external_id": submittedTask.ExternalID,
			"url":         submittedTask.ExternalURL,
		})

		return

	case "sync":
		// Sync task from provider: sync <task-id>
		task := cond.GetActiveTask()
		if task == nil {
			s.writeError(w, http.StatusBadRequest, "no active task to sync")

			return
		}
		// For now, indicate that sync is available - full implementation would require
		// provider fetch and delta spec generation
		message = fmt.Sprintf("Sync requested for task %s. Use CLI for full provider sync.", task.ID)

	default:
		s.writeError(w, http.StatusBadRequest, "unknown command: "+req.Command)

		return
	}

	// Handle cancellation gracefully
	if errors.Is(err, context.Canceled) {
		state := ""
		if task := cond.GetActiveTask(); task != nil {
			state = task.State
		}
		s.writeJSON(w, http.StatusOK, commandResponse{
			Success: true,
			Message: req.Command + " cancelled",
			State:   state,
		})

		return
	}

	if err != nil {
		slog.Error("command error", "command", req.Command, "error", err)
		s.writeJSON(w, http.StatusInternalServerError, commandResponse{
			Success: false,
			Message: err.Error(),
		})

		return
	}

	// Get current state
	state := ""
	taskID := ""
	title := ""
	if task := cond.GetActiveTask(); task != nil {
		state = task.State
		taskID = task.ID
		if work := cond.GetTaskWork(); work != nil {
			title = work.Metadata.Title
		}
	}

	s.writeJSON(w, http.StatusOK, commandResponse{
		Success: true,
		Message: message,
		State:   state,
	})

	// Also publish state update
	if taskID != "" {
		s.config.EventBus.PublishRaw(eventbus.Event{
			Type: events.TypeStateChanged,
			Data: map[string]any{
				"task_id": taskID,
				"state":   state,
				"title":   title,
			},
		})
	}
}
