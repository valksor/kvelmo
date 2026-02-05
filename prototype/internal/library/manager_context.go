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

			// Calculate relevance score (uses embeddings if available, falls back to keywords)
			score := m.scorer.ScoreForPath(ctx, pageTitle, content, keywords)

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

			// Score based on query matching (uses embeddings if available, falls back to keywords)
			score := m.scorer.ScoreForQuery(ctx, pageTitle, content, query)
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
