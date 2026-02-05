package library

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestManager_GetDocsForPaths(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	// Create test docs
	docsDir := filepath.Join(tmpDir, "vscode-docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(docsDir, "extensions.md"), []byte("# VS Code Extensions\n\nHow to build extensions."), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := NewManager(ctx, tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Pull with path patterns
	_, err = m.Pull(ctx, docsDir, &PullOptions{
		Name:        "VS Code API",
		IncludeMode: IncludeModeAuto,
		Paths:       []string{"ide/vscode/**"},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Get docs for matching path
	docs, err := m.GetDocsForPaths(ctx, []string{"ide/vscode/src/extension.ts"}, 10000)
	if err != nil {
		t.Fatal(err)
	}

	if len(docs.Pages) == 0 {
		t.Error("expected matching docs for vscode path")
	}

	// Get docs for non-matching path
	docs, err = m.GetDocsForPaths(ctx, []string{"ide/jetbrains/src/Plugin.kt"}, 10000)
	if err != nil {
		t.Fatal(err)
	}

	if len(docs.Pages) != 0 {
		t.Error("expected no docs for non-matching path")
	}
}

func TestManager_GetExplicitDocs(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	// Create test file
	testFile := filepath.Join(tmpDir, "guide.md")
	if err := os.WriteFile(testFile, []byte("# User Guide\n\nThis is a guide."), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := NewManager(ctx, tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	result, err := m.Pull(ctx, testFile, &PullOptions{
		Name:        "User Guide",
		IncludeMode: IncludeModeExplicit, // Only include when explicitly requested
	})
	if err != nil {
		t.Fatal(err)
	}

	// Get by explicit name
	docs, err := m.GetExplicitDocs(ctx, []string{result.Collection.ID}, 10000)
	if err != nil {
		t.Fatal(err)
	}

	if len(docs.Pages) != 1 {
		t.Errorf("expected 1 page, got %d", len(docs.Pages))
	}
	if docs.Pages[0].Title != "User Guide" {
		t.Errorf("title = %q, want %q", docs.Pages[0].Title, "User Guide")
	}
}

func TestManager_GetDocsForQuery(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	// Create test files with different content
	docsDir := filepath.Join(tmpDir, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(docsDir, "auth.md"), []byte("# Authentication\n\nHow to authenticate users with JWT tokens."), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "routing.md"), []byte("# Routing\n\nHow to set up routes."), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := NewManager(ctx, tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	_, err = m.Pull(ctx, docsDir, &PullOptions{Name: "API Docs"})
	if err != nil {
		t.Fatal(err)
	}

	// Query for authentication
	docs, err := m.GetDocsForQuery(ctx, "how to authenticate users", 10000)
	if err != nil {
		t.Fatal(err)
	}

	if len(docs.Pages) == 0 {
		t.Fatal("expected matching docs for auth query")
	}

	// First result should be auth doc
	if docs.Pages[0].Title != "Authentication" {
		t.Errorf("expected Authentication first, got %q", docs.Pages[0].Title)
	}
}

func TestBudgetPages(t *testing.T) {
	candidates := []*PageContent{
		{Title: "High Score", Content: "A", TokenCount: 100, Score: 0.9},
		{Title: "Medium Score", Content: "B", TokenCount: 100, Score: 0.5},
		{Title: "Low Score", Content: "C", TokenCount: 100, Score: 0.1},
	}

	// Budget allows all pages
	result, err := budgetPages(candidates, 500, 10)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Pages) != 3 {
		t.Errorf("expected 3 pages, got %d", len(result.Pages))
	}
	if result.TotalTokens != 300 {
		t.Errorf("total tokens = %d, want 300", result.TotalTokens)
	}
	if result.Truncated {
		t.Error("should not be truncated")
	}

	// Budget limits pages
	result, err = budgetPages(candidates, 150, 10)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Pages) != 1 {
		t.Errorf("expected 1 page, got %d", len(result.Pages))
	}
}

func TestBudgetPages_MaxPagesLimit(t *testing.T) {
	candidates := make([]*PageContent, 10)
	for i := range candidates {
		candidates[i] = &PageContent{
			Title:      "Page",
			Content:    "Content",
			TokenCount: 10,
			Score:      1.0,
		}
	}

	result, err := budgetPages(candidates, 10000, 3)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Pages) != 3 {
		t.Errorf("expected 3 pages, got %d", len(result.Pages))
	}
	if !result.Truncated {
		t.Error("should be truncated when max pages exceeded")
	}
}

// Note: Tests for calculateRelevanceScoreFromContent, calculateQueryScoreFromContent,
// and extractQueryKeywords have been removed as those functions are now internal to
// EmbeddingScorer. See embedding_test.go for coverage of the scoring logic.

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		content string
		want    int
	}{
		{"", 0},
		{"test", 1},
		{"hello world", 2}, // 11 chars / 4 = 2
		{"This is a longer piece of content that should have more tokens.", 15},
	}

	for _, tt := range tests {
		t.Run(tt.content, func(t *testing.T) {
			got := estimateTokens(tt.content)
			if got != tt.want {
				t.Errorf("estimateTokens(%q) = %d, want %d", tt.content, got, tt.want)
			}
		})
	}
}

func TestTruncateToTokens(t *testing.T) {
	content := "This is the first sentence. This is the second sentence. This is the third sentence."

	// No truncation needed
	result := truncateToTokens(content, 100)
	if result != content {
		t.Error("should not truncate when under limit")
	}

	// Truncation needed
	result = truncateToTokens(content, 10)
	if len(result) > 70 { // 10 tokens * 4 chars + truncation marker (24 chars) + buffer
		t.Errorf("content too long after truncation: %d chars", len(result))
	}
	if !containsString(result, "[Content truncated...]") {
		t.Error("should contain truncation marker")
	}
}

func TestFormatDocsForPrompt(t *testing.T) {
	docs := &DocContext{
		Pages: []*PageContent{
			{
				CollectionName: "React Docs",
				Title:          "Hooks Guide",
				Content:        "How to use React hooks.",
			},
		},
		TotalTokens: 100,
		Truncated:   false,
	}

	output := FormatDocsForPrompt(docs)

	if !containsString(output, "## Relevant Documentation") {
		t.Error("should have documentation header")
	}
	if !containsString(output, "### Hooks Guide") {
		t.Error("should have page title")
	}
	if !containsString(output, "(React Docs)") {
		t.Error("should have collection name")
	}
	if !containsString(output, "How to use React hooks.") {
		t.Error("should have content")
	}
}

func TestFormatDocsForPrompt_Truncated(t *testing.T) {
	docs := &DocContext{
		Pages: []*PageContent{
			{CollectionName: "Docs", Title: "Page", Content: "Content"},
		},
		Truncated: true,
	}

	output := FormatDocsForPrompt(docs)

	if !containsString(output, "truncated") {
		t.Error("should mention truncation")
	}
}

func TestFormatDocsForPrompt_Empty(t *testing.T) {
	if output := FormatDocsForPrompt(nil); output != "" {
		t.Error("nil docs should return empty string")
	}

	if output := FormatDocsForPrompt(&DocContext{}); output != "" {
		t.Error("empty docs should return empty string")
	}
}

// Helper function.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}
