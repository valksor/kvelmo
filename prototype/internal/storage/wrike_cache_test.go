package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ──────────────────────────────────────────────────────────────────────────────
// WrikeCache tests
// ──────────────────────────────────────────────────────────────────────────────

func TestWrikeCachePath(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)

	path := ws.WrikeCachePath()
	expected := filepath.Join(ws.workspaceRoot, "wrike_cache.json")

	if path != expected {
		t.Errorf("WrikeCachePath() = %q, want %q", path, expected)
	}
}

func TestLoadWrikeCache_NotExists(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)

	cache, err := ws.LoadWrikeCache()
	if err != nil {
		t.Errorf("LoadWrikeCache() error = %v, want nil for non-existent file", err)
	}
	if cache != nil {
		t.Errorf("LoadWrikeCache() cache = %v, want nil for non-existent file", cache)
	}
}

func TestSaveAndLoadWrikeCache(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)

	// Create workspace directory first
	if err := os.MkdirAll(ws.workspaceRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", ws.workspaceRoot, err)
	}

	// Create cache with all fields
	cache := &WrikeCache{
		ResolvedAt: time.Now(),
		Space: &WrikeCachedEntity{
			NumericID: "824404493",
			APIID:     "IEAAJSPACEID",
			Title:     "Test Space",
			Type:      "space",
			Permalink: "https://www.wrike.com/open.htm?id=824404493",
			Scope:     "WsSpace",
			IsProject: false,
		},
		Folder: &WrikeCachedEntity{
			NumericID: "1635167041",
			APIID:     "IEAAJFOLDERID",
			Title:     "Test Folder",
			Type:      "folder",
			Permalink: "https://www.wrike.com/open.htm?id=1635167041",
			Scope:     "WsFolder",
			IsProject: false,
		},
		Project: &WrikeCachedEntity{
			NumericID: "4352950154",
			APIID:     "IEAAJPROJECTID",
			Title:     "Test Project",
			Type:      "project",
			Permalink: "https://www.wrike.com/open.htm?id=4352950154",
			Scope:     "WsProject",
			IsProject: true,
		},
	}

	// Save cache
	if err := ws.SaveWrikeCache(cache); err != nil {
		t.Fatalf("SaveWrikeCache() error = %v", err)
	}

	// Verify file exists
	if !ws.HasWrikeCache() {
		t.Error("HasWrikeCache() = false after save, want true")
	}

	// Load cache and verify
	loaded, err := ws.LoadWrikeCache()
	if err != nil {
		t.Fatalf("LoadWrikeCache() error = %v", err)
	}
	if loaded == nil {
		t.Fatal("LoadWrikeCache() returned nil")
	}

	// Verify Space
	if loaded.Space == nil {
		t.Fatal("loaded.Space is nil")
	}
	if loaded.Space.NumericID != "824404493" {
		t.Errorf("Space.NumericID = %q, want %q", loaded.Space.NumericID, "824404493")
	}
	if loaded.Space.APIID != "IEAAJSPACEID" {
		t.Errorf("Space.APIID = %q, want %q", loaded.Space.APIID, "IEAAJSPACEID")
	}
	if loaded.Space.Title != "Test Space" {
		t.Errorf("Space.Title = %q, want %q", loaded.Space.Title, "Test Space")
	}

	// Verify Folder
	if loaded.Folder == nil {
		t.Fatal("loaded.Folder is nil")
	}
	if loaded.Folder.NumericID != "1635167041" {
		t.Errorf("Folder.NumericID = %q, want %q", loaded.Folder.NumericID, "1635167041")
	}
	if loaded.Folder.APIID != "IEAAJFOLDERID" {
		t.Errorf("Folder.APIID = %q, want %q", loaded.Folder.APIID, "IEAAJFOLDERID")
	}

	// Verify Project
	if loaded.Project == nil {
		t.Fatal("loaded.Project is nil")
	}
	if loaded.Project.NumericID != "4352950154" {
		t.Errorf("Project.NumericID = %q, want %q", loaded.Project.NumericID, "4352950154")
	}
	if loaded.Project.APIID != "IEAAJPROJECTID" {
		t.Errorf("Project.APIID = %q, want %q", loaded.Project.APIID, "IEAAJPROJECTID")
	}
	if !loaded.Project.IsProject {
		t.Error("Project.IsProject = false, want true")
	}
}

func TestSaveWrikeCache_UpdatesResolvedAt(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)

	// Create workspace directory
	if err := os.MkdirAll(ws.workspaceRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", ws.workspaceRoot, err)
	}

	oldTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	cache := &WrikeCache{
		ResolvedAt: oldTime,
		Project: &WrikeCachedEntity{
			NumericID: "1234567890",
			APIID:     "IEAAJTEST",
			Title:     "Test",
			Type:      "project",
		},
	}

	// Save should update ResolvedAt
	if err := ws.SaveWrikeCache(cache); err != nil {
		t.Fatalf("SaveWrikeCache() error = %v", err)
	}

	// Load and verify ResolvedAt was updated
	loaded, err := ws.LoadWrikeCache()
	if err != nil {
		t.Fatalf("LoadWrikeCache() error = %v", err)
	}

	if loaded.ResolvedAt.Equal(oldTime) {
		t.Error("ResolvedAt was not updated during save")
	}
	if loaded.ResolvedAt.Before(time.Now().Add(-time.Minute)) {
		t.Errorf("ResolvedAt = %v, expected recent time", loaded.ResolvedAt)
	}
}

func TestClearWrikeCache(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)

	// Create workspace directory
	if err := os.MkdirAll(ws.workspaceRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", ws.workspaceRoot, err)
	}

	// Save cache first
	cache := &WrikeCache{
		Project: &WrikeCachedEntity{
			NumericID: "1234567890",
			APIID:     "IEAAJTEST",
			Title:     "Test",
			Type:      "project",
		},
	}
	if err := ws.SaveWrikeCache(cache); err != nil {
		t.Fatalf("SaveWrikeCache() error = %v", err)
	}

	// Verify cache exists
	if !ws.HasWrikeCache() {
		t.Fatal("HasWrikeCache() = false after save, want true")
	}

	// Clear cache
	if err := ws.ClearWrikeCache(); err != nil {
		t.Fatalf("ClearWrikeCache() error = %v", err)
	}

	// Verify cache no longer exists
	if ws.HasWrikeCache() {
		t.Error("HasWrikeCache() = true after clear, want false")
	}
}

func TestClearWrikeCache_NotExists(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)

	// Clear non-existent cache should not error
	if err := ws.ClearWrikeCache(); err != nil {
		t.Errorf("ClearWrikeCache() error = %v, want nil for non-existent file", err)
	}
}

func TestHasWrikeCache(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)

	// Initially no cache
	if ws.HasWrikeCache() {
		t.Error("HasWrikeCache() = true, want false (no cache exists)")
	}

	// Create workspace directory and cache
	if err := os.MkdirAll(ws.workspaceRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", ws.workspaceRoot, err)
	}

	cache := &WrikeCache{
		Project: &WrikeCachedEntity{
			NumericID: "1234567890",
			APIID:     "IEAAJTEST",
			Title:     "Test",
			Type:      "project",
		},
	}
	if err := ws.SaveWrikeCache(cache); err != nil {
		t.Fatalf("SaveWrikeCache() error = %v", err)
	}

	// Now cache exists
	if !ws.HasWrikeCache() {
		t.Error("HasWrikeCache() = false after save, want true")
	}
}

func TestLoadWrikeCache_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)

	// Create workspace directory
	if err := os.MkdirAll(ws.workspaceRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", ws.workspaceRoot, err)
	}

	// Write invalid JSON to cache file
	if err := os.WriteFile(ws.WrikeCachePath(), []byte("not valid json"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Load should return error for invalid JSON
	_, err := ws.LoadWrikeCache()
	if err == nil {
		t.Error("LoadWrikeCache() expected error for invalid JSON, got nil")
	}
}

func TestSaveWrikeCache_PartialData(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)

	// Create workspace directory
	if err := os.MkdirAll(ws.workspaceRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", ws.workspaceRoot, err)
	}

	// Create cache with only project (no space or folder)
	cache := &WrikeCache{
		Project: &WrikeCachedEntity{
			NumericID: "4352950154",
			APIID:     "IEAAJPROJECTID",
			Title:     "Test Project",
			Type:      "project",
			IsProject: true,
		},
	}

	// Save and load
	if err := ws.SaveWrikeCache(cache); err != nil {
		t.Fatalf("SaveWrikeCache() error = %v", err)
	}

	loaded, err := ws.LoadWrikeCache()
	if err != nil {
		t.Fatalf("LoadWrikeCache() error = %v", err)
	}

	// Verify partial data is preserved
	if loaded.Space != nil {
		t.Error("loaded.Space should be nil")
	}
	if loaded.Folder != nil {
		t.Error("loaded.Folder should be nil")
	}
	if loaded.Project == nil {
		t.Fatal("loaded.Project should not be nil")
	}
	if loaded.Project.APIID != "IEAAJPROJECTID" {
		t.Errorf("Project.APIID = %q, want %q", loaded.Project.APIID, "IEAAJPROJECTID")
	}
}
