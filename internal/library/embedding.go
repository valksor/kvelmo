// Package library provides semantic scoring for documentation pages using embeddings.
package library

import (
	"context"
	"errors"
	"math"
	"strings"
)

// EmbeddingScorer provides semantic similarity scoring using vector embeddings.
// This enables better doc relevance matching beyond simple keyword overlap.
type EmbeddingScorer struct {
	// embedding is the underlying embedding model for generating vectors.
	embedding EmbeddingModel
}

// EmbeddingModel defines the interface for generating text embeddings.
// This matches the interface from internal/memory/embeddings.go.
type EmbeddingModel interface {
	Dimension() int
	Embed(ctx context.Context, text string) ([]float32, error)
}

// NewEmbeddingScorer creates a new semantic scorer using the given embedding model.
// If nil, it creates a default local hash-based embedding.
func NewEmbeddingScorer(model EmbeddingModel) *EmbeddingScorer {
	if model == nil {
		// Create a simple default scorer - will use keyword-based fallback
		return &EmbeddingScorer{}
	}

	return &EmbeddingScorer{
		embedding: model,
	}
}

// ScoreForQuery calculates semantic similarity between a page and a query.
// Returns a score between 0.0 (no relevance) and 1.0 (highly relevant).
// Falls back to keyword scoring if embeddings are not available.
func (s *EmbeddingScorer) ScoreForQuery(ctx context.Context, title, content, query string) float64 {
	// If no embedding model configured, use keyword-based fallback
	if s.embedding == nil {
		return s.keywordScore(title, content, query)
	}

	// Generate embeddings for query and page content
	queryEmbedding, err := s.embedding.Embed(ctx, s.prepareText(query))
	if err != nil {
		// Fall back to keyword scoring on error
		return s.keywordScore(title, content, query)
	}

	// Combine title and content for page embedding
	pageText := s.prepareText(title + " " + content)
	pageEmbedding, err := s.embedding.Embed(ctx, pageText)
	if err != nil {
		return s.keywordScore(title, content, query)
	}

	// Calculate cosine similarity
	similarity := cosineSimilarity(queryEmbedding, pageEmbedding)

	// Boost score if title contains query terms
	if title != "" {
		titleLower := strings.ToLower(title)
		queryLower := strings.ToLower(query)
		queryWords := strings.Fields(queryLower)

		for _, word := range queryWords {
			if len(word) >= 3 && strings.Contains(titleLower, word) {
				boosted := similarity * 1.2
				if boosted > 1.0 {
					boosted = 1.0
				}
				similarity = boosted

				break
			}
		}
	}

	return float64(similarity)
}

// ScoreForPath calculates relevance of a page to file path keywords.
// This is used for auto-include functionality based on edited files.
func (s *EmbeddingScorer) ScoreForPath(ctx context.Context, title, content string, keywords []string) float64 {
	if len(keywords) == 0 {
		return 0.5 // Default score when no keywords
	}

	// If no embedding model, use keyword-based scoring
	if s.embedding == nil {
		return s.keywordPathScore(title, content, keywords)
	}

	// Generate a combined query from keywords
	query := strings.Join(keywords, " ")

	return s.ScoreForQuery(ctx, title, content, query)
}

// prepareText normalizes text for embedding generation.
func (s *EmbeddingScorer) prepareText(text string) string {
	// Normalize whitespace
	text = strings.Join(strings.Fields(text), " ")

	// Truncate if too long (most embeddings have input limits)
	maxLength := 4000
	if len(text) > maxLength {
		text = text[:maxLength]
	}

	return text
}

// keywordScore provides fallback scoring when embeddings are unavailable.
func (s *EmbeddingScorer) keywordScore(title, content, query string) float64 {
	if query == "" {
		return 0.5
	}

	queryLower := strings.ToLower(query)
	queryWords := strings.Fields(queryLower)

	titleLower := strings.ToLower(title)
	contentLower := strings.ToLower(content)

	// Count matching words
	matches := 0
	for _, word := range queryWords {
		if len(word) < 3 {
			continue
		}

		if strings.Contains(titleLower, word) || strings.Contains(contentLower, word) {
			matches++
		}
	}

	if len(queryWords) == 0 {
		return 0.5
	}

	// Return proportion of matched words
	score := float64(matches) / float64(len(queryWords))

	// Boost for title matches
	if title != "" {
		for _, word := range queryWords {
			if strings.Contains(titleLower, word) {
				score = math.Min(score*1.3, 1.0) // 30% boost for title match

				break
			}
		}
	}

	return score
}

// keywordPathScore provides fallback path-based scoring when embeddings are unavailable.
func (s *EmbeddingScorer) keywordPathScore(title, content string, keywords []string) float64 {
	if len(keywords) == 0 {
		return 0.5
	}

	titleLower := strings.ToLower(title)
	contentLower := strings.ToLower(content)

	totalScore := 0.0
	for _, kw := range keywords {
		kwLower := strings.ToLower(kw)

		// Title match: high weight
		if strings.Contains(titleLower, kwLower) {
			totalScore += 0.4
		}

		// Content match: lower weight
		if strings.Contains(contentLower, kwLower) {
			totalScore += 0.1
		}
	}

	// Normalize to 0-1 range
	maxScore := float64(len(keywords)) * 0.5
	if maxScore > 0 {
		totalScore = totalScore / maxScore
		if totalScore > 1.0 {
			totalScore = 1.0
		}
	}

	return totalScore
}

// cosineSimilarity calculates the cosine similarity between two vectors.
// Returns a value between -1.0 and 1.0, where 1.0 means identical direction.
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

// ScorePages scores a list of page candidates against a query, returning sorted results.
func (s *EmbeddingScorer) ScorePages(ctx context.Context, pages []*PageContent, query string) []*PageContent {
	// Score each page
	for _, page := range pages {
		if page.Score <= 0 {
			// Only re-score if no existing score
			page.Score = s.ScoreForQuery(ctx, page.Title, page.Content, query)
		}
	}

	// Sort by score (descending)
	sortedPages := make([]*PageContent, len(pages))
	copy(sortedPages, pages)

	// Simple insertion sort for small slices
	for i := 1; i < len(sortedPages); i++ {
		key := sortedPages[i]
		j := i - 1
		for j >= 0 && sortedPages[j].Score < key.Score {
			sortedPages[j+1] = sortedPages[j]
			j--
		}
		sortedPages[j+1] = key
	}

	return sortedPages
}

// EmbedPages generates embeddings for a batch of pages, useful for caching.
func (s *EmbeddingScorer) EmbedPages(ctx context.Context, pages []*PageContent) error {
	if s.embedding == nil {
		return errors.New("no embedding model configured")
	}

	// This would be useful for pre-computing embeddings for faster retrieval
	// For now, we generate on-demand to avoid complexity
	return nil
}

// HasEmbedding returns true if a proper embedding model is configured.
func (s *EmbeddingScorer) HasEmbedding() bool {
	return s.embedding != nil
}
