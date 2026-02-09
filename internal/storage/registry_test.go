package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectRegistry_LoadEmpty(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Load registry from non-existent file
	registry, err := LoadRegistryWithOverride(tmpDir)
	require.NoError(t, err)
	assert.NotNil(t, registry)
	assert.Equal(t, RegistryVersion, registry.Version)
	assert.Empty(t, registry.Projects)
}

func TestProjectRegistry_RegisterAndSave(t *testing.T) {
	tmpDir := t.TempDir()

	// Load empty registry
	registry, err := LoadRegistryWithOverride(tmpDir)
	require.NoError(t, err)

	// Register a project
	err = registry.Register("github.com-user-repo", "/path/to/repo", "https://github.com/user/repo", "repo")
	require.NoError(t, err)

	// Save registry
	err = registry.Save()
	require.NoError(t, err)

	// Verify file exists (path is tmpDir/.valksor/mehrhof/projects.yaml)
	registryPath := filepath.Join(tmpDir, ".valksor", "mehrhof", RegistryFile)
	_, err = os.Stat(registryPath)
	require.NoError(t, err)

	// Load registry again and verify
	registry2, err := LoadRegistryWithOverride(tmpDir)
	require.NoError(t, err)
	assert.Len(t, registry2.Projects, 1)

	meta := registry2.Lookup("github.com-user-repo")
	require.NotNil(t, meta)
	assert.Equal(t, "github.com-user-repo", meta.ID)
	assert.Equal(t, "/path/to/repo", meta.Path)
	assert.Equal(t, "https://github.com/user/repo", meta.RemoteURL)
	assert.Equal(t, "repo", meta.Name)
	assert.False(t, meta.RegisteredAt.IsZero())
}

func TestProjectRegistry_Unregister(t *testing.T) {
	tmpDir := t.TempDir()

	registry, err := LoadRegistryWithOverride(tmpDir)
	require.NoError(t, err)

	// Register two projects
	err = registry.Register("project-1", "/path/1", "", "Project 1")
	require.NoError(t, err)
	err = registry.Register("project-2", "/path/2", "", "Project 2")
	require.NoError(t, err)

	assert.Equal(t, 2, registry.Count())

	// Unregister one
	removed := registry.Unregister("project-1")
	assert.True(t, removed)
	assert.Equal(t, 1, registry.Count())

	// Verify it's gone
	assert.Nil(t, registry.Lookup("project-1"))
	assert.NotNil(t, registry.Lookup("project-2"))

	// Unregister non-existent
	removed = registry.Unregister("non-existent")
	assert.False(t, removed)
}

func TestProjectRegistry_List(t *testing.T) {
	tmpDir := t.TempDir()

	registry, err := LoadRegistryWithOverride(tmpDir)
	require.NoError(t, err)

	// Register projects
	err = registry.Register("project-a", "/path/a", "", "Project A")
	require.NoError(t, err)
	err = registry.Register("project-b", "/path/b", "", "Project B")
	require.NoError(t, err)

	// List all
	list := registry.List()
	assert.Len(t, list, 2)

	// Verify both are in list
	ids := make(map[string]bool)
	for _, meta := range list {
		ids[meta.ID] = true
	}
	assert.True(t, ids["project-a"])
	assert.True(t, ids["project-b"])
}

func TestProjectRegistry_UpdateExisting(t *testing.T) {
	tmpDir := t.TempDir()

	registry, err := LoadRegistryWithOverride(tmpDir)
	require.NoError(t, err)

	// Register project
	err = registry.Register("project-1", "/old/path", "", "Old Name")
	require.NoError(t, err)

	meta := registry.Lookup("project-1")
	require.NotNil(t, meta)
	originalRegTime := meta.RegisteredAt

	// Update same project
	err = registry.Register("project-1", "/new/path", "https://example.com", "New Name")
	require.NoError(t, err)

	// Verify update
	meta = registry.Lookup("project-1")
	require.NotNil(t, meta)
	assert.Equal(t, "/new/path", meta.Path)
	assert.Equal(t, "https://example.com", meta.RemoteURL)
	assert.Equal(t, "New Name", meta.Name)
	// RegisteredAt should be preserved
	assert.Equal(t, originalRegTime, meta.RegisteredAt)
	// Still only one project
	assert.Equal(t, 1, registry.Count())
}

func TestProjectRegistry_UpdateLastAccess(t *testing.T) {
	tmpDir := t.TempDir()

	registry, err := LoadRegistryWithOverride(tmpDir)
	require.NoError(t, err)

	// Register project
	err = registry.Register("project-1", "/path", "", "Project")
	require.NoError(t, err)

	meta := registry.Lookup("project-1")
	require.NotNil(t, meta)
	initialAccess := meta.LastAccess

	// Update last access
	registry.UpdateLastAccess("project-1")

	meta = registry.Lookup("project-1")
	require.NotNil(t, meta)
	assert.True(t, meta.LastAccess.After(initialAccess) || meta.LastAccess.Equal(initialAccess))
}

func TestProjectRegistry_UpdateLastAccess_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()

	registry, err := LoadRegistryWithOverride(tmpDir)
	require.NoError(t, err)

	// Should not panic on non-existent project
	registry.UpdateLastAccess("non-existent")
}

func TestProjectRegistry_LookupNonExistent(t *testing.T) {
	tmpDir := t.TempDir()

	registry, err := LoadRegistryWithOverride(tmpDir)
	require.NoError(t, err)

	meta := registry.Lookup("non-existent")
	assert.Nil(t, meta)
}

func TestProjectRegistry_Persistence(t *testing.T) {
	tmpDir := t.TempDir()

	// Create registry and add projects
	registry1, err := LoadRegistryWithOverride(tmpDir)
	require.NoError(t, err)

	err = registry1.Register("project-1", "/path/1", "https://example.com/1", "Project 1")
	require.NoError(t, err)
	err = registry1.Register("project-2", "/path/2", "https://example.com/2", "Project 2")
	require.NoError(t, err)

	// Save to disk
	err = registry1.Save()
	require.NoError(t, err)

	// Load fresh registry
	registry2, err := LoadRegistryWithOverride(tmpDir)
	require.NoError(t, err)

	// Verify both projects persisted
	assert.Equal(t, 2, registry2.Count())
	assert.NotNil(t, registry2.Lookup("project-1"))
	assert.NotNil(t, registry2.Lookup("project-2"))
}

func TestProjectRegistry_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()

	registry, err := LoadRegistryWithOverride(tmpDir)
	require.NoError(t, err)

	// Concurrent reads and writes
	done := make(chan bool)
	for i := range 10 {
		go func(id int) {
			projectID := "project-" + string(rune('0'+id))
			_ = registry.Register(projectID, "/path", "", "Project")
			_ = registry.Lookup(projectID)
			_ = registry.List()
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for range 10 {
		<-done
	}
}

func TestProjectRegistry_EmptyFields(t *testing.T) {
	tmpDir := t.TempDir()

	registry, err := LoadRegistryWithOverride(tmpDir)
	require.NoError(t, err)

	// Register with empty optional fields
	err = registry.Register("project-1", "/path", "", "")
	require.NoError(t, err)

	meta := registry.Lookup("project-1")
	require.NotNil(t, meta)
	assert.Equal(t, "", meta.RemoteURL)
	assert.Equal(t, "", meta.Name)
}

func TestProjectRegistry_Register_RedactsRemoteCredentials(t *testing.T) {
	tmpDir := t.TempDir()

	registry, err := LoadRegistryWithOverride(tmpDir)
	require.NoError(t, err)

	err = registry.Register(
		"project-1",
		"/path",
		"https://ghp_secret123@github.com/user/repo.git",
		"Project",
	)
	require.NoError(t, err)

	meta := registry.Lookup("project-1")
	require.NotNil(t, meta)
	assert.Equal(t, "https://github.com/user/repo.git", meta.RemoteURL)
	assert.NotContains(t, meta.RemoteURL, "ghp_secret123")
}

func TestProjectRegistry_FindByPath(t *testing.T) {
	tmpDir := t.TempDir()

	registry, err := LoadRegistryWithOverride(tmpDir)
	require.NoError(t, err)

	// Register projects with different paths
	err = registry.Register("project-1", "/path/to/project1", "https://github.com/user/repo1.git", "Project 1")
	require.NoError(t, err)
	err = registry.Register("project-2", "/path/to/project2", "https://github.com/user/repo2.git", "Project 2")
	require.NoError(t, err)

	// Find existing project by exact path
	meta, ok := registry.FindByPath("/path/to/project1")
	assert.True(t, ok)
	assert.Equal(t, "project-1", meta.ID)
	assert.Equal(t, "Project 1", meta.Name)

	// Find with trailing slash (cleaned path should match)
	meta, ok = registry.FindByPath("/path/to/project2/")
	assert.True(t, ok)
	assert.Equal(t, "project-2", meta.ID)

	// Not found for non-existent path
	_, ok = registry.FindByPath("/nonexistent/path")
	assert.False(t, ok)
}

func TestProjectRegistry_FindByPath_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()

	registry, err := LoadRegistryWithOverride(tmpDir)
	require.NoError(t, err)

	// Register a project
	err = registry.Register("project-1", "/path/to/project", "", "Project")
	require.NoError(t, err)

	// Concurrent reads via FindByPath
	done := make(chan bool)
	for range 10 {
		go func() {
			_, _ = registry.FindByPath("/path/to/project")
			done <- true
		}()
	}

	// Wait for all goroutines
	for range 10 {
		<-done
	}
}

func TestProjectRegistry_ToggleFavorite(t *testing.T) {
	tmpDir := t.TempDir()

	registry, err := LoadRegistryWithOverride(tmpDir)
	require.NoError(t, err)

	// Register a project
	err = registry.Register("project-1", "/path/to/project", "", "Project")
	require.NoError(t, err)

	// Initially not a favorite
	meta, ok := registry.FindByPath("/path/to/project")
	require.True(t, ok)
	assert.False(t, meta.IsFavorite)

	// Toggle to favorite
	isFav, err := registry.ToggleFavorite("/path/to/project")
	require.NoError(t, err)
	assert.True(t, isFav)

	// Verify it's now a favorite
	meta, ok = registry.FindByPath("/path/to/project")
	require.True(t, ok)
	assert.True(t, meta.IsFavorite)

	// Toggle back to non-favorite
	isFav, err = registry.ToggleFavorite("/path/to/project")
	require.NoError(t, err)
	assert.False(t, isFav)

	// Verify it's no longer a favorite
	meta, ok = registry.FindByPath("/path/to/project")
	require.True(t, ok)
	assert.False(t, meta.IsFavorite)

	// Error for non-existent project
	_, err = registry.ToggleFavorite("/nonexistent/path")
	assert.Error(t, err)
}

func TestProjectRegistry_GetFavorites(t *testing.T) {
	tmpDir := t.TempDir()

	registry, err := LoadRegistryWithOverride(tmpDir)
	require.NoError(t, err)

	// Register multiple projects
	err = registry.Register("project-1", "/path/to/project1", "", "Project 1")
	require.NoError(t, err)
	err = registry.Register("project-2", "/path/to/project2", "", "Project 2")
	require.NoError(t, err)
	err = registry.Register("project-3", "/path/to/project3", "", "Project 3")
	require.NoError(t, err)

	// Initially no favorites
	favorites := registry.GetFavorites()
	assert.Empty(t, favorites)

	// Mark project1 and project3 as favorites
	_, err = registry.ToggleFavorite("/path/to/project1")
	require.NoError(t, err)
	_, err = registry.ToggleFavorite("/path/to/project3")
	require.NoError(t, err)

	// Get favorites
	favorites = registry.GetFavorites()
	assert.Len(t, favorites, 2)
	assert.Contains(t, favorites, "/path/to/project1")
	assert.Contains(t, favorites, "/path/to/project3")
	assert.NotContains(t, favorites, "/path/to/project2")
}

func TestProjectRegistry_ListRecent(t *testing.T) {
	tmpDir := t.TempDir()

	registry, err := LoadRegistryWithOverride(tmpDir)
	require.NoError(t, err)

	// Register projects with different access times
	now := time.Now()
	err = registry.Register("project-old", "/path/to/old", "", "Old Project")
	require.NoError(t, err)

	// Update the old project's last access to be in the past
	if meta, ok := registry.FindByPath("/path/to/old"); ok {
		meta.LastAccess = now.Add(-time.Hour)
		registry.Projects[meta.ID] = meta
	}

	time.Sleep(10 * time.Millisecond) // Ensure different timestamps

	err = registry.Register("project-new", "/path/to/new", "", "New Project")
	require.NoError(t, err)

	// ListRecent should return newest first
	projects := registry.ListRecent(0) // 0 = no limit
	require.Len(t, projects, 2)
	assert.Equal(t, "project-new", projects[0].ID)
	assert.Equal(t, "project-old", projects[1].ID)

	// Test with limit
	projects = registry.ListRecent(1)
	require.Len(t, projects, 1)
	assert.Equal(t, "project-new", projects[0].ID)
}

func TestProjectRegistry_IsFavorite_Persistence(t *testing.T) {
	tmpDir := t.TempDir()

	registry, err := LoadRegistryWithOverride(tmpDir)
	require.NoError(t, err)

	// Register and favorite a project
	err = registry.Register("project-1", "/path/to/project", "", "Project")
	require.NoError(t, err)
	_, err = registry.ToggleFavorite("/path/to/project")
	require.NoError(t, err)
	err = registry.Save()
	require.NoError(t, err)

	// Reload registry
	registry2, err := LoadRegistryWithOverride(tmpDir)
	require.NoError(t, err)

	// Verify favorite status persisted
	meta, ok := registry2.FindByPath("/path/to/project")
	require.True(t, ok)
	assert.True(t, meta.IsFavorite)

	favorites := registry2.GetFavorites()
	assert.Len(t, favorites, 1)
	assert.Contains(t, favorites, "/path/to/project")
}
