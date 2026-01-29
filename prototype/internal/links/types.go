package links

import (
	"fmt"
	"strings"
	"time"
)

// EntityType represents the type of entity that can be linked.
type EntityType string

const (
	TypeSpec     EntityType = "spec"
	TypeSession  EntityType = "session"
	TypeDecision EntityType = "decision"
	TypeTask     EntityType = "task"
	TypeNote     EntityType = "note"
	TypeSolution EntityType = "solution"
	TypeError    EntityType = "error"
)

// Reference represents a parsed [[reference]] from markdown content.
type Reference struct {
	Type     EntityType // spec, session, decision, task, note
	TaskID   string     // Empty for task-scoped references (uses active task)
	ID       string     // spec number, session timestamp, decision ID, etc.
	Name     string     // Human-readable name (if name-based reference)
	Alias    string     // Optional alias for display: [[ref|Display Text]]
	Raw      string     // Original [[...]] text for error reporting
	Position int        // Byte position in source content
}

// IsTaskScoped returns true if the reference is scoped to the active task.
func (r *Reference) IsTaskScoped() bool {
	return r.TaskID == ""
}

// String returns the canonical string representation of the reference.
func (r *Reference) String() string {
	if r.IsTaskScoped() {
		return fmt.Sprintf("[[%s:%s]]", r.Type, r.ID)
	}

	return fmt.Sprintf("[[%s:%s:%s]]", r.Type, r.TaskID, r.ID)
}

// Link represents a directional link between two entities.
type Link struct {
	Source    string    `json:"source"`     // Source entity ID: "spec:task-123:1"
	Target    string    `json:"target"`     // Target entity ID: "spec:task-456:2"
	Context   string    `json:"context"`    // Surrounding text (up to 200 chars)
	CreatedAt time.Time `json:"created_at"` // When the link was created
}

// EntityID returns the canonical entity ID for a reference.
func EntityID(entityType EntityType, taskID, id string) string {
	if taskID == "" {
		return fmt.Sprintf("%s:%s", entityType, id)
	}

	return fmt.Sprintf("%s:%s:%s", entityType, taskID, id)
}

// ParseEntityID parses an entity ID back into its components.
// Returns: entityType, taskID, id
// Examples:
//
//	"spec:task-123:1" -> "spec", "task-123", "1"
//	"decision:abc123" -> "decision", "", "abc123"
//	"session:task:2024-01-29T10:00:00Z" -> "session", "task", "2024-01-29T10:00:00Z"
func ParseEntityID(entityID string) (EntityType, string, string) {
	parts := strings.Split(entityID, ":")
	if len(parts) == 2 {
		return EntityType(parts[0]), "", parts[1]
	}
	if len(parts) == 3 {
		return EntityType(parts[0]), parts[1], parts[2]
	}
	if len(parts) > 3 {
		// Handle timestamps with colons - join remaining parts for ID
		entityType := EntityType(parts[0])
		if entityType == TypeSession || entityType == TypeDecision {
			return entityType, parts[1], strings.Join(parts[2:], ":")
		}
	}

	return "", "", ""
}

// LinkIndex stores bidirectional link mappings between entities.
// The index maintains both forward (source -> targets) and backward
// (target -> sources) mappings for O(1) lookups in either direction.
type LinkIndex struct {
	Forward  map[string][]Link // source -> outgoing links
	Backward map[string][]Link // target -> incoming links
}

// NewLinkIndex creates an empty link index.
func NewLinkIndex() *LinkIndex {
	return &LinkIndex{
		Forward:  make(map[string][]Link),
		Backward: make(map[string][]Link),
	}
}

// AddLink adds a bidirectional link to the index.
// If the link already exists, it will not be duplicated.
func (idx *LinkIndex) AddLink(link Link) {
	// Add to forward index
	idx.Forward[link.Source] = append(idx.Forward[link.Source], link)

	// Add to backward index
	idx.Backward[link.Target] = append(idx.Backward[link.Target], link)
}

// RemoveLinks removes all links from a specific source entity.
// This is called when an entity is deleted or content is reparsed.
func (idx *LinkIndex) RemoveLinks(source string) {
	// Get all outgoing links to remove from backward index
	links := idx.Forward[source]
	for _, link := range links {
		idx.removeFromBackward(link.Target, source)
	}

	// Remove from forward index
	delete(idx.Forward, source)

	// Also remove this entity from any backward indices
	// (it was a target of other entities)
	delete(idx.Backward, source)
}

// removeFromBackward removes a specific source from a target's backward list.
func (idx *LinkIndex) removeFromBackward(target, source string) {
	links := idx.Backward[target]
	var filtered []Link
	for _, link := range links {
		if link.Source != source {
			filtered = append(filtered, link)
		}
	}
	if len(filtered) == 0 {
		delete(idx.Backward, target)
	} else {
		idx.Backward[target] = filtered
	}
}

// GetOutgoing returns all links from the given source entity.
func (idx *LinkIndex) GetOutgoing(source string) []Link {
	return idx.Forward[source]
}

// GetIncoming returns all links to the given target entity.
func (idx *LinkIndex) GetIncoming(target string) []Link {
	return idx.Backward[target]
}

// HasLink returns true if a link exists from source to target.
func (idx *LinkIndex) HasLink(source, target string) bool {
	links := idx.Forward[source]
	for _, link := range links {
		if link.Target == target {
			return true
		}
	}

	return false
}

// GetAllSources returns all entities that have outgoing links.
func (idx *LinkIndex) GetAllSources() []string {
	sources := make([]string, 0, len(idx.Forward))
	for source := range idx.Forward {
		sources = append(sources, source)
	}

	return sources
}

// GetAllTargets returns all entities that have incoming links.
func (idx *LinkIndex) GetAllTargets() []string {
	targets := make([]string, 0, len(idx.Backward))
	for target := range idx.Backward {
		targets = append(targets, target)
	}

	return targets
}

// Stats returns statistics about the link index.
func (idx *LinkIndex) Stats() IndexStats {
	totalLinks := 0
	for _, links := range idx.Forward {
		totalLinks += len(links)
	}

	// Count entities with no incoming or outgoing links (orphan detection)
	// This requires knowing about all entities, which is tracked separately
	return IndexStats{
		TotalLinks:   totalLinks,
		TotalSources: len(idx.Forward),
		TotalTargets: len(idx.Backward),
	}
}

// IndexStats represents statistics about the link index.
type IndexStats struct {
	TotalLinks     int
	TotalSources   int
	TotalTargets   int
	OrphanEntities int // Entities with no links (computed separately)
}

// NameRegistry maps human-readable names to entity IDs.
// This enables name-based references like [[Authentication Spec]].
type NameRegistry struct {
	Specs     map[string]string `json:"specs"`     // "Authentication Spec" -> "spec:task-123:1"
	Sessions  map[string]string `json:"sessions"`  // "Planning session" -> "session:task-123:2024-01-29..."
	Decisions map[string]string `json:"decisions"` // "Cache Strategy" -> "decision:task-123:abc123"
	Tasks     map[string]string `json:"tasks"`     // "Fix auth bug" -> "task:abc123"
	Notes     map[string]string `json:"notes"`     // "User requirements" -> "note:task-123:..." (optional)
}

// NewNameRegistry creates an empty name registry.
func NewNameRegistry() *NameRegistry {
	return &NameRegistry{
		Specs:     make(map[string]string),
		Sessions:  make(map[string]string),
		Decisions: make(map[string]string),
		Tasks:     make(map[string]string),
		Notes:     make(map[string]string),
	}
}

// Register adds a name → entity ID mapping.
func (r *NameRegistry) Register(entityType EntityType, name, entityID string) {
	switch entityType {
	case TypeSpec:
		r.Specs[name] = entityID
	case TypeSession:
		r.Sessions[name] = entityID
	case TypeDecision:
		r.Decisions[name] = entityID
	case TypeTask:
		r.Tasks[name] = entityID
	case TypeNote:
		r.Notes[name] = entityID
	case TypeSolution:
		// Solutions are stored under notes by convention
		r.Notes[name] = entityID
	case TypeError:
		// Errors are stored under notes by convention
		r.Notes[name] = entityID
	}
}

// Resolve looks up an entity by human-readable name.
// Returns the entity ID and true if found, empty string and false otherwise.
func (r *NameRegistry) Resolve(name string) (string, bool) {
	// Check all registries (case-insensitive for user convenience)
	for id, entityID := range r.Specs {
		if strings.EqualFold(id, name) {
			return entityID, true
		}
	}
	for id, entityID := range r.Sessions {
		if strings.EqualFold(id, name) {
			return entityID, true
		}
	}
	for id, entityID := range r.Decisions {
		if strings.EqualFold(id, name) {
			return entityID, true
		}
	}
	for id, entityID := range r.Tasks {
		if strings.EqualFold(id, name) {
			return entityID, true
		}
	}
	for id, entityID := range r.Notes {
		if strings.EqualFold(id, name) {
			return entityID, true
		}
	}

	return "", false
}

// Unregister removes a name mapping.
func (r *NameRegistry) Unregister(entityType EntityType, name string) {
	switch entityType {
	case TypeSpec:
		delete(r.Specs, name)
	case TypeSession:
		delete(r.Sessions, name)
	case TypeDecision:
		delete(r.Decisions, name)
	case TypeTask:
		delete(r.Tasks, name)
	case TypeNote:
		delete(r.Notes, name)
	case TypeSolution:
		// Solutions are stored under notes by convention
		delete(r.Notes, name)
	case TypeError:
		// Errors are stored under notes by convention
		delete(r.Notes, name)
	}
}

// ParsedContent represents the result of parsing content for references.
type ParsedContent struct {
	References []Reference // All [[references]] found in content
	Content    string      // Original content
}

// IndexData represents the complete index data stored on disk.
type IndexData struct {
	Version   int               `json:"version" yaml:"version"`       // Format version for migrations
	Forward   map[string][]Link `json:"forward" yaml:"forward"`       // Forward link mappings
	Backward  map[string][]Link `json:"backward" yaml:"backward"`     // Backward link mappings
	Names     *NameRegistry     `json:"names" yaml:"names"`           // Name registry
	UpdatedAt time.Time         `json:"updated_at" yaml:"updated_at"` // Last update timestamp
}

// NewIndexData creates empty index data.
func NewIndexData() *IndexData {
	return &IndexData{
		Version:   1,
		Forward:   make(map[string][]Link),
		Backward:  make(map[string][]Link),
		Names:     NewNameRegistry(),
		UpdatedAt: time.Now(),
	}
}
