package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-mehrhof/internal/template"
	"github.com/valksor/go-mehrhof/internal/workflow"
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

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		s.writeError(w, http.StatusServiceUnavailable, "workspace not initialized")

		return
	}

	// Load task work
	work, err := ws.LoadWork(req.TaskID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "task not found: "+err.Error())

		return
	}

	taskDir := ws.WorkPath(req.TaskID)
	if taskDir == "" {
		s.writeError(w, http.StatusInternalServerError, "task directory is empty")

		return
	}

	// Load source content to reconstruct current work unit
	var sourcePath string
	if len(work.Source.Files) > 0 {
		sourcePath = work.Source.Files[0]
		if filepath.IsAbs(sourcePath) {
			s.writeError(w, http.StatusInternalServerError, "source file path is absolute, expected relative")

			return
		}
		sourcePath = filepath.Join(taskDir, sourcePath)
	} else {
		sourcePath = filepath.Join(taskDir, "source", work.Source.Type+".txt")
	}

	sourceContent, err := os.ReadFile(sourcePath)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to read source file: "+err.Error())

		return
	}

	// Reconstruct work unit from work metadata
	workUnit := &provider.WorkUnit{
		ID:          work.Metadata.ID,
		ExternalID:  work.Source.Ref,
		Provider:    work.Source.Type,
		Title:       work.Metadata.Title,
		Description: string(sourceContent),
	}

	// Fetch updated version from provider
	updated, err := s.fetchUpdatedFromProvider(r.Context(), workUnit)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to fetch from provider: "+err.Error())

		return
	}

	// Detect changes
	changes := provider.DetectChanges(workUnit, updated)

	if !changes.HasChanges {
		s.writeJSON(w, http.StatusOK, syncResponse{
			Success:    true,
			HasChanges: false,
			Message:    "no changes detected",
		})

		return
	}

	// Generate delta specification
	gen := workflow.NewGenerator(taskDir)

	// Backup original source file
	_ = gen.BackupSourceFile(sourcePath)

	// Write diff file
	_ = gen.WriteDiffFile(changes)

	// Extract content for comparison
	oldContent := extractWorkUnitContent(workUnit)
	newContent := extractWorkUnitContent(updated)

	// Generate delta specification with timeout
	genCtx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()

	specPath, err := gen.GenerateDeltaSpecification(genCtx, changes, oldContent, newContent)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to generate delta specification: "+err.Error())

		return
	}

	s.writeJSON(w, http.StatusOK, syncResponse{
		Success:        true,
		HasChanges:     true,
		ChangesSummary: changes.Summary(),
		SpecGenerated:  specPath,
		Message:        "changes detected and delta specification generated",
	})
}

// fetchUpdatedFromProvider fetches the updated version of the task from the provider.
func (s *Server) fetchUpdatedFromProvider(ctx context.Context, old *provider.WorkUnit) (*provider.WorkUnit, error) {
	registry := s.config.Conductor.GetProviderRegistry()
	providerInstance, id, err := registry.Resolve(ctx, old.ExternalID, provider.NewConfig(), provider.ResolveOptions{})
	if err != nil {
		return nil, fmt.Errorf("resolve provider: %w", err)
	}

	reader, ok := providerInstance.(provider.Reader)
	if !ok {
		return nil, fmt.Errorf("provider %s does not support reading", old.Provider)
	}

	updated, err := reader.Fetch(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetch from provider: %w", err)
	}

	return updated, nil
}

// extractWorkUnitContent extracts the main content from a work unit.
func extractWorkUnitContent(wu *provider.WorkUnit) string {
	var content string

	if wu.Title != "" {
		content += fmt.Sprintf("# %s\n\n", wu.Title)
	}

	if wu.Description != "" {
		content += wu.Description + "\n"
	}

	if len(wu.Comments) > 0 {
		content += "\n## Comments\n\n"
		var contentSb182 strings.Builder
		for _, comment := range wu.Comments {
			author := provider.ResolveAuthor(comment)
			if author == "" {
				author = comment.Author.ID
			}
			contentSb182.WriteString("### " + author + "\n\n" + comment.Body + "\n\n")
		}
		content += contentSb182.String()
	}

	return content
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
