package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/valksor/go-mehrhof/internal/server/api"
	"github.com/valksor/go-mehrhof/internal/storage"
)

// handleGetI18nOverrides returns merged i18n overrides (global + project).
// GET /api/v1/i18n/overrides.
func (s *Server) handleGetI18nOverrides(w http.ResponseWriter, r *http.Request) {
	projectName := s.getCurrentProjectName(r.Context())

	overrides, err := storage.LoadMergedI18nOverrides(projectName)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, err.Error())

		return
	}

	api.WriteSuccess(w, overrides)
}

// handleGetI18nOverridesGlobal returns global i18n overrides only.
// GET /api/v1/i18n/overrides/global.
func (s *Server) handleGetI18nOverridesGlobal(w http.ResponseWriter, _ *http.Request) {
	overrides, err := storage.LoadI18nOverrides("")
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, err.Error())

		return
	}

	api.WriteSuccess(w, overrides)
}

// handleGetI18nOverridesProject returns project-specific i18n overrides only.
// GET /api/v1/i18n/overrides/project.
func (s *Server) handleGetI18nOverridesProject(w http.ResponseWriter, r *http.Request) {
	projectName := s.getCurrentProjectName(r.Context())
	if projectName == "" {
		api.WriteSuccess(w, storage.NewI18nOverrides())

		return
	}

	overrides, err := storage.LoadI18nOverrides(projectName)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, err.Error())

		return
	}

	api.WriteSuccess(w, overrides)
}

// handleSaveI18nOverridesGlobal saves global i18n overrides.
// POST /api/v1/i18n/overrides/global.
func (s *Server) handleSaveI18nOverridesGlobal(w http.ResponseWriter, r *http.Request) {
	var overrides storage.I18nOverrides
	if err := json.NewDecoder(r.Body).Decode(&overrides); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid JSON: "+err.Error())

		return
	}

	if err := validateI18nOverrides(&overrides); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "validation failed: "+err.Error())

		return
	}

	if err := storage.SaveI18nOverrides("", &overrides); err != nil {
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, err.Error())

		return
	}

	api.WriteSuccessMessage(w, "global overrides saved")
}

// handleSaveI18nOverridesProject saves project-specific i18n overrides.
// POST /api/v1/i18n/overrides/project.
func (s *Server) handleSaveI18nOverridesProject(w http.ResponseWriter, r *http.Request) {
	projectName := s.getCurrentProjectName(r.Context())
	if projectName == "" {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "no project context")

		return
	}

	var overrides storage.I18nOverrides
	if err := json.NewDecoder(r.Body).Decode(&overrides); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid JSON: "+err.Error())

		return
	}

	if err := validateI18nOverrides(&overrides); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "validation failed: "+err.Error())

		return
	}

	if err := storage.SaveI18nOverrides(projectName, &overrides); err != nil {
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, err.Error())

		return
	}

	api.WriteSuccessMessage(w, "project overrides saved")
}

// handleGetI18nKeys returns all available translation keys for the override editor.
// GET /api/v1/i18n/keys.
func (s *Server) handleGetI18nKeys(w http.ResponseWriter, _ *http.Request) {
	// Return a list of all translation keys for the key override editor.
	// These are extracted from the bundled translation files.
	// For now, we return a static list of commonly overridden keys.
	// In a full implementation, this could parse the actual JSON files.
	keys := []string{
		// Navigation
		"nav.dashboard",
		"nav.projects",
		"nav.work",
		"nav.project",
		"nav.quick",
		"nav.history",
		"nav.advanced",
		"nav.find",
		"nav.review",
		"nav.commit",
		"nav.simplify",
		"nav.chat",
		"nav.library",
		"nav.links",
		"nav.tools",
		"nav.admin",
		"nav.settings",
		"nav.license",
		// Status
		"status.connected",
		"status.reconnecting",
		"status.loading",
		// Actions
		"actions.save",
		"actions.saveChanges",
		"actions.cancel",
		"actions.confirm",
		"actions.delete",
		"actions.edit",
		// Workflow states
		"workflow:states.idle",
		"workflow:states.planning",
		"workflow:states.implementing",
		"workflow:states.reviewing",
		"workflow:states.waiting",
		"workflow:states.done",
		"workflow:states.failed",
		// Task
		"workflow:task.title",
		"workflow:task.noActiveTask",
		"workflow:task.activeTask",
		"workflow:task.specifications",
		"workflow:task.reviews",
		"workflow:task.notes",
		"workflow:task.costs",
		// Settings
		"settings:title",
		"settings:tabs.work",
		"settings:tabs.advanced",
		"settings:sections.git.title",
		"settings:sections.agent.title",
		"settings:sections.workflow.title",
		"settings:sections.appearance.title",
	}

	api.WriteSuccess(w, map[string][]string{"keys": keys})
}

// getCurrentProjectName returns the current project name for i18n overrides.
// Returns empty string if no project context is available.
func (s *Server) getCurrentProjectName(ctx context.Context) string {
	if s.config.Conductor == nil {
		return ""
	}

	ws := s.config.Conductor.GetWorkspace()
	if ws == nil {
		return ""
	}

	// Generate project ID from workspace root
	projectID, err := storage.GenerateProjectID(ctx, ws.Root())
	if err != nil {
		return ""
	}

	return projectID
}

// validateI18nOverrides validates the i18n overrides before saving.
// Returns an error if validation fails.
func validateI18nOverrides(o *storage.I18nOverrides) error {
	// Validate terminology entries
	for find, replace := range o.Terminology {
		if strings.TrimSpace(find) == "" {
			return errors.New("terminology 'find' value cannot be empty")
		}

		if strings.TrimSpace(replace) == "" {
			return errors.New("terminology 'replace' value cannot be empty")
		}
	}

	// Validate key overrides
	for lang, keys := range o.Keys {
		if !isValidLanguageCode(lang) {
			return fmt.Errorf("invalid language code: %s (must be 2-3 lowercase letters)", lang)
		}

		for key := range keys {
			if strings.TrimSpace(key) == "" {
				return errors.New("translation key cannot be empty")
			}
		}
	}

	return nil
}

// isValidLanguageCode checks if a language code is valid (2-3 lowercase letters).
func isValidLanguageCode(code string) bool {
	if len(code) < 2 || len(code) > 3 {
		return false
	}

	for _, r := range code {
		if r < 'a' || r > 'z' {
			return false
		}
	}

	return true
}
