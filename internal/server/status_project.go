package server

import (
	"path/filepath"

	"github.com/valksor/go-mehrhof/internal/storage"
)

type statusProjectInfo struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name"`
	Path      string `json:"path"`
	RemoteURL string `json:"remote_url,omitempty"`
}

func (s *Server) currentProjectStatusInfo() *statusProjectInfo {
	if s.config.Mode != ModeProject || s.config.WorkspaceRoot == "" {
		return nil
	}

	workspacePath := filepath.Clean(s.config.WorkspaceRoot)
	project := &statusProjectInfo{
		Name: filepath.Base(workspacePath),
		Path: workspacePath,
	}

	registry, err := storage.LoadRegistry()
	if err != nil {
		return project
	}

	for _, registered := range registry.List() {
		if filepath.Clean(registered.Path) != workspacePath {
			continue
		}

		project.ID = registered.ID
		if registered.Name != "" {
			project.Name = registered.Name
		}
		project.RemoteURL = storage.SanitizeRemoteURL(registered.RemoteURL)

		break
	}

	return project
}
