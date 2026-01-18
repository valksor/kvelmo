package memory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ChromaDBStore implements VectorStore with local file persistence.
// IMPORTANT: This is NOT an actual ChromaDB integration. This is a file-based
// vector store that provides a similar API for local-only operation without external dependencies.
// Documents are persisted to disk as JSON files for durability.
type ChromaDBStore struct {
	mu         sync.RWMutex
	path       string
	collection string
	documents  map[string]*Document
	embedding  EmbeddingModel
	dirty      bool // tracks if unsaved changes exist
	closed     bool // tracks if the store has been closed
}

// NewChromaDBStore creates a new ChromaDB-compatible vector store.
// Loads existing documents from disk if available.
func NewChromaDBStore(path, collection string, model EmbeddingModel) (*ChromaDBStore, error) {
	if path == "" {
		path = filepath.Join(".mehrhof", "vectors")
	}

	// Ensure directory exists
	if err := os.MkdirAll(path, 0o755); err != nil {
		return nil, fmt.Errorf("create vector directory: %w", err)
	}

	store := &ChromaDBStore{
		path:       path,
		collection: collection,
		documents:  make(map[string]*Document),
		embedding:  model,
		dirty:      false,
		closed:     false,
	}

	// Load existing documents from disk
	if err := store.loadFromDisk(); err != nil {
		// Only log if it's not a "not found" error (first run is ok)
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("load vectors from disk: %w", err)
		}
		// First run - directory doesn't exist yet, that's ok
	}

	return store, nil
}

// Close gracefully shuts down the store, saving any pending changes.
func (c *ChromaDBStore) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if already closed
	if c.closed {
		return nil
	}

	// Save any pending changes
	if c.dirty {
		if err := c.saveToDisk(); err != nil {
			return fmt.Errorf("save pending changes before shutdown: %w", err)
		}
	}

	// Mark as closed instead of setting documents to nil
	c.closed = true

	return nil
}

// loadFromDisk loads all documents from the storage directory.
func (c *ChromaDBStore) loadFromDisk() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	storePath := filepath.Join(c.path, c.collection)
	entries, err := os.ReadDir(storePath)
	if err != nil {
		if os.IsNotExist(err) {
			// First run - directory doesn't exist yet
			return os.MkdirAll(storePath, 0o755)
		}

		return fmt.Errorf("read store directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Skip non-JSON files
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		// Load document from file
		filePath := filepath.Join(storePath, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("read document file %s: %w", entry.Name(), err)
		}

		var doc Document
		if err := json.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("unmarshal document %s: %w", entry.Name(), err)
		}

		c.documents[doc.ID] = &doc
	}

	c.dirty = false

	return nil
}

// saveToDisk writes all documents to disk.
func (c *ChromaDBStore) saveToDisk() error {
	storePath := filepath.Join(c.path, c.collection)
	if err := os.MkdirAll(storePath, 0o755); err != nil {
		return fmt.Errorf("create store directory: %w", err)
	}

	// Write each document to a separate file
	for id, doc := range c.documents {
		filePath := filepath.Join(storePath, id+".json")
		data, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal document %s: %w", id, err)
		}

		if err := os.WriteFile(filePath, data, 0o644); err != nil {
			return fmt.Errorf("write document %s: %w", id, err)
		}
	}

	c.dirty = false

	return nil
}

// Insert stores documents in the vector store and persists to disk.
func (c *ChromaDBStore) Insert(ctx context.Context, docs []*Document) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if store is closed
	if c.closed {
		return errors.New("store is closed")
	}

	// Check context cancellation
	if ctx.Err() != nil {
		return ctx.Err()
	}

	for _, doc := range docs {
		// Check context before each expensive operation
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Generate ID if not present
		if doc.ID == "" {
			doc.ID = uuid.New().String()
		}

		// Generate embedding if not present
		if len(doc.Embedding) == 0 && doc.Content != "" {
			embedding, err := c.embedding.Embed(ctx, doc.Content)
			if err != nil {
				return fmt.Errorf("generate embedding for %s: %w", doc.ID, err)
			}
			doc.Embedding = embedding
		}

		// Store document
		c.documents[doc.ID] = doc
	}

	c.dirty = true

	// Persist to disk
	if err := c.saveToDisk(); err != nil {
		return fmt.Errorf("save to disk: %w", err)
	}

	return nil
}

// Search performs similarity search using cosine similarity.
// The filter map supports special keys for optimized filtering:
// - "min_score": float32 - minimum similarity threshold (applied during scoring)
// - "time_from": time.Time - only include documents created after this time
// - "time_to": time.Time - only include documents created before this time
// - "type": string or []string - filter by document type(s).
func (c *ChromaDBStore) Search(ctx context.Context, queryEmbedding []float32, limit int, filter map[string]interface{}) ([]*SearchResult, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Check if store is closed
	if c.closed {
		return nil, errors.New("store is closed")
	}

	// Check if we have any documents
	if len(c.documents) == 0 {
		return []*SearchResult{}, nil // Empty slice, not nil
	}

	// Check context cancellation
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Extract minScore from filter for early termination
	var minScore float32
	if ms, ok := filter["min_score"]; ok {
		if f, ok := ms.(float32); ok {
			minScore = f
		} else if f, ok := ms.(float64); ok {
			minScore = float32(f)
		}
	}

	// Calculate similarity with all documents
	type scoredDoc struct {
		doc   *Document
		score float32
	}

	var scored []scoredDoc
	for _, doc := range c.documents {
		// Apply filters (including time range)
		if !matchesFilter(doc, filter) {
			continue
		}

		// Skip documents without embeddings
		if len(doc.Embedding) == 0 {
			continue
		}

		// Calculate cosine similarity
		score := cosineSimilarity(queryEmbedding, doc.Embedding)

		// Early termination: skip if below minimum score
		if minScore > 0 && score < minScore {
			continue
		}

		scored = append(scored, scoredDoc{
			doc:   doc,
			score: score,
		})
	}

	// Sort by score (descending) using standard library sort
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// Return top N results
	if len(scored) == 0 {
		return []*SearchResult{}, nil // No matches found
	}

	count := len(scored)
	if limit > 0 && limit < count {
		count = limit
	}

	results := make([]*SearchResult, count)
	for i := range count {
		results[i] = &SearchResult{
			Document: scored[i].doc,
			Score:    scored[i].score,
		}
	}

	return results, nil
}

// Delete removes documents by ID and updates disk storage.
func (c *ChromaDBStore) Delete(ctx context.Context, ids []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if store is closed
	if c.closed {
		return errors.New("store is closed")
	}

	for _, id := range ids {
		delete(c.documents, id)

		// Remove the file from disk
		filePath := filepath.Join(c.path, c.collection, id+".json")
		if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("delete document file %s: %w", id, err)
		}
	}

	return nil
}

// Update updates a document and persists to disk.
func (c *ChromaDBStore) Update(ctx context.Context, doc *Document) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if store is closed
	if c.closed {
		return errors.New("store is closed")
	}

	if _, exists := c.documents[doc.ID]; !exists {
		return fmt.Errorf("document not found: %s", doc.ID)
	}

	// Regenerate embedding if content changed
	if len(doc.Embedding) == 0 && doc.Content != "" {
		embedding, err := c.embedding.Embed(ctx, doc.Content)
		if err != nil {
			return fmt.Errorf("generate embedding: %w", err)
		}
		doc.Embedding = embedding
	}

	c.documents[doc.ID] = doc
	c.dirty = true

	if err := c.saveToDisk(); err != nil {
		return fmt.Errorf("save to disk: %w", err)
	}

	return nil
}

// Get retrieves a document by ID.
func (c *ChromaDBStore) Get(ctx context.Context, id string) (*Document, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Check if store is closed
	if c.closed {
		return nil, errors.New("store is closed")
	}

	doc, exists := c.documents[id]
	if !exists {
		return nil, fmt.Errorf("document not found: %s", id)
	}

	return doc, nil
}

// matchesFilter checks if a document matches the filter criteria.
func matchesFilter(doc *Document, filter map[string]interface{}) bool {
	if len(filter) == 0 {
		return true
	}

	// Filter by document type
	if typeFilter, exists := filter["type"]; exists {
		if !matchTypeFilter(doc.Type, typeFilter) {
			return false
		}
	}

	// Filter by time range (optimized pushdown)
	if timeFrom, exists := filter["time_from"]; exists {
		if t, ok := timeFrom.(time.Time); ok {
			if doc.CreatedAt.Before(t) {
				return false
			}
		}
	}
	if timeTo, exists := filter["time_to"]; exists {
		if t, ok := timeTo.(time.Time); ok {
			if doc.CreatedAt.After(t) {
				return false
			}
		}
	}

	// Filter by metadata
	for key, value := range filter {
		if key == "type" || key == "time_from" || key == "time_to" || key == "min_score" {
			continue // Already handled
		}
		if docValue, ok := doc.Metadata[key]; ok {
			if fmt.Sprintf("%v", docValue) != fmt.Sprintf("%v", value) {
				return false
			}
		} else {
			return false
		}
	}

	return true
}

// matchTypeFilter checks if the document type matches the type filter.
// Supports both string and []string filter values.
func matchTypeFilter(docType DocumentType, typeFilter interface{}) bool {
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

// cosineSimilarity calculates the cosine similarity between two vectors.
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct float32
	var normA float32
	var normB float32

	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}
