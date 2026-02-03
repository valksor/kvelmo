package storage

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valksor/go-mehrhof/internal/links"
)

// TestNewLinkManager tests creating a new link manager.
func TestNewLinkManager(t *testing.T) {
	tests := []struct {
		name           string
		config         *LinksSettings
		expectManager  bool
		expectNilStats bool
		expectNilIndex bool
		expectNilNames bool
	}{
		{
			name:           "nil config",
			config:         nil,
			expectManager:  false,
			expectNilStats: true,
			expectNilIndex: true,
			expectNilNames: true,
		},
		{
			name:           "disabled config",
			config:         &LinksSettings{Enabled: false},
			expectManager:  false,
			expectNilStats: true,
			expectNilIndex: true,
			expectNilNames: true,
		},
		{
			name: "enabled config",
			config: &LinksSettings{
				Enabled: true,
			},
			expectManager:  true,  // Manager is created
			expectNilStats: false, // Stats returns non-nil (empty) when enabled
			expectNilIndex: false, // Index returns non-nil (empty) when enabled
			expectNilNames: false, // Names returns non-nil (empty) when enabled
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			mgr := NewLinkManager(tmpDir, tt.config)

			if mgr == nil {
				t.Fatal("NewLinkManager returned nil")
			}

			// Verify manager methods don't panic
			_ = mgr.GetOutgoing("test")
			_ = mgr.GetIncoming("test")
			stats := mgr.GetStats()

			if tt.expectNilStats {
				assert.Nil(t, stats)
			} else {
				assert.NotNil(t, stats)
				// Verify stats are empty (no links indexed yet)
				assert.Equal(t, 0, stats.TotalLinks)
			}

			index := mgr.GetIndex()
			if tt.expectNilIndex {
				assert.Nil(t, index)
			} else {
				assert.NotNil(t, index)
				assert.Equal(t, 0, len(index.Forward))
				assert.Equal(t, 0, len(index.Backward))
			}

			names := mgr.GetNames()
			if tt.expectNilNames {
				assert.Nil(t, names)
			} else {
				assert.NotNil(t, names)
				assert.Equal(t, 0, len(names.Specs))
				assert.Equal(t, 0, len(names.Sessions))
			}
		})
	}
}

// TestLinkManager_GetLinksConfig tests getting links configuration.
func TestLinkManager_GetLinksConfig(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		config         *WorkspaceConfig
		wantEnabled    bool
		wantAutoIndex  bool
		wantCaseSens   bool
		wantMaxContext int
	}{
		{
			name:           "default config",
			config:         &WorkspaceConfig{},
			wantEnabled:    true,
			wantAutoIndex:  true,
			wantCaseSens:   false,
			wantMaxContext: 200,
		},
		{
			name: "custom config",
			config: &WorkspaceConfig{
				Links: &LinksSettings{
					Enabled:          false,
					AutoIndex:        false,
					CaseSensitive:    true,
					MaxContextLength: 100,
				},
			},
			wantEnabled:    false,
			wantAutoIndex:  false,
			wantCaseSens:   true,
			wantMaxContext: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tt.config.Storage.HomeDir = t.TempDir()
			ws, err := OpenWorkspace(ctx, tmpDir, tt.config)
			require.NoError(t, err)

			// Save the config so LoadConfig will read it back
			err = ws.SaveConfig(tt.config)
			require.NoError(t, err)

			cfg := GetLinksConfig(ctx, ws)
			require.NotNil(t, cfg)

			assert.Equal(t, tt.wantEnabled, cfg.Enabled)
			assert.Equal(t, tt.wantAutoIndex, cfg.AutoIndex)
			assert.Equal(t, tt.wantCaseSens, cfg.CaseSensitive)
			assert.Equal(t, tt.wantMaxContext, cfg.MaxContextLength)
		})
	}
}

// TestLinkManager_IndexContent tests indexing content for links.
func TestLinkManager_IndexContent(t *testing.T) {
	tests := []struct {
		name         string
		enabled      bool
		autoIndex    bool
		sourceID     string
		content      string
		activeTaskID string
		wantOutgoing int
	}{
		{
			name:         "disabled links",
			enabled:      false,
			autoIndex:    true,
			sourceID:     "spec:test:1",
			content:      "See [[spec:test:2]] for details",
			wantOutgoing: 0,
		},
		{
			name:         "auto index disabled",
			enabled:      true,
			autoIndex:    false,
			sourceID:     "spec:test:1",
			content:      "See [[spec:test:2]] for details",
			wantOutgoing: 0,
		},
		{
			name:         "valid link",
			enabled:      true,
			autoIndex:    true,
			sourceID:     "spec:test:1",
			content:      "See [[spec:test:2]] for details",
			activeTaskID: "test",
			wantOutgoing: 1,
		},
		{
			name:         "multiple links",
			enabled:      true,
			autoIndex:    true,
			sourceID:     "spec:test:1",
			content:      "See [[spec:test:2]] and [[spec:test:3]]",
			activeTaskID: "test",
			wantOutgoing: 2,
		},
		{
			name:         "no links",
			enabled:      true,
			autoIndex:    true,
			sourceID:     "spec:test:1",
			content:      "No links here",
			wantOutgoing: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			config := &LinksSettings{
				Enabled:   tt.enabled,
				AutoIndex: tt.autoIndex,
			}
			mgr := NewLinkManager(tmpDir, config)

			err := mgr.IndexContent(tt.sourceID, tt.content, tt.activeTaskID)
			require.NoError(t, err)

			outgoing := mgr.GetOutgoing(tt.sourceID)
			assert.Equal(t, tt.wantOutgoing, len(outgoing))
		})
	}
}

// TestLinkManager_GetOutgoing tests getting outgoing links.
func TestLinkManager_GetOutgoing(t *testing.T) {
	tmpDir := t.TempDir()
	config := &LinksSettings{Enabled: true, AutoIndex: true}
	mgr := NewLinkManager(tmpDir, config)

	// Index some content
	err := mgr.IndexContent("spec:test:1", "See [[spec:test:2]]", "test")
	require.NoError(t, err)

	outgoing := mgr.GetOutgoing("spec:test:1")
	assert.Equal(t, 1, len(outgoing))
	if len(outgoing) > 0 {
		assert.Equal(t, "spec:test:2", outgoing[0].Target)
	}

	// Non-existent source
	empty := mgr.GetOutgoing("nonexistent")
	assert.Nil(t, empty)
}

// TestLinkManager_GetIncoming tests getting incoming links.
func TestLinkManager_GetIncoming(t *testing.T) {
	tmpDir := t.TempDir()
	config := &LinksSettings{Enabled: true, AutoIndex: true}
	mgr := NewLinkManager(tmpDir, config)

	// Index some content
	err := mgr.IndexContent("spec:test:1", "See [[spec:test:2]]", "test")
	require.NoError(t, err)

	incoming := mgr.GetIncoming("spec:test:2")
	assert.Equal(t, 1, len(incoming))
	if len(incoming) > 0 {
		assert.Equal(t, "spec:test:1", incoming[0].Source)
	}

	// Non-existent target
	empty := mgr.GetIncoming("nonexistent")
	assert.Nil(t, empty)
}

// TestLinkManager_RegisterName tests registering entity names.
func TestLinkManager_RegisterName(t *testing.T) {
	tests := []struct {
		name       string
		entityID   string
		entityName string
		entityTyp  links.EntityType
	}{
		{
			name:       "register spec",
			entityID:   "spec:task-123:1",
			entityName: "Authentication Flow",
			entityTyp:  links.TypeSpec,
		},
		{
			name:       "register decision",
			entityID:   "decision:task-123:cache",
			entityName: "Cache Strategy",
			entityTyp:  links.TypeDecision,
		},
		{
			name:       "register session",
			entityID:   "session:task-123:timestamp",
			entityName: "Planning Session",
			entityTyp:  links.TypeSession,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			config := &LinksSettings{Enabled: true}
			mgr := NewLinkManager(tmpDir, config)

			err := mgr.RegisterName(tt.entityTyp, tt.entityName, tt.entityID)
			require.NoError(t, err)

			// Verify lookup
			entityID, found := mgr.ResolveName(tt.entityName)
			assert.True(t, found, "name should be found")
			assert.Equal(t, tt.entityID, entityID)
		})
	}
}

// TestLinkManager_ResolveName tests resolving entity names.
func TestLinkManager_ResolveName(t *testing.T) {
	tmpDir := t.TempDir()
	config := &LinksSettings{Enabled: true}
	mgr := NewLinkManager(tmpDir, config)

	// Register some names
	_ = mgr.RegisterName(links.TypeSpec, "Auth Spec", "spec:task-123:1")
	_ = mgr.RegisterName(links.TypeDecision, "Cache Strategy", "decision:task-123:cache")

	tests := []struct {
		name         string
		search       string
		wantFound    bool
		wantEntityID string
	}{
		{
			name:         "exact match spec",
			search:       "Auth Spec",
			wantFound:    true,
			wantEntityID: "spec:task-123:1",
		},
		{
			name:         "exact match decision",
			search:       "Cache Strategy",
			wantFound:    true,
			wantEntityID: "decision:task-123:cache",
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

// TestLinkManager_GetStats tests getting link statistics.
func TestLinkManager_GetStats(t *testing.T) {
	tmpDir := t.TempDir()
	config := &LinksSettings{Enabled: true, AutoIndex: true}
	mgr := NewLinkManager(tmpDir, config)

	// Initially no stats
	stats := mgr.GetStats()
	assert.NotNil(t, stats)
	assert.Equal(t, 0, stats.TotalLinks)

	// Index some content
	_ = mgr.IndexContent("spec:test:1", "See [[spec:test:2]] and [[spec:test:3]]", "test")

	stats = mgr.GetStats()
	assert.NotNil(t, stats)
	assert.Equal(t, 2, stats.TotalLinks)
	assert.Equal(t, 1, stats.TotalSources)
}

// TestLinkManager_Save tests saving the link index.
func TestLinkManager_Save(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{
			name:    "enabled",
			enabled: true,
		},
		{
			name:    "disabled",
			enabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			config := &LinksSettings{Enabled: tt.enabled}
			mgr := NewLinkManager(tmpDir, config)

			// Save should not error
			err := mgr.Save()
			assert.NoError(t, err)
		})
	}
}

// TestLinkManager_GetIndex tests getting the link index.
func TestLinkManager_GetIndex(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{
			name:    "enabled",
			enabled: true,
		},
		{
			name:    "disabled",
			enabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			config := &LinksSettings{Enabled: tt.enabled}
			mgr := NewLinkManager(tmpDir, config)

			index := mgr.GetIndex()

			if tt.enabled {
				assert.NotNil(t, index)
				// Empty index has no forward/backward links
				assert.Equal(t, 0, len(index.Forward))
				assert.Equal(t, 0, len(index.Backward))
			} else {
				assert.Nil(t, index)
			}
		})
	}
}

// TestLinkManager_GetNames tests getting the name registry.
func TestLinkManager_GetNames(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{
			name:    "enabled",
			enabled: true,
		},
		{
			name:    "disabled",
			enabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			config := &LinksSettings{Enabled: tt.enabled}
			mgr := NewLinkManager(tmpDir, config)

			names := mgr.GetNames()

			if tt.enabled {
				assert.NotNil(t, names)
				// Empty registry has no entries
				assert.Equal(t, 0, len(names.Specs))
				assert.Equal(t, 0, len(names.Decisions))
			} else {
				assert.Nil(t, names)
			}
		})
	}
}

// TestLinkManager_Rebuild tests rebuilding the link index.
func TestLinkManager_Rebuild(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{
			name:    "enabled",
			enabled: true,
		},
		{
			name:    "disabled",
			enabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			config := &LinksSettings{Enabled: tt.enabled}
			mgr := NewLinkManager(tmpDir, config)

			// Rebuild should not error (may be empty)
			err := mgr.Rebuild()
			assert.NoError(t, err)
		})
	}
}

// TestIndexSpecification tests indexing a specification.
func TestIndexSpecification(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	ws, err := OpenWorkspace(ctx, tmpDir, &WorkspaceConfig{
		Links:   &LinksSettings{Enabled: true, AutoIndex: true},
		Storage: StorageSettings{HomeDir: t.TempDir()},
	})
	require.NoError(t, err)

	// Create a task
	taskID := "test-task-spec"
	_, err = ws.CreateWork(taskID, SourceInfo{Type: "file", Ref: "test.md"})
	require.NoError(t, err)

	tests := []struct {
		name      string
		content   string
		specNum   int
		wantLinks int
	}{
		{
			name:      "no links",
			content:   "# Specification\n\nNo links here.",
			specNum:   1,
			wantLinks: 0,
		},
		{
			name:      "with link",
			content:   "# Specification\n\nSee [[spec:2]] for details.",
			specNum:   1,
			wantLinks: 1,
		},
		{
			name:      "with YAML title",
			content:   "---\ntitle: Auth Flow\n---\n\nSee [[spec:2]].",
			specNum:   1,
			wantLinks: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			IndexSpecification(ctx, ws, taskID, tt.specNum, tt.content)

			linkMgr := GetLinkManager(ctx, ws)
			entityID := links.EntityID(links.TypeSpec, taskID, string(rune('0'+tt.specNum)))
			outgoing := linkMgr.GetOutgoing(entityID)

			assert.Equal(t, tt.wantLinks, len(outgoing))
		})
	}
}

// TestIndexNote tests indexing a note.
func TestIndexNote(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	ws, err := OpenWorkspace(ctx, tmpDir, &WorkspaceConfig{
		Links:   &LinksSettings{Enabled: true, AutoIndex: true},
		Storage: StorageSettings{HomeDir: t.TempDir()},
	})
	require.NoError(t, err)

	// Create a task
	taskID := "test-task-note"
	_, err = ws.CreateWork(taskID, SourceInfo{Type: "file", Ref: "test.md"})
	require.NoError(t, err)

	tests := []struct {
		name      string
		content   string
		wantLinks int
	}{
		{
			name:      "no links",
			content:   "# Notes\n\nNo links here.",
			wantLinks: 0,
		},
		{
			name:      "with link",
			content:   "# Notes\n\nSee [[spec:test-task-note:1]] for details.",
			wantLinks: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			IndexNote(ctx, ws, taskID, tt.content)

			linkMgr := GetLinkManager(ctx, ws)
			noteID := links.EntityID(links.TypeNote, taskID, "notes")
			outgoing := linkMgr.GetOutgoing(noteID)

			assert.Equal(t, tt.wantLinks, len(outgoing))
		})
	}
}

// TestIndexSession tests indexing a session.
func TestIndexSession(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	ws, err := OpenWorkspace(ctx, tmpDir, &WorkspaceConfig{
		Links:   &LinksSettings{Enabled: true, AutoIndex: true},
		Storage: StorageSettings{HomeDir: t.TempDir()},
	})
	require.NoError(t, err)

	// Create a task
	taskID := "test-task-session"
	_, err = ws.CreateWork(taskID, SourceInfo{Type: "file", Ref: "test.md"})
	require.NoError(t, err)

	tests := []struct {
		name      string
		content   string
		wantLinks int
	}{
		{
			name:      "no links",
			content:   "# Session\n\nNo links here.",
			wantLinks: 0,
		},
		{
			name:      "with link",
			content:   "# Session\n\nSee [[decision:cache]] for details.",
			wantLinks: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			IndexSession(ctx, ws, taskID, []byte(tt.content))

			linkMgr := GetLinkManager(ctx, ws)
			sessionID := links.EntityID(links.TypeSession, taskID, "latest")
			outgoing := linkMgr.GetOutgoing(sessionID)

			assert.Equal(t, tt.wantLinks, len(outgoing))
		})
	}
}

// TestExtractSpecTitle tests extracting titles from specification content.
func TestExtractSpecTitle(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantTitle string
	}{
		{
			name:      "no title",
			content:   "Just some content",
			wantTitle: "",
		},
		{
			name:      "YAML frontmatter title",
			content:   "---\ntitle: Authentication Flow\n---\n\nContent here.",
			wantTitle: "Authentication Flow",
		},
		{
			name:      "YAML frontmatter with quoted title",
			content:   "---\ntitle: \"Authentication Flow\"\n---\n\nContent here.",
			wantTitle: "Authentication Flow",
		},
		{
			name:      "first heading",
			content:   "# Authentication Flow\n\nContent here.",
			wantTitle: "Authentication Flow",
		},
		{
			name:      "YAML title takes precedence",
			content:   "---\ntitle: YAML Title\n---\n\n# Heading Title\n\nContent.",
			wantTitle: "YAML Title",
		},
		{
			name:      "YAML with single quotes",
			content:   "---\ntitle: 'Auth Flow'\n---\n\nContent.",
			wantTitle: "Auth Flow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSpecTitle(tt.content)
			assert.Equal(t, tt.wantTitle, got)
		})
	}
}

// TestExtractSpecTitle_case_insensitive tests case-insensitive title key matching.
func TestExtractSpecTitle_case_insensitive(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantTitle string
	}{
		{
			name:      "lowercase title key",
			content:   "---\ntitle: Test Title\n---\n\nContent.",
			wantTitle: "Test Title",
		},
		{
			name:      "uppercase title key",
			content:   "---\nTITLE: Test Title\n---\n\nContent.",
			wantTitle: "Test Title",
		},
		{
			name:      "mixed case title key",
			content:   "---\nTitle: Test Title\n---\n\nContent.",
			wantTitle: "Test Title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSpecTitle(tt.content)
			assert.Equal(t, tt.wantTitle, got)
		})
	}
}

// TestGetLinkManager tests getting the link manager for a workspace.
func TestGetLinkManager(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	tests := []struct {
		name   string
		config *WorkspaceConfig
	}{
		{
			name:   "nil links config",
			config: &WorkspaceConfig{},
		},
		{
			name: "disabled links",
			config: &WorkspaceConfig{
				Links: &LinksSettings{Enabled: false},
			},
		},
		{
			name: "enabled links",
			config: &WorkspaceConfig{
				Links: &LinksSettings{Enabled: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.config.Storage.HomeDir = t.TempDir()
			ws, err := OpenWorkspace(ctx, tmpDir, tt.config)
			require.NoError(t, err)

			mgr := GetLinkManager(ctx, ws)
			assert.NotNil(t, mgr)
		})
	}
}

// TestLinkManager_Integration tests the full integration workflow.
func TestLinkManager_Integration(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	config := &WorkspaceConfig{
		Links: &LinksSettings{
			Enabled:          true,
			AutoIndex:        true,
			CaseSensitive:    false,
			MaxContextLength: 200,
		},
		Storage: StorageSettings{HomeDir: t.TempDir()},
	}
	ws, err := OpenWorkspace(ctx, tmpDir, config)
	require.NoError(t, err)

	// Save the config so GetLinksConfig will read it back from disk
	err = ws.SaveConfig(config)
	require.NoError(t, err)

	// Create a task
	taskID := "test-task-integration"
	_, err = ws.CreateWork(taskID, SourceInfo{Type: "file", Ref: "test.md"})
	require.NoError(t, err)

	// Index a specification with links
	content := `---
title: Authentication Flow
---

# Authentication Flow

This spec describes the login process. See [[spec:test-task-integration:2]] for API details.

Related decisions:
- [[decision:cache-strategy]] for session caching
`
	IndexSpecification(ctx, ws, taskID, 1, content)

	// Verify links were created
	linkMgr := GetLinkManager(ctx, ws)
	entityID := "spec:test-task-integration:1"
	outgoing := linkMgr.GetOutgoing(entityID)

	assert.Equal(t, 2, len(outgoing), "should have 2 outgoing links")

	// Verify title was registered
	names := linkMgr.GetNames()
	assert.NotNil(t, names)
	title, found := names.Specs["Authentication Flow"]
	assert.True(t, found, "title should be registered")
	assert.Equal(t, entityID, title)
}

// TestLinkManager_ContextExtraction tests context extraction from content.
func TestLinkManager_ContextExtraction(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	config := &WorkspaceConfig{
		Links: &LinksSettings{
			Enabled:          true,
			AutoIndex:        true,
			MaxContextLength: 100, // Enough to include full link + surrounding text
		},
		Storage: StorageSettings{HomeDir: t.TempDir()},
	}
	ws, err := OpenWorkspace(ctx, tmpDir, config)
	require.NoError(t, err)

	// Save the config so GetLinksConfig will read it back from disk
	err = ws.SaveConfig(config)
	require.NoError(t, err)

	// Create a task
	taskID := "test-task-context"
	_, err = ws.CreateWork(taskID, SourceInfo{Type: "file", Ref: "test.md"})
	require.NoError(t, err)

	// Content with link in the middle - use short link ref to fit in context
	content := "Some text before the link. Here: [[spec:2]]. And more text after the link reference."
	IndexSpecification(ctx, ws, taskID, 1, content)

	// Verify context was extracted
	linkMgr := GetLinkManager(ctx, ws)
	entityID := "spec:test-task-context:1"
	outgoing := linkMgr.GetOutgoing(entityID)

	assert.Equal(t, 1, len(outgoing), "should have 1 outgoing link")
	if len(outgoing) > 0 {
		context := outgoing[0].Context
		assert.NotEmpty(t, context, "context should not be empty")
		// Context may include ellipsis (...) so check raw length is reasonable
		assert.True(t, len(context) <= 120, "context should be limited to MaxContextLength (+ellipsis)")
		assert.Contains(t, context, "[[spec:2]]", "context should contain the link")
	}
}

// TestLinkManager_TaskScopedReferences tests task-scoped reference resolution.
func TestLinkManager_TaskScopedReferences(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	config := &WorkspaceConfig{
		Links: &LinksSettings{
			Enabled:   true,
			AutoIndex: true,
		},
		Storage: StorageSettings{HomeDir: t.TempDir()},
	}
	ws, err := OpenWorkspace(ctx, tmpDir, config)
	require.NoError(t, err)

	// Save the config so GetLinksConfig will read it back from disk
	err = ws.SaveConfig(config)
	require.NoError(t, err)

	// Create a task
	taskID := "test-task-scoped"
	_, err = ws.CreateWork(taskID, SourceInfo{Type: "file", Ref: "test.md"})
	require.NoError(t, err)

	// Content with task-scoped reference
	content := "See [[spec:2]] for details."
	IndexSpecification(ctx, ws, taskID, 1, content)

	// Verify link was created with full entity ID
	linkMgr := GetLinkManager(ctx, ws)
	entityID := "spec:test-task-scoped:1"
	outgoing := linkMgr.GetOutgoing(entityID)

	assert.Equal(t, 1, len(outgoing), "should have 1 outgoing link")
	if len(outgoing) > 0 {
		assert.Equal(t, "spec:test-task-scoped:2", outgoing[0].Target, "task-scoped reference should be resolved to full entity ID")
	}
}

// TestLinkManager_NameBasedReferences tests name-based reference resolution.
func TestLinkManager_NameBasedReferences(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	config := &WorkspaceConfig{
		Links: &LinksSettings{
			Enabled:   true,
			AutoIndex: true,
		},
		Storage: StorageSettings{HomeDir: t.TempDir()},
	}
	ws, err := OpenWorkspace(ctx, tmpDir, config)
	require.NoError(t, err)

	// Save the config so GetLinksConfig will read it back from disk
	err = ws.SaveConfig(config)
	require.NoError(t, err)

	// Create a task
	taskID := "test-task-name"
	_, err = ws.CreateWork(taskID, SourceInfo{Type: "file", Ref: "test.md"})
	require.NoError(t, err)

	// First, register a name for spec 2
	linkMgr := GetLinkManager(ctx, ws)
	_ = linkMgr.RegisterName(links.TypeSpec, "API Design", "spec:test-task-name:2")
	// Save to ensure name is persisted before IndexSpecification loads a new manager
	_ = linkMgr.Save()

	// Content with name-based reference
	content := "See [[API Design]] for details."
	IndexSpecification(ctx, ws, taskID, 1, content)

	// Verify link was created - get fresh manager to load persisted state
	linkMgr = GetLinkManager(ctx, ws)
	entityID := "spec:test-task-name:1"
	outgoing := linkMgr.GetOutgoing(entityID)

	assert.Equal(t, 1, len(outgoing), "should have 1 outgoing link")
	if len(outgoing) > 0 {
		assert.Equal(t, "spec:test-task-name:2", outgoing[0].Target, "name-based reference should be resolved")
	}
}

// TestLinkManager_UpdateReindex tests reindexing content updates links.
func TestLinkManager_UpdateReindex(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	config := &WorkspaceConfig{
		Links: &LinksSettings{
			Enabled:   true,
			AutoIndex: true,
		},
		Storage: StorageSettings{HomeDir: t.TempDir()},
	}
	ws, err := OpenWorkspace(ctx, tmpDir, config)
	require.NoError(t, err)

	// Save the config so GetLinksConfig will read it back from disk
	err = ws.SaveConfig(config)
	require.NoError(t, err)

	// Create a task
	taskID := "test-task-reindex"
	_, err = ws.CreateWork(taskID, SourceInfo{Type: "file", Ref: "test.md"})
	require.NoError(t, err)

	entityID := "spec:test-task-reindex:1"

	// Index initial content
	content1 := "See [[spec:2]] for details."
	IndexSpecification(ctx, ws, taskID, 1, content1)

	// Get fresh manager to see persisted state
	linkMgr := GetLinkManager(ctx, ws)
	outgoing1 := linkMgr.GetOutgoing(entityID)
	assert.Equal(t, 1, len(outgoing1), "should have 1 link")

	// Reindex with different content
	content2 := "See [[spec:2]] and [[spec:3]] for details."
	IndexSpecification(ctx, ws, taskID, 1, content2)

	// Refresh manager to see updated state
	linkMgr = GetLinkManager(ctx, ws)
	outgoing2 := linkMgr.GetOutgoing(entityID)
	assert.Equal(t, 2, len(outgoing2), "should have 2 links after reindex")

	// Reindex with no links
	content3 := "No links here."
	IndexSpecification(ctx, ws, taskID, 1, content3)

	// Refresh manager to see updated state
	linkMgr = GetLinkManager(ctx, ws)
	outgoing3 := linkMgr.GetOutgoing(entityID)
	assert.Equal(t, 0, len(outgoing3), "should have no links after reindexing with no links")
}

// TestLinkManager_Persistence tests saving and loading link data.
func TestLinkManager_Persistence(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	homeDir := t.TempDir()

	// Create first manager and add data
	ws1, err := OpenWorkspace(ctx, tmpDir, &WorkspaceConfig{
		Links:   &LinksSettings{Enabled: true},
		Storage: StorageSettings{HomeDir: homeDir},
	})
	require.NoError(t, err)

	taskID := "test-task-persist"
	_, err = ws1.CreateWork(taskID, SourceInfo{Type: "file", Ref: "test.md"})
	require.NoError(t, err)

	content := "See [[spec:2]] for details."
	IndexSpecification(ctx, ws1, taskID, 1, content)

	// Save the link manager
	linkMgr1 := GetLinkManager(ctx, ws1)
	err = linkMgr1.Save()
	require.NoError(t, err)

	// Create a new manager for the same workspace
	ws2, err := OpenWorkspace(ctx, tmpDir, &WorkspaceConfig{
		Links:   &LinksSettings{Enabled: true},
		Storage: StorageSettings{HomeDir: homeDir},
	})
	require.NoError(t, err)

	linkMgr2 := GetLinkManager(ctx, ws2)
	entityID := "spec:test-task-persist:1"
	outgoing := linkMgr2.GetOutgoing(entityID)

	assert.Equal(t, 1, len(outgoing), "should have loaded persisted link")
	if len(outgoing) > 0 {
		assert.Equal(t, "spec:test-task-persist:2", outgoing[0].Target)
	}
}

// TestSplitLines tests the splitLines helper.
func TestSplitLines(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantLen int
	}{
		{
			name:    "empty string",
			input:   "",
			wantLen: 1, // Split returns []string{""}
		},
		{
			name:    "single line",
			input:   "single line",
			wantLen: 1,
		},
		{
			name:    "multiple lines",
			input:   "line1\nline2\nline3",
			wantLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitLines(tt.input)
			assert.Equal(t, tt.wantLen, len(result))
		})
	}
}

// TestIndexOfString tests the indexOfString helper.
func TestIndexOfString(t *testing.T) {
	tests := []struct {
		name    string
		s       string
		substr  string
		wantIdx int
	}{
		{
			name:    "found",
			s:       "Hello World",
			substr:  "world",
			wantIdx: 6,
		},
		{
			name:    "not found",
			s:       "Hello World",
			substr:  "xyz",
			wantIdx: -1,
		},
		{
			name:    "case insensitive",
			s:       "HELLO WORLD",
			substr:  "world",
			wantIdx: 6,
		},
		{
			name:    "empty substring",
			s:       "Hello",
			substr:  "",
			wantIdx: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := indexOfString(tt.s, tt.substr)
			assert.Equal(t, tt.wantIdx, got)
		})
	}
}

// TestLinkManager_ConcurrentIndexing tests concurrent indexing operations.
func TestLinkManager_ConcurrentIndexing(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	ws, err := OpenWorkspace(ctx, tmpDir, &WorkspaceConfig{
		Links: &LinksSettings{
			Enabled:   true,
			AutoIndex: true,
		},
		Storage: StorageSettings{HomeDir: t.TempDir()},
	})
	require.NoError(t, err)

	// Create multiple tasks
	const numTasks = 10
	const specsPerTask = 5

	for i := range numTasks {
		taskID := strings.Join([]string{"test-task-concurrent", strconv.Itoa(int(rune('a' + i)))}, "-")
		_, err := ws.CreateWork(taskID, SourceInfo{Type: "file", Ref: "test.md"})
		require.NoError(t, err)

		// Index multiple specs concurrently
		done := make(chan bool, specsPerTask)
		for j := range specsPerTask {
			go func(specNum int) {
				defer func() { done <- true }()
				content := "See [[spec:2]] for details."
				IndexSpecification(ctx, ws, taskID, specNum, content)
			}(j)
		}

		// Wait for all goroutines
		for range specsPerTask {
			<-done
		}
	}

	// Verify stats
	linkMgr := GetLinkManager(ctx, ws)
	stats := linkMgr.GetStats()
	assert.NotNil(t, stats)
	// We expect at least some links to be created
	assert.GreaterOrEqual(t, stats.TotalSources, 1)
}
