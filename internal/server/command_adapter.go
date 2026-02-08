package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/valksor/go-mehrhof/internal/conductor/commands"
)

// CommandRoute maps an HTTP route to a router command invocation.
type CommandRoute struct {
	Command           string
	ParseFn           func(r *http.Request) (commands.Invocation, error)
	InjectFn          func(r *http.Request, inv *commands.Invocation) // Optional: inject server context after parsing
	UnwrapData        bool                                            // When true, write result.Data directly as response
	AllowNilConductor bool                                            // When true, skip conductor nil check (for global-mode endpoints)
}

func (s *Server) handleViaRouter(route CommandRoute) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !route.AllowNilConductor && s.config.Conductor == nil {
			s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

			return
		}

		info, ok := commands.GetCommandInfo(route.Command)
		if !ok {
			s.writeError(w, http.StatusInternalServerError, "unknown router command: "+route.Command)

			return
		}
		inv := commands.Invocation{Source: commands.SourceAPI}
		if route.ParseFn != nil {
			parsed, err := route.ParseFn(r)
			if err != nil {
				s.writeError(w, http.StatusBadRequest, err.Error())

				return
			}
			inv = parsed
		}
		if inv.Source == "" {
			inv.Source = commands.SourceAPI
		}
		if route.InjectFn != nil {
			route.InjectFn(r, &inv)
		}

		// 1. Execute command (transitions state immediately, may set Executor for long-running ops)
		result, err := commands.Execute(r.Context(), s.config.Conductor, route.Command, inv)
		if err != nil {
			s.mapErrorToHTTP(w, err)

			return
		}

		// 2. Publish INITIAL state change BEFORE executor blocks
		//    State is already transitioned (e.g., "implementing") after cond.Implement() in Execute()
		if info.MutatesState && result != nil && result.TaskID != "" && result.State != "" {
			s.publishStateChangeEvent(r.Context())
		}

		// 3. Run executor (blocks for long-running commands like implement, plan, review)
		if result != nil && result.Executor != nil {
			if execErr := result.Executor(r.Context()); execErr != nil {
				result = commands.ClassifyError(result, execErr)
				// Fall through to normal response writing (handles waiting, paused, errors)
			}
		}

		// 4. Handle waiting/redirect/response
		if result != nil && result.Type == commands.ResultWaiting {
			commands.EnrichWaitingResult(result, s.config.Conductor)
		}

		if route.Command == "start" && strings.Contains(r.Header.Get("Accept"), "text/html") &&
			(result == nil || (result.Type != commands.ResultError && result.Type != commands.ResultConflict)) {
			http.Redirect(w, r, "/", http.StatusSeeOther)

			return
		}

		if route.UnwrapData {
			s.writeCommandDataOrResult(w, result)
		} else {
			s.writeCommandResult(w, result)
		}

		// 5. Publish FINAL state change after executor completes (only for commands with executors)
		if info.MutatesState && result != nil && result.Executor != nil {
			s.publishStateChangeEvent(r.Context())
		}
	}
}

func (s *Server) writeCommandResult(w http.ResponseWriter, result *commands.Result) {
	if result == nil {
		s.writeJSON(w, http.StatusOK, map[string]any{"success": true})

		return
	}

	response := map[string]any{
		"success": true,
		"message": result.Message,
	}
	if result.State != "" {
		response["state"] = result.State
	}
	if result.TaskID != "" {
		response["task_id"] = result.TaskID
	}

	status := http.StatusOK

	switch result.Type {
	case commands.ResultMessage,
		commands.ResultStatus,
		commands.ResultList,
		commands.ResultCost,
		commands.ResultSpecifications,
		commands.ResultBudget,
		commands.ResultQuestion,
		commands.ResultChat,
		commands.ResultHelp,
		commands.ResultExit:
		if result.Data != nil {
			if payload, ok := result.Data.(map[string]any); ok {
				for k, v := range payload {
					if _, exists := response[k]; !exists {
						response[k] = v
					}
				}
			} else {
				response["data"] = result.Data
			}
		}
	case commands.ResultWaiting:
		response["status"] = "waiting"
		if waiting, ok := result.Data.(commands.WaitingData); ok {
			response["question"] = waiting.Question
			response["options"] = waiting.Options
			if waiting.Phase != "" {
				response["phase"] = waiting.Phase
			}
		}
	case commands.ResultPaused:
		response["status"] = "paused"
	case commands.ResultStopped:
		response["success"] = false
		response["status"] = "stopped"
	case commands.ResultConflict:
		response["success"] = false
		response["status"] = "conflict"
		status = http.StatusConflict
		if result.Data != nil {
			if payload, ok := result.Data.(map[string]any); ok {
				for k, v := range payload {
					response[k] = v
				}
			} else {
				response["conflict"] = result.Data
			}
		}
	case commands.ResultError:
		response["success"] = false
		status = http.StatusInternalServerError
	}

	s.writeJSON(w, status, response)
}

// writeCommandDataOrResult writes the result's Data directly as the response body,
// otherwise falls back to the standard command response envelope.
// Use this when the command returns a fully-formed API response in Data.
// Handles both map[string]any and struct types (e.g., *storage.WorkspaceConfig).
func (s *Server) writeCommandDataOrResult(w http.ResponseWriter, result *commands.Result) {
	if result != nil && result.Data != nil {
		status := http.StatusOK

		switch result.Type { //nolint:exhaustive // Only error/conflict need non-200; all others are OK
		case commands.ResultConflict:
			status = http.StatusConflict
		case commands.ResultError:
			status = http.StatusInternalServerError
		}

		s.writeJSON(w, status, result.Data)

		return
	}

	s.writeCommandResult(w, result)
}

func (s *Server) mapErrorToHTTP(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, commands.ErrNoActiveTask):
		s.writeError(w, http.StatusBadRequest, "no active task")
	case errors.Is(err, commands.ErrUnknownCommand):
		s.writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, commands.ErrBadRequest):
		s.writeError(w, http.StatusBadRequest, err.Error())
	default:
		s.writeError(w, http.StatusInternalServerError, err.Error())
	}
}

func (s *Server) parseStartInvocation(r *http.Request) (commands.Invocation, error) {
	inv := commands.Invocation{Source: commands.SourceAPI}
	contentType := r.Header.Get("Content-Type")

	var (
		ref      string
		template string
		noBranch bool
	)

	switch {
	case strings.HasPrefix(contentType, "multipart/form-data"):
		taskRef, err := s.handleFileUpload(r)
		if err != nil {
			return inv, err
		}
		ref = taskRef
		template = r.FormValue("template")
		noBranch = r.FormValue("no_branch") == "true"
	case strings.HasPrefix(contentType, "application/x-www-form-urlencoded"):
		if err := r.ParseForm(); err != nil {
			return inv, errors.New("invalid form data: " + err.Error())
		}
		content := r.FormValue("content")
		refVal := r.FormValue("ref")
		template = r.FormValue("template")
		noBranch = r.FormValue("no_branch") == "true"
		switch {
		case content != "":
			taskRef, err := s.saveContentToFile(content)
			if err != nil {
				return inv, errors.New("failed to save content: " + err.Error())
			}
			ref = taskRef
		case refVal != "":
			ref = refVal
		default:
			return inv, errors.New("ref or content is required")
		}
	default:
		var req struct {
			Ref      string `json:"ref"`
			Content  string `json:"content"`
			Template string `json:"template"`
			NoBranch bool   `json:"no_branch"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return inv, errors.New("invalid request body: " + err.Error())
		}

		template = req.Template
		noBranch = req.NoBranch
		switch {
		case req.Content != "":
			taskRef, err := s.saveContentToFile(req.Content)
			if err != nil {
				return inv, errors.New("failed to save content: " + err.Error())
			}
			ref = taskRef
		case req.Ref != "":
			ref = req.Ref
		default:
			return inv, errors.New("ref or content is required")
		}
	}

	inv.Args = []string{ref}
	inv.Options = map[string]any{
		"ref":       ref,
		"template":  template,
		"no_branch": noBranch,
	}

	return inv, nil
}

func parseImplementInvocation(r *http.Request) (commands.Invocation, error) {
	options := map[string]any{}
	component := r.URL.Query().Get("component")
	parallel := r.URL.Query().Get("parallel")
	if component != "" {
		options["component"] = component
	}
	if parallel != "" {
		options["parallel"] = parallel
	}

	return commands.Invocation{
		Source:  commands.SourceAPI,
		Options: options,
	}, nil
}

func parseImplementReviewInvocation(r *http.Request) (commands.Invocation, error) {
	nStr := r.PathValue("n")
	if nStr == "" {
		return commands.Invocation{}, errors.New("review number is required")
	}
	reviewNumber, err := strconv.Atoi(nStr)
	if err != nil || reviewNumber <= 0 {
		return commands.Invocation{}, errors.New("invalid review number: must be a positive integer")
	}

	return commands.Invocation{
		Source: commands.SourceAPI,
		Args:   []string{"review", nStr},
	}, nil
}

func parseFinishInvocation(r *http.Request) (commands.Invocation, error) {
	inv := commands.Invocation{Source: commands.SourceAPI}

	var req finishRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		return inv, errors.New("invalid request body: " + err.Error())
	}

	inv.Options = map[string]any{
		"squash_merge":  req.SquashMerge,
		"delete_branch": req.DeleteBranch,
		"target_branch": req.TargetBranch,
		"push_after":    req.PushAfter,
		"force_merge":   req.ForceMerge,
		"draft_pr":      req.DraftPR,
		"pr_title":      req.PRTitle,
		"pr_body":       req.PRBody,
		"finish_action": req.FinishAction,
	}

	return inv, nil
}

func parseContinueInvocation(r *http.Request) (commands.Invocation, error) {
	inv := commands.Invocation{Source: commands.SourceAPI}
	if r.Body == nil {
		return inv, nil
	}

	var req struct {
		Auto bool `json:"auto"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if errors.Is(err, io.EOF) {
			return inv, nil
		}

		return inv, errors.New("invalid request body: " + err.Error())
	}

	inv.Options = map[string]any{
		"auto": req.Auto,
	}

	return inv, nil
}

func parseAutoInvocation(r *http.Request) (commands.Invocation, error) {
	inv := commands.Invocation{Source: commands.SourceAPI}
	if r.Body == nil {
		return inv, errors.New("ref is required")
	}

	var req struct {
		Ref           string `json:"ref"`
		MaxRetries    int    `json:"max_retries"`
		NoPush        bool   `json:"no_push"`
		NoDelete      bool   `json:"no_delete"`
		NoSquash      bool   `json:"no_squash"`
		TargetBranch  string `json:"target_branch"`
		QualityTarget string `json:"quality_target"`
		NoQuality     bool   `json:"no_quality"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if errors.Is(err, io.EOF) {
			return inv, errors.New("ref is required")
		}

		return inv, errors.New("invalid request body: " + err.Error())
	}

	req.Ref = strings.TrimSpace(req.Ref)
	if req.Ref == "" {
		return inv, errors.New("ref is required")
	}

	if req.MaxRetries == 0 {
		req.MaxRetries = 3
	}
	if strings.TrimSpace(req.QualityTarget) == "" {
		req.QualityTarget = "quality"
	}
	if req.NoQuality {
		req.MaxRetries = 0
	}

	inv.Args = []string{req.Ref}
	inv.Options = map[string]any{
		"ref":            req.Ref,
		"max_retries":    req.MaxRetries,
		"no_push":        req.NoPush,
		"no_delete":      req.NoDelete,
		"no_squash":      req.NoSquash,
		"target_branch":  req.TargetBranch,
		"quality_target": req.QualityTarget,
		"no_quality":     req.NoQuality,
	}

	return inv, nil
}

func parseAnswerInvocation(r *http.Request) (commands.Invocation, error) {
	inv := commands.Invocation{Source: commands.SourceAPI}

	var answer string
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
		if err := r.ParseForm(); err != nil {
			return inv, errors.New("invalid form data: " + err.Error())
		}
		answer = r.FormValue("answer")
	} else {
		var req struct {
			Answer string `json:"answer"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return inv, errors.New("invalid request body: " + err.Error())
		}
		answer = req.Answer
	}
	if strings.TrimSpace(answer) == "" {
		return inv, errors.New("answer is required")
	}

	inv.Args = []string{answer}

	return inv, nil
}

func parseNoteInvocation(r *http.Request) (commands.Invocation, error) {
	inv := commands.Invocation{Source: commands.SourceAPI}

	taskID := strings.TrimSpace(r.PathValue("id"))
	if taskID == "" {
		return inv, errors.New("task ID is required")
	}

	var req struct {
		Note string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return inv, errors.New("invalid request body: " + err.Error())
	}

	note := strings.TrimSpace(req.Note)
	if note == "" {
		return inv, errors.New("note is required")
	}

	inv.Args = []string{note}
	inv.Options = map[string]any{
		"task_id": taskID,
	}

	return inv, nil
}

func parseNotesInvocation(r *http.Request) (commands.Invocation, error) {
	taskID := strings.TrimSpace(r.PathValue("id"))
	if taskID == "" {
		return commands.Invocation{}, errors.New("task ID is required")
	}

	return commands.Invocation{
		Source: commands.SourceAPI,
		Options: map[string]any{
			"task_id": taskID,
		},
	}, nil
}

func parseTaskIDInvocation(r *http.Request) (commands.Invocation, error) {
	taskID := strings.TrimSpace(r.PathValue("id"))
	if taskID == "" {
		return commands.Invocation{}, errors.New("task ID is required")
	}

	return commands.Invocation{
		Source: commands.SourceAPI,
		Options: map[string]any{
			"task_id": taskID,
		},
	}, nil
}

func parseWorkByIDInvocation(r *http.Request) (commands.Invocation, error) {
	taskID := strings.TrimSpace(r.PathValue("id"))
	if taskID == "" {
		return commands.Invocation{}, errors.New("task ID is required")
	}

	return commands.Invocation{
		Source: commands.SourceAPI,
		Args:   []string{taskID},
		Options: map[string]any{
			"id": taskID,
		},
	}, nil
}

func parseAggregateCostsInvocation(_ *http.Request) (commands.Invocation, error) {
	return commands.Invocation{
		Source: commands.SourceAPI,
		Options: map[string]any{
			"aggregate": true,
		},
	}, nil
}

func parseSpecificationInvocation(r *http.Request) (commands.Invocation, error) {
	inv := commands.Invocation{
		Source: commands.SourceAPI,
	}

	number := strings.TrimSpace(r.URL.Query().Get("number"))
	if number != "" {
		n, err := strconv.Atoi(number)
		if err != nil || n <= 0 {
			return commands.Invocation{}, errors.New("invalid number: must be a positive integer")
		}
		inv.Args = []string{number}
	}

	return inv, nil
}
