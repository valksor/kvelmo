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
)

// VectorStore is a file-backed, in-memory vector store that persists documents
// as individual JSON files under a configurable directory.  Search uses cosine
// similarity.
type VectorStore struct {
	mu        sync.RWMutex
	dir       string
	embedder  Embedder
	documents map[string]*Document
}

// NewVectorStore creates a VectorStore rooted at dir.  Existing JSON documents
// are loaded eagerly.  embedder is used when a document is stored without a
// pre-computed embedding.
func NewVectorStore(dir string, embedder Embedder) (*VectorStore, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create vector store dir: %w", err)
	}

	vs := &VectorStore{
		dir:       dir,
		embedder:  embedder,
		documents: make(map[string]*Document),
	}

	if err := vs.load(); err != nil {
		return nil, fmt.Errorf("load existing documents: %w", err)
	}

	return vs, nil
}

// Store saves a document to the store.  If the document has no embedding and
// non-empty content the store generates one via the configured Embedder.
func (vs *VectorStore) Store(ctx context.Context, doc *Document) error {
	if doc.ID == "" {
		return errors.New("document must have an ID")
	}

	// Generate embedding when absent.
	if len(doc.Embedding) == 0 && doc.Content != "" {
		emb, err := vs.embedder.Embed(ctx, doc.Content)
		if err != nil {
			return fmt.Errorf("generate embedding: %w", err)
		}
		doc.Embedding = emb
	}

	if doc.CreatedAt.IsZero() {
		doc.CreatedAt = time.Now()
	}

	vs.mu.Lock()
	vs.documents[doc.ID] = doc
	vs.mu.Unlock()

	return vs.persist(doc)
}

// Search performs cosine-similarity search against all stored embeddings.
func (vs *VectorStore) Search(ctx context.Context, query string, opts SearchOptions) ([]*SearchResult, error) {
	queryEmb, err := vs.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}

	vs.mu.RLock()
	defer vs.mu.RUnlock()

	type candidate struct {
		doc   *Document
		score float32
	}

	var candidates []candidate
	for _, doc := range vs.documents {
		if !matchesFilter(doc, opts) {
			continue
		}
		if len(doc.Embedding) == 0 {
			continue
		}
		score := cosineSimilarity(queryEmb, doc.Embedding)
		if opts.MinScore > 0 && score < opts.MinScore {
			continue
		}
		candidates = append(candidates, candidate{doc: doc, score: score})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	count := len(candidates)
	if opts.Limit > 0 && opts.Limit < count {
		count = opts.Limit
	}

	results := make([]*SearchResult, count)
	for i := range count {
		results[i] = &SearchResult{
			Document: candidates[i].doc,
			Score:    candidates[i].score,
		}
	}

	return results, nil
}

// Delete removes a document by ID from memory and disk.
func (vs *VectorStore) Delete(_ context.Context, id string) error {
	vs.mu.Lock()
	delete(vs.documents, id)
	vs.mu.Unlock()

	path := vs.filePath(id)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove document file: %w", err)
	}

	return nil
}

// Get retrieves a document by ID.
func (vs *VectorStore) Get(_ context.Context, id string) (*Document, error) {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	doc, ok := vs.documents[id]
	if !ok {
		return nil, fmt.Errorf("document not found: %s", id)
	}

	return doc, nil
}

// Clear removes all documents from memory and disk.
func (vs *VectorStore) Clear(_ context.Context) error {
	vs.mu.Lock()
	vs.documents = make(map[string]*Document)
	vs.mu.Unlock()

	entries, err := os.ReadDir(vs.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("read store dir: %w", err)
	}

	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		if err := os.Remove(filepath.Join(vs.dir, e.Name())); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove %s: %w", e.Name(), err)
		}
	}

	return nil
}

// Stats returns aggregate counts for the store, including the active embedder.
func (vs *VectorStore) Stats() Stats {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	byType := make(map[string]int)
	for _, doc := range vs.documents {
		byType[string(doc.Type)]++
	}

	return Stats{
		TotalDocuments: len(vs.documents),
		ByType:         byType,
		Embedder:       vs.embedder.Name(),
	}
}

// --- internal helpers ---

func (vs *VectorStore) filePath(id string) string {
	// Sanitise the ID so it is safe as a filename.
	safe := filepath.Base(id)
	if safe == "." || safe == "/" {
		safe = "_"
	}

	return filepath.Join(vs.dir, safe+".json")
}

func (vs *VectorStore) persist(doc *Document) error {
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal document: %w", err)
	}
	if err := os.WriteFile(vs.filePath(doc.ID), data, 0o644); err != nil {
		return fmt.Errorf("write document: %w", err)
	}

	return nil
}

func (vs *VectorStore) load() error {
	entries, err := os.ReadDir(vs.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("read store dir: %w", err)
	}

	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(vs.dir, e.Name()))
		if err != nil {
			return fmt.Errorf("read %s: %w", e.Name(), err)
		}
		var doc Document
		if err := json.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("unmarshal %s: %w", e.Name(), err)
		}
		vs.documents[doc.ID] = &doc
	}

	return nil
}

// matchesFilter returns true when doc passes all SearchOptions filters.
func matchesFilter(doc *Document, opts SearchOptions) bool {
	if len(opts.DocumentTypes) > 0 {
		matched := false
		for _, dt := range opts.DocumentTypes {
			if doc.Type == dt {
				matched = true

				break
			}
		}
		if !matched {
			return false
		}
	}

	if opts.TimeRange != nil {
		if !opts.TimeRange.From.IsZero() && doc.CreatedAt.Before(opts.TimeRange.From) {
			return false
		}
		if !opts.TimeRange.To.IsZero() && doc.CreatedAt.After(opts.TimeRange.To) {
			return false
		}
	}

	for k, v := range opts.MetadataFilters {
		docVal, ok := doc.Metadata[k]
		if !ok {
			return false
		}
		if fmt.Sprintf("%v", docVal) != fmt.Sprintf("%v", v) {
			return false
		}
	}

	return true
}

// cosineSimilarity computes the cosine similarity between two vectors.
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dot, normA, normB float32
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}

	return dot / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}
