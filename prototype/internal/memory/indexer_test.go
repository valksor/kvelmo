package memory

import (
	"context"
	"testing"

	"github.com/valksor/go-mehrhof/internal/storage"
)

func TestNewIndexer(t *testing.T) {
	mem := &MockMemory{}
	ws := &storage.Workspace{}

	indexer := NewIndexer(mem, ws, nil)

	if indexer == nil {
		t.Fatal("NewIndexer returned nil")
	}
	if indexer.memory != mem {
		t.Error("memory not set correctly")
	}
	if indexer.workspace != ws {
		t.Error("workspace not set correctly")
	}
	if indexer.git != nil {
		t.Error("git should be nil")
	}
}

func TestIndexer_SearchSimilar(t *testing.T) {
	ctx := context.Background()
	mem := &MockMemory{
		searchResults: []*SearchResult{
			{
				Document: &Document{
					TaskID:  "task1",
					Type:    TypeSpecification,
					Content: "Spec for auth",
				},
				Score: 0.8,
			},
			{
				Document: &Document{
					TaskID:  "task2",
					Type:    TypeCodeChange,
					Content: "Code for auth",
				},
				Score: 0.75,
			},
		},
	}

	indexer := &Indexer{
		memory: mem,
	}

	results, err := indexer.SearchSimilar(ctx, "auth", 5)
	if err != nil {
		t.Fatalf("SearchSimilar failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	// Verify correct types were searched (specs and code changes)
	if results[0].Document.Type != TypeSpecification {
		t.Error("first result should be specification")
	}
}

func TestIndexer_SearchSimilar_EmptyResults(t *testing.T) {
	ctx := context.Background()
	mem := &MockMemory{
		searchResults: []*SearchResult{},
	}

	indexer := &Indexer{
		memory: mem,
	}

	results, err := indexer.SearchSimilar(ctx, "nonexistent", 5)
	if err != nil {
		t.Fatalf("SearchSimilar failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestIndexer_SearchSimilar_SearchError(t *testing.T) {
	ctx := context.Background()
	mem := &MockMemory{
		searchError: &testError{"search failed"},
	}

	indexer := &Indexer{
		memory: mem,
	}

	_, err := indexer.SearchSimilar(ctx, "test", 5)

	if err == nil {
		t.Error("expected error when search fails")
	}
}

func TestIndexer_FormatResults(t *testing.T) {
	results := []*SearchResult{
		{
			Document: &Document{
				TaskID:  "task1",
				Type:    TypeSolution,
				Content: "This is a solution for authentication",
			},
			Score: 0.85,
		},
		{
			Document: &Document{
				TaskID:  "task2",
				Type:    TypeCodeChange,
				Content: "Code changes",
			},
			Score: 0.72,
		},
	}

	indexer := &Indexer{}

	formatted := indexer.FormatResults(results)

	if formatted == "" {
		t.Fatal("FormatResults returned empty string")
	}

	// Check for expected content
	if !containsString(formatted, "Found 2 similar task") {
		t.Error("should mention found 2 tasks")
	}
	if !containsString(formatted, "task1") {
		t.Error("should contain task1")
	}
	if !containsString(formatted, "0.85") {
		t.Error("should contain score")
	}
}

func TestIndexer_FormatResults_Empty(t *testing.T) {
	indexer := &Indexer{}

	formatted := indexer.FormatResults([]*SearchResult{})

	if formatted != "No similar tasks found." {
		t.Errorf("expected 'No similar tasks found.', got %q", formatted)
	}
}

func TestIndexer_FormatResults_LongContentTruncated(t *testing.T) {
	longContent := string(make([]byte, 600))
	for i := range longContent {
		longContent = longContent[:i] + "x" + longContent[i+1:]
	}

	results := []*SearchResult{
		{
			Document: &Document{
				TaskID:  "task1",
				Type:    TypeSolution,
				Content: longContent,
			},
			Score: 0.9,
		},
	}

	indexer := &Indexer{}

	formatted := indexer.FormatResults(results)

	// Long content should be truncated
	if !containsString(formatted, "...") {
		t.Error("long content should be truncated with '...'")
	}

	// Full content should not be present
	if containsString(formatted, longContent) {
		t.Error("full long content should not be present")
	}
}

func TestIndexer_GetStats(t *testing.T) {
	ctx := context.Background()
	mem := &MockMemory{
		searchResults: []*SearchResult{
			{
				Document: &Document{
					Type: TypeSpecification,
				},
			},
			{
				Document: &Document{
					Type: TypeSolution,
				},
			},
			{
				Document: &Document{
					Type: TypeCodeChange,
				},
			},
			{
				Document: &Document{
					Type: TypeSolution, // Second solution
				},
			},
		},
	}

	indexer := &Indexer{
		memory: mem,
	}

	stats, err := indexer.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	if stats.TotalDocuments != 4 {
		t.Errorf("expected 4 total documents, got %d", stats.TotalDocuments)
	}

	if stats.ByType["solution"] != 2 {
		t.Errorf("expected 2 solutions, got %d", stats.ByType["solution"])
	}

	if stats.ByType["specification"] != 1 {
		t.Errorf("expected 1 specification, got %d", stats.ByType["specification"])
	}

	if stats.ByType["code_change"] != 1 {
		t.Errorf("expected 1 code_change, got %d", stats.ByType["code_change"])
	}
}

func TestIndexer_GetStats_Empty(t *testing.T) {
	ctx := context.Background()
	mem := &MockMemory{
		searchResults: []*SearchResult{},
	}

	indexer := &Indexer{
		memory: mem,
	}

	stats, err := indexer.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	if stats.TotalDocuments != 0 {
		t.Errorf("expected 0 total documents, got %d", stats.TotalDocuments)
	}

	if len(stats.ByType) != 0 {
		t.Errorf("expected empty ByType map, got %v", stats.ByType)
	}
}

func TestIndexer_ClearForTask(t *testing.T) {
	ctx := context.Background()

	// Create a mock that tracks deletions
	var deletedIDs []string
	trackingMem := &DeleteTrackingMock{
		searchResults: []*SearchResult{
			{
				Document: &Document{
					ID:     "doc1",
					TaskID: "task1",
				},
			},
			{
				Document: &Document{
					ID:     "doc2",
					TaskID: "task1",
				},
			},
		},
		onDelete: func(id string) error {
			deletedIDs = append(deletedIDs, id)

			return nil
		},
	}

	indexer := &Indexer{
		memory: trackingMem,
	}

	err := indexer.ClearForTask(ctx, "task1")
	if err != nil {
		t.Fatalf("ClearForTask failed: %v", err)
	}

	if len(deletedIDs) != 2 {
		t.Errorf("expected 2 deletions, got %d", len(deletedIDs))
	}
}

// DeleteTrackingMock is a mock Memory that tracks deletions.
type DeleteTrackingMock struct {
	searchResults []*SearchResult
	onDelete      func(id string) error
}

func (m *DeleteTrackingMock) Store(ctx context.Context, doc *Document) error {
	return nil
}

func (m *DeleteTrackingMock) Search(ctx context.Context, query string, opts SearchOptions) ([]*SearchResult, error) {
	return m.searchResults, nil
}

func (m *DeleteTrackingMock) Delete(ctx context.Context, id string) error {
	if m.onDelete != nil {
		return m.onDelete(id)
	}

	return nil
}

func (m *DeleteTrackingMock) Get(ctx context.Context, id string) (*Document, error) {
	return &Document{ID: id}, nil
}

func (m *DeleteTrackingMock) Clear(ctx context.Context) error {
	return nil
}

// testError is a simple error type for testing.
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
