// Package handlers provides HTTP handler utilities and shared functionality.
// This package bridges the existing server handlers with the new views and api packages.
package handlers

import (
	"net/http"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/server/api"
	"github.com/valksor/go-mehrhof/internal/server/views"
	"github.com/valksor/go-mehrhof/internal/storage"
)

// Context provides common dependencies for handlers.
type Context struct {
	Conductor *conductor.Conductor
	Workspace *storage.Workspace
	Renderer  *views.Renderer

	// Page configuration
	Mode             string
	IsGlobalMode     bool
	AuthEnabled      bool
	CanSwitchProject bool
	CurrentUser      string
	IsViewer         bool
}

// PageData creates a PageData struct from the handler context.
func (c *Context) PageData() views.PageData {
	return views.ComputePageData(
		c.Mode,
		c.IsGlobalMode,
		c.AuthEnabled,
		c.CanSwitchProject,
		c.IsViewer,
		c.CurrentUser,
	)
}

// DashboardData computes and returns the full dashboard data.
func (c *Context) DashboardData() views.DashboardData {
	return views.ComputeDashboard(c.Conductor, c.Workspace, c.PageData())
}

// RenderDashboard renders the dashboard page.
func (c *Context) RenderDashboard(w http.ResponseWriter) error {
	return c.Renderer.RenderDashboard(w, c.DashboardData())
}

// WriteSuccess writes a successful API response.
func WriteSuccess(w http.ResponseWriter, data interface{}) {
	api.WriteSuccess(w, data)
}

// WriteError writes an error API response.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	api.WriteError(w, status, code, message)
}

// WriteBadRequest writes a 400 Bad Request error.
func WriteBadRequest(w http.ResponseWriter, message string) {
	api.WriteBadRequest(w, message)
}

// WriteNotFound writes a 404 Not Found error.
func WriteNotFound(w http.ResponseWriter, message string) {
	api.WriteNotFound(w, message)
}

// WriteUnauthorized writes a 401 Unauthorized error.
func WriteUnauthorized(w http.ResponseWriter) {
	api.WriteUnauthorized(w)
}

// WriteNoActiveTask writes a "no active task" error.
func WriteNoActiveTask(w http.ResponseWriter) {
	api.WriteNoActiveTask(w)
}

// WriteInvalidState writes an "invalid state" error.
func WriteInvalidState(w http.ResponseWriter, current, required string) {
	api.WriteInvalidState(w, current, required)
}

// WriteInternal writes a 500 Internal Server Error.
func WriteInternal(w http.ResponseWriter, err error) {
	api.WriteInternal(w, err)
}

// RequireConductor returns an error response if conductor is nil.
func RequireConductor(w http.ResponseWriter, c *conductor.Conductor) bool {
	if c == nil {
		api.WriteNotConfigured(w, "conductor")

		return false
	}

	return true
}

// RequireActiveTask returns an error response if there's no active task.
func RequireActiveTask(w http.ResponseWriter, c *conductor.Conductor) bool {
	if c == nil || c.GetActiveTask() == nil {
		api.WriteNoActiveTask(w)

		return false
	}

	return true
}

// RequireWorkspace returns an error response if workspace is nil.
func RequireWorkspace(w http.ResponseWriter, ws *storage.Workspace) bool {
	if ws == nil {
		api.WriteNotConfigured(w, "workspace")

		return false
	}

	return true
}
