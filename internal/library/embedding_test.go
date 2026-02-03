package library

import (
	"context"
	"math"
	"strings"
	"testing"
)

// mockEmbedding is a simple mock for testing.
type mockEmbedding struct {
	dimension int
}

func (m *mockEmbedding) Dimension() int {
	return m.dimension
}

func (m *mockEmbedding) Embed(ctx context.Context, text string) ([]float32, error) {
	// Simple deterministic embedding based on text length
	result := make([]float32, m.dimension)
	for i := range result {
		// Use text length to create different vectors
		val := float32(len(text)%100) * 0.01
		result[i] = val + float32(i)*0.001
	}

	return result, nil
}

func TestNewEmbeddingScorer(t *testing.T) {
	tests := []struct {
		name  string
		model EmbeddingModel
		want  *EmbeddingScorer
	}{
		{
			name:  "nil model creates scorer",
			model: nil,
			want:  &EmbeddingScorer{},
		},
		{
			name:  "with model",
			model: &mockEmbedding{dimension: 10},
			want:  &EmbeddingScorer{embedding: &mockEmbedding{dimension: 10}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewEmbeddingScorer(tt.model)
			if tt.model == nil {
				if got.embedding != nil {
					t.Errorf("NewEmbeddingScorer() with nil model should have nil embedding")
				}
			} else if got.embedding == nil && tt.model != nil {
				t.Errorf("NewEmbeddingScorer() should preserve model")
			}
		})
	}
}

func TestScoreForQuery(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		model    EmbeddingModel
		title    string
		content  string
		query    string
		minScore float64
		maxScore float64
	}{
		{
			name:     "exact match in title",
			model:    &mockEmbedding{dimension: 10},
			title:    "Go Programming Language",
			content:  "Some content about Go",
			query:    "Go Programming",
			minScore: 0.0,
			maxScore: 1.0,
		},
		{
			name:     "no embedding - keyword fallback",
			model:    nil,
			title:    "TypeScript Handbook",
			content:  "TypeScript is a typed superset of JavaScript",
			query:    "TypeScript",
			minScore: 0.3, // Should have some keyword match
			maxScore: 1.0,
		},
		{
			name:     "no match",
			model:    nil,
			title:    "Python Documentation",
			content:  "Python is a programming language",
			query:    "golang rust java",
			minScore: 0.0,
			maxScore: 0.2, // Low score for no match
		},
		{
			name:     "empty query",
			model:    nil,
			title:    "Some Title",
			content:  "Some content",
			query:    "",
			minScore: 0.5, // Default score
			maxScore: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scorer := NewEmbeddingScorer(tt.model)
			score := scorer.ScoreForQuery(ctx, tt.title, tt.content, tt.query)

			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("ScoreForQuery() = %v, want between %v and %v", score, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestScoreForPath(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		model    EmbeddingModel
		title    string
		content  string
		keywords []string
		minScore float64
		maxScore float64
	}{
		{
			name:     "matching keyword in title",
			model:    nil,
			title:    "React Component Guide",
			content:  "How to build components",
			keywords: []string{"react", "component"},
			minScore: 0.4,
			maxScore: 1.0,
		},
		{
			name:     "no keywords",
			model:    nil,
			title:    "Some Title",
			content:  "Some content",
			keywords: []string{},
			minScore: 0.5,
			maxScore: 0.5,
		},
		{
			name:     "keyword in content",
			model:    nil,
			title:    "API Reference",
			content:  "This documentation covers the vscode extension API",
			keywords: []string{"vscode", "api"},
			minScore: 0.1,
			maxScore: 1.0,
		},
		{
			name:     "with embedding model",
			model:    &mockEmbedding{dimension: 10},
			title:    "Go Documentation",
			content:  "Go is a programming language",
			keywords: []string{"go", "language"},
			minScore: 0.0,
			maxScore: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scorer := NewEmbeddingScorer(tt.model)
			score := scorer.ScoreForPath(ctx, tt.title, tt.content, tt.keywords)

			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("ScoreForPath() = %v, want between %v and %v", score, tt.minScore, tt.maxScore)
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
			a:    []float32{1, 2, 3},
			b:    []float32{1, 2, 3},
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
			a:    []float32{1, 1, 1},
			b:    []float32{-1, -1, -1},
			want: -1.0,
		},
		{
			name: "different lengths - returns 0",
			a:    []float32{1, 2},
			b:    []float32{1, 2, 3},
			want: 0.0,
		},
		{
			name: "zero vector a",
			a:    []float32{0, 0, 0},
			b:    []float32{1, 2, 3},
			want: 0.0,
		},
		{
			name: "zero vector b",
			a:    []float32{1, 2, 3},
			b:    []float32{0, 0, 0},
			want: 0.0,
		},
		{
			name: "similar vectors",
			a:    []float32{1, 2, 3},
			b:    []float32{1.1, 2.1, 3.1},
			want: 0.999, // Very close to 1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cosineSimilarity(tt.a, tt.b)
			if tt.want != 0 && math.Abs(float64(got-tt.want)) > 0.01 {
				t.Errorf("cosineSimilarity() = %v, want %v", got, tt.want)
			} else if tt.want == 0 && got != 0 {
				t.Errorf("cosineSimilarity() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScorePages(t *testing.T) {
	ctx := context.Background()

	pages := []*PageContent{
		{
			Title:   "Go Tutorial",
			Content: "Learn Go programming",
			Score:   0,
		},
		{
			Title:   "Python Guide",
			Content: "Learn Python programming",
			Score:   0,
		},
		{
			Title:   "JavaScript Basics",
			Content: "Learn JavaScript",
			Score:   0,
		},
	}

	scorer := NewEmbeddingScorer(nil)
	sorted := scorer.ScorePages(ctx, pages, "Go programming")

	// First result should be Go-related (highest score)
	if !strings.Contains(sorted[0].Title, "Go") && sorted[0].Score > 0 {
		t.Errorf("ScorePages() first result = %v, want Go-related page", sorted[0].Title)
	}
}

func TestHasEmbedding(t *testing.T) {
	tests := []struct {
		name  string
		model EmbeddingModel
		want  bool
	}{
		{
			name:  "nil model",
			model: nil,
			want:  false,
		},
		{
			name:  "with model",
			model: &mockEmbedding{dimension: 10},
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scorer := NewEmbeddingScorer(tt.model)
			if got := scorer.HasEmbedding(); got != tt.want {
				t.Errorf("HasEmbedding() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKeywordScore(t *testing.T) {
	scorer := NewEmbeddingScorer(nil)

	tests := []struct {
		name     string
		title    string
		content  string
		query    string
		minScore float64
	}{
		{
			name:     "title match",
			title:    "React Hooks Guide",
			content:  "Content about hooks",
			query:    "react hooks",
			minScore: 0.7,
		},
		{
			name:     "content match only",
			title:    "Some Title",
			content:  "This talks about TypeScript types",
			query:    "typescript",
			minScore: 0.3,
		},
		{
			name:     "partial match",
			title:    "API Documentation",
			content:  "Reference for the extension API",
			query:    "extension api reference",
			minScore: 0.3,
		},
		{
			name:     "no match",
			title:    "Go Programming",
			content:  "Go language guide",
			query:    "python rust",
			minScore: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := scorer.keywordScore(tt.title, tt.content, tt.query)
			if score < tt.minScore {
				t.Errorf("keywordScore() = %v, want >= %v", score, tt.minScore)
			}
		})
	}
}

func TestPrepareText(t *testing.T) {
	scorer := NewEmbeddingScorer(nil)

	tests := []struct {
		name     string
		text     string
		maxLen   int
		wantTrim bool
	}{
		{
			name:     "short text unchanged",
			text:     "Hello world",
			maxLen:   4000,
			wantTrim: false,
		},
		{
			name:     "whitespace normalized",
			text:     "Hello    world\n\n\ttest",
			maxLen:   4000,
			wantTrim: false,
		},
		{
			name:     "long text truncated",
			text:     string(make([]byte, 5000)),
			maxLen:   4000,
			wantTrim: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scorer.prepareText(tt.text)
			// Check whitespace normalization for short text
			if !tt.wantTrim {
				// Should not have multiple consecutive spaces
				for i := range len(got) - 1 {
					if got[i] == ' ' && got[i+1] == ' ' {
						t.Error("prepareText() should normalize whitespace")

						break
					}
				}
			}
			// Check max length
			if len(got) > tt.maxLen {
				t.Errorf("prepareText() length = %v, want <= %v", len(got), tt.maxLen)
			}
		})
	}
}
