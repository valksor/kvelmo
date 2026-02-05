//go:build cgo

package memory

import (
	"context"
	"testing"

	"github.com/valksor/go-mehrhof/internal/storage"
)

func TestNewONNXEmbedding(t *testing.T) {
	opts := ONNXEmbeddingOptions{
		ModelName: "all-MiniLM-L6-v2",
		MaxLength: 128,
	}

	e, err := NewONNXEmbedding(opts)
	if err != nil {
		t.Fatalf("NewONNXEmbedding: %v", err)
	}

	if e.dimension != 384 {
		t.Errorf("dimension: got %d, want 384", e.dimension)
	}

	if e.maxLength != 128 {
		t.Errorf("maxLength: got %d, want 128", e.maxLength)
	}
}

func TestNewONNXEmbedding_DefaultOptions(t *testing.T) {
	opts := ONNXEmbeddingOptions{}

	e, err := NewONNXEmbedding(opts)
	if err != nil {
		t.Fatalf("NewONNXEmbedding: %v", err)
	}

	if e.dimension != 384 { // all-MiniLM-L6-v2 default
		t.Errorf("dimension: got %d, want 384", e.dimension)
	}

	if e.maxLength != 256 { // Default max length
		t.Errorf("maxLength: got %d, want 256", e.maxLength)
	}
}

func TestNewONNXEmbedding_UnknownModel(t *testing.T) {
	opts := ONNXEmbeddingOptions{
		ModelName: "nonexistent-model",
	}

	_, err := NewONNXEmbedding(opts)
	if err == nil {
		t.Error("expected error for unknown model")
	}
}

func TestONNXEmbedding_Dimension(t *testing.T) {
	opts := ONNXEmbeddingOptions{
		ModelName: "all-MiniLM-L6-v2",
	}

	e, _ := NewONNXEmbedding(opts)

	if e.Dimension() != 384 {
		t.Errorf("Dimension(): got %d, want 384", e.Dimension())
	}
}

func TestNewONNXEmbeddingFromConfig(t *testing.T) {
	cfg := storage.VectorDBSettings{
		ONNX: storage.ONNXSettings{
			Model:     "all-MiniLM-L6-v2",
			MaxLength: 128,
		},
	}

	e, err := NewONNXEmbeddingFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewONNXEmbeddingFromConfig: %v", err)
	}

	if e.Dimension() != 384 {
		t.Errorf("Dimension: got %d, want 384", e.Dimension())
	}
}

func TestMeanPooling(t *testing.T) {
	// Test with simple 2x3 output (2 tokens, 3 dims)
	output := []float32{
		1.0, 2.0, 3.0, // Token 0
		4.0, 5.0, 6.0, // Token 1
	}
	attentionMask := []int64{1, 1} // Both tokens are real

	result := meanPooling(output, attentionMask, 2, 3)

	// Expected: (1+4)/2, (2+5)/2, (3+6)/2 = 2.5, 3.5, 4.5
	expected := []float32{2.5, 3.5, 4.5}
	for i, v := range result {
		if v != expected[i] {
			t.Errorf("meanPooling[%d]: got %f, want %f", i, v, expected[i])
		}
	}
}

func TestMeanPooling_WithPadding(t *testing.T) {
	// Test with padding (only first token is real)
	output := []float32{
		1.0, 2.0, 3.0, // Token 0 (real)
		4.0, 5.0, 6.0, // Token 1 (padding)
	}
	attentionMask := []int64{1, 0} // Only first token is real

	result := meanPooling(output, attentionMask, 2, 3)

	// Expected: only token 0's values
	expected := []float32{1.0, 2.0, 3.0}
	for i, v := range result {
		if v != expected[i] {
			t.Errorf("meanPooling with padding[%d]: got %f, want %f", i, v, expected[i])
		}
	}
}

func TestL2Normalize(t *testing.T) {
	// Test with [3, 4] which should normalize to [0.6, 0.8]
	vec := []float32{3.0, 4.0}
	result := l2Normalize(vec)

	// Norm = 5, so [3/5, 4/5] = [0.6, 0.8]
	if result[0] != 0.6 || result[1] != 0.8 {
		t.Errorf("l2Normalize: got [%f, %f], want [0.6, 0.8]", result[0], result[1])
	}
}

func TestL2Normalize_ZeroVector(t *testing.T) {
	// Zero vector should remain zero
	vec := []float32{0.0, 0.0, 0.0}
	result := l2Normalize(vec)

	for i, v := range result {
		if v != 0 {
			t.Errorf("l2Normalize zero[%d]: got %f, want 0", i, v)
		}
	}
}

func TestL2Normalize_UnitVector(t *testing.T) {
	// Already normalized vector should stay the same
	vec := []float32{1.0, 0.0, 0.0}
	result := l2Normalize(vec)

	if result[0] != 1.0 || result[1] != 0.0 || result[2] != 0.0 {
		t.Errorf("l2Normalize unit: got %v", result)
	}
}

// Integration test - only runs if ONNX Runtime is available.
func TestONNXEmbedding_Integration(t *testing.T) {
	if !IsONNXAvailable() {
		t.Skip("ONNX Runtime not available, skipping integration test")
	}

	opts := ONNXEmbeddingOptions{
		ModelName: "all-MiniLM-L6-v2",
		MaxLength: 64,
	}

	e, err := NewONNXEmbedding(opts)
	if err != nil {
		t.Fatalf("NewONNXEmbedding: %v", err)
	}
	defer func() { _ = e.Close() }()

	ctx := context.Background()

	// Test single embed
	embedding, err := e.Embed(ctx, "hello world")
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}

	if len(embedding) != 384 {
		t.Errorf("embedding length: got %d, want 384", len(embedding))
	}

	// Test batch embed
	texts := []string{"hello", "world", "test"}
	embeddings, err := e.EmbedBatch(ctx, texts)
	if err != nil {
		t.Fatalf("EmbedBatch: %v", err)
	}

	if len(embeddings) != 3 {
		t.Errorf("batch count: got %d, want 3", len(embeddings))
	}

	// Verify all embeddings have correct dimension
	for i, emb := range embeddings {
		if len(emb) != 384 {
			t.Errorf("embeddings[%d] length: got %d, want 384", i, len(emb))
		}
	}
}

// Test semantic similarity (only with ONNX Runtime).
func TestONNXEmbedding_SemanticSimilarity(t *testing.T) {
	if !IsONNXAvailable() {
		t.Skip("ONNX Runtime not available, skipping semantic similarity test")
	}

	opts := ONNXEmbeddingOptions{
		ModelName: "all-MiniLM-L6-v2",
		MaxLength: 64,
	}

	e, err := NewONNXEmbedding(opts)
	if err != nil {
		t.Fatalf("NewONNXEmbedding: %v", err)
	}
	defer func() { _ = e.Close() }()

	ctx := context.Background()

	// Similar sentences should have high similarity
	e1, _ := e.Embed(ctx, "The cat sat on the mat")
	e2, _ := e.Embed(ctx, "A cat is sitting on a mat")
	e3, _ := e.Embed(ctx, "Python is a programming language")

	sim12 := testCosineSimilarity(e1, e2)
	sim13 := testCosineSimilarity(e1, e3)

	// Similar sentences should have higher similarity
	if sim12 <= sim13 {
		t.Errorf("expected similar sentences to have higher similarity: sim(cat/cat)=%f, sim(cat/python)=%f",
			sim12, sim13)
	}

	// Similar sentences should have similarity > 0.5
	if sim12 < 0.5 {
		t.Errorf("expected similarity > 0.5 for similar sentences, got %f", sim12)
	}
}

// testCosineSimilarity calculates cosine similarity between two vectors (test helper).
func testCosineSimilarity(a, b []float32) float32 {
	// Use the package's cosineSimilarity function
	return cosineSimilarity(a, b)
}
