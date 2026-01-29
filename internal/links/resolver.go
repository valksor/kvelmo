package links

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Resolver resolves references to entity IDs by consulting the name registry
// and inspecting the workspace for entity metadata.
type Resolver struct {
	workspace string // Workspace directory path
	registry  *NameRegistry
}

// NewResolver creates a new resolver for the given workspace.
func NewResolver(workspace string, registry *NameRegistry) *Resolver {
	return &Resolver{
		workspace: workspace,
		registry:  registry,
	}
}

// Resolve resolves a reference to its target entity ID.
// Returns the entity ID and true if resolved, empty string and false otherwise.
func (r *Resolver) Resolve(ref Reference, activeTaskID string) (string, bool) {
	// If it's a typed reference, construct entity ID directly
	if ref.Type != "" {
		taskID := ref.TaskID
		if ref.IsTaskScoped() {
			// Use active task ID
			if activeTaskID == "" {
				return "", false // Cannot resolve task-scoped ref without active task
			}
			taskID = activeTaskID
		}

		return EntityID(ref.Type, taskID, ref.ID), true
	}

	// Name-based reference: look up in registry
	if ref.Name != "" {
		entityID, found := r.registry.Resolve(ref.Name)
		if found {
			return entityID, true
		}

		// Try to resolve by searching the workspace
		return r.resolveByName(ref.Name, activeTaskID)
	}

	return "", false
}

// resolveByName searches the workspace for an entity by human-readable name.
// This scans specification files, session files, and other sources.
func (r *Resolver) resolveByName(name string, activeTaskID string) (string, bool) {
	// Normalize the name for matching
	searchName := strings.ToLower(strings.TrimSpace(name))

	// Search in specifications
	if entityID, found := r.searchSpecs(searchName, activeTaskID); found {
		return entityID, true
	}

	// Search in sessions (if task-scoped)
	if activeTaskID != "" {
		if entityID, found := r.searchSessions(searchName, activeTaskID); found {
			return entityID, true
		}
	}

	// Search in decisions (stored as notes with special format)
	if entityID, found := r.searchDecisions(searchName, activeTaskID); found {
		return entityID, true
	}

	return "", false
}

// searchSpecs searches specifications for a matching title.
func (r *Resolver) searchSpecs(name string, activeTaskID string) (string, bool) {
	if activeTaskID == "" {
		return "", false
	}

	// Specs directory: ~/.valksor/mehrhof/workspaces/<project>/work/<task>/specifications/
	specsDir := filepath.Join(r.workspace, "work", activeTaskID, "specifications")
	if _, err := os.Stat(specsDir); os.IsNotExist(err) {
		return "", false
	}

	// List spec files
	entries, err := os.ReadDir(specsDir)
	if err != nil {
		return "", false
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Parse spec file to get title
		specPath := filepath.Join(specsDir, entry.Name())
		entityID, title, err := r.parseSpecTitle(specPath, activeTaskID)
		if err != nil {
			continue
		}

		// Check if title matches
		if strings.Contains(strings.ToLower(title), name) {
			return entityID, true
		}
	}

	return "", false
}

// parseSpecTitle reads a specification file and extracts its title.
// Returns: entityID, title, error.
func (r *Resolver) parseSpecTitle(specPath, taskID string) (string, string, error) {
	content, err := os.ReadFile(specPath)
	if err != nil {
		return "", "", err
	}

	// Extract spec number from filename
	base := filepath.Base(specPath)
	// Expected format: specification-N.md
	var specNum string
	if _, err := fmt.Sscanf(base, "specification-%s.md", &specNum); err == nil {
		// Successfully parsed spec number
	} else {
		// Fallback: use full base name without extension
		specNum = strings.TrimSuffix(base, ".md")
	}

	// Check for YAML frontmatter
	var title string
	if strings.HasPrefix(string(content), "---\n") {
		endIdx := strings.Index(string(content)[4:], "\n---")
		if endIdx > 0 {
			// Parse frontmatter for title
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

// searchSessions searches sessions for a matching name/title.
func (r *Resolver) searchSessions(name, activeTaskID string) (string, bool) {
	if activeTaskID == "" {
		return "", false
	}

	// Sessions directory: ~/.valksor/mehrhof/workspaces/<project>/work/<task>/sessions/
	sessionsDir := filepath.Join(r.workspace, "work", activeTaskID, "sessions")
	if _, err := os.Stat(sessionsDir); os.IsNotExist(err) {
		return "", false
	}

	// List session files
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		return "", false
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		// Parse session file to get metadata
		sessionPath := filepath.Join(sessionsDir, entry.Name())
		entityID, sessionName, err := r.parseSessionMeta(sessionPath, activeTaskID)
		if err != nil {
			continue
		}

		// Check if name matches
		// Use the filename timestamp as the ID
		if strings.Contains(strings.ToLower(sessionName), name) {
			return entityID, true
		}
	}

	return "", false
}

// parseSessionMeta reads a session file and extracts its metadata.
// Returns: entityID, session name, error.
func (r *Resolver) parseSessionMeta(sessionPath, taskID string) (string, string, error) {
	content, err := os.ReadFile(sessionPath)
	if err != nil {
		return "", "", err
	}

	// Parse YAML
	var session struct {
		Metadata struct {
			Type string `yaml:"type"` // planning, implementing, reviewing
		} `yaml:"metadata"`
	}

	if err := yaml.Unmarshal(content, &session); err != nil {
		return "", "", err
	}

	// Extract timestamp from filename for ID
	base := filepath.Base(sessionPath)
	id := strings.TrimSuffix(base, ".yaml")

	// Build a descriptive name from type
	sessionName := session.Metadata.Type + " session"
	if sessionName == "" {
		sessionName = "session"
	}

	entityID := EntityID(TypeSession, taskID, id)

	return entityID, sessionName, nil
}

// searchDecisions searches for decisions in notes.
// Decisions are identified by a special format in notes.
func (r *Resolver) searchDecisions(name, activeTaskID string) (string, bool) {
	if activeTaskID == "" {
		return "", false
	}

	// Notes file: ~/.valksor/mehrhof/workspaces/<project>/work/<task>/notes.md
	notesPath := filepath.Join(r.workspace, "work", activeTaskID, "notes.md")
	if _, err := os.Stat(notesPath); os.IsNotExist(err) {
		return "", false
	}

	content, err := os.ReadFile(notesPath)
	if err != nil {
		return "", false
	}

	// Parse notes for decisions (marked with "decision:" prefix)
	notes := string(content)
	lines := strings.Split(notes, "\n")

	for i, line := range lines {
		// Look for decision markers
		if strings.HasPrefix(strings.ToLower(line), "decision:") {
			decisionText := strings.TrimPrefix(strings.ToLower(line), "decision:")
			decisionText = strings.TrimSpace(decisionText)

			// Extract decision name (first word or phrase)
			decisionName := decisionText
			if spaceIdx := strings.Index(decisionName, " "); spaceIdx >= 0 {
				decisionName = decisionName[:spaceIdx]
			}

			// Check if matches
			if strings.Contains(strings.ToLower(decisionName), name) {
				// Generate a decision ID
				decisionID := fmt.Sprintf("decision-%d", i)
				entityID := EntityID(TypeDecision, activeTaskID, decisionID)

				return entityID, true
			}
		}
	}

	return "", false
}

// IndexWorkspace scans the workspace and populates the name registry
// with all discoverable entities (specs, sessions, decisions).
func (r *Resolver) IndexWorkspace() error {
	// Get list of tasks in workspace
	workDir := filepath.Join(r.workspace, "work")
	entries, err := os.ReadDir(workDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No work directory yet
		}

		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		taskID := entry.Name()

		// Index specs
		r.indexSpecs(taskID)

		// Index sessions
		r.indexSessions(taskID)

		// Index decisions from notes
		r.indexDecisions(taskID)
	}

	return nil
}

// indexSpecs indexes all specifications for a task.
func (r *Resolver) indexSpecs(taskID string) {
	specsDir := filepath.Join(r.workspace, "work", taskID, "specifications")
	if _, err := os.Stat(specsDir); os.IsNotExist(err) {
		return
	}

	entries, err := os.ReadDir(specsDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		specPath := filepath.Join(specsDir, entry.Name())
		entityID, title, err := r.parseSpecTitle(specPath, taskID)
		if err != nil {
			continue
		}

		// Register in name registry
		r.registry.Register(TypeSpec, title, entityID)
	}
}

// indexSessions indexes all sessions for a task.
func (r *Resolver) indexSessions(taskID string) {
	sessionsDir := filepath.Join(r.workspace, "work", taskID, "sessions")
	if _, err := os.Stat(sessionsDir); os.IsNotExist(err) {
		return
	}

	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		sessionPath := filepath.Join(sessionsDir, entry.Name())
		entityID, sessionName, err := r.parseSessionMeta(sessionPath, taskID)
		if err != nil {
			continue
		}

		// Register in name registry
		r.registry.Register(TypeSession, sessionName, entityID)
	}
}

// indexDecisions indexes all decisions from notes for a task.
func (r *Resolver) indexDecisions(taskID string) {
	notesPath := filepath.Join(r.workspace, "work", taskID, "notes.md")
	if _, err := os.Stat(notesPath); os.IsNotExist(err) {
		return
	}

	content, err := os.ReadFile(notesPath)
	if err != nil {
		return
	}

	notes := string(content)
	lines := strings.Split(notes, "\n")

	for i, line := range lines {
		// Look for decision markers
		if strings.HasPrefix(strings.ToLower(line), "decision:") {
			decisionText := strings.TrimPrefix(strings.ToLower(line), "decision:")
			decisionText = strings.TrimSpace(decisionText)

			// Extract decision name
			decisionName := decisionText
			if spaceIdx := strings.Index(decisionName, " "); spaceIdx >= 0 {
				decisionName = decisionName[:spaceIdx]
			}

			// Generate entity ID
			decisionID := fmt.Sprintf("decision-%d", i)
			entityID := EntityID(TypeDecision, taskID, decisionID)

			// Register in name registry
			r.registry.Register(TypeDecision, decisionName, entityID)
		}
	}
}
