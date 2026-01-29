package links

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// IndexVersion is the current index format version.
	IndexVersion = 1
	// IndexFileName is the name of the link index file.
	IndexFileName = "index.json"
	// NamesFileName is the name of the name registry file.
	NamesFileName = "names.json"
)

// Manager manages the link index and name registry with file persistence.
type Manager struct {
	workspace        string
	linksDir         string
	index            *LinkIndex
	names            *NameRegistry
	mu               sync.RWMutex
	autoSave         bool // Automatically save after modifications
	maxContextLength int  // Maximum context length for link extraction (0 = default 200)
}

// NewManager creates a new link index manager for the given workspace.
// The workspace path should be the root of the mehrhof workspace
// (e.g., ~/.valksor/mehrhof/workspaces/<project-id>).
func NewManager(workspace string) *Manager {
	linksDir := filepath.Join(workspace, "links")

	return &Manager{
		workspace: workspace,
		linksDir:  linksDir,
		index:     NewLinkIndex(),
		names:     NewNameRegistry(),
		autoSave:  true,
	}
}

// SetMaxContextLength sets the maximum context length for link extraction.
// If set to 0 or negative, defaults to 200 characters.
func (m *Manager) SetMaxContextLength(maxLen int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.maxContextLength = maxLen
}

// getMaxContextLength returns the configured max context length or default.
func (m *Manager) getMaxContextLength() int {
	if m.maxContextLength <= 0 {
		return 200 // default
	}

	return m.maxContextLength
}

// Load loads the index from disk. If no index exists, returns an empty one.
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Ensure links directory exists
	if err := os.MkdirAll(m.linksDir, 0o755); err != nil {
		return fmt.Errorf("create links directory: %w", err)
	}

	// Load index
	if err := m.loadIndex(); err != nil {
		// If index doesn't exist, initialize empty
		if os.IsNotExist(err) {
			m.index = NewLinkIndex()
		} else {
			return err
		}
	}

	// Load name registry (only if not already loaded from index)
	// Names are now stored in IndexData, but we check for legacy separate file too
	if m.names == nil || (len(m.names.Specs) == 0 && len(m.names.Sessions) == 0 && len(m.names.Decisions) == 0) {
		if err := m.loadNames(); err != nil {
			// If registry doesn't exist, initialize empty
			if os.IsNotExist(err) {
				if m.names == nil {
					m.names = NewNameRegistry()
				}
			} else {
				return err
			}
		}
	}

	return nil
}

// loadIndex loads the link index from disk.
func (m *Manager) loadIndex() error {
	path := filepath.Join(m.linksDir, IndexFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var indexData IndexData
	if err := json.Unmarshal(data, &indexData); err != nil {
		return err
	}

	// Version check
	if indexData.Version != IndexVersion {
		// Handle migration if needed
		return fmt.Errorf("unsupported index version: %d", indexData.Version)
	}

	m.index = &LinkIndex{
		Forward:  indexData.Forward,
		Backward: indexData.Backward,
	}
	if indexData.Names != nil {
		m.names = indexData.Names
	}

	return nil
}

// loadNames loads the name registry from disk.
func (m *Manager) loadNames() error {
	path := filepath.Join(m.linksDir, NamesFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, m.names)
}

// Save saves the index to disk.
func (m *Manager) Save() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.saveUnsafe()
}

// saveUnsafe saves without locking (internal use only, caller must hold lock).
func (m *Manager) saveUnsafe() error {
	// Save index
	indexData := IndexData{
		Version:   IndexVersion,
		Forward:   m.index.Forward,
		Backward:  m.index.Backward,
		Names:     m.names,
		UpdatedAt: time.Now(),
	}

	data, err := json.MarshalIndent(indexData, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal index: %w", err)
	}

	// Write to temporary file first
	indexPath := filepath.Join(m.linksDir, IndexFileName)
	tmpPath := indexPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("write index temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, indexPath); err != nil {
		//nolint:errcheck // Best-effort cleanup, rename already failed
		os.Remove(tmpPath)

		return fmt.Errorf("rename index file: %w", err)
	}

	return nil
}

// IndexContent parses content for references and adds links to the index.
// The sourceEntityID is the canonical ID of the entity being indexed.
// The activeTaskID is used to resolve task-scoped references.
func (m *Manager) IndexContent(sourceEntityID string, content string, activeTaskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Remove existing links from this source
	m.index.RemoveLinks(sourceEntityID)

	// Parse and create new links
	parsed := Parse(content)
	for _, ref := range parsed.References {
		targetID, ok := m.resolveReference(ref, activeTaskID)
		if !ok {
			continue // Skip unresolvable references
		}

		// Add link
		link := Link{
			Source:    sourceEntityID,
			Target:    targetID,
			Context:   ExtractContext(content, ref.Position, m.getMaxContextLength()),
			CreatedAt: time.Now(),
		}
		m.index.AddLink(link)
	}

	// Auto-save if enabled
	if m.autoSave {
		return m.saveUnsafe()
	}

	return nil
}

// resolveReference resolves a reference using the name registry.
func (m *Manager) resolveReference(ref Reference, activeTaskID string) (string, bool) {
	// If it's a typed reference, construct entity ID directly
	if ref.Type != "" {
		taskID := ref.TaskID
		if ref.IsTaskScoped() {
			if activeTaskID == "" {
				return "", false
			}
			taskID = activeTaskID
		}

		return EntityID(ref.Type, taskID, ref.ID), true
	}

	// Name-based reference: look up in registry
	if ref.Name != "" {
		entityID, found := m.names.Resolve(ref.Name)
		if found {
			return entityID, true
		}
	}

	return "", false
}

// GetOutgoing returns all outgoing links from the given entity.
func (m *Manager) GetOutgoing(entityID string) []Link {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.index.GetOutgoing(entityID)
}

// GetIncoming returns all incoming links to the given entity.
func (m *Manager) GetIncoming(entityID string) []Link {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.index.GetIncoming(entityID)
}

// RegisterName registers a name → entity ID mapping.
func (m *Manager) RegisterName(entityType EntityType, name, entityID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.names.Register(entityType, name, entityID)

	if m.autoSave {
		return m.saveUnsafe()
	}

	return nil
}

// UnregisterName removes a name mapping.
func (m *Manager) UnregisterName(entityType EntityType, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.names.Unregister(entityType, name)

	if m.autoSave {
		return m.saveUnsafe()
	}

	return nil
}

// ResolveName looks up an entity by human-readable name.
func (m *Manager) ResolveName(name string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.names.Resolve(name)
}

// GetStats returns statistics about the link index.
func (m *Manager) GetStats() IndexStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.index.Stats()
}

// Rebuild rebuilds the index by scanning all workspace content.
// This is useful after manual edits or migration.
func (m *Manager) Rebuild() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clear existing index
	m.index = NewLinkIndex()
	m.names = NewNameRegistry()

	// Scan workspace for content
	workDir := filepath.Join(m.workspace, "work")
	entries, err := os.ReadDir(workDir)
	if err != nil {
		if os.IsNotExist(err) {
			return m.saveUnsafe() // Save empty index
		}

		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		taskID := entry.Name()
		if err := m.indexTask(taskID); err != nil {
			// Log but continue with other tasks
			continue
		}
	}

	return m.saveUnsafe()
}

// indexTask indexes all content for a single task.
func (m *Manager) indexTask(taskID string) error {
	// Index specifications
	specsDir := filepath.Join(m.workspace, "work", taskID, "specifications")
	if _, err := os.Stat(specsDir); err == nil {
		entries, err := os.ReadDir(specsDir)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			specPath := filepath.Join(specsDir, entry.Name())
			content, err := os.ReadFile(specPath)
			if err != nil {
				continue
			}

			// Extract spec number
			entityID, title, err := m.parseSpecTitle(specPath, taskID)
			if err != nil {
				continue
			}

			// Register name
			m.names.Register(TypeSpec, title, entityID)

			// Index content
			parsed := Parse(string(content))
			for _, ref := range parsed.References {
				targetID, ok := m.resolveReferenceUnsafe(ref, taskID)
				if !ok {
					continue
				}

				m.index.AddLink(Link{
					Source:    entityID,
					Target:    targetID,
					Context:   ExtractContext(string(content), ref.Position, m.getMaxContextLength()),
					CreatedAt: time.Now(),
				})
			}
		}
	}

	// Index notes
	notesPath := filepath.Join(m.workspace, "work", taskID, "notes.md")
	if content, err := os.ReadFile(notesPath); err == nil {
		// Generate note entity ID
		noteID := fmt.Sprintf("note:%s:notes", taskID)

		parsed := Parse(string(content))
		for _, ref := range parsed.References {
			targetID, ok := m.resolveReferenceUnsafe(ref, taskID)
			if !ok {
				continue
			}

			m.index.AddLink(Link{
				Source:    noteID,
				Target:    targetID,
				Context:   ExtractContext(string(content), ref.Position, m.getMaxContextLength()),
				CreatedAt: time.Now(),
			})
		}
	}

	return nil
}

// parseSpecTitle reads a specification file and extracts its title.
// Returns: entityID, title, error.
func (m *Manager) parseSpecTitle(specPath, taskID string) (string, string, error) {
	content, err := os.ReadFile(specPath)
	if err != nil {
		return "", "", err
	}

	// Extract spec number from filename
	base := filepath.Base(specPath)
	var specNum string
	if _, err := fmt.Sscanf(base, "specification-%s.md", &specNum); err == nil {
		// Successfully parsed
	} else {
		specNum = strings.TrimSuffix(base, ".md")
	}

	// Check for YAML frontmatter
	var title string
	if strings.HasPrefix(string(content), "---\n") {
		endIdx := strings.Index(string(content)[4:], "\n---")
		if endIdx > 0 {
			frontmatter := string(content[4 : 4+endIdx])
			var frontmatterMap map[string]interface{}
			if err := yaml.Unmarshal([]byte(frontmatter), &frontmatterMap); err == nil {
				if t, ok := frontmatterMap["title"].(string); ok {
					title = t
				}
			}
		}
	}

	// If no title in frontmatter, extract from first heading
	if title == "" {
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "# ") {
				title = strings.TrimPrefix(line, "# ")

				break
			}
		}
	}

	// If still no title, use spec number
	if title == "" {
		title = "Specification " + specNum
	}

	entityID := EntityID(TypeSpec, taskID, specNum)

	return entityID, title, nil
}

// resolveReferenceUnsafe resolves a reference without locking (internal use).
func (m *Manager) resolveReferenceUnsafe(ref Reference, activeTaskID string) (string, bool) {
	// If it's a typed reference, construct entity ID directly
	if ref.Type != "" {
		taskID := ref.TaskID
		if ref.IsTaskScoped() {
			if activeTaskID == "" {
				return "", false
			}
			taskID = activeTaskID
		}

		return EntityID(ref.Type, taskID, ref.ID), true
	}

	// Name-based reference: look up in registry
	if ref.Name != "" {
		entityID, found := m.names.Resolve(ref.Name)
		if found {
			return entityID, true
		}
	}

	return "", false
}

// SetAutoSave controls whether the index is automatically saved after modifications.
func (m *Manager) SetAutoSave(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.autoSave = enabled
}

// GetIndex returns a copy of the current link index (for inspection).
func (m *Manager) GetIndex() *LinkIndex {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to avoid race conditions
	result := NewLinkIndex()
	for source, links := range m.index.Forward {
		result.Forward[source] = append([]Link{}, links...)
	}
	for target, links := range m.index.Backward {
		result.Backward[target] = append([]Link{}, links...)
	}

	return result
}

// GetNames returns a copy of the current name registry (for inspection).
func (m *Manager) GetNames() *NameRegistry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to avoid race conditions
	result := NewNameRegistry()
	for k, v := range m.names.Specs {
		result.Specs[k] = v
	}
	for k, v := range m.names.Sessions {
		result.Sessions[k] = v
	}
	for k, v := range m.names.Decisions {
		result.Decisions[k] = v
	}
	for k, v := range m.names.Tasks {
		result.Tasks[k] = v
	}
	for k, v := range m.names.Notes {
		result.Notes[k] = v
	}

	return result
}
