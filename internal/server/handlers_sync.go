package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/valksor/go-mehrhof/internal/template"
)

// handleWorkflowSync syncs a task from the provider and generates delta spec if changed.
func (s *Server) handleWorkflowSync(w http.ResponseWriter, r *http.Request) {
	if s.isViewer(r) {
		s.writeError(w, http.StatusForbidden, "viewers cannot modify workflow")

		return
	}

	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	var req syncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

		return
	}

	if req.TaskID == "" {
		s.writeError(w, http.StatusBadRequest, "task_id is required")

		return
	}

	result, err := s.config.Conductor.SyncTask(r.Context(), req.TaskID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "sync failed: "+err.Error())

		return
	}

	if !result.HasChanges {
		s.writeJSON(w, http.StatusOK, syncResponse{
			Success:    true,
			HasChanges: false,
			Message:    "no changes detected",
		})

		return
	}

	s.writeJSON(w, http.StatusOK, syncResponse{
		Success:              true,
		HasChanges:           true,
		ChangesSummary:       result.ChangesSummary,
		SpecGenerated:        result.SpecGenerated,
		SourceUpdated:        result.SourceUpdated,
		PreviousSnapshotPath: result.PreviousSnapshotPath,
		DiffPath:             result.DiffPath,
		Warnings:             result.Warnings,
		Message:              "changes detected and delta specification generated",
	})
}

// handleWorkflowSimplify auto-simplifies content based on workflow state.
func (s *Server) handleWorkflowSimplify(w http.ResponseWriter, r *http.Request) {
	if s.isViewer(r) {
		s.writeError(w, http.StatusForbidden, "viewers cannot modify workflow")

		return
	}

	if s.config.Conductor == nil {
		s.writeError(w, http.StatusServiceUnavailable, "conductor not initialized")

		return
	}

	activeTask := s.config.Conductor.GetActiveTask()
	if activeTask == nil {
		s.writeError(w, http.StatusBadRequest, "no active task")

		return
	}

	var req simplifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Default to checkpoint enabled
		req = simplifyRequest{}
	}

	// Call simplify
	err := s.config.Conductor.Simplify(r.Context(), req.Agent, !req.NoCheckpoint)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "simplification failed: "+err.Error())

		return
	}

	// Determine what was simplified based on state
	ws := s.config.Conductor.GetWorkspace()
	specs, _ := ws.ListSpecifications(activeTask.ID)
	hasSpecs := len(specs) > 0

	simplified := "task_input"
	if hasSpecs {
		simplified = "specifications"
	}

	s.writeJSON(w, http.StatusOK, simplifyResponse{
		Success:    true,
		Simplified: simplified,
		Message:    "simplification complete",
	})
}

// handleListTemplates returns a list of available templates.
func (s *Server) handleListTemplates(w http.ResponseWriter, r *http.Request) {
	names := template.BuiltInTemplates()

	var templates []templateInfo
	for _, name := range names {
		tpl, err := template.LoadBuiltIn(name)
		if err != nil {
			continue
		}
		templates = append(templates, templateInfo{
			Name:        name,
			Description: tpl.GetDescription(),
		})
	}

	s.writeJSON(w, http.StatusOK, templatesListResponse{
		Templates: templates,
		Count:     len(templates),
	})
}

// handleGetTemplate returns details for a specific template.
func (s *Server) handleGetTemplate(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		s.writeError(w, http.StatusBadRequest, "template name is required")

		return
	}

	tpl, err := template.LoadBuiltIn(name)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "template not found: "+err.Error())

		return
	}

	// Convert AgentSteps to map[string]any
	agentSteps := make(map[string]any)
	for step, cfg := range tpl.AgentSteps {
		agentSteps[step] = cfg
	}

	// Convert Workflow to map[string]any
	workflowCfg := make(map[string]any)
	for k, v := range tpl.Workflow {
		workflowCfg[k] = v
	}

	s.writeJSON(w, http.StatusOK, templateShowResponse{
		Name:        tpl.Name,
		Description: tpl.Description,
		Frontmatter: tpl.Frontmatter,
		Agent:       tpl.Agent,
		AgentSteps:  agentSteps,
		Git:         tpl.Git,
		Workflow:    workflowCfg,
	})
}

// handleApplyTemplate applies a template to a file.
func (s *Server) handleApplyTemplate(w http.ResponseWriter, r *http.Request) {
	var req templateApplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())

		return
	}

	if req.TemplateName == "" {
		s.writeError(w, http.StatusBadRequest, "template_name is required")

		return
	}

	if req.FilePath == "" {
		s.writeError(w, http.StatusBadRequest, "file_path is required")

		return
	}

	tpl, err := template.LoadBuiltIn(req.TemplateName)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "template not found: "+err.Error())

		return
	}

	// Read existing content
	var content string
	data, err := os.ReadFile(req.FilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			s.writeError(w, http.StatusInternalServerError, "failed to read file: "+err.Error())

			return
		}
		content = "# Task Title\n\nDescribe your task here.\n"
	} else {
		content = string(data)
	}

	// Apply template
	newContent := tpl.ApplyToContent(content)

	// Write back
	if err := os.WriteFile(req.FilePath, []byte(newContent), 0o644); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to write file: "+err.Error())

		return
	}

	s.writeJSON(w, http.StatusOK, templateApplyResponse{
		Success:     true,
		Frontmatter: tpl.Frontmatter,
		Message:     fmt.Sprintf("applied template '%s' to %s", tpl.Name, req.FilePath),
	})
}
