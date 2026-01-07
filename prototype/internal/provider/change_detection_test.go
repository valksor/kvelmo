package provider

import (
	"strings"
	"testing"
)

func TestDetectChanges_NoChanges(t *testing.T) {
	old := &WorkUnit{
		Title:       "Test Task",
		Description: "Test description",
		Status:      StatusOpen,
		Priority:    PriorityNormal,
		Comments:    []Comment{},
		Attachments: []Attachment{},
	}

	updated := &WorkUnit{
		Title:       "Test Task",
		Description: "Test description",
		Status:      StatusOpen,
		Priority:    PriorityNormal,
		Comments:    []Comment{},
		Attachments: []Attachment{},
	}

	changes := DetectChanges(old, updated)

	if changes.HasChanges {
		t.Error("expected no changes")
	}
}

func TestDetectChanges_DescriptionChanged(t *testing.T) {
	old := &WorkUnit{
		Title:       "Test Task",
		Description: "Original description",
		Status:      StatusOpen,
	}

	updated := &WorkUnit{
		Title:       "Test Task",
		Description: "Updated description",
		Status:      StatusOpen,
	}

	changes := DetectChanges(old, updated)

	if !changes.HasChanges {
		t.Error("expected changes to be detected")
	}
	if !changes.DescriptionChanged {
		t.Error("expected DescriptionChanged to be true")
	}
}

func TestDetectChanges_StatusChanged(t *testing.T) {
	old := &WorkUnit{
		Title:  "Test Task",
		Status: StatusOpen,
	}

	updated := &WorkUnit{
		Title:  "Test Task",
		Status: StatusDone,
	}

	changes := DetectChanges(old, updated)

	if !changes.StatusChanged {
		t.Error("expected StatusChanged to be true")
	}
	if changes.OldStatus != StatusOpen {
		t.Error("expected OldStatus to be Open")
	}
	if changes.NewStatus != StatusDone {
		t.Error("expected NewStatus to be Done")
	}
}

func TestDetectChanges_NewComments(t *testing.T) {
	old := &WorkUnit{
		Title: "Test Task",
		Comments: []Comment{
			{ID: "1", Body: "Old comment"},
		},
	}

	updated := &WorkUnit{
		Title: "Test Task",
		Comments: []Comment{
			{ID: "1", Body: "Old comment"},
			{ID: "2", Body: "New comment"},
		},
	}

	changes := DetectChanges(old, updated)

	if len(changes.NewComments) != 1 {
		t.Errorf("expected 1 new comment, got %d", len(changes.NewComments))
	}
	if changes.NewComments[0].ID != "2" {
		t.Errorf("expected new comment ID to be 2, got %s", changes.NewComments[0].ID)
	}
}

func TestDetectChanges_NewAttachments(t *testing.T) {
	old := &WorkUnit{
		Title: "Test Task",
		Attachments: []Attachment{
			{ID: "A1", Name: "old.pdf"},
		},
	}

	updated := &WorkUnit{
		Title: "Test Task",
		Attachments: []Attachment{
			{ID: "A1", Name: "old.pdf"},
			{ID: "A2", Name: "new.pdf"},
		},
	}

	changes := DetectChanges(old, updated)

	if len(changes.NewAttachments) != 1 {
		t.Errorf("expected 1 new attachment, got %d", len(changes.NewAttachments))
	}
	if changes.NewAttachments[0].ID != "A2" {
		t.Errorf("expected new attachment ID to be A2, got %s", changes.NewAttachments[0].ID)
	}
}

func TestDetectChanges_RemovedAttachments(t *testing.T) {
	old := &WorkUnit{
		Title: "Test Task",
		Attachments: []Attachment{
			{ID: "A1", Name: "file1.pdf"},
			{ID: "A2", Name: "file2.pdf"},
		},
	}

	updated := &WorkUnit{
		Title: "Test Task",
		Attachments: []Attachment{
			{ID: "A1", Name: "file1.pdf"},
		},
	}

	changes := DetectChanges(old, updated)

	if len(changes.RemovedAttachments) != 1 {
		t.Errorf("expected 1 removed attachment, got %d", len(changes.RemovedAttachments))
	}
	if changes.RemovedAttachments[0].ID != "A2" {
		t.Errorf("expected removed attachment ID to be A2, got %s", changes.RemovedAttachments[0].ID)
	}
}

func TestChangeSet_Summary(t *testing.T) {
	changes := ChangeSet{
		HasChanges:         true,
		DescriptionChanged: true,
		StatusChanged:      true,
		NewComments: []Comment{
			{ID: "1"},
			{ID: "2"},
		},
		NewAttachments: []Attachment{
			{ID: "A1"},
		},
		NewStatus: StatusDone,
	}

	summary := changes.Summary()
	if summary == "" {
		t.Error("expected non-empty summary")
	}
}

func TestChangeSet_FormatDiff(t *testing.T) {
	changes := ChangeSet{
		HasChanges:         true,
		DescriptionChanged: true,
		NewComments: []Comment{
			{ID: "1", Body: "New comment text here", Author: Person{ID: "user1"}},
		},
		NewAttachments: []Attachment{
			{ID: "A1", Name: "document.pdf"},
		},
	}

	diff := changes.FormatDiff()
	if diff == "" {
		t.Error("expected non-empty diff")
	}
}

func TestFindNewComments_Empty(t *testing.T) {
	result := findNewComments(nil, nil)
	if result != nil {
		t.Error("expected nil for empty slices")
	}

	result = findNewComments([]Comment{}, []Comment{})
	if result != nil {
		t.Error("expected nil for empty slices")
	}
}

func TestFindUpdatedComments(t *testing.T) {
	old := []Comment{
		{ID: "1", Body: "Original text"},
		{ID: "2", Body: "Another comment"},
	}

	updatedComments := []Comment{
		{ID: "1", Body: "Updated text"},
		{ID: "2", Body: "Another comment"},
	}

	updated := findUpdatedComments(old, updatedComments)
	if len(updated) != 1 {
		t.Errorf("expected 1 updated comment, got %d", len(updated))
	}
	if updated[0].ID != "1" {
		t.Errorf("expected updated comment ID to be 1, got %s", updated[0].ID)
	}
}

func TestResolveAuthor(t *testing.T) {
	tests := []struct {
		name     string
		comment  Comment
		expected string
	}{
		{
			name: "author with name",
			comment: Comment{
				Author: Person{
					ID:   "123",
					Name: "John Doe",
				},
			},
			expected: "John Doe",
		},
		{
			name: "author with ID only",
			comment: Comment{
				Author: Person{
					ID: "123",
				},
			},
			expected: "123",
		},
		{
			name:     "empty author",
			comment:  Comment{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveAuthor(tt.comment)
			if result != tt.expected {
				t.Errorf("ResolveAuthor() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestDetectChanges_LabelsChanged(t *testing.T) {
	old := &WorkUnit{
		Title:  "Test Task",
		Labels: []string{"bug", "urgent"},
	}

	updated := &WorkUnit{
		Title:  "Test Task",
		Labels: []string{"bug", "enhancement"},
	}

	changes := DetectChanges(old, updated)

	if !changes.LabelsChanged {
		t.Error("expected LabelsChanged to be true")
	}
	if !changes.HasChanges {
		t.Error("expected HasChanges to be true")
	}
}

func TestDetectChanges_AssigneesChanged(t *testing.T) {
	old := &WorkUnit{
		Title:     "Test Task",
		Assignees: []Person{{ID: "user1", Name: "Alice"}},
	}

	updated := &WorkUnit{
		Title:     "Test Task",
		Assignees: []Person{{ID: "user2", Name: "Bob"}},
	}

	changes := DetectChanges(old, updated)

	if !changes.AssigneesChanged {
		t.Error("expected AssigneesChanged to be true")
	}
	if !changes.HasChanges {
		t.Error("expected HasChanges to be true")
	}
}

func TestDetectChanges_LabelsOrderInsensitive(t *testing.T) {
	old := &WorkUnit{
		Title:  "Test Task",
		Labels: []string{"bug", "urgent", "enhancement"},
	}

	updated := &WorkUnit{
		Title:  "Test Task",
		Labels: []string{"enhancement", "bug", "urgent"},
	}

	changes := DetectChanges(old, updated)

	if changes.LabelsChanged {
		t.Error("expected LabelsChanged to be false when order differs but labels are same")
	}
}

func TestDetectChanges_AssigneesOrderInsensitive(t *testing.T) {
	old := &WorkUnit{
		Title:     "Test Task",
		Assignees: []Person{{ID: "user1"}, {ID: "user2"}},
	}

	updated := &WorkUnit{
		Title:     "Test Task",
		Assignees: []Person{{ID: "user2"}, {ID: "user1"}},
	}

	changes := DetectChanges(old, updated)

	if changes.AssigneesChanged {
		t.Error("expected AssigneesChanged to be false when order differs but assignees are same")
	}
}

func TestEqualStringSlices_NilSafety(t *testing.T) {
	tests := []struct {
		name     string
		a        []string
		b        []string
		expected bool
	}{
		{
			name:     "both nil",
			a:        nil,
			b:        nil,
			expected: true,
		},
		{
			name:     "first nil, second empty",
			a:        nil,
			b:        []string{},
			expected: true,
		},
		{
			name:     "first empty, second nil",
			a:        []string{},
			b:        nil,
			expected: true,
		},
		{
			name:     "both empty",
			a:        []string{},
			b:        []string{},
			expected: true,
		},
		{
			name:     "nil vs non-empty",
			a:        nil,
			b:        []string{"a"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EqualStringSlices(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("EqualStringSlices() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEqualPersonSlices_Duplicates(t *testing.T) {
	tests := []struct {
		name     string
		a        []Person
		b        []Person
		expected bool
	}{
		{
			name:     "with duplicates in both",
			a:        []Person{{ID: "1"}, {ID: "1"}, {ID: "2"}},
			b:        []Person{{ID: "1"}, {ID: "1"}, {ID: "2"}},
			expected: true,
		},
		{
			name:     "different duplicate counts are equal after deduplication",
			a:        []Person{{ID: "1"}, {ID: "1"}, {ID: "2"}},
			b:        []Person{{ID: "1"}, {ID: "2"}},
			expected: true, // Now equal after deduplication
		},
		{
			name:     "nil safety",
			a:        nil,
			b:        nil,
			expected: true,
		},
		{
			name:     "nil vs empty",
			a:        nil,
			b:        []Person{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := equalPersonSlices(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("equalPersonSlices() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestPersonNames(t *testing.T) {
	tests := []struct {
		name     string
		persons  []Person
		expected []string
	}{
		{
			name:     "all have names",
			persons:  []Person{{ID: "1", Name: "Alice"}, {ID: "2", Name: "Bob"}},
			expected: []string{"Alice", "Bob"},
		},
		{
			name:     "mixed names and IDs",
			persons:  []Person{{ID: "1", Name: "Alice"}, {ID: "2"}},
			expected: []string{"Alice", "2"},
		},
		{
			name:     "all IDs only",
			persons:  []Person{{ID: "1"}, {ID: "2"}},
			expected: []string{"1", "2"},
		},
		{
			name:     "empty slice",
			persons:  []Person{},
			expected: []string{},
		},
		{
			name:     "nil slice",
			persons:  nil,
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PersonNames(tt.persons)
			if len(result) != len(tt.expected) {
				t.Errorf("PersonNames() length = %d, want %d", len(result), len(tt.expected))

				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("PersonNames()[%d] = %q, want %q", i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestChangeSet_Summary_StatusAndPriority(t *testing.T) {
	changes := ChangeSet{
		HasChanges:      true,
		StatusChanged:   true,
		PriorityChanged: true,
		OldStatus:       StatusOpen,
		NewStatus:       StatusDone,
		OldPriority:     PriorityNormal,
		NewPriority:     PriorityHigh,
	}

	summary := changes.Summary()
	if summary == "" {
		t.Error("expected non-empty summary")
	}

	// Check that it shows "old → new" format
	if !strings.Contains(summary, "status changed from") {
		t.Errorf("Summary() should show status change in 'from → to' format, got: %s", summary)
	}
	if !strings.Contains(summary, "priority changed from") {
		t.Errorf("Summary() should show priority change in 'from → to' format, got: %s", summary)
	}
}
