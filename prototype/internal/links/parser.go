package links

import (
	"regexp"
	"strings"
	"time"
	"unicode"
)

// linkPattern matches [[reference]] syntax.
// Supports: [[type:id]], [[type:task-id:id]], [[name]], [[type:id|alias]].
var linkPattern = regexp.MustCompile(`\[\[([^\]]+)\]\]`)

// Parse extracts all [[references]] from the given content.
// Returns a ParsedContent containing all found references.
func Parse(content string) *ParsedContent {
	matches := linkPattern.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return &ParsedContent{
			References: []Reference{},
			Content:    content,
		}
	}

	refs := make([]Reference, 0, len(matches))
	for _, match := range matches {
		// match[0] is the full [[...]] match positions
		// match[1] is the inner content positions
		fullStart, fullEnd := match[0], match[1]
		innerStart, innerEnd := match[2], match[3]

		raw := content[fullStart:fullEnd]
		inner := content[innerStart:innerEnd]

		ref := parseReference(inner, fullStart)
		if ref != nil {
			ref.Raw = raw
			refs = append(refs, *ref)
		}
	}

	return &ParsedContent{
		References: refs,
		Content:    content,
	}
}

// parseReference parses a single reference from the inner [[...]] content.
// Returns nil if the reference is invalid.
func parseReference(inner string, position int) *Reference {
	// Check for alias syntax: [[ref|Display Text]]
	var alias string
	if pipeIdx := strings.Index(inner, "|"); pipeIdx >= 0 {
		alias = strings.TrimSpace(inner[pipeIdx+1:])
		inner = strings.TrimSpace(inner[:pipeIdx])
	}

	// Split by colon to get components
	parts := strings.Split(inner, ":")
	if len(parts) < 2 {
		// Invalid format, might be a plain name reference
		return parseNameReference(inner, position, alias)
	}

	// Parse entity type
	entityType := EntityType(strings.ToLower(parts[0]))
	if !isValidEntityType(entityType) {
		// Unknown type, might be a name reference with colons
		return parseNameReference(inner, position, alias)
	}

	// Handle timestamps with colons (e.g., session:task:2024-01-29T10:00:00Z)
	// If we have more than 3 parts and the entity type is session or decision,
	// the remaining parts might be a timestamp or ID with colons
	if len(parts) > 3 && (entityType == TypeSession || entityType == TypeDecision) {
		// Join the remaining parts back together for the ID
		id := strings.Join(parts[2:], ":")

		return &Reference{
			Type:     entityType,
			TaskID:   parts[1],
			ID:       id,
			Alias:    alias,
			Position: position,
		}
	}

	// Parse the rest based on number of parts
	switch len(parts) {
	case 2:
		// Task-scoped: [[spec:1]]
		return &Reference{
			Type:     entityType,
			TaskID:   "",
			ID:       parts[1],
			Alias:    alias,
			Position: position,
		}
	case 3:
		// Fully qualified: [[spec:task-123:1]]
		return &Reference{
			Type:     entityType,
			TaskID:   parts[1],
			ID:       parts[2],
			Alias:    alias,
			Position: position,
		}
	default:
		// Too many parts, treat as name reference if valid
		return parseNameReference(inner, position, alias)
	}
}

// parseNameReference parses a name-based reference like [[Authentication Spec]].
// These are resolved later via the NameRegistry.
func parseNameReference(name string, position int, alias string) *Reference {
	// Name must not be empty and must be reasonably valid
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}

	// Reject obviously invalid names (too short, special chars only, etc.)
	if !isValidName(name) {
		return nil
	}

	return &Reference{
		Type:     "", // Type determined during resolution
		TaskID:   "",
		ID:       "",
		Name:     name,
		Alias:    alias,
		Position: position,
	}
}

// isValidEntityType returns true if the entity type is recognized.
func isValidEntityType(entityType EntityType) bool {
	switch entityType {
	case TypeSpec, TypeSession, TypeDecision, TypeTask, TypeNote, TypeSolution, TypeError:
		return true
	default:
		return false
	}
}

// isValidName returns true if the name is valid for a name-based reference.
// Names should be reasonable human-readable identifiers.
func isValidName(name string) bool {
	if len(name) < 1 {
		return false
	}

	// Must contain at least some alphanumeric characters
	hasAlnum := false
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			hasAlnum = true

			break
		}
	}
	if !hasAlnum {
		return false
	}

	// Reject obviously invalid patterns
	// (all special chars, etc.)
	return true
}

// ExtractContext extracts the surrounding text around a reference.
// Returns up to maxChars characters from the content surrounding the position.
func ExtractContext(content string, position int, maxChars int) string {
	if maxChars <= 0 {
		maxChars = 200 // default
	}

	// Calculate context boundaries
	start := position - maxChars/2
	if start < 0 {
		start = 0
	}

	end := position + maxChars/2
	if end > len(content) {
		end = len(content)
	}

	// Extract and clean context
	context := content[start:end]

	// Trim to nearest word boundary
	if start > 0 {
		if firstSpace := strings.IndexByte(context, ' '); firstSpace >= 0 && firstSpace < 20 {
			context = context[firstSpace+1:]
		}
	}

	if end < len(content) {
		if lastSpace := strings.LastIndexByte(context, ' '); lastSpace >= 0 && len(context)-lastSpace < 20 {
			context = context[:lastSpace]
		}
	}

	// Clean up whitespace
	context = strings.TrimSpace(context)
	context = strings.Join(strings.Fields(context), " ") // normalize whitespace

	// Add ellipsis if truncated
	if start > 0 {
		context = "..." + context
	}
	if end < len(content) {
		context = context + "..."
	}

	return context
}

// ParseAndIndex parses content and creates links for all references.
// This is a convenience function that combines Parse and Link creation.
//
// The activeTaskID is used to resolve task-scoped references.
// If activeTaskID is empty, task-scoped references will fail to resolve.
func ParseAndIndex(content string, sourceEntityID string, activeTaskID string, names *NameRegistry) []Link {
	parsed := Parse(content)
	if len(parsed.References) == 0 {
		return nil
	}

	links := make([]Link, 0, len(parsed.References))
	for _, ref := range parsed.References {
		targetID, ok := resolveReference(ref, activeTaskID, names)
		if !ok {
			continue // Skip unresolvable references
		}

		// Create link
		links = append(links, Link{
			Source:    sourceEntityID,
			Target:    targetID,
			Context:   ExtractContext(content, ref.Position, 200),
			CreatedAt: now(),
		})
	}

	return links
}

// resolveReference resolves a reference to its target entity ID.
// Returns the entity ID and true if resolved, empty string and false otherwise.
func resolveReference(ref Reference, activeTaskID string, names *NameRegistry) (string, bool) {
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
		entityID, found := names.Resolve(ref.Name)
		if found {
			return entityID, true
		}
	}

	return "", false
}

// now returns the current time. Extracted for testability.
func now() time.Time {
	return time.Now()
}
