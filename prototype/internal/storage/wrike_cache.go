package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const wrikeCacheFile = "wrike_cache.json"

// WrikeCache stores resolved Wrike IDs and metadata for reuse.
// This avoids repeated API calls when numeric IDs are configured.
type WrikeCache struct {
	ResolvedAt time.Time          `json:"resolved_at"`
	Space      *WrikeCachedEntity `json:"space,omitempty"`
	Folder     *WrikeCachedEntity `json:"folder,omitempty"`
	Project    *WrikeCachedEntity `json:"project,omitempty"`
}

// WrikeCachedEntity stores resolved ID and useful metadata.
type WrikeCachedEntity struct {
	NumericID string `json:"numeric_id"`      // Original from config (e.g., "4352950154")
	APIID     string `json:"api_id"`          // Resolved API ID (e.g., "IEAAJXXXX")
	Title     string `json:"title"`           // Entity name from Wrike
	Type      string `json:"type"`            // "folder", "project", "space"
	Permalink string `json:"permalink"`       // Full Wrike URL
	Scope     string `json:"scope,omitempty"` // "WsFolder", "WsProject", etc.
	IsProject bool   `json:"is_project"`      // True if this folder is actually a project
}

// WrikeCachePath returns the path to wrike_cache.json file.
func (w *Workspace) WrikeCachePath() string {
	return filepath.Join(w.workspaceRoot, wrikeCacheFile)
}

// LoadWrikeCache loads the cached Wrike IDs and metadata.
// Returns nil, nil if the cache file doesn't exist (not an error).
func (w *Workspace) LoadWrikeCache() (*WrikeCache, error) {
	data, err := os.ReadFile(w.WrikeCachePath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil //nolint:nilnil // Non-existent cache is not an error
		}

		return nil, fmt.Errorf("read wrike cache: %w", err)
	}

	var cache WrikeCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("parse wrike cache: %w", err)
	}

	return &cache, nil
}

// SaveWrikeCache saves the Wrike cache using atomic write pattern.
func (w *Workspace) SaveWrikeCache(cache *WrikeCache) error {
	cache.ResolvedAt = time.Now()

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal wrike cache: %w", err)
	}

	// Use atomic write pattern: write to temp file, then rename
	path := w.WrikeCachePath()
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("write wrike cache: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath) // Clean up on error

		return fmt.Errorf("save wrike cache: %w", err)
	}

	return nil
}

// ClearWrikeCache removes the Wrike cache file.
func (w *Workspace) ClearWrikeCache() error {
	err := os.Remove(w.WrikeCachePath())
	if os.IsNotExist(err) {
		return nil
	}

	return err
}

// HasWrikeCache checks if there's a cached Wrike configuration.
func (w *Workspace) HasWrikeCache() bool {
	_, err := os.Stat(w.WrikeCachePath())

	return err == nil
}
