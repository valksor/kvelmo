package storage

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/valksor/go-mehrhof/internal/links"
)

// LinkManager manages the link index for a workspace.
// This is a lightweight wrapper around the links.Manager
// that integrates with the storage layer.
type LinkManager struct {
	manager *links.Manager
	config  *LinksSettings
}

// NewLinkManager creates a new link manager for a workspace.
func NewLinkManager(workspace string, config *LinksSettings) *LinkManager {
	var mgr *links.Manager
	if config != nil && config.Enabled {
		mgr = links.NewManager(workspace)
		if err := mgr.Load(); err != nil {
			slog.Warn("failed to load link index", "error", err)
			// Continue with empty manager
		}
		// Apply config settings
		if config.MaxContextLength > 0 {
			mgr.SetMaxContextLength(config.MaxContextLength)
		}
	}

	return &LinkManager{
		manager: mgr,
		config:  config,
	}
}

// IndexContent indexes content for references.
// If links are disabled, this is a no-op.
func (lm *LinkManager) IndexContent(sourceEntityID, content, activeTaskID string) error {
	if lm == nil || lm.manager == nil || lm.config == nil || !lm.config.Enabled {
		return nil // Links disabled
	}

	if !lm.config.AutoIndex {
		return nil // Auto-indexing disabled
	}

	return lm.manager.IndexContent(sourceEntityID, content, activeTaskID)
}

// GetOutgoing returns outgoing links for an entity.
func (lm *LinkManager) GetOutgoing(entityID string) []links.Link {
	if lm == nil || lm.manager == nil {
		return nil
	}

	return lm.manager.GetOutgoing(entityID)
}

// GetIncoming returns incoming links for an entity.
func (lm *LinkManager) GetIncoming(entityID string) []links.Link {
	if lm == nil || lm.manager == nil {
		return nil
	}

	return lm.manager.GetIncoming(entityID)
}

// RegisterName registers a name → entity ID mapping.
func (lm *LinkManager) RegisterName(entityType links.EntityType, name, entityID string) error {
	if lm == nil || lm.manager == nil {
		return nil
	}

	return lm.manager.RegisterName(entityType, name, entityID)
}

// ResolveName looks up an entity by human-readable name.
func (lm *LinkManager) ResolveName(name string) (string, bool) {
	if lm == nil || lm.manager == nil {
		return "", false
	}

	return lm.manager.ResolveName(name)
}

// Rebuild rebuilds the link index from workspace content.
func (lm *LinkManager) Rebuild() error {
	if lm == nil || lm.manager == nil {
		return nil
	}

	return lm.manager.Rebuild()
}

// Save saves the link index to disk.
func (lm *LinkManager) Save() error {
	if lm == nil || lm.manager == nil {
		return nil
	}

	return lm.manager.Save()
}

// GetStats returns statistics about the link index.
func (lm *LinkManager) GetStats() *links.IndexStats {
	if lm == nil || lm.manager == nil {
		return nil
	}

	stats := lm.manager.GetStats()

	return &stats
}

// GetIndex returns a copy of the link index.
func (lm *LinkManager) GetIndex() *links.LinkIndex {
	if lm == nil || lm.manager == nil {
		return nil
	}

	return lm.manager.GetIndex()
}

// GetNames returns a copy of the name registry.
func (lm *LinkManager) GetNames() *links.NameRegistry {
	if lm == nil || lm.manager == nil {
		return nil
	}

	return lm.manager.GetNames()
}

// GetLinksConfig returns the links configuration from workspace config.
// Returns nil if links are not configured.
func GetLinksConfig(ctx context.Context, ws *Workspace) *LinksSettings {
	cfg, err := ws.LoadConfig()
	if err != nil {
		return nil
	}

	if cfg.Links == nil {
		// Return default config
		return &LinksSettings{
			Enabled:          true,
			AutoIndex:        true,
			CaseSensitive:    false,
			MaxContextLength: 200,
		}
	}

	// Apply sensible defaults when links are enabled
	result := *cfg.Links // Copy to avoid modifying original
	if result.Enabled {
		// AutoIndex defaults to true when links are enabled
		result.AutoIndex = true
		if result.MaxContextLength == 0 {
			result.MaxContextLength = 200
		}
	}

	return &result
}

// IndexSpecification indexes a specification for links.
// This is called after saving a specification.
func IndexSpecification(ctx context.Context, ws *Workspace, taskID string, number int, content string) {
	linksCfg := GetLinksConfig(ctx, ws)
	if linksCfg == nil || !linksCfg.Enabled {
		return
	}

	lm := NewLinkManager(ws.workspaceRoot, linksCfg)
	entityID := fmt.Sprintf("spec:%s:%d", taskID, number)

	if err := lm.IndexContent(entityID, content, taskID); err != nil {
		slog.Warn("failed to index specification links",
			"task_id", taskID,
			"spec_number", number,
			"error", err,
		)
	}

	// Also register the spec by title if we can extract it
	title := extractSpecTitle(content)
	if title != "" {
		if err := lm.RegisterName(links.TypeSpec, title, entityID); err != nil {
			slog.Warn("failed to register spec name", "title", title, "error", err)
		}
	}

	if err := lm.Save(); err != nil {
		slog.Warn("failed to save link index", "error", err)
	}
}

// IndexNote indexes a note for links.
// This is called after appending a note.
func IndexNote(ctx context.Context, ws *Workspace, taskID, content string) {
	linksCfg := GetLinksConfig(ctx, ws)
	if linksCfg == nil || !linksCfg.Enabled {
		return
	}

	lm := NewLinkManager(ws.workspaceRoot, linksCfg)
	entityID := fmt.Sprintf("note:%s:notes", taskID)

	if err := lm.IndexContent(entityID, content, taskID); err != nil {
		slog.Warn("failed to index note links",
			"task_id", taskID,
			"error", err,
		)
	}

	if err := lm.Save(); err != nil {
		slog.Warn("failed to save link index", "error", err)
	}
}

// IndexSession indexes a session for links.
// This is called after saving a session.
func IndexSession(ctx context.Context, ws *Workspace, taskID string, sessionContent []byte) {
	linksCfg := GetLinksConfig(ctx, ws)
	if linksCfg == nil || !linksCfg.Enabled {
		return
	}

	lm := NewLinkManager(ws.workspaceRoot, linksCfg)
	// Use session timestamp as ID (or generate one)
	entityID := fmt.Sprintf("session:%s:latest", taskID)

	if err := lm.IndexContent(entityID, string(sessionContent), taskID); err != nil {
		slog.Warn("failed to index session links",
			"task_id", taskID,
			"error", err,
		)
	}

	if err := lm.Save(); err != nil {
		slog.Warn("failed to save link index", "error", err)
	}
}

// extractSpecTitle extracts the title from a specification's content.
// It checks YAML frontmatter first, then falls back to the first heading.
func extractSpecTitle(content string) string {
	// Check for YAML frontmatter
	if len(content) > 4 && content[:4] == "---\n" {
		endIdx := -1
		for i := 4; i < len(content)-3; i++ {
			if content[i:i+4] == "\n---" {
				endIdx = i

				break
			}
		}

		if endIdx > 0 {
			frontmatter := content[4:endIdx]
			lines := splitLines(frontmatter)
			for _, line := range lines {
				if idx := indexOfString(line, "title:"); idx >= 0 {
					title := strings.TrimSpace(line[idx+6:])
					// Remove quotes if present
					if len(title) > 0 && (title[0] == '"' || title[0] == '\'') {
						title = title[1:]
					}
					if len(title) > 0 && (title[len(title)-1] == '"' || title[len(title)-1] == '\'') {
						title = title[:len(title)-1]
					}

					return title
				}
			}
		}
	}

	// Fall back to first heading
	lines := splitLines(content)
	for _, line := range lines {
		if len(line) > 2 && line[0:2] == "# " {
			return strings.TrimSpace(line[2:])
		}
	}

	return ""
}

// splitLines splits content into lines.
func splitLines(content string) []string {
	return strings.Split(content, "\n")
}

// indexOfString finds the index of a substring in a string (case-insensitive).
func indexOfString(s, substr string) int {
	low := strings.ToLower(s)
	lowSub := strings.ToLower(substr)
	idx := strings.Index(low, lowSub)

	return idx
}

// GetLinkManager returns a link manager for the workspace.
// This is useful for external access to the link system.
func GetLinkManager(ctx context.Context, ws *Workspace) *LinkManager {
	linksCfg := GetLinksConfig(ctx, ws)

	return NewLinkManager(ws.workspaceRoot, linksCfg)
}
