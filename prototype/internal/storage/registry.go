package storage

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// RegistryFile is the filename for the project registry.
	RegistryFile = "projects.yaml"
	// RegistryVersion is the current version of the registry format.
	RegistryVersion = "1"
)

// ProjectRegistry maps project IDs to their metadata for remote access.
// Stored at ~/.valksor/mehrhof/projects.yaml.
type ProjectRegistry struct {
	Version  string                     `yaml:"version"`
	Projects map[string]ProjectMetadata `yaml:"projects"` // key = project ID

	mu   sync.RWMutex `yaml:"-"`
	path string       `yaml:"-"` // path to registry file
}

// ProjectMetadata holds information about a registered project.
type ProjectMetadata struct {
	ID           string    `yaml:"id"`
	Path         string    `yaml:"path"`                  // Absolute filesystem path to repo
	RemoteURL    string    `yaml:"remote_url,omitempty"`  // Git remote URL if available
	Name         string    `yaml:"name"`                  // Human-readable name (repo name or dir name)
	RegisteredAt time.Time `yaml:"registered_at"`         // When project was registered
	LastAccess   time.Time `yaml:"last_access,omitempty"` // Last time project was accessed
}

// LoadRegistry loads the project registry from disk.
// If the registry file doesn't exist, returns an empty registry.
func LoadRegistry() (*ProjectRegistry, error) {
	return LoadRegistryWithOverride("")
}

// LoadRegistryWithOverride loads the registry with an optional home directory override.
// Used for testing.
func LoadRegistryWithOverride(homeDirOverride string) (*ProjectRegistry, error) {
	globalRoot, err := GetGlobalWorkspaceRootWithOverride(homeDirOverride)
	if err != nil {
		return nil, fmt.Errorf("get global workspace root: %w", err)
	}

	path := filepath.Join(globalRoot, RegistryFile)

	registry := &ProjectRegistry{
		Version:  RegistryVersion,
		Projects: make(map[string]ProjectMetadata),
		path:     path,
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		// Return empty registry if file doesn't exist
		return registry, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read registry: %w", err)
	}

	if err := yaml.Unmarshal(data, registry); err != nil {
		return nil, fmt.Errorf("parse registry: %w", err)
	}

	// Ensure Projects map is initialized even if empty in file
	if registry.Projects == nil {
		registry.Projects = make(map[string]ProjectMetadata)
	}

	// Never expose stored credentials in remote URLs.
	for id, meta := range registry.Projects {
		meta.RemoteURL = SanitizeRemoteURL(meta.RemoteURL)
		registry.Projects[id] = meta
	}

	registry.path = path

	return registry, nil
}

// Save writes the registry to disk using atomic write pattern.
func (r *ProjectRegistry) Save() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Ensure directory exists
	dir := filepath.Dir(r.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create registry directory: %w", err)
	}

	data, err := yaml.Marshal(r)
	if err != nil {
		return fmt.Errorf("marshal registry: %w", err)
	}

	// Use atomic write pattern: write to temp file, then rename
	tmpPath := r.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("write registry: %w", err)
	}

	if err := os.Rename(tmpPath, r.path); err != nil {
		// Clean up temp file on error
		if removeErr := os.Remove(tmpPath); removeErr != nil {
			slog.Warn("failed to clean up temp file after rename error", "path", tmpPath, "error", removeErr)
		}

		return fmt.Errorf("save registry: %w", err)
	}

	return nil
}

// Register adds or updates a project in the registry.
func (r *ProjectRegistry) Register(id, path, remoteURL, name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	sanitizedRemoteURL := SanitizeRemoteURL(remoteURL)

	existing, exists := r.Projects[id]
	if exists {
		// Update existing entry
		existing.Path = path
		existing.RemoteURL = sanitizedRemoteURL
		existing.Name = name
		existing.LastAccess = now
		r.Projects[id] = existing
	} else {
		// Create new entry
		r.Projects[id] = ProjectMetadata{
			ID:           id,
			Path:         path,
			RemoteURL:    sanitizedRemoteURL,
			Name:         name,
			RegisteredAt: now,
			LastAccess:   now,
		}
	}

	return nil
}

// Unregister removes a project from the registry.
// Returns true if the project was found and removed, false otherwise.
func (r *ProjectRegistry) Unregister(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.Projects[id]; exists {
		delete(r.Projects, id)

		return true
	}

	return false
}

// Lookup retrieves a project by ID.
// Returns nil if the project is not registered.
func (r *ProjectRegistry) Lookup(id string) *ProjectMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if meta, exists := r.Projects[id]; exists {
		return &meta
	}

	return nil
}

// List returns all registered projects.
func (r *ProjectRegistry) List() []ProjectMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]ProjectMetadata, 0, len(r.Projects))
	for _, meta := range r.Projects {
		result = append(result, meta)
	}

	return result
}

// Count returns the number of registered projects.
func (r *ProjectRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.Projects)
}

// UpdateLastAccess updates the last access time for a project.
func (r *ProjectRegistry) UpdateLastAccess(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if meta, exists := r.Projects[id]; exists {
		meta.LastAccess = time.Now()
		r.Projects[id] = meta
	}
}
