package memory

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"
)

func TestNewLocalHashEmbedding(t *testing.T) {
	model, err := NewLocalHashEmbedding("openai", "text-embedding-3-small")
	if err != nil {
		t.Fatalf("NewLocalHashEmbedding failed: %v", err)
	}

	if model == nil {
		t.Fatal("expected non-nil model")
	}

	if model.model != "text-embedding-3-small" {
		t.Errorf("model = %q, want %q", model.model, "text-embedding-3-small")
	}
}

func TestNewLocalHashEmbedding_EmptyModel(t *testing.T) {
	model, err := NewLocalHashEmbedding("", "")
	if err != nil {
		t.Fatalf("NewLocalHashEmbedding failed: %v", err)
	}

	if model.model != "default" {
		t.Errorf("empty model should default to 'default', got %q", model.model)
	}
}

func TestLocalHashEmbedding_Dimension(t *testing.T) {
	model, _ := NewLocalHashEmbedding("test", "model")

	if model.Dimension() != defaultEmbeddingDim {
		t.Errorf("Dimension() = %d, want %d", model.Dimension(), defaultEmbeddingDim)
	}
}

func TestLocalHashEmbedding_Embed(t *testing.T) {
	ctx := context.Background()
	model, _ := NewLocalHashEmbedding("test", "model")

	tests := []struct {
		name string
		text string
	}{
		{"simple text", "hello world"},
		{"empty string", ""},
		{"long text", string(make([]byte, 10000))},
		{"special characters", "hello\nworld\t!@#$%^&*()"},
		{"unicode", "Hello 世界 🌍"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			embedding, err := model.Embed(ctx, tt.text)
			if err != nil {
				t.Fatalf("Embed() failed: %v", err)
			}

			if len(embedding) != defaultEmbeddingDim {
				t.Errorf("Embed() length = %d, want %d", len(embedding), defaultEmbeddingDim)
			}

			// Verify embedding is normalized
			var norm float32
			for _, v := range embedding {
				norm += v * v
			}
			norm = float32(math.Sqrt(float64(norm)))

			if norm < 0.99 || norm > 1.01 {
				t.Errorf("embedding not normalized, norm = %f", norm)
			}
		})
	}
}

func TestLocalHashEmbedding_Embed_Truncation(t *testing.T) {
	ctx := context.Background()
	model, _ := NewLocalHashEmbedding("test", "model")

	// Create text longer than maxTextLength
	longText := string(make([]byte, maxTextLength+1000))
	for i := range longText {
		longText = longText[:i] + "a" + longText[i+1:]
	}

	embedding, err := model.Embed(ctx, longText)
	if err != nil {
		t.Fatalf("Embed() failed: %v", err)
	}

	// Should succeed with truncated text
	if len(embedding) != defaultEmbeddingDim {
		t.Errorf("Embed() length = %d, want %d", len(embedding), defaultEmbeddingDim)
	}
}

func TestLocalHashEmbedding_Embed_Deterministic(t *testing.T) {
	ctx := context.Background()
	model, _ := NewLocalHashEmbedding("test", "model")

	text := "test document"

	emb1, _ := model.Embed(ctx, text)
	emb2, _ := model.Embed(ctx, text)

	// Same input should produce same output
	for i := range emb1 {
		if emb1[i] != emb2[i] {
			t.Errorf("embedding not deterministic at index %d: %v != %v", i, emb1[i], emb2[i])
		}
	}
}

func TestLocalHashEmbedding_Embed_DifferentInputs(t *testing.T) {
	ctx := context.Background()
	model, _ := NewLocalHashEmbedding("test", "model")

	emb1, _ := model.Embed(ctx, "hello")
	emb2, _ := model.Embed(ctx, "world")

	// Different inputs should produce different embeddings
	different := false
	for i := range emb1 {
		if emb1[i] != emb2[i] {
			different = true

			break
		}
	}

	if !different {
		t.Error("different inputs should produce different embeddings")
	}
}

func TestLocalHashEmbedding_Embed_CaseSensitivity(t *testing.T) {
	ctx := context.Background()
	model, _ := NewLocalHashEmbedding("test", "model")

	// hashEmbedding lowercases input, so case shouldn't matter for the hash part
	// But position mixing may cause differences
	emb1, _ := model.Embed(ctx, "HELLO")
	emb2, _ := model.Embed(ctx, "hello")

	// Due to lowercasing in hashEmbedding, these should be identical
	for i := range emb1 {
		if emb1[i] != emb2[i] {
			t.Errorf("case should not affect embedding at index %d: %v != %v", i, emb1[i], emb2[i])
		}
	}
}

func TestLocalHashEmbedding_EmbedBatch(t *testing.T) {
	ctx := context.Background()
	model, _ := NewLocalHashEmbedding("test", "model")

	texts := []string{"doc1", "doc2", "doc3"}

	embeddings, err := model.EmbedBatch(ctx, texts)
	if err != nil {
		t.Fatalf("EmbedBatch() failed: %v", err)
	}

	if len(embeddings) != len(texts) {
		t.Errorf("EmbedBatch() returned %d embeddings, want %d", len(embeddings), len(texts))
	}

	for i, emb := range embeddings {
		if len(emb) != defaultEmbeddingDim {
			t.Errorf("embedding %d has length %d, want %d", i, len(emb), defaultEmbeddingDim)
		}
	}
}

func TestLocalHashEmbedding_EmptyBatch(t *testing.T) {
	ctx := context.Background()
	model, _ := NewLocalHashEmbedding("test", "model")

	embeddings, err := model.EmbedBatch(ctx, []string{})
	if err != nil {
		t.Fatalf("EmbedBatch() failed: %v", err)
	}

	if len(embeddings) != 0 {
		t.Errorf("EmbedBatch() with empty slice returned %d embeddings, want 0", len(embeddings))
	}
}

func TestNewSimpleEmbedding(t *testing.T) {
	model := NewSimpleEmbedding(512)

	if model.dimension != 512 {
		t.Errorf("dimension = %d, want 512", model.dimension)
	}
}

func TestNewSimpleEmbedding_ZeroDimension(t *testing.T) {
	model := NewSimpleEmbedding(0)

	if model.dimension != defaultEmbeddingDim {
		t.Errorf("zero dimension should default to %d, got %d", defaultEmbeddingDim, model.dimension)
	}
}

func TestSimpleEmbedding_Dimension(t *testing.T) {
	model := NewSimpleEmbedding(256)

	if model.Dimension() != 256 {
		t.Errorf("Dimension() = %d, want 256", model.Dimension())
	}
}

func TestSimpleEmbedding_Embed(t *testing.T) {
	ctx := context.Background()
	model := NewSimpleEmbedding(256)

	text := "test document"

	embedding, err := model.Embed(ctx, text)
	if err != nil {
		t.Fatalf("Embed() failed: %v", err)
	}

	if len(embedding) != 256 {
		t.Errorf("Embed() length = %d, want 256", len(embedding))
	}

	// Verify normalization
	var norm float32
	for _, v := range embedding {
		norm += v * v
	}
	norm = float32(math.Sqrt(float64(norm)))

	if norm < 0.99 || norm > 1.01 {
		t.Errorf("embedding not normalized, norm = %f", norm)
	}
}

func TestSimpleEmbedding_EmbedBatch(t *testing.T) {
	ctx := context.Background()
	model := NewSimpleEmbedding(128)

	texts := []string{"a", "b", "c"}

	embeddings, err := model.EmbedBatch(ctx, texts)
	if err != nil {
		t.Fatalf("EmbedBatch() failed: %v", err)
	}

	if len(embeddings) != 3 {
		t.Errorf("EmbedBatch() returned %d embeddings, want 3", len(embeddings))
	}

	for i, emb := range embeddings {
		if len(emb) != 128 {
			t.Errorf("embedding %d has length %d, want 128", i, len(emb))
		}
	}
}

func TestGenerateHashEmbedding(t *testing.T) {
	hashBytes := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}

	embedding := generateHashEmbedding(hashBytes, 10)

	if len(embedding) != 10 {
		t.Errorf("embedding length = %d, want 10", len(embedding))
	}

	// Verify all values are in valid range
	for _, v := range embedding {
		if v < -1 || v > 1 {
			t.Errorf("embedding value %f out of range [-1, 1]", v)
		}
	}

	// Verify normalization
	var norm float32
	for _, v := range embedding {
		norm += v * v
	}
	norm = float32(math.Sqrt(float64(norm)))

	if norm < 0.99 || norm > 1.01 {
		t.Errorf("embedding not normalized, norm = %f", norm)
	}
}

func TestGenerateHashEmbedding_DifferentDimensions(t *testing.T) {
	hashBytes := []byte{0x01, 0x02, 0x03, 0x04}

	dimensions := []int{10, 50, 100, 1536}

	for _, dim := range dimensions {
		t.Run(fmt.Sprintf("dim_%d", dim), func(t *testing.T) {
			embedding := generateHashEmbedding(hashBytes, dim)

			if len(embedding) != dim {
				t.Errorf("embedding length = %d, want %d", len(embedding), dim)
			}
		})
	}
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name string
		a    []float32
		b    []float32
		want float32
	}{
		{
			name: "identical vectors",
			a:    []float32{1, 0, 0},
			b:    []float32{1, 0, 0},
			want: 1.0,
		},
		{
			name: "orthogonal vectors",
			a:    []float32{1, 0, 0},
			b:    []float32{0, 1, 0},
			want: 0.0,
		},
		{
			name: "opposite vectors",
			a:    []float32{1, 0, 0},
			b:    []float32{-1, 0, 0},
			want: -1.0,
		},
		{
			name: "45 degree angle",
			a:    []float32{1, 1, 0},
			b:    []float32{1, 0, 0},
			want: float32(1 / math.Sqrt(2)),
		},
		{
			name: "different lengths",
			a:    []float32{1, 0},
			b:    []float32{1, 0, 0},
			want: 0,
		},
		{
			name: "zero vector a",
			a:    []float32{0, 0, 0},
			b:    []float32{1, 0, 0},
			want: 0,
		},
		{
			name: "zero vector b",
			a:    []float32{1, 0, 0},
			b:    []float32{0, 0, 0},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cosineSimilarity(tt.a, tt.b)

			if math.Abs(float64(got-tt.want)) > 0.001 {
				t.Errorf("cosineSimilarity() = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestEmbedBatch_Error(t *testing.T) {
	ctx := context.Background()
	_, _ = NewLocalHashEmbedding("test", "model")

	// embedBatch with error function
	errorFunc := func(context.Context, string) ([]float32, error) {
		return nil, errors.New("test error")
	}

	_, err := embedBatch(ctx, []string{"test"}, errorFunc)

	if err == nil {
		t.Error("embedBatch with error function should return error")
	}

	if !errors.Is(err, errors.New("test error")) {
		// Check the error message contains our expected text
		if !strings.Contains(err.Error(), "test error") {
			t.Errorf("error = %v, should contain 'test error'", err)
		}
	}
}
