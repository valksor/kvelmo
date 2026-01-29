package links

import (
	"strings"
	"testing"
)

func TestParse_FullyQualifiedReferences(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []Reference
	}{
		{
			name:    "single spec reference",
			content: "See [[spec:task-123:1]] for details",
			expected: []Reference{
				{Type: TypeSpec, TaskID: "task-123", ID: "1", Raw: "[[spec:task-123:1]]"},
			},
		},
		{
			name:    "session reference",
			content: "[[session:abc:2024-01-29T10:00:00Z]]",
			expected: []Reference{
				{Type: TypeSession, TaskID: "abc", ID: "2024-01-29T10:00:00Z", Raw: "[[session:abc:2024-01-29T10:00:00Z]]"},
			},
		},
		{
			name:    "decision reference",
			content: "Based on [[decision:task-456:cache-strategy]]",
			expected: []Reference{
				{Type: TypeDecision, TaskID: "task-456", ID: "cache-strategy"},
			},
		},
		{
			name:    "multiple references",
			content: "[[spec:task-1:1]] and [[spec:task-1:2]]",
			expected: []Reference{
				{Type: TypeSpec, TaskID: "task-1", ID: "1"},
				{Type: TypeSpec, TaskID: "task-1", ID: "2"},
			},
		},
		{
			name:    "all entity types",
			content: "[[spec:t:1]] [[session:t:s]] [[decision:t:d]] [[task:t]] [[note:t:n]]",
			expected: []Reference{
				{Type: TypeSpec, TaskID: "t", ID: "1"},
				{Type: TypeSession, TaskID: "t", ID: "s"},
				{Type: TypeDecision, TaskID: "t", ID: "d"},
				{Type: TypeTask, TaskID: "", ID: "t"},
				{Type: TypeNote, TaskID: "t", ID: "n"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Parse(tt.content)
			if len(result.References) != len(tt.expected) {
				t.Fatalf("expected %d references, got %d", len(tt.expected), len(result.References))
			}

			for i, ref := range result.References {
				expected := tt.expected[i]
				if ref.Type != expected.Type {
					t.Errorf("[%d] expected type %q, got %q", i, expected.Type, ref.Type)
				}
				if ref.TaskID != expected.TaskID {
					t.Errorf("[%d] expected taskID %q, got %q", i, expected.TaskID, ref.TaskID)
				}
				if ref.ID != expected.ID {
					t.Errorf("[%d] expected ID %q, got %q", i, expected.ID, ref.ID)
				}
				if expected.Raw != "" && ref.Raw != expected.Raw {
					t.Errorf("[%d] expected raw %q, got %q", i, expected.Raw, ref.Raw)
				}
			}
		})
	}
}

func TestParse_TaskScopedReferences(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []Reference
	}{
		{
			name:    "task-scoped spec",
			content: "See [[spec:1]] for implementation",
			expected: []Reference{
				{Type: TypeSpec, TaskID: "", ID: "1", Raw: "[[spec:1]]"},
			},
		},
		{
			name:    "task-scoped decision",
			content: "Using [[decision:jwt-choice]] from earlier",
			expected: []Reference{
				{Type: TypeDecision, TaskID: "", ID: "jwt-choice", Raw: "[[decision:jwt-choice]]"},
			},
		},
		{
			name:    "mixed task-scoped and fully-qualified",
			content: "[[spec:1]] vs [[spec:other:2]]",
			expected: []Reference{
				{Type: TypeSpec, TaskID: "", ID: "1"},
				{Type: TypeSpec, TaskID: "other", ID: "2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Parse(tt.content)
			if len(result.References) != len(tt.expected) {
				t.Fatalf("expected %d references, got %d", len(tt.expected), len(result.References))
			}

			for i, ref := range result.References {
				expected := tt.expected[i]
				if ref.Type != expected.Type {
					t.Errorf("[%d] expected type %q, got %q", i, expected.Type, ref.Type)
				}
				if ref.TaskID != expected.TaskID {
					t.Errorf("[%d] expected taskID %q, got %q", i, expected.TaskID, ref.TaskID)
				}
				if ref.ID != expected.ID {
					t.Errorf("[%d] expected ID %q, got %q", i, expected.ID, ref.ID)
				}
			}
		})
	}
}

func TestParse_NameReferences(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []Reference
	}{
		{
			name:    "simple name reference",
			content: "See [[Authentication Spec]] for details",
			expected: []Reference{
				{Type: "", TaskID: "", ID: "", Name: "Authentication Spec"},
			},
		},
		{
			name:    "name with spaces",
			content: "[[JWT Middleware Decision]] made earlier",
			expected: []Reference{
				{Type: "", TaskID: "", ID: "", Name: "JWT Middleware Decision"},
			},
		},
		{
			name:    "name with special chars",
			content: "[[OAuth 2.0 Flow]] implementation",
			expected: []Reference{
				{Type: "", TaskID: "", ID: "", Name: "OAuth 2.0 Flow"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Parse(tt.content)
			if len(result.References) != len(tt.expected) {
				t.Fatalf("expected %d references, got %d", len(tt.expected), len(result.References))
			}

			for i, ref := range result.References {
				expected := tt.expected[i]
				if ref.Type != expected.Type {
					t.Errorf("[%d] expected type %q, got %q", i, expected.Type, ref.Type)
				}
				if ref.Name != expected.Name {
					t.Errorf("[%d] expected name %q, got %q", i, expected.Name, ref.Name)
				}
			}
		})
	}
}

func TestParse_AliasSyntax(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []Reference
	}{
		{
			name:    "typed reference with alias",
			content: "[[spec:1|the implementation spec]]",
			expected: []Reference{
				{Type: TypeSpec, TaskID: "", ID: "1", Alias: "the implementation spec"},
			},
		},
		{
			name:    "name reference with alias",
			content: "[[Auth Spec|authentication flow]]",
			expected: []Reference{
				{Type: "", TaskID: "", ID: "", Name: "Auth Spec", Alias: "authentication flow"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Parse(tt.content)
			if len(result.References) != len(tt.expected) {
				t.Fatalf("expected %d references, got %d", len(tt.expected), len(result.References))
			}

			for i, ref := range result.References {
				expected := tt.expected[i]
				if ref.Alias != expected.Alias {
					t.Errorf("[%d] expected alias %q, got %q", i, expected.Alias, ref.Alias)
				}
			}
		})
	}
}

func TestParse_InvalidReferences(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		expectRefLen int
	}{
		{
			name:         "empty content",
			content:      "",
			expectRefLen: 0,
		},
		{
			name:         "no references",
			content:      "This is just plain text with no links",
			expectRefLen: 0,
		},
		{
			name:         "incomplete bracket",
			content:      "This has [[open bracket",
			expectRefLen: 0,
		},
		{
			name:         "invalid entity type",
			content:      "[[invalid:type:id]]",
			expectRefLen: 1, // Unknown types are treated as name refs
		},
		{
			name:         "too many colons",
			content:      "[[spec:too:many:colons:here]]",
			expectRefLen: 1, // Treated as name reference
		},
		{
			name:         "empty brackets",
			content:      "[[]]",
			expectRefLen: 0, // Invalid name
		},
		{
			name:         "single special char name",
			content:      "[[!]]",
			expectRefLen: 0, // Invalid name
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Parse(tt.content)
			if len(result.References) != tt.expectRefLen {
				t.Errorf("expected %d references, got %d", tt.expectRefLen, len(result.References))
			}
		})
	}
}

func TestParse_ReferencePosition(t *testing.T) {
	content := "Start [[spec:1]] middle [[decision:d]] end"
	result := Parse(content)

	if len(result.References) != 2 {
		t.Fatalf("expected 2 references, got %d", len(result.References))
	}

	// First reference position should be near "Start " (6 chars)
	if result.References[0].Position < 5 || result.References[0].Position > 10 {
		t.Errorf("first reference position %d seems wrong for content %q", result.References[0].Position, content)
	}

	// Second reference should be after first
	if result.References[1].Position <= result.References[0].Position {
		t.Errorf("second reference position %d should be after first %d", result.References[1].Position, result.References[0].Position)
	}
}

func TestExtractContext(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		position  int
		maxChars  int
		wantStart bool
		wantEnd   bool
	}{
		{
			name:      "middle of content",
			content:   "This is a long piece of content with [[spec:1]] reference in the middle of some text",
			position:  42, // Position of "[[spec:1]]"
			maxChars:  50,
			wantStart: false,
			wantEnd:   false,
		},
		{
			name:      "near start",
			content:   "[[spec:1]] at the start",
			position:  0,
			maxChars:  50,
			wantStart: false,
			wantEnd:   false,
		},
		{
			name:      "near end",
			content:   "Content near the end [[spec:1]]",
			position:  30,
			maxChars:  50,
			wantStart: false,
			wantEnd:   false,
		},
		{
			name:      "zero maxChars",
			content:   "Some text [[spec:1]] more text",
			position:  10,
			maxChars:  0, // Should use default 200
			wantStart: false,
			wantEnd:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			context := ExtractContext(tt.content, tt.position, tt.maxChars)

			// Context should be reasonably long
			if len(context) < 10 && len(tt.content) > 20 {
				t.Errorf("context too short: %q", context)
			}

			// Check for ellipsis indicating truncation
			hasStartEllipsis := strings.HasPrefix(context, "...")
			hasEndEllipsis := strings.HasSuffix(context, "...")

			if tt.wantStart && !hasStartEllipsis {
				t.Errorf("expected start ellipsis, got: %q", context)
			}
			if !tt.wantStart && tt.position > 50 && hasStartEllipsis {
				// OK, context was truncated at start
			} else if !tt.wantStart && tt.position > 50 && !hasStartEllipsis {
				// Also OK - position > 50 but full context fits
				t.Logf("full context for position %d: %q", tt.position, context)
			}
			if tt.wantEnd && !hasEndEllipsis {
				t.Errorf("expected end ellipsis, got: %q", context)
			}
		})
	}
}

func TestEntityID(t *testing.T) {
	tests := []struct {
		name     string
		typ      EntityType
		taskID   string
		id       string
		expected string
	}{
		{
			name:     "task-scoped",
			typ:      TypeSpec,
			taskID:   "",
			id:       "1",
			expected: "spec:1",
		},
		{
			name:     "fully qualified",
			typ:      TypeSpec,
			taskID:   "task-123",
			id:       "1",
			expected: "spec:task-123:1",
		},
		{
			name:     "decision without task",
			typ:      TypeDecision,
			taskID:   "",
			id:       "jwt-choice",
			expected: "decision:jwt-choice",
		},
		{
			name:     "session with task",
			typ:      TypeSession,
			taskID:   "abc",
			id:       "2024-01-29T10:00:00Z",
			expected: "session:abc:2024-01-29T10:00:00Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EntityID(tt.typ, tt.taskID, tt.id)
			if result != tt.expected {
				t.Errorf("EntityID() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestParseEntityID(t *testing.T) {
	tests := []struct {
		name         string
		entityID     string
		expectType   EntityType
		expectTaskID string
		expectID     string
	}{
		{
			name:         "fully qualified",
			entityID:     "spec:task-123:1",
			expectType:   TypeSpec,
			expectTaskID: "task-123",
			expectID:     "1",
		},
		{
			name:         "task-scoped",
			entityID:     "decision:jwt-choice",
			expectType:   TypeDecision,
			expectTaskID: "",
			expectID:     "jwt-choice",
		},
		{
			name:         "session",
			entityID:     "session:abc:2024-01-29T10:00:00Z",
			expectType:   TypeSession,
			expectTaskID: "abc",
			expectID:     "2024-01-29T10:00:00Z",
		},
		{
			name:         "invalid",
			entityID:     "invalid",
			expectType:   "",
			expectTaskID: "",
			expectID:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typ, taskID, id := ParseEntityID(tt.entityID)
			if typ != tt.expectType {
				t.Errorf("type = %q, want %q", typ, tt.expectType)
			}
			if taskID != tt.expectTaskID {
				t.Errorf("taskID = %q, want %q", taskID, tt.expectTaskID)
			}
			if id != tt.expectID {
				t.Errorf("id = %q, want %q", id, tt.expectID)
			}
		})
	}
}

func TestReference_IsTaskScoped(t *testing.T) {
	tests := []struct {
		name     string
		ref      Reference
		expected bool
	}{
		{
			name:     "task-scoped",
			ref:      Reference{Type: TypeSpec, TaskID: "", ID: "1"},
			expected: true,
		},
		{
			name:     "fully qualified",
			ref:      Reference{Type: TypeSpec, TaskID: "task-123", ID: "1"},
			expected: false,
		},
		{
			name:     "name reference",
			ref:      Reference{Name: "Some Spec"},
			expected: true, // Name refs are task-scoped by default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ref.IsTaskScoped(); got != tt.expected {
				t.Errorf("IsTaskScoped() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestLinkIndex_AddLink(t *testing.T) {
	idx := NewLinkIndex()

	link1 := Link{
		Source:  "spec:task-1:1",
		Target:  "spec:task-1:2",
		Context: "See next spec",
	}
	link2 := Link{
		Source:  "spec:task-1:1",
		Target:  "decision:task-1:jwt",
		Context: "Based on this decision",
	}

	idx.AddLink(link1)
	idx.AddLink(link2)

	// Check forward index
	outgoing := idx.GetOutgoing("spec:task-1:1")
	if len(outgoing) != 2 {
		t.Fatalf("expected 2 outgoing links, got %d", len(outgoing))
	}

	// Check backward index
	incoming := idx.GetIncoming("spec:task-1:2")
	if len(incoming) != 1 {
		t.Fatalf("expected 1 incoming link to spec:task-1:2, got %d", len(incoming))
	}

	// Verify HasLink
	if !idx.HasLink("spec:task-1:1", "spec:task-1:2") {
		t.Error("expected HasLink to return true")
	}
}

func TestLinkIndex_RemoveLinks(t *testing.T) {
	idx := NewLinkIndex()

	// Add some links
	idx.AddLink(Link{Source: "spec:task-1:1", Target: "spec:task-1:2"})
	idx.AddLink(Link{Source: "spec:task-1:1", Target: "decision:task-1:d"})
	idx.AddLink(Link{Source: "note:task-1:n", Target: "spec:task-1:1"})

	// Remove links from spec:task-1:1
	idx.RemoveLinks("spec:task-1:1")

	// Check forward index
	if outgoing := idx.GetOutgoing("spec:task-1:1"); len(outgoing) != 0 {
		t.Errorf("expected 0 outgoing links after removal, got %d", len(outgoing))
	}

	// Check backward index - spec:task-1:1 should not be a target anymore
	if incoming := idx.GetIncoming("spec:task-1:1"); len(incoming) != 0 {
		t.Errorf("expected 0 incoming links after removal, got %d", len(incoming))
	}

	// Other backward links should remain
	if incoming := idx.GetIncoming("spec:task-1:2"); len(incoming) != 0 {
		// spec:task-1:2 is no longer a target since the only link to it was removed
		t.Errorf("expected 0 incoming links to spec:task-1:2, got %d", len(incoming))
	}
}

func TestLinkIndex_Stats(t *testing.T) {
	idx := NewLinkIndex()

	// Initial stats
	stats := idx.Stats()
	if stats.TotalLinks != 0 {
		t.Errorf("expected 0 total links, got %d", stats.TotalLinks)
	}
	if stats.TotalSources != 0 {
		t.Errorf("expected 0 total sources, got %d", stats.TotalSources)
	}

	// Add links
	idx.AddLink(Link{Source: "spec:task-1:1", Target: "spec:task-1:2"})
	idx.AddLink(Link{Source: "spec:task-1:1", Target: "decision:task-1:d"})
	idx.AddLink(Link{Source: "note:task-1:n", Target: "spec:task-1:2"})

	stats = idx.Stats()
	if stats.TotalLinks != 3 {
		t.Errorf("expected 3 total links, got %d", stats.TotalLinks)
	}
	if stats.TotalSources != 2 { // spec:task-1:1 and note:task-1:n
		t.Errorf("expected 2 total sources, got %d", stats.TotalSources)
	}
	if stats.TotalTargets != 2 { // spec:task-1:2 and decision:task-1:d
		t.Errorf("expected 2 total targets, got %d", stats.TotalTargets)
	}
}

func TestNameRegistry_RegisterAndResolve(t *testing.T) {
	registry := NewNameRegistry()

	// Register some names
	registry.Register(TypeSpec, "Authentication Spec", "spec:task-1:1")
	registry.Register(TypeDecision, "JWT Choice", "decision:task-1:jwt")

	// Resolve exact match
	entityID, found := registry.Resolve("Authentication Spec")
	if !found {
		t.Error("expected to find Authentication Spec")
	}
	if entityID != "spec:task-1:1" {
		t.Errorf("expected spec:task-1:1, got %s", entityID)
	}

	// Resolve case-insensitive
	entityID, found = registry.Resolve("authentication spec")
	if !found {
		t.Error("expected to find authentication spec (case-insensitive)")
	}
	if entityID != "spec:task-1:1" {
		t.Errorf("expected spec:task-1:1, got %s", entityID)
	}

	// Resolve unknown
	_, found = registry.Resolve("Unknown Spec")
	if found {
		t.Error("expected not to find Unknown Spec")
	}
}

func TestNameRegistry_Unregister(t *testing.T) {
	registry := NewNameRegistry()

	registry.Register(TypeSpec, "Test Spec", "spec:task-1:1")

	// Verify it exists
	if _, found := registry.Resolve("Test Spec"); !found {
		t.Error("expected to find Test Spec before unregister")
	}

	// Unregister
	registry.Unregister(TypeSpec, "Test Spec")

	// Verify it's gone
	if _, found := registry.Resolve("Test Spec"); found {
		t.Error("expected not to find Test Spec after unregister")
	}
}

func TestParseAndIndex(t *testing.T) {
	content := "See [[spec:1]] and [[decision:jwt]] and [[Named Spec]]"
	sourceEntityID := "spec:task-abc:1"
	activeTaskID := "task-abc"

	names := NewNameRegistry()
	names.Register(TypeSpec, "Named Spec", "spec:task-abc:2")

	links := ParseAndIndex(content, sourceEntityID, activeTaskID, names)

	if len(links) != 3 {
		t.Fatalf("expected 3 links (all resolvable), got %d", len(links))
	}

	// Check first link
	if links[0].Source != sourceEntityID {
		t.Errorf("expected source %s, got %s", sourceEntityID, links[0].Source)
	}
	if links[0].Target != "spec:task-abc:1" {
		t.Errorf("expected target spec:task-abc:1, got %s", links[0].Target)
	}

	// Check second link (decision)
	if links[1].Target != "decision:task-abc:jwt" {
		t.Errorf("expected target decision:task-abc:jwt, got %s", links[1].Target)
	}

	// Check third link (named spec)
	if links[2].Target != "spec:task-abc:2" {
		t.Errorf("expected target spec:task-abc:2 (Named Spec), got %s", links[2].Target)
	}
}

func TestIsValidEntityType(t *testing.T) {
	tests := []struct {
		typ      EntityType
		expected bool
	}{
		{TypeSpec, true},
		{TypeSession, true},
		{TypeDecision, true},
		{TypeTask, true},
		{TypeNote, true},
		{TypeSolution, true},
		{TypeError, true},
		{EntityType("invalid"), false},
		{EntityType(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.typ), func(t *testing.T) {
			if got := isValidEntityType(tt.typ); got != tt.expected {
				t.Errorf("isValidEntityType(%q) = %v, want %v", tt.typ, got, tt.expected)
			}
		})
	}
}

func TestIsValidName(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"Authentication Spec", true},
		{"JWT Decision", true},
		{"OAuth 2.0", true},
		{"A", true},   // Single letter is OK
		{"", false},   // Empty is invalid
		{"!", false},  // Single special char is invalid
		{"123", true}, // Numbers only are OK
		{"Test_With-Underscore", true},
		{"   ", false}, // Only spaces is invalid
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidName(tt.name); got != tt.expected {
				t.Errorf("isValidName(%q) = %v, want %v", tt.name, got, tt.expected)
			}
		})
	}
}
