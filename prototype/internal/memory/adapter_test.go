package memory

import (
	"context"
	"errors"
	"testing"
)

// MockMemory is a mock Memory implementation for testing.
type MockMemory struct {
	storeError  error
	searchError error
	deleteError error
	getError    error
	clearError  error

	searchResults []*SearchResult
	storedDocs    []*Document
	deletedIDs    []string
}

func (m *MockMemory) Store(ctx context.Context, doc *Document) error {
	m.storedDocs = append(m.storedDocs, doc)

	return m.storeError
}

func (m *MockMemory) Search(ctx context.Context, query string, opts SearchOptions) ([]*SearchResult, error) {
	if m.searchError != nil {
		return nil, m.searchError
	}

	return m.searchResults, nil
}

func (m *MockMemory) Delete(ctx context.Context, id string) error {
	m.deletedIDs = append(m.deletedIDs, id)

	return m.deleteError
}

func (m *MockMemory) Get(ctx context.Context, id string) (*Document, error) {
	if m.getError != nil {
		return nil, m.getError
	}

	return &Document{ID: id}, nil
}

func (m *MockMemory) Clear(ctx context.Context) error {
	return m.clearError
}

func TestMemoryTool_LearnFromCorrection(t *testing.T) {
	ctx := context.Background()
	mem := &MockMemory{}

	// Create tool with nil indexer - not all methods require indexer
	tool := &MemoryTool{
		memory:  mem,
		indexer: nil,
	}

	err := tool.LearnFromCorrection(ctx, "task1", "bug in auth", "fix: add validation")
	if err != nil {
		t.Fatalf("LearnFromCorrection failed: %v", err)
	}

	if len(mem.storedDocs) != 1 {
		t.Errorf("expected 1 stored document, got %d", len(mem.storedDocs))
	}

	doc := mem.storedDocs[0]
	if doc.Type != TypeSolution {
		t.Errorf("document type = %s, want %s", doc.Type, TypeSolution)
	}

	if doc.TaskID != "task1" {
		t.Error("task ID mismatch")
	}

	if !containsString(doc.Content, "bug in auth") {
		t.Error("content should contain problem")
	}

	if !containsString(doc.Content, "fix: add validation") {
		t.Error("content should contain solution")
	}

	expectedTags := []string{"solution", "fix", "learned"}
	for _, tag := range expectedTags {
		if !containsTag(doc.Tags, tag) {
			t.Errorf("missing tag: %s", tag)
		}
	}
}

func TestMemoryTool_LearnFromCorrection_StoreError(t *testing.T) {
	ctx := context.Background()
	mem := &MockMemory{
		storeError: errors.New("store failed"),
	}

	tool := &MemoryTool{
		memory:  mem,
		indexer: nil,
	}

	err := tool.LearnFromCorrection(ctx, "task1", "problem", "solution")

	if err == nil {
		t.Error("expected error when store fails")
	}
}

func TestMemoryTool_ClearMemory(t *testing.T) {
	ctx := context.Background()
	mem := &MockMemory{}

	tool := &MemoryTool{
		memory:  mem,
		indexer: nil,
	}

	err := tool.ClearMemory(ctx)
	if err != nil {
		t.Fatalf("ClearMemory failed: %v", err)
	}
}

func TestMemoryTool_ClearMemory_Error(t *testing.T) {
	ctx := context.Background()
	mem := &MockMemory{
		clearError: errors.New("clear failed"),
	}

	tool := &MemoryTool{
		memory:  mem,
		indexer: nil,
	}

	err := tool.ClearMemory(ctx)

	if err == nil {
		t.Error("expected error when clear fails")
	}
}

func TestFormatResult(t *testing.T) {
	result := &SearchResult{
		Document: &Document{
			TaskID:  "task1",
			Type:    TypeCodeChange,
			Content: "Fixed the bug in authentication",
			Metadata: map[string]interface{}{
				"file":     "auth.go",
				"lines":    10,
				"language": "go",
			},
		},
		Score: 0.85,
	}

	formatted := formatResult(result)

	if formatted == "" {
		t.Fatal("formatResult returned empty string")
	}

	// Check for expected content
	expectedStrings := []string{
		"task1",
		"0.85",
		"code_change",
		"Fixed the bug",
		"auth.go",
	}

	for _, s := range expectedStrings {
		if !containsString(formatted, s) {
			t.Errorf("formatted result should contain %q", s)
		}
	}
}

func TestFormatResult_LongContentTruncated(t *testing.T) {
	longContent := string(make([]byte, 300))
	for i := range longContent {
		longContent = longContent[:i] + "x" + longContent[i+1:]
	}

	result := &SearchResult{
		Document: &Document{
			TaskID:  "task1",
			Type:    TypeCodeChange,
			Content: longContent,
		},
		Score: 0.9,
	}

	formatted := formatResult(result)

	// Long content should be truncated
	if !containsString(formatted, "...") {
		t.Error("long content should be truncated with '...'")
	}

	if containsString(formatted, longContent) {
		t.Error("full long content should not be present")
	}
}

func TestFormatResult_NoMetadata(t *testing.T) {
	result := &SearchResult{
		Document: &Document{
			TaskID:   "task1",
			Type:     TypeCodeChange,
			Content:  "Simple fix",
			Metadata: nil,
		},
		Score: 0.9,
	}

	formatted := formatResult(result)

	if formatted == "" {
		t.Fatal("formatResult returned empty string")
	}

	// Should still format even without metadata
	if !containsString(formatted, "task1") {
		t.Error("result should contain task ID")
	}
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()

	if id1 == "" {
		t.Error("generateID should not return empty string")
	}

	// IDs are timestamp-based, so they might not always be unique if called very quickly
	// Just verify the format is a valid Unix nano timestamp string
	for _, c := range id1 {
		if c < '0' || c > '9' {
			t.Errorf("generateID should return numeric string, got %q", id1)
		}
	}
}

func TestMemoryTool_GetMemoryStats_NoIndexer(t *testing.T) {
	ctx := context.Background()
	tool := &MemoryTool{
		memory:  &MockMemory{},
		indexer: nil,
	}

	_, err := tool.GetMemoryStats(ctx)

	if err == nil {
		t.Error("expected error when no indexer")
	}

	if !containsString(err.Error(), "indexer not available") {
		t.Errorf("error message should mention indexer, got: %v", err)
	}
}

func TestNewMemoryTool(t *testing.T) {
	mem := &MockMemory{}
	var indexer *Indexer = nil // Using nil for test

	tool := NewMemoryTool(mem, indexer)

	if tool == nil {
		t.Fatal("NewMemoryTool returned nil")
	}
	if tool.memory != mem {
		t.Error("memory not set correctly")
	}
	if tool.indexer != indexer {
		t.Error("indexer not set correctly")
	}
}

func TestMemoryTool_SearchSimilarTasks(t *testing.T) {
	ctx := context.Background()
	mem := &MockMemory{
		searchResults: []*SearchResult{
			{
				Document: &Document{
					TaskID:  "task1",
					Type:    TypeSolution,
					Content: "Fixed auth bug",
				},
				Score: 0.8,
			},
			{
				Document: &Document{
					TaskID:  "task2",
					Type:    TypeSpecification,
					Content: "Spec for auth",
				},
				Score: 0.75,
			},
		},
	}

	tool := &MemoryTool{
		memory:  mem,
		indexer: nil,
	}

	results, err := tool.SearchSimilarTasks(ctx, "auth bug", 5)
	if err != nil {
		t.Fatalf("SearchSimilarTasks failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	// Check results are formatted
	if !containsString(results[0], "task1") {
		t.Error("first result should contain task1")
	}
}

func TestMemoryTool_SearchSimilarTasks_NoResults(t *testing.T) {
	ctx := context.Background()
	mem := &MockMemory{
		searchResults: []*SearchResult{},
	}

	tool := &MemoryTool{
		memory:  mem,
		indexer: nil,
	}

	results, err := tool.SearchSimilarTasks(ctx, "nonexistent", 5)
	if err != nil {
		t.Fatalf("SearchSimilarTasks failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestMemoryTool_SearchSimilarTasks_SearchError(t *testing.T) {
	ctx := context.Background()
	mem := &MockMemory{
		searchError: errors.New("search failed"),
	}

	tool := &MemoryTool{
		memory:  mem,
		indexer: nil,
	}

	_, err := tool.SearchSimilarTasks(ctx, "test", 5)

	if err == nil {
		t.Error("expected error when search fails")
	}
}

func TestMemoryTool_AugmentPrompt(t *testing.T) {
	ctx := context.Background()
	mem := &MockMemory{
		searchResults: []*SearchResult{
			{
				Document: &Document{
					TaskID:  "task1",
					Type:    TypeSolution,
					Content: "This is a solution for the auth problem",
				},
				Score: 0.8,
			},
		},
	}

	tool := &MemoryTool{
		memory:  mem,
		indexer: nil,
	}

	augmented, err := tool.AugmentPrompt(ctx, "auth fix", "fix authentication bug")
	if err != nil {
		t.Fatalf("AugmentPrompt failed: %v", err)
	}

	if augmented == "" {
		t.Error("AugmentPrompt returned empty string")
	}

	// Check for expected sections
	if !containsString(augmented, "Relevant Context from Similar Tasks") {
		t.Error("augmented prompt should contain context header")
	}
	if !containsString(augmented, "task1") {
		t.Error("augmented prompt should contain task ID")
	}
	if !containsString(augmented, "80%") {
		t.Error("augmented prompt should contain similarity score")
	}
}

func TestMemoryTool_AugmentPrompt_NoResults(t *testing.T) {
	ctx := context.Background()
	mem := &MockMemory{
		searchResults: []*SearchResult{},
	}

	tool := &MemoryTool{
		memory:  mem,
		indexer: nil,
	}

	_, err := tool.AugmentPrompt(ctx, "test", "description")

	if err == nil {
		t.Error("expected error when no similar tasks found")
	}

	if !containsString(err.Error(), "no similar tasks") {
		t.Errorf("error should mention no similar tasks, got: %v", err)
	}
}

func TestMemoryTool_AugmentPrompt_LongContentTruncated(t *testing.T) {
	ctx := context.Background()
	longContent := string(make([]byte, 400))
	for i := range longContent {
		longContent = longContent[:i] + "x" + longContent[i+1:]
	}

	mem := &MockMemory{
		searchResults: []*SearchResult{
			{
				Document: &Document{
					TaskID:  "task1",
					Type:    TypeSolution,
					Content: longContent,
				},
				Score: 0.8,
			},
		},
	}

	tool := &MemoryTool{
		memory:  mem,
		indexer: nil,
	}

	augmented, err := tool.AugmentPrompt(ctx, "test", "description")
	if err != nil {
		t.Fatalf("AugmentPrompt failed: %v", err)
	}

	// Long content should be truncated
	if !containsString(augmented, "...") {
		t.Error("long content should be truncated with '...'")
	}

	// Full content should not be present
	if containsString(augmented, longContent) {
		t.Error("full long content should not be present")
	}
}

func TestMemoryTool_GetCodeExamples(t *testing.T) {
	ctx := context.Background()
	mem := &MockMemory{
		searchResults: []*SearchResult{
			{
				Document: &Document{
					Type:    TypeCodeChange,
					Content: "func example() { return true }",
				},
				Score: 0.7,
			},
			{
				Document: &Document{
					Type:    TypeCodeChange,
					Content: "func another() { return false }",
				},
				Score: 0.65,
			},
		},
	}

	tool := &MemoryTool{
		memory:  mem,
		indexer: nil,
	}

	examples, err := tool.GetCodeExamples(ctx, "go", "authentication")
	if err != nil {
		t.Fatalf("GetCodeExamples failed: %v", err)
	}

	if len(examples) != 2 {
		t.Errorf("expected 2 examples, got %d", len(examples))
	}

	if examples[0] != "func example() { return true }" {
		t.Error("first example content mismatch")
	}
}

func TestMemoryTool_GetCodeExamples_NoResults(t *testing.T) {
	ctx := context.Background()
	mem := &MockMemory{
		searchResults: []*SearchResult{},
	}

	tool := &MemoryTool{
		memory:  mem,
		indexer: nil,
	}

	_, err := tool.GetCodeExamples(ctx, "go", "nonexistent")

	if err == nil {
		t.Error("expected error when no code examples found")
	}

	if !containsString(err.Error(), "no code examples") {
		t.Errorf("error should mention no code examples, got: %v", err)
	}
}

// Helper functions

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsInString(s, substr))
}

func containsInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}

func containsTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}

	return false
}
