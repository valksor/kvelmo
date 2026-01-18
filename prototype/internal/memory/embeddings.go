package memory

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math"
	"strings"
)

const (
	// defaultEmbeddingDim is the default embedding dimension.
	defaultEmbeddingDim = 1536
	// maxTextLength is the maximum character length for text embeddings.
	maxTextLength = 8191
)

// LocalHashEmbedding implements a deterministic hash-based embedding for local-only operation.
// This generates embeddings using SHA256 hashing without external API calls.
type LocalHashEmbedding struct {
	model string
}

// NewLocalHashEmbedding creates a new local hash-based embedding model.
func NewLocalHashEmbedding(_, model string) (*LocalHashEmbedding, error) {
	if model == "" {
		model = "default"
	}

	return &LocalHashEmbedding{
		model: model,
	}, nil
}

// Dimension returns the embedding dimension for the model.
func (l *LocalHashEmbedding) Dimension() int {
	return defaultEmbeddingDim
}

// Embed generates an embedding for a single text using local hash-based approach.
func (l *LocalHashEmbedding) Embed(ctx context.Context, text string) ([]float32, error) {
	// Truncate text if too long
	if len(text) > maxTextLength {
		text = text[:maxTextLength]
	}

	return l.hashEmbedding(text), nil
}

// EmbedBatch generates embeddings for multiple texts.
func (l *LocalHashEmbedding) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	return embedBatch(ctx, texts, l.Embed)
}

// hashEmbedding generates a deterministic embedding from text using hashing.
// This is a local-only implementation using SHA256 - no external APIs are called.
func (l *LocalHashEmbedding) hashEmbedding(text string) []float32 {
	// Preprocess text
	text = strings.ToLower(text)
	text = strings.TrimSpace(text)

	// Create hash
	hash := sha256.Sum256([]byte(text))

	// Convert hash to embedding vector
	return generateHashEmbedding(hash[:], l.Dimension())
}

// generateHashEmbedding creates a normalized embedding vector from hash bytes.
func generateHashEmbedding(hashBytes []byte, dimension int) []float32 {
	embedding := make([]float32, dimension)

	// Expand hash to dimension
	for i := range dimension {
		// Use different bytes of the hash
		idx := (i * 4) % len(hashBytes)
		val := float32(hashBytes[idx])

		// Mix in position to create variation
		val += float32(i) * 0.01

		// Normalize to roughly [-1, 1] range
		embedding[i] = (val / 128.0) - 1.0
	}

	// Normalize the vector
	norm := float32(0)
	for _, v := range embedding {
		norm += v * v
	}
	norm = float32(math.Sqrt(float64(norm)))

	for i := range embedding {
		embedding[i] /= norm
	}

	return embedding
}

// SimpleEmbedding provides a simple hash-based embedding without external dependencies.
type SimpleEmbedding struct {
	dimension int
}

// NewSimpleEmbedding creates a new simple embedding model.
func NewSimpleEmbedding(dimension int) *SimpleEmbedding {
	if dimension == 0 {
		dimension = defaultEmbeddingDim
	}

	return &SimpleEmbedding{
		dimension: dimension,
	}
}

// Dimension returns the embedding dimension.
func (s *SimpleEmbedding) Dimension() int {
	return s.dimension
}

// Embed generates a simple hash-based embedding.
func (s *SimpleEmbedding) Embed(ctx context.Context, text string) ([]float32, error) {
	// Preprocess text
	text = strings.ToLower(text)
	text = strings.TrimSpace(text)

	// Create hash
	hash := sha256.Sum256([]byte(text))

	// Convert hash to embedding vector
	return generateHashEmbedding(hash[:], s.dimension), nil
}

// EmbedBatch generates embeddings for multiple texts.
func (s *SimpleEmbedding) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	return embedBatch(ctx, texts, s.Embed)
}

// embedBatch is a helper for batch embedding generation.
func embedBatch(ctx context.Context, texts []string, embedFunc func(context.Context, string) ([]float32, error)) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		embedding, err := embedFunc(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("embed text %d: %w", i, err)
		}
		embeddings[i] = embedding
	}

	return embeddings, nil
}
