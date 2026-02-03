package library

import (
	"context"
	"sort"
	"strings"
)

// GetDocsForPaths returns documentation relevant to the given file paths.
// This is the auto-include functionality that matches collections to working files.
func (m *Manager) GetDocsForPaths(ctx context.Context, filePaths []string, maxTokens int) (*DocContext, error) {
	if maxTokens <= 0 {
		maxTokens = m.config.MaxTokenBudget
	}

	collections, err := m.List(ctx, nil)
	if err != nil {
		return nil, err
	}

	// Extract keywords from file paths for relevance scoring
	keywords := ExtractKeywords(filePaths)

	var candidates []*PageContent

	for _, coll := range collections {
		// Determine which store has this collection
		isShared := coll.Location == string(LocationShared)
		store := m.GetStore(isShared)
		if store == nil {
			continue
		}

		// Skip collections in explicit mode unless explicitly requested
		if coll.IncludeMode == IncludeModeExplicit {
			continue
		}

		// For auto mode, check path patterns
		if coll.IncludeMode == IncludeModeAuto && len(coll.Paths) > 0 {
			if !MatchesAnyPath(coll.Paths, filePaths) {
				continue
			}
		}

		// Get pages from this collection
		pagePaths, err := store.ListPageFiles(coll.ID)
		if err != nil {
			continue
		}

		// Load collection metadata to get page titles
		meta, err := store.LoadCollectionMeta(coll.ID)
		if err != nil {
			continue
		}

		for _, pagePath := range pagePaths {
			content, err := store.ReadPage(coll.ID, pagePath)
			if err != nil {
				continue
			}

			// Find page metadata
			var pageTitle string
			for _, p := range meta.Pages {
				if p.Path == pagePath {
					pageTitle = p.Title

					break
				}
			}

			// Calculate relevance score based on keywords
			score := calculateRelevanceScoreFromContent(pageTitle, pagePath, content, keywords)

			candidates = append(candidates, &PageContent{
				CollectionID:   coll.ID,
				CollectionName: coll.Name,
				Path:           pagePath,
				Title:          pageTitle,
				Content:        content,
				TokenCount:     estimateTokens(content),
				Score:          score,
			})
		}
	}

	// Sort by score (descending)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	// Budget tokens and select pages
	return budgetPages(candidates, maxTokens, m.config.MaxPagesPerPrompt)
}

// GetExplicitDocs returns documentation for explicitly named collections.
func (m *Manager) GetExplicitDocs(ctx context.Context, names []string, maxTokens int) (*DocContext, error) {
	if maxTokens <= 0 {
		maxTokens = m.config.MaxTokenBudget
	}

	var candidates []*PageContent

	for _, name := range names {
		coll, err := m.Show(ctx, name)
		if err != nil {
			continue // Skip collections that don't exist
		}

		// Determine which store has this collection
		isShared := coll.Location == string(LocationShared)
		store := m.GetStore(isShared)
		if store == nil {
			continue
		}

		pagePaths, err := store.ListPageFiles(coll.ID)
		if err != nil {
			continue
		}

		// Load collection metadata to get page titles
		meta, err := store.LoadCollectionMeta(coll.ID)
		if err != nil {
			continue
		}

		for _, pagePath := range pagePaths {
			content, err := store.ReadPage(coll.ID, pagePath)
			if err != nil {
				continue
			}

			// Find page metadata
			var pageTitle string
			for _, p := range meta.Pages {
				if p.Path == pagePath {
					pageTitle = p.Title

					break
				}
			}

			candidates = append(candidates, &PageContent{
				CollectionID:   coll.ID,
				CollectionName: coll.Name,
				Path:           pagePath,
				Title:          pageTitle,
				Content:        content,
				TokenCount:     estimateTokens(content),
				Score:          1.0, // Explicit requests get full score
			})
		}
	}

	return budgetPages(candidates, maxTokens, m.config.MaxPagesPerPrompt)
}

// GetDocsForQuery returns documentation matching a search query.
func (m *Manager) GetDocsForQuery(ctx context.Context, query string, maxTokens int) (*DocContext, error) {
	if maxTokens <= 0 {
		maxTokens = m.config.MaxTokenBudget
	}

	collections, err := m.List(ctx, nil)
	if err != nil {
		return nil, err
	}

	queryKeywords := extractQueryKeywords(query)
	var candidates []*PageContent

	for _, coll := range collections {
		isShared := coll.Location == string(LocationShared)
		store := m.GetStore(isShared)
		if store == nil {
			continue
		}

		pagePaths, err := store.ListPageFiles(coll.ID)
		if err != nil {
			continue
		}

		// Load collection metadata to get page titles
		meta, err := store.LoadCollectionMeta(coll.ID)
		if err != nil {
			continue
		}

		for _, pagePath := range pagePaths {
			content, err := store.ReadPage(coll.ID, pagePath)
			if err != nil {
				continue
			}

			// Find page metadata
			var pageTitle string
			for _, p := range meta.Pages {
				if p.Path == pagePath {
					pageTitle = p.Title

					break
				}
			}

			// Score based on query matching
			score := calculateQueryScoreFromContent(pageTitle, content, queryKeywords)
			if score == 0 {
				continue
			}

			candidates = append(candidates, &PageContent{
				CollectionID:   coll.ID,
				CollectionName: coll.Name,
				Path:           pagePath,
				Title:          pageTitle,
				Content:        content,
				TokenCount:     estimateTokens(content),
				Score:          score,
			})
		}
	}

	// Sort by score (descending)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	return budgetPages(candidates, maxTokens, m.config.MaxPagesPerPrompt)
}

// budgetPages selects pages that fit within token and count limits.
func budgetPages(candidates []*PageContent, maxTokens, maxPages int) (*DocContext, error) {
	result := &DocContext{
		Pages: make([]*PageContent, 0),
	}

	totalTokens := 0

	for _, page := range candidates {
		if len(result.Pages) >= maxPages {
			result.Truncated = true

			break
		}

		if totalTokens+page.TokenCount > maxTokens {
			// Try to include partial content for important pages
			if page.Score > 0.8 && len(result.Pages) < maxPages/2 {
				// Truncate content to fit
				remaining := maxTokens - totalTokens
				if remaining > 500 { // Minimum useful content
					truncatedContent := truncateToTokens(page.Content, remaining)
					page.Content = truncatedContent
					page.TokenCount = remaining
					result.Pages = append(result.Pages, page)
					totalTokens += remaining
					result.Truncated = true
				}
			}

			continue
		}

		result.Pages = append(result.Pages, page)
		totalTokens += page.TokenCount
	}

	result.TotalTokens = totalTokens

	return result, nil
}

// calculateRelevanceScoreFromContent scores based on keyword matches in title, path, and content.
func calculateRelevanceScoreFromContent(title, path, content string, keywords []string) float64 {
	if len(keywords) == 0 {
		return 0.5 // Default score when no keywords
	}

	score := 0.0
	titleLower := strings.ToLower(title)
	pathLower := strings.ToLower(path)
	contentLower := strings.ToLower(content)

	for _, kw := range keywords {
		kwLower := strings.ToLower(kw)

		// Title match: high weight
		if strings.Contains(titleLower, kwLower) {
			score += 0.4
		}

		// Path match: medium weight
		if strings.Contains(pathLower, kwLower) {
			score += 0.3
		}

		// Content match: lower weight (just presence)
		if strings.Contains(contentLower, kwLower) {
			score += 0.1
		}
	}

	// Normalize to 0-1 range
	maxScore := float64(len(keywords)) * 0.8
	if maxScore > 0 {
		score = score / maxScore
		if score > 1.0 {
			score = 1.0
		}
	}

	return score
}

// calculateQueryScoreFromContent scores based on query keyword matches in title and content.
func calculateQueryScoreFromContent(title, content string, queryKeywords []string) float64 {
	if len(queryKeywords) == 0 {
		return 0
	}

	matches := 0
	titleLower := strings.ToLower(title)
	contentLower := strings.ToLower(content)

	for _, kw := range queryKeywords {
		kwLower := strings.ToLower(kw)
		if strings.Contains(titleLower, kwLower) || strings.Contains(contentLower, kwLower) {
			matches++
		}
	}

	// Return proportion of keywords matched
	return float64(matches) / float64(len(queryKeywords))
}

// extractQueryKeywords extracts search keywords from a query string.
func extractQueryKeywords(query string) []string {
	// Simple word splitting - could be enhanced with NLP
	words := strings.Fields(strings.ToLower(query))

	// Filter common words
	stopWords := map[string]bool{
		"a": true, "an": true, "the": true, "is": true, "are": true,
		"in": true, "on": true, "at": true, "to": true, "for": true,
		"of": true, "and": true, "or": true, "how": true, "what": true,
		"where": true, "when": true, "why": true, "which": true,
	}

	var keywords []string
	for _, w := range words {
		if len(w) >= 2 && !stopWords[w] {
			keywords = append(keywords, w)
		}
	}

	return keywords
}

// estimateTokens provides a rough token count estimate.
// Rule of thumb: ~4 characters per token for English text.
func estimateTokens(content string) int {
	return len(content) / 4
}

// truncateToTokens truncates content to approximately the given token count.
func truncateToTokens(content string, maxTokens int) string {
	maxChars := maxTokens * 4
	if len(content) <= maxChars {
		return content
	}

	// Try to truncate at a sentence boundary
	truncated := content[:maxChars]
	lastPeriod := strings.LastIndex(truncated, ". ")
	if lastPeriod > maxChars/2 {
		truncated = truncated[:lastPeriod+1]
	}

	return truncated + "\n\n[Content truncated...]"
}

// FormatDocsForPrompt formats the documentation context for AI prompt inclusion.
func FormatDocsForPrompt(docs *DocContext) string {
	if docs == nil || len(docs.Pages) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Relevant Documentation\n\n")

	for _, page := range docs.Pages {
		sb.WriteString("### ")
		if page.Title != "" {
			sb.WriteString(page.Title)
		} else {
			sb.WriteString(page.Path)
		}
		sb.WriteString(" (")
		sb.WriteString(page.CollectionName)
		sb.WriteString(")\n\n")
		sb.WriteString(page.Content)
		sb.WriteString("\n\n---\n\n")
	}

	if docs.Truncated {
		sb.WriteString("*Note: Some documentation was truncated to fit token limits.*\n")
	}

	return sb.String()
}
