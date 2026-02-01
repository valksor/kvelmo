package links

import (
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewManager tests creating a new link manager.
func TestNewManager(t *testing.T) {
	tmpDir := t.TempDir()

	mgr := NewManager(tmpDir)

	assert.NotNil(t, mgr)
	assert.Equal(t, tmpDir, mgr.workspace)
	assert.Equal(t, filepath.Join(tmpDir, "links"), mgr.linksDir)
	assert.NotNil(t, mgr.index)
	assert.NotNil(t, mgr.names)
}

// TestManager_Load_EmptyWorkspace tests loading from an empty workspace.
func TestManager_Load_EmptyWorkspace(t *testing.T) {
	tmpDir := t.TempDir()

	mgr := NewManager(tmpDir)
	err := mgr.Load()

	assert.NoError(t, err)
	assert.NotNil(t, mgr.index)
	assert.NotNil(t, mgr.names)
}

// TestManager_Load_ExistingIndex tests loading from an existing index.
func TestManager_Load_ExistingIndex(t *testing.T) {
	tmpDir := t.TempDir()
	linksDir := filepath.Join(tmpDir, "links")

	// Create links directory
	err := os.MkdirAll(linksDir, 0o755)
	require.NoError(t, err)

	// Create a manager and add some data
	mgr1 := NewManager(tmpDir)
	mgr1.index.Forward["spec:task:1"] = []Link{
		{
			Source:    "spec:task:1",
			Target:    "spec:task:2",
			Context:   "see also",
			CreatedAt: time.Now(),
		},
	}
	mgr1.index.Backward["spec:task:2"] = []Link{
		{
			Source:    "spec:task:1",
			Target:    "spec:task:2",
			Context:   "see also",
			CreatedAt: time.Now(),
		},
	}
	_ = mgr1.Save()

	// Load into a new manager
	mgr2 := NewManager(tmpDir)
	err = mgr2.Load()

	assert.NoError(t, err)
	assert.NotNil(t, mgr2.index)
	assert.Equal(t, 1, len(mgr2.index.Forward))
}

// TestManager_Save tests saving the link index.
func TestManager_Save(t *testing.T) {
	tmpDir := t.TempDir()

	mgr := NewManager(tmpDir)
	_ = mgr.Load() // Create the links directory

	// Add some data using the public API
	content := `See [[spec:task:2]] for details.`
	err := mgr.IndexContent("spec:task:1", content, "task")
	require.NoError(t, err)
	err = mgr.RegisterName(TypeSpec, "Test Spec", "spec:task:1")
	require.NoError(t, err)

	// Save
	err = mgr.Save()
	assert.NoError(t, err)

	// Verify file exists
	indexPath := filepath.Join(tmpDir, "links", IndexFileName)
	_, err = os.Stat(indexPath)
	assert.NoError(t, err, "index file should exist")
}

// TestManager_Save_Atomic tests that save uses atomic writes.
func TestManager_Save_Atomic(t *testing.T) {
	tmpDir := t.TempDir()

	mgr := NewManager(tmpDir)
	_ = mgr.Load() // Create the links directory

	// Add some data using the public API
	content := `See [[spec:task:2]] for details.`
	err := mgr.IndexContent("spec:task:1", content, "task")
	require.NoError(t, err)

	// Save multiple times to ensure atomic rename works
	for range 5 {
		err := mgr.Save()
		assert.NoError(t, err)
	}

	// Verify file exists and is valid
	indexPath := filepath.Join(tmpDir, "links", IndexFileName)
	data, err := os.ReadFile(indexPath)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "spec:task:1")
}

// TestManager_IndexContent tests indexing content for links.
func TestManager_IndexContent(t *testing.T) {
	tmpDir := t.TempDir()

	mgr := NewManager(tmpDir)
	_ = mgr.Load()

	tests := []struct {
		name         string
		sourceID     string
		content      string
		activeTaskID string
		wantLinks    int
	}{
		{
			name:         "no links",
			sourceID:     "spec:task:1",
			content:      "No links here",
			activeTaskID: "task",
			wantLinks:    0,
		},
		{
			name:         "single link",
			sourceID:     "spec:task:1",
			content:      "See [[spec:task:2]] for details",
			activeTaskID: "task",
			wantLinks:    1,
		},
		{
			name:         "multiple links",
			sourceID:     "spec:task:1",
			content:      "See [[spec:task:2]] and [[spec:task:3]]",
			activeTaskID: "task",
			wantLinks:    2,
		},
		{
			name:         "task-scoped link",
			sourceID:     "spec:task:1",
			content:      "See [[spec:2]] for details",
			activeTaskID: "task",
			wantLinks:    1,
		},
		{
			name:         "task-scoped without active task",
			sourceID:     "spec:task:1",
			content:      "See [[spec:2]] for details",
			activeTaskID: "",
			wantLinks:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mgr.IndexContent(tt.sourceID, tt.content, tt.activeTaskID)
			assert.NoError(t, err)

			outgoing := mgr.GetOutgoing(tt.sourceID)
			assert.Equal(t, tt.wantLinks, len(outgoing))
		})
	}
}

// TestManager_IndexContent_Reindex tests that reindexing replaces existing links.
func TestManager_IndexContent_Reindex(t *testing.T) {
	tmpDir := t.TempDir()

	mgr := NewManager(tmpDir)
	_ = mgr.Load()

	sourceID := "spec:task:1"

	// Index initial content
	_ = mgr.IndexContent(sourceID, "See [[spec:task:2]]", "task")
	outgoing1 := mgr.GetOutgoing(sourceID)
	assert.Equal(t, 1, len(outgoing1))

	// Reindex with different content
	_ = mgr.IndexContent(sourceID, "See [[spec:task:3]]", "task")
	outgoing2 := mgr.GetOutgoing(sourceID)
	assert.Equal(t, 1, len(outgoing2))
	assert.Equal(t, "spec:task:3", outgoing2[0].Target)

	// Verify old target no longer has incoming link
	incoming := mgr.GetIncoming("spec:task:2")
	assert.Equal(t, 0, len(incoming))
}

// TestManager_RegisterName tests registering entity names.
func TestManager_RegisterName(t *testing.T) {
	tmpDir := t.TempDir()

	mgr := NewManager(tmpDir)
	_ = mgr.Load()

	tests := []struct {
		name       string
		entityTyp  EntityType
		entityName string
		entityID   string
	}{
		{
			name:       "register spec",
			entityTyp:  TypeSpec,
			entityName: "Authentication Flow",
			entityID:   "spec:task:1",
		},
		{
			name:       "register decision",
			entityTyp:  TypeDecision,
			entityName: "Cache Strategy",
			entityID:   "decision:task:cache",
		},
		{
			name:       "register session",
			entityTyp:  TypeSession,
			entityName: "Planning Session",
			entityID:   "session:task:2024-01-29T10:00:00Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mgr.RegisterName(tt.entityTyp, tt.entityName, tt.entityID)
			assert.NoError(t, err)

			// Verify lookup
			entityID, found := mgr.ResolveName(tt.entityName)
			assert.True(t, found)
			assert.Equal(t, tt.entityID, entityID)
		})
	}
}

// TestManager_ResolveName tests resolving entity names.
func TestManager_ResolveName(t *testing.T) {
	tmpDir := t.TempDir()

	mgr := NewManager(tmpDir)
	_ = mgr.Load()

	// Register some names
	_ = mgr.RegisterName(TypeSpec, "Auth Spec", "spec:task:1")
	_ = mgr.RegisterName(TypeDecision, "cache", "decision:task:cache")

	tests := []struct {
		name         string
		search       string
		wantFound    bool
		wantEntityID string
	}{
		{
			name:         "exact match",
			search:       "Auth Spec",
			wantFound:    true,
			wantEntityID: "spec:task:1",
		},
		{
			name:         "case insensitive",
			search:       "auth spec",
			wantFound:    true,
			wantEntityID: "spec:task:1",
		},
		{
			name:      "not found",
			search:    "Nonexistent",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entityID, found := mgr.ResolveName(tt.search)
			assert.Equal(t, tt.wantFound, found)
			if found {
				assert.Equal(t, tt.wantEntityID, entityID)
			}
		})
	}
}

// TestManager_UnregisterName tests unregistering entity names.
func TestManager_UnregisterName(t *testing.T) {
	tmpDir := t.TempDir()

	mgr := NewManager(tmpDir)
	_ = mgr.Load()

	// Register a name
	_ = mgr.RegisterName(TypeSpec, "Test Spec", "spec:task:1")

	// Verify it's registered
	_, found := mgr.ResolveName("Test Spec")
	assert.True(t, found)

	// Unregister
	_ = mgr.UnregisterName(TypeSpec, "Test Spec")

	// Verify it's gone
	_, found = mgr.ResolveName("Test Spec")
	assert.False(t, found)
}

// TestManager_GetOutgoing tests getting outgoing links.
func TestManager_GetOutgoing(t *testing.T) {
	tmpDir := t.TempDir()

	mgr := NewManager(tmpDir)
	_ = mgr.Load()

	_ = mgr.IndexContent("spec:task:1", "See [[spec:task:2]]", "task")

	outgoing := mgr.GetOutgoing("spec:task:1")
	assert.Equal(t, 1, len(outgoing))
	assert.Equal(t, "spec:task:2", outgoing[0].Target)

	// Non-existent source
	empty := mgr.GetOutgoing("nonexistent")
	assert.Nil(t, empty)
}

// TestManager_GetIncoming tests getting incoming links.
func TestManager_GetIncoming(t *testing.T) {
	tmpDir := t.TempDir()

	mgr := NewManager(tmpDir)
	_ = mgr.Load()

	_ = mgr.IndexContent("spec:task:1", "See [[spec:task:2]]", "task")

	incoming := mgr.GetIncoming("spec:task:2")
	assert.Equal(t, 1, len(incoming))
	assert.Equal(t, "spec:task:1", incoming[0].Source)

	// Non-existent target
	empty := mgr.GetIncoming("nonexistent")
	assert.Nil(t, empty)
}

// TestManager_GetStats tests getting link statistics.
func TestManager_GetStats(t *testing.T) {
	tmpDir := t.TempDir()

	mgr := NewManager(tmpDir)
	_ = mgr.Load()

	// Initially no stats
	stats := mgr.GetStats()
	assert.Equal(t, 0, stats.TotalLinks)
	assert.Equal(t, 0, stats.TotalSources)
	assert.Equal(t, 0, stats.TotalTargets)

	// Add some links
	_ = mgr.IndexContent("spec:task:1", "See [[spec:task:2]] and [[spec:task:3]]", "task")

	stats = mgr.GetStats()
	assert.Equal(t, 2, stats.TotalLinks)
	assert.Equal(t, 1, stats.TotalSources)
	assert.Equal(t, 2, stats.TotalTargets)
}

// TestManager_Rebuild tests rebuilding the index from workspace.
func TestManager_Rebuild(t *testing.T) {
	tests := []struct {
		name      string
		setupWork func(t *testing.T, workspace string)
		wantLinks int
	}{
		{
			name:      "no work directory",
			setupWork: func(t *testing.T, workspace string) { t.Helper() },
			wantLinks: 0,
		},
		{
			name: "with specifications",
			setupWork: func(t *testing.T, workspace string) {
				t.Helper()
				workDir := filepath.Join(workspace, "work", "test-task")
				specDir := filepath.Join(workDir, "specifications")
				err := os.MkdirAll(specDir, 0o755)
				require.NoError(t, err)

				// Create a spec file
				content := `---
title: Test Spec
---

# Test Spec

See [[spec:2]] for details.`
				err = os.WriteFile(filepath.Join(specDir, "specification-1.md"), []byte(content), 0o644)
				require.NoError(t, err)
			},
			wantLinks: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tt.setupWork(t, tmpDir)

			mgr := NewManager(tmpDir)
			_ = mgr.Load()

			err := mgr.Rebuild()
			assert.NoError(t, err)

			stats := mgr.GetStats()
			assert.Equal(t, tt.wantLinks, stats.TotalLinks)
		})
	}
}

// TestManager_Rebuild_NoWorkDir tests rebuilding when work directory doesn't exist.
func TestManager_Rebuild_NoWorkDir(t *testing.T) {
	tmpDir := t.TempDir()

	mgr := NewManager(tmpDir)
	_ = mgr.Load()

	err := mgr.Rebuild()
	assert.NoError(t, err)

	stats := mgr.GetStats()
	assert.Equal(t, 0, stats.TotalLinks)
}

// TestManager_SetAutoSave tests controlling auto-save behavior.
func TestManager_SetAutoSave(t *testing.T) {
	tmpDir := t.TempDir()

	mgr := NewManager(tmpDir)
	_ = mgr.Load()

	// Disable auto-save
	mgr.SetAutoSave(false)

	// Add some content (should not auto-save)
	_ = mgr.IndexContent("spec:task:1", "See [[spec:task:2]]", "task")

	// Verify index file wasn't created (or is empty)
	indexPath := filepath.Join(tmpDir, "links", IndexFileName)
	if _, err := os.Stat(indexPath); err == nil {
		data, _ := os.ReadFile(indexPath)
		// File should be empty or not contain our data
		assert.NotContains(t, string(data), "spec:task:1")
	}

	// Enable auto-save
	mgr.SetAutoSave(true)

	// Add more content (should auto-save)
	_ = mgr.IndexContent("spec:task:2", "No links", "task")

	// Verify file now exists and contains data
	data, err := os.ReadFile(indexPath)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)
}

// TestManager_GetIndex tests getting a copy of the index.
func TestManager_GetIndex(t *testing.T) {
	tmpDir := t.TempDir()

	mgr := NewManager(tmpDir)
	_ = mgr.Load()

	_ = mgr.IndexContent("spec:task:1", "See [[spec:task:2]]", "task")

	index := mgr.GetIndex()
	assert.NotNil(t, index)
	assert.Equal(t, 1, len(index.Forward))

	// Modifying the returned index should not affect the manager
	index.Forward["test"] = []Link{}

	original := mgr.GetIndex()
	_, found := original.Forward["test"]
	assert.False(t, found, "modifying returned index should not affect manager")
}

// TestManager_GetNames tests getting a copy of the name registry.
func TestManager_GetNames(t *testing.T) {
	tmpDir := t.TempDir()

	mgr := NewManager(tmpDir)
	_ = mgr.Load()

	_ = mgr.RegisterName(TypeSpec, "Test", "spec:task:1")

	names := mgr.GetNames()
	assert.NotNil(t, names)
	assert.Equal(t, 1, len(names.Specs))

	// Modifying the returned registry should not affect the manager
	names.Specs["Another"] = "spec:task:2"

	original := mgr.GetNames()
	_, found := original.Specs["Another"]
	assert.False(t, found, "modifying returned registry should not affect manager")
}

// TestManager_ConcurrentAccess tests concurrent access to the manager.
func TestManager_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()

	mgr := NewManager(tmpDir)
	_ = mgr.Load()

	const numGoroutines = 100
	const opsPerGoroutine = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 3) // 3 types of operations

	// Concurrent writes
	for i := range numGoroutines {
		go func(n int) {
			defer wg.Done()
			for range opsPerGoroutine {
				_ = mgr.IndexContent(EntityID(TypeSpec, "task", string(rune('a'+n%26))+string(rune('0'+n%10))), "See [[spec:2]]", "task")
			}
		}(i)
	}

	// Concurrent name registrations
	for i := range numGoroutines {
		go func(n int) {
			defer wg.Done()
			for j := range opsPerGoroutine {
				_ = mgr.RegisterName(TypeSpec, "Spec"+string(rune('a'+n%26))+strconv.Itoa(int(rune('0'+j%10))), "spec:task:"+strconv.Itoa(int(rune('0'+j%10))))
			}
		}(i)
	}

	// Concurrent reads
	for i := range numGoroutines {
		go func(n int) {
			defer wg.Done()
			for range opsPerGoroutine {
				_ = mgr.GetOutgoing("spec:task:1")
				_ = mgr.GetIncoming("spec:task:2")
				_ = mgr.GetStats()
				_, _ = mgr.ResolveName("Spec" + string(rune('a'+n%26)))
			}
		}(i)
	}

	wg.Wait()

	// Verify manager is still functional
	stats := mgr.GetStats()
	assert.NotNil(t, stats)
}

// TestManager_parseSpecTitle tests parsing spec titles from files.
func TestManager_parseSpecTitle(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)
	_ = mgr.Load() // Create the links directory

	tests := []struct {
		name      string
		content   string
		wantTitle string
	}{
		{
			name: "YAML title",
			content: `---
title: Authentication Flow
---
# Header

Content`,
			wantTitle: "Authentication Flow",
		},
		{
			name: "first heading",
			content: `# Authentication Flow

Content here`,
			wantTitle: "Authentication Flow",
		},
		{
			name: "YAML title with quotes",
			content: `---
title: "Authentication Flow"
---
Content`,
			wantTitle: "Authentication Flow",
		},
		{
			name:      "no title",
			content:   `Just content`,
			wantTitle: "Specification 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a spec file with correct naming (1.md, not specification-1.md)
			specPath := filepath.Join(tmpDir, "1.md")
			err := os.WriteFile(specPath, []byte(tt.content), 0o644)
			require.NoError(t, err)

			entityID, title, err := mgr.parseSpecTitle(specPath, "task-123")
			require.NoError(t, err)
			assert.Equal(t, tt.wantTitle, title)
			assert.Equal(t, "spec:task-123:1", entityID)
		})
	}
}

// TestManager_indexTask_WithNotes tests indexing notes.
func TestManager_indexTask_WithNotes(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Create task directory structure
	workDir := filepath.Join(tmpDir, "work", "test-task")
	err := os.MkdirAll(workDir, 0o755)
	require.NoError(t, err)

	// Create a notes file
	content := `# Notes

See [[spec:2]] for details.`
	err = os.WriteFile(filepath.Join(workDir, "notes.md"), []byte(content), 0o644)
	require.NoError(t, err)

	// Index the task
	err = mgr.indexTask("test-task")
	assert.NoError(t, err)

	// Verify links were created
	outgoing := mgr.GetOutgoing("note:test-task:notes")
	assert.Equal(t, 1, len(outgoing))
}

// TestNameRegistry_SolutionAndErrorTypes tests solution and error types.
func TestNameRegistry_SolutionAndErrorTypes(t *testing.T) {
	reg := NewNameRegistry()

	// Solutions and errors are stored under notes
	reg.Register(TypeSolution, "Fix Auth Bug", "solution:task:123")
	reg.Register(TypeError, "Login Error", "error:task:456")

	assert.Equal(t, 2, len(reg.Notes))
	assert.Equal(t, "solution:task:123", reg.Notes["Fix Auth Bug"])
	assert.Equal(t, "error:task:456", reg.Notes["Login Error"])
}

// TestNameRegistry_UnregisterSolutionAndErrorTypes tests unregistering solutions and errors.
func TestNameRegistry_UnregisterSolutionAndErrorTypes(t *testing.T) {
	reg := NewNameRegistry()

	reg.Register(TypeSolution, "Fix", "solution:task:1")
	reg.Register(TypeError, "Error", "error:task:2")

	// Verify registered
	assert.Equal(t, 2, len(reg.Notes))

	// Unregister
	reg.Unregister(TypeSolution, "Fix")
	reg.Unregister(TypeError, "Error")

	// Verify gone
	assert.Equal(t, 0, len(reg.Notes))
}
