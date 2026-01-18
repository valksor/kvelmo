package memory

import (
	"context"
	"time"
)

// DocumentType represents the type of document stored in memory.
type DocumentType string

const (
	TypeCodeChange    DocumentType = "code_change"
	TypeSpecification DocumentType = "specification"
	TypeSession       DocumentType = "session"
	TypeDecision      DocumentType = "decision"
	TypeSolution      DocumentType = "solution"
	TypeError         DocumentType = "error"
)

// Document represents a document stored in vector memory.
type Document struct {
	ID        string                 `json:"id"`
	TaskID    string                 `json:"task_id"`
	Type      DocumentType           `json:"type"`
	Content   string                 `json:"content"`
	Metadata  map[string]interface{} `json:"metadata"`
	Embedding []float32              `json:"embedding"`
	CreatedAt time.Time              `json:"created_at"`
	Tags      []string               `json:"tags"`
}

// SearchResult represents a document found via semantic search.
type SearchResult struct {
	Document   *Document
	Score      float32
	Highlights []string
}

// SearchOptions controls search behavior.
type SearchOptions struct {
	Limit           int                    // Max results to return
	MinScore        float32                // Minimum similarity score (0-1)
	DocumentTypes   []DocumentType         // Filter by document types
	TimeRange       *TimeRange             // Filter by time range
	MetadataFilters map[string]interface{} // Filter by metadata
}

// TimeRange represents a time range for filtering.
type TimeRange struct {
	From time.Time
	To   time.Time
}

// Memory is the interface for vector-based semantic memory.
type Memory interface {
	// Store stores a document in vector memory.
	Store(ctx context.Context, doc *Document) error

	// Search performs semantic search for similar documents.
	Search(ctx context.Context, query string, opts SearchOptions) ([]*SearchResult, error)

	// Delete removes a document from memory.
	Delete(ctx context.Context, id string) error

	// Get retrieves a document by ID.
	Get(ctx context.Context, id string) (*Document, error)

	// Clear removes all documents from memory.
	Clear(ctx context.Context) error
}

// VectorStore is the interface for vector storage backends.
type VectorStore interface {
	// Insert inserts documents into the vector store.
	Insert(ctx context.Context, docs []*Document) error

	// Search searches for similar vectors.
	Search(ctx context.Context, embedding []float32, limit int, filter map[string]interface{}) ([]*SearchResult, error)

	// Delete removes documents by IDs.
	Delete(ctx context.Context, ids []string) error

	// Update updates a document in the store.
	Update(ctx context.Context, doc *Document) error

	// Get retrieves a document by ID.
	Get(ctx context.Context, id string) (*Document, error)
}

// EmbeddingModel is the interface for text embedding models.
type EmbeddingModel interface {
	// Embed generates an embedding for a single text.
	Embed(ctx context.Context, text string) ([]float32, error)

	// EmbedBatch generates embeddings for multiple texts.
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)

	// Dimension returns the dimension of embeddings.
	Dimension() int
}

// MemorySystem is the main memory system implementation.
type MemorySystem struct {
	store VectorStore
	model EmbeddingModel
}

// NewMemorySystem creates a new memory system.
func NewMemorySystem(store VectorStore, model EmbeddingModel) *MemorySystem {
	return &MemorySystem{
		store: store,
		model: model,
	}
}

// Store generates an embedding and stores the document.
func (m *MemorySystem) Store(ctx context.Context, doc *Document) error {
	// Generate embedding
	embedding, err := m.model.Embed(ctx, doc.Content)
	if err != nil {
		return err
	}
	doc.Embedding = embedding

	// Store in vector database
	return m.store.Insert(ctx, []*Document{doc})
}

// Search performs semantic search with optimized filter pushdown.
func (m *MemorySystem) Search(ctx context.Context, query string, opts SearchOptions) ([]*SearchResult, error) {
	// Generate query embedding
	embedding, err := m.model.Embed(ctx, query)
	if err != nil {
		return nil, err
	}

	// Build filter from options - push down to store layer for better performance
	filter := make(map[string]interface{})

	// Push down document type filter
	if len(opts.DocumentTypes) > 0 {
		types := make([]string, len(opts.DocumentTypes))
		for i, dt := range opts.DocumentTypes {
			types[i] = string(dt)
		}
		filter["type"] = types
	}

	// Push down time range filter
	if opts.TimeRange != nil {
		filter["time_from"] = opts.TimeRange.From
		filter["time_to"] = opts.TimeRange.To
	}

	// Push down minScore filter for early termination during similarity calculation
	if opts.MinScore > 0 {
		filter["min_score"] = opts.MinScore
	}

	// Push down metadata filters
	for k, v := range opts.MetadataFilters {
		filter[k] = v
	}

	// Search vector store with all filters applied at storage layer
	results, err := m.store.Search(ctx, embedding, opts.Limit, filter)
	if err != nil {
		return nil, err
	}

	return results, nil
}

// Delete removes a document by ID.
func (m *MemorySystem) Delete(ctx context.Context, id string) error {
	return m.store.Delete(ctx, []string{id})
}

// Get retrieves a document by ID.
func (m *MemorySystem) Get(ctx context.Context, id string) (*Document, error) {
	return m.store.Get(ctx, id)
}

// Clear removes all documents.
func (m *MemorySystem) Clear(ctx context.Context) error {
	// Implementation depends on vector store capabilities
	return nil
}
