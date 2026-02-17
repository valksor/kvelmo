package conductor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valksor/go-toolkit/workunit"
)

// mockParentFetcher is a mock provider that implements ParentFetcher.
type mockParentFetcher struct {
	parent *workunit.WorkUnit
	err    error
}

func (m *mockParentFetcher) FetchParent(ctx context.Context, workUnitID string) (*workunit.WorkUnit, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.parent, nil
}

// mockSubtaskFetcher is a mock provider that implements SubtaskFetcher.
type mockSubtaskFetcher struct {
	subtasks []*workunit.WorkUnit
	err      error
}

func (m *mockSubtaskFetcher) FetchSubtasks(ctx context.Context, parentID string) ([]*workunit.WorkUnit, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.subtasks, nil
}

// mockParentAndSubtaskFetcher combines both interfaces.
type mockParentAndSubtaskFetcher struct {
	*mockParentFetcher
	*mockSubtaskFetcher
}

func TestFetchHierarchicalContext_NotASubtask(t *testing.T) {
	c := &Conductor{}

	// Create a work unit that is NOT a subtask
	workUnit := &workunit.WorkUnit{
		ID:    "task-123",
		Title: "Regular Task",
		Metadata: map[string]any{
			"is_subtask": false,
		},
	}

	result, err := c.FetchHierarchicalContext(context.Background(), nil, workUnit, false)

	require.ErrorIs(t, err, ErrNotASubtask)
	assert.Nil(t, result, "Should return nil for non-subtask")
}

func TestFetchHierarchicalContext_NoParentFetcher(t *testing.T) {
	c := &Conductor{}

	// Create a subtask work unit
	workUnit := &workunit.WorkUnit{
		ID:    "subtask-123",
		Title: "Subtask",
		Metadata: map[string]any{
			"is_subtask": true,
			"parent_id":  "parent-123",
		},
	}

	// Use a provider that doesn't implement ParentFetcher
	providerWithoutParent := struct{}{}

	result, err := c.FetchHierarchicalContext(context.Background(), providerWithoutParent, workUnit, false)

	require.ErrorIs(t, err, ErrNotASubtask)
	assert.Nil(t, result, "Should return nil when provider doesn't support parent fetching")
}

func TestFetchHierarchicalContext_WithParent(t *testing.T) {
	c := &Conductor{}

	// Create a subtask work unit
	workUnit := &workunit.WorkUnit{
		ID:    "subtask-123",
		Title: "Subtask",
		Metadata: map[string]any{
			"is_subtask": true,
			"parent_id":  "parent-123",
		},
	}

	// Create a mock parent fetcher
	parentWorkUnit := &workunit.WorkUnit{
		ID:          "parent-123",
		Title:       "Parent Task",
		Description: "This is the parent task description",
		Metadata: map[string]any{
			"subtask_count": 3,
		},
	}

	mockProvider := &mockParentFetcher{
		parent: parentWorkUnit,
	}

	result, err := c.FetchHierarchicalContext(context.Background(), mockProvider, workUnit, false)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotNil(t, result.Parent)
	assert.Equal(t, "parent-123", result.Parent.ID)
	assert.Equal(t, "Parent Task", result.Parent.Title)
	assert.Nil(t, result.Siblings, "Siblings should be nil when includeSiblings is false")
}

func TestFetchHierarchicalContext_WithParentAndSiblings(t *testing.T) {
	c := &Conductor{}

	// Create a subtask work unit
	workUnit := &workunit.WorkUnit{
		ID:    "subtask-2",
		Title: "Subtask 2",
		Metadata: map[string]any{
			"is_subtask": true,
			"parent_id":  "parent-123",
		},
	}

	// Create mock siblings (including the current task)
	siblings := []*workunit.WorkUnit{
		{ID: "subtask-1", Title: "Subtask 1", Metadata: map[string]any{"state": "done"}},
		{ID: "subtask-2", Title: "Subtask 2", Metadata: map[string]any{"state": "in_progress"}}, // Current task
		{ID: "subtask-3", Title: "Subtask 3", Metadata: map[string]any{"state": "todo"}},
	}

	parentWorkUnit := &workunit.WorkUnit{
		ID:          "parent-123",
		Title:       "Parent Task",
		Description: "Parent description",
	}

	mockProvider := &mockParentAndSubtaskFetcher{
		mockParentFetcher:  &mockParentFetcher{parent: parentWorkUnit},
		mockSubtaskFetcher: &mockSubtaskFetcher{subtasks: siblings},
	}

	result, err := c.FetchHierarchicalContext(context.Background(), mockProvider, workUnit, true)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotNil(t, result.Parent)
	assert.Equal(t, "parent-123", result.Parent.ID)

	// Siblings should exclude the current task
	require.NotNil(t, result.Siblings)
	assert.Len(t, result.Siblings, 2, "Current task should be filtered from siblings")
	assert.NotContains(t, []string{result.Siblings[0].ID, result.Siblings[1].ID}, "subtask-2")
}

func TestFetchHierarchicalContext_ParentFetchError(t *testing.T) {
	c := &Conductor{}

	workUnit := &workunit.WorkUnit{
		ID:    "subtask-123",
		Title: "Subtask",
		Metadata: map[string]any{
			"is_subtask": true,
			"parent_id":  "parent-123",
		},
	}

	mockProvider := &mockParentFetcher{
		err: assert.AnError,
	}

	result, err := c.FetchHierarchicalContext(context.Background(), mockProvider, workUnit, false)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestFetchHierarchicalContext_SubtaskFetchError(t *testing.T) {
	c := &Conductor{}

	workUnit := &workunit.WorkUnit{
		ID:    "subtask-123",
		Title: "Subtask",
		Metadata: map[string]any{
			"is_subtask": true,
			"parent_id":  "parent-123",
		},
	}

	parentWorkUnit := &workunit.WorkUnit{
		ID:    "parent-123",
		Title: "Parent Task",
	}

	mockProvider := &mockParentAndSubtaskFetcher{
		mockParentFetcher:  &mockParentFetcher{parent: parentWorkUnit},
		mockSubtaskFetcher: &mockSubtaskFetcher{err: assert.AnError},
	}

	result, err := c.FetchHierarchicalContext(context.Background(), mockProvider, workUnit, true)

	// Should not fail - just return parent without siblings
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotNil(t, result.Parent)
	assert.Nil(t, result.Siblings, "Siblings should be nil when subtask fetch fails")
}

func TestBuildHierarchyMetadata_NilHierarchy(t *testing.T) {
	result := BuildHierarchyMetadata(nil, nil)
	assert.Nil(t, result)
}

func TestBuildHierarchyMetadata_WithParent(t *testing.T) {
	workUnit := &workunit.WorkUnit{
		ID:    "subtask-1",
		Title: "Subtask 1",
	}

	hierarchy := &HierarchicalContext{
		Parent: &workunit.WorkUnit{
			ID:          "parent-123",
			Title:       "Parent Task",
			Description: "Parent description",
		},
	}

	result := BuildHierarchyMetadata(workUnit, hierarchy)

	require.NotNil(t, result)
	assert.Equal(t, "parent-123", result.ParentID)
	assert.Equal(t, "Parent Task", result.ParentTitle)
	assert.Nil(t, result.SiblingIDs)
}

func TestBuildHierarchyMetadata_WithSiblings(t *testing.T) {
	workUnit := &workunit.WorkUnit{
		ID:    "subtask-1",
		Title: "Subtask 1",
	}

	hierarchy := &HierarchicalContext{
		Parent: &workunit.WorkUnit{
			ID:    "parent-123",
			Title: "Parent Task",
		},
		Siblings: []*workunit.WorkUnit{
			{ID: "subtask-2", Title: "Subtask 2"},
			{ID: "subtask-3", Title: "Subtask 3"},
		},
	}

	result := BuildHierarchyMetadata(workUnit, hierarchy)

	require.NotNil(t, result)
	assert.Equal(t, "parent-123", result.ParentID)
	assert.Equal(t, "Parent Task", result.ParentTitle)
	require.NotNil(t, result.SiblingIDs)
	assert.Equal(t, []string{"subtask-2", "subtask-3"}, result.SiblingIDs)
}

func TestBuildHierarchyMetadata_NilParent(t *testing.T) {
	workUnit := &workunit.WorkUnit{
		ID:    "task-1",
		Title: "Task 1",
	}

	hierarchy := &HierarchicalContext{
		Parent: nil,
		Siblings: []*workunit.WorkUnit{
			{ID: "sibling-1", Title: "Sibling 1"},
		},
	}

	result := BuildHierarchyMetadata(workUnit, hierarchy)

	require.NotNil(t, result)
	assert.Empty(t, result.ParentID)
	assert.Empty(t, result.ParentTitle)
	assert.Equal(t, []string{"sibling-1"}, result.SiblingIDs)
}

func TestIsSubtask_WithIsSubtaskFlag(t *testing.T) {
	tests := []struct {
		name     string
		workUnit *workunit.WorkUnit
		want     bool
	}{
		{
			name: "explicit true flag",
			workUnit: &workunit.WorkUnit{
				ID: "task-1",
				Metadata: map[string]any{
					"is_subtask": true,
				},
			},
			want: true,
		},
		{
			name: "explicit false flag",
			workUnit: &workunit.WorkUnit{
				ID: "task-2",
				Metadata: map[string]any{
					"is_subtask": false,
				},
			},
			want: false,
		},
		{
			name: "parent_id present",
			workUnit: &workunit.WorkUnit{
				ID: "task-3",
				Metadata: map[string]any{
					"parent_id": "parent-123",
				},
			},
			want: true,
		},
		{
			name: "GitHub task pattern",
			workUnit: &workunit.WorkUnit{
				ID:       "github:123:task-456",
				Metadata: map[string]any{},
			},
			want: true,
		},
		{
			name: "GitLab task pattern",
			workUnit: &workunit.WorkUnit{
				ID:       "gitlab:123-task-456",
				Metadata: map[string]any{},
			},
			want: true,
		},
		{
			name: "regular task",
			workUnit: &workunit.WorkUnit{
				ID:       "task-123",
				Metadata: map[string]any{},
			},
			want: false,
		},
		{
			name:     "nil work unit",
			workUnit: nil,
			want:     false,
		},
		{
			name: "nil metadata",
			workUnit: &workunit.WorkUnit{
				ID:       "task-123",
				Metadata: nil,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSubtask(tt.workUnit)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestFilterSelf(t *testing.T) {
	siblings := []*workunit.WorkUnit{
		{ID: "sibling-1", Title: "Sibling 1"},
		{ID: "self-id", Title: "Self"},
		{ID: "sibling-2", Title: "Sibling 2"},
		{ID: "sibling-3", Title: "Sibling 3"},
	}

	result := filterSelf(siblings, "self-id")

	assert.Len(t, result, 3, "Self should be filtered out")
	assert.NotContains(t, getIDs(result), "self-id")
	assert.Contains(t, getIDs(result), "sibling-1")
	assert.Contains(t, getIDs(result), "sibling-2")
	assert.Contains(t, getIDs(result), "sibling-3")
}

// Helper function to extract IDs from work units.
func getIDs(workUnits []*workunit.WorkUnit) []string {
	ids := make([]string, len(workUnits))
	for i, wu := range workUnits {
		ids[i] = wu.ID
	}

	return ids
}
