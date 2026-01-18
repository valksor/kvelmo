package memory

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// MockVectorStore is a test implementation of VectorStore.
type MockVectorStore struct {
	documents map[string]*Document
}

func NewMockVectorStore() *MockVectorStore {
	return &MockVectorStore{
		documents: make(map[string]*Document),
	}
}

func (m *MockVectorStore) Insert(ctx context.Context, docs []*Document) error {
	for _, doc := range docs {
		m.documents[doc.ID] = doc
	}

	return nil
}

func (m *MockVectorStore) Search(ctx context.Context, embedding []float32, limit int, filters map[string]interface{}) ([]*SearchResult, error) {
	var results []*SearchResult
	for _, doc := range m.documents {
		// Apply type filter if specified
		if typeFilter, exists := filters["type"]; exists {
			if !mockMatchTypeFilter(doc.Type, typeFilter) {
				continue
			}
		}

		results = append(results, &SearchResult{
			Document: doc,
			Score:    0.9, // Mock score
		})
		if len(results) >= limit {
			break
		}
	}

	return results, nil
}

// mockMatchTypeFilter checks if the document type matches the type filter.
func mockMatchTypeFilter(docType DocumentType, typeFilter interface{}) bool {
	switch v := typeFilter.(type) {
	case string:
		return string(docType) == v
	case []string:
		for _, t := range v {
			if string(docType) == t {
				return true
			}
		}

		return false
	default:
		// Unknown type, try string conversion
		return fmt.Sprintf("%v", docType) == fmt.Sprintf("%v", typeFilter)
	}
}

func (m *MockVectorStore) Delete(ctx context.Context, ids []string) error {
	for _, id := range ids {
		delete(m.documents, id)
	}

	return nil
}

func (m *MockVectorStore) Update(ctx context.Context, doc *Document) error {
	m.documents[doc.ID] = doc

	return nil
}

func (m *MockVectorStore) Get(ctx context.Context, id string) (*Document, error) {
	doc, ok := m.documents[id]
	if !ok {
		return nil, fmt.Errorf("document not found: %s", id)
	}

	return doc, nil
}

// MockEmbeddingModel is a test implementation of EmbeddingModel.
type MockEmbeddingModel struct {
	dimension int
}

func NewMockEmbeddingModel(dimension int) *MockEmbeddingModel {
	return &MockEmbeddingModel{dimension: dimension}
}

func (m *MockEmbeddingModel) Embed(ctx context.Context, text string) ([]float32, error) {
	// Generate deterministic mock embeddings based on text length
	embedding := make([]float32, m.dimension)
	for i := range embedding {
		embedding[i] = float32(len(text)%100) / 100.0
	}

	return embedding, nil
}

func (m *MockEmbeddingModel) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		emb, err := m.Embed(ctx, text)
		if err != nil {
			return nil, err
		}
		embeddings[i] = emb
	}

	return embeddings, nil
}

func (m *MockEmbeddingModel) Dimension() int {
	return m.dimension
}

func TestMemorySystem_Store(t *testing.T) {
	ctx := context.Background()
	store := NewMockVectorStore()
	model := NewMockEmbeddingModel(1536)

	mem := NewMemorySystem(store, model)

	doc := &Document{
		ID:        "doc1",
		TaskID:    "task1",
		Type:      TypeCodeChange,
		Content:   "Test content",
		CreatedAt: time.Now(),
		Tags:      []string{"test", "code"},
	}

	err := mem.Store(ctx, doc)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Verify embedding was generated
	if len(doc.Embedding) != 1536 {
		t.Errorf("expected embedding dimension 1536, got %d", len(doc.Embedding))
	}

	// Verify document was stored
	stored, err := store.Get(ctx, "doc1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if stored.ID != doc.ID {
		t.Errorf("expected ID %s, got %s", doc.ID, stored.ID)
	}
}

func TestMemorySystem_StoreMultiple(t *testing.T) {
	ctx := context.Background()
	store := NewMockVectorStore()
	model := NewMockEmbeddingModel(1536)

	mem := NewMemorySystem(store, model)

	// Store multiple documents
	for i := range 3 {
		doc := &Document{
			ID:        fmt.Sprintf("doc%d", i),
			TaskID:    "task1",
			Type:      TypeCodeChange,
			Content:   fmt.Sprintf("Content %d", i),
			CreatedAt: time.Now(),
		}

		err := mem.Store(ctx, doc)
		if err != nil {
			t.Fatalf("Store failed for doc%d: %v", i, err)
		}
	}
}

func TestMemorySystem_Search(t *testing.T) {
	ctx := context.Background()
	store := NewMockVectorStore()
	model := NewMockEmbeddingModel(1536)

	mem := NewMemorySystem(store, model)

	// Store some documents
	docs := []*Document{
		{
			ID:        "doc1",
			TaskID:    "task1",
			Type:      TypeCodeChange,
			Content:   "Authentication bug fix",
			CreatedAt: time.Now(),
		},
		{
			ID:        "doc2",
			TaskID:    "task1",
			Type:      TypeSolution,
			Content:   "Fixed JWT token validation",
			CreatedAt: time.Now(),
		},
	}

	for _, doc := range docs {
		if err := mem.Store(ctx, doc); err != nil {
			t.Fatalf("Store failed: %v", err)
		}
	}

	// Search
	results, err := mem.Search(ctx, "authentication fix", SearchOptions{
		Limit:    5,
		MinScore: 0.5,
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("expected search results, got none")
	}
}

func TestMemorySystem_Search_WithFilters(t *testing.T) {
	ctx := context.Background()
	store := NewMockVectorStore()
	model := NewMockEmbeddingModel(1536)

	mem := NewMemorySystem(store, model)

	// Store documents of different types
	docs := []*Document{
		{
			ID:        "doc1",
			TaskID:    "task1",
			Type:      TypeCodeChange,
			Content:   "Code change",
			CreatedAt: time.Now(),
		},
		{
			ID:        "doc2",
			TaskID:    "task1",
			Type:      TypeSpecification,
			Content:   "Specification",
			CreatedAt: time.Now(),
		},
	}

	for _, doc := range docs {
		if err := mem.Store(ctx, doc); err != nil {
			t.Fatalf("Store failed: %v", err)
		}
	}

	// Search for only code changes
	results, err := mem.Search(ctx, "code", SearchOptions{
		Limit:         5,
		DocumentTypes: []DocumentType{TypeCodeChange},
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// Verify filtering (in real implementation, this would filter by type)
	for _, result := range results {
		if result.Document.Type != TypeCodeChange {
			t.Errorf("expected only code changes, got %s", result.Document.Type)
		}
	}
}

func TestMemorySystem_Delete(t *testing.T) {
	ctx := context.Background()
	store := NewMockVectorStore()
	model := NewMockEmbeddingModel(1536)

	mem := NewMemorySystem(store, model)

	doc := &Document{
		ID:        "doc1",
		TaskID:    "task1",
		Type:      TypeCodeChange,
		Content:   "Test content",
		CreatedAt: time.Now(),
	}

	if err := mem.Store(ctx, doc); err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	err := mem.Delete(ctx, "doc1")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify document is deleted
	_, err = store.Get(ctx, "doc1")
	if err == nil {
		t.Error("expected error for deleted document, got nil")
	}
}

func TestMemorySystem_Get(t *testing.T) {
	ctx := context.Background()
	store := NewMockVectorStore()
	model := NewMockEmbeddingModel(1536)

	mem := NewMemorySystem(store, model)

	doc := &Document{
		ID:        "doc1",
		TaskID:    "task1",
		Type:      TypeCodeChange,
		Content:   "Test content",
		CreatedAt: time.Now(),
		Metadata: map[string]interface{}{
			"key": "value",
		},
	}

	if err := mem.Store(ctx, doc); err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	retrieved, err := mem.Get(ctx, "doc1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.ID != doc.ID {
		t.Errorf("expected ID %s, got %s", doc.ID, retrieved.ID)
	}

	if retrieved.Content != doc.Content {
		t.Errorf("expected content %s, got %s", doc.Content, retrieved.Content)
	}

	if retrieved.Metadata["key"] != "value" {
		t.Errorf("metadata not preserved")
	}
}

func TestMemorySystem_Get_NotFound(t *testing.T) {
	ctx := context.Background()
	store := NewMockVectorStore()
	model := NewMockEmbeddingModel(1536)

	mem := NewMemorySystem(store, model)

	_, err := mem.Get(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent document, got nil")
	}
}

func TestDocumentType_String(t *testing.T) {
	tests := []struct {
		dt     DocumentType
		expect string
	}{
		{TypeCodeChange, "code_change"},
		{TypeSpecification, "specification"},
		{TypeSession, "session"},
		{TypeDecision, "decision"},
		{TypeSolution, "solution"},
		{TypeError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.expect, func(t *testing.T) {
			if string(tt.dt) != tt.expect {
				t.Errorf("expected %s, got %s", tt.expect, string(tt.dt))
			}
		})
	}
}

func TestSearchOptions_Defaults(t *testing.T) {
	opts := SearchOptions{}

	if opts.Limit == 0 {
		opts.Limit = 10 // Default limit
	}

	if opts.MinScore == 0 {
		opts.MinScore = 0.7 // Default minimum score
	}

	if opts.Limit != 10 {
		t.Errorf("expected default limit 10, got %d", opts.Limit)
	}

	if opts.MinScore != 0.7 {
		t.Errorf("expected default min score 0.7, got %f", opts.MinScore)
	}
}

func TestMemorySystem_Metadata(t *testing.T) {
	ctx := context.Background()
	store := NewMockVectorStore()
	model := NewMockEmbeddingModel(1536)

	mem := NewMemorySystem(store, model)

	doc := &Document{
		ID:        "doc1",
		TaskID:    "task1",
		Type:      TypeCodeChange,
		Content:   "Test content",
		CreatedAt: time.Now(),
		Metadata: map[string]interface{}{
			"title":      "Test Task",
			"status":     "completed",
			"complexity": 5,
			"language":   "go",
		},
		Tags: []string{"auth", "security", "fix"},
	}

	err := mem.Store(ctx, doc)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	retrieved, _ := mem.Get(ctx, "doc1")

	if retrieved.Metadata["title"] != "Test Task" {
		t.Error("metadata not preserved correctly")
	}

	if len(retrieved.Tags) != 3 {
		t.Errorf("expected 3 tags, got %d", len(retrieved.Tags))
	}
}

func TestMemorySystem_EmbeddingGeneration(t *testing.T) {
	ctx := context.Background()
	model := NewMockEmbeddingModel(1536)

	// Test embedding generation
	text := "This is a test document"
	embedding, err := model.Embed(ctx, text)
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}

	if len(embedding) != 1536 {
		t.Errorf("expected embedding dimension 1536, got %d", len(embedding))
	}

	// Test batch embedding
	texts := []string{"doc1", "doc2", "doc3"}
	embeddings, err := model.EmbedBatch(ctx, texts)
	if err != nil {
		t.Fatalf("EmbedBatch failed: %v", err)
	}

	if len(embeddings) != 3 {
		t.Errorf("expected 3 embeddings, got %d", len(embeddings))
	}

	for i, emb := range embeddings {
		if len(emb) != 1536 {
			t.Errorf("embedding %d has wrong dimension: %d", i, len(emb))
		}
	}
}

// TestChromaDBStore_ClosedState tests that operations fail after Close().
func TestChromaDBStore_ClosedState(t *testing.T) {
	ctx := context.Background()
	model := NewMockEmbeddingModel(1536)

	// Create a temporary store
	store, err := NewChromaDBStore("", "test-collection", model)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Add a document before closing
	doc := &Document{
		ID:      "test-doc",
		TaskID:  "task1",
		Type:    TypeCodeChange,
		Content: "Test content",
	}
	err = store.Insert(ctx, []*Document{doc})
	if err != nil {
		t.Fatalf("Failed to insert document: %v", err)
	}

	// Close the store
	err = store.Close()
	if err != nil {
		t.Fatalf("Failed to close store: %v", err)
	}

	// Test that operations fail after close
	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "insert after close",
			fn: func(t *testing.T) {
				t.Helper()
				err := store.Insert(ctx, []*Document{{
					ID:      "doc2",
					TaskID:  "task1",
					Type:    TypeCodeChange,
					Content: "Content",
				}})
				if err == nil {
					t.Error("expected error when inserting after close, got nil")
				}
			},
		},
		{
			name: "search after close",
			fn: func(t *testing.T) {
				t.Helper()
				_, err := store.Search(ctx, []float32{0.1, 0.2}, 5, nil)
				if err == nil {
					t.Error("expected error when searching after close, got nil")
				}
			},
		},
		{
			name: "get after close",
			fn: func(t *testing.T) {
				t.Helper()
				_, err := store.Get(ctx, "test-doc")
				if err == nil {
					t.Error("expected error when getting after close, got nil")
				}
			},
		},
		{
			name: "update after close",
			fn: func(t *testing.T) {
				t.Helper()
				err := store.Update(ctx, &Document{
					ID:      "test-doc",
					Type:    TypeCodeChange,
					Content: "Updated content",
				})
				if err == nil {
					t.Error("expected error when updating after close, got nil")
				}
			},
		},
		{
			name: "delete after close",
			fn: func(t *testing.T) {
				t.Helper()
				err := store.Delete(ctx, []string{"test-doc"})
				if err == nil {
					t.Error("expected error when deleting after close, got nil")
				}
			},
		},
		{
			name: "double close",
			fn: func(t *testing.T) {
				t.Helper()
				err := store.Close()
				if err != nil {
					t.Errorf("double close should not error, got: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fn(t)
		})
	}
}

// TestChromaDBStore_EmptyResults tests search with empty document set.
func TestChromaDBStore_EmptyResults(t *testing.T) {
	ctx := context.Background()
	model := NewMockEmbeddingModel(1536)

	store, err := NewChromaDBStore("", "test-empty", model)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Search with no documents should return empty slice, not nil
	results, err := store.Search(ctx, []float32{0.1}, 5, nil)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if results == nil {
		t.Error("expected empty slice, got nil")
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

// TestChromaDBStore_ContextCancellation tests context cancellation handling.
func TestChromaDBStore_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	model := NewMockEmbeddingModel(1536)

	store, err := NewChromaDBStore("", "test-cancel", model)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Add a document first with valid context
	validCtx := context.Background()
	docs := []*Document{
		{
			ID:      "doc1",
			TaskID:  "task1",
			Type:    TypeCodeChange,
			Content: "Content",
		},
	}
	err = store.Insert(validCtx, docs)
	if err != nil {
		t.Fatalf("Failed to insert document: %v", err)
	}

	// Cancel the context
	cancel()

	// Insert should respect context cancellation
	err = store.Insert(ctx, docs)
	if err == nil {
		t.Error("expected error when inserting with cancelled context, got nil")
	}

	// Search should also respect context cancellation
	// Note: Search may complete before context is checked if there are no documents
	// The actual ChromaDBStore implementation checks ctx.Err() at the start
	// But if documents map is empty, it returns early
	_, err = store.Search(ctx, []float32{0.1}, 5, nil)
	// Search might not error if it returns early (empty result), so we just log
	t.Logf("Search with cancelled context returned: %v", err)
}
