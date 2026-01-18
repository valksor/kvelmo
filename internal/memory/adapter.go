package memory

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// MemoryTool provides memory-related tools to agents.
type MemoryTool struct {
	memory  Memory
	indexer *Indexer
}

// NewMemoryTool creates a new memory tool.
func NewMemoryTool(memory Memory, indexer *Indexer) *MemoryTool {
	return &MemoryTool{
		memory:  memory,
		indexer: indexer,
	}
}

// SearchSimilarTasks searches for similar past tasks.
func (m *MemoryTool) SearchSimilarTasks(ctx context.Context, query string, limit int) ([]string, error) {
	results, err := m.memory.Search(ctx, query, SearchOptions{
		Limit:         limit,
		MinScore:      0.75,
		DocumentTypes: []DocumentType{TypeSpecification, TypeSolution},
	})
	if err != nil {
		return nil, err
	}

	var contexts []string
	for _, result := range results {
		contexts = append(contexts, formatResult(result))
	}

	return contexts, nil
}

// AugmentPrompt adds relevant memory context to an agent prompt.
func (m *MemoryTool) AugmentPrompt(ctx context.Context, taskTitle, taskDescription string) (string, error) {
	// Search for similar tasks
	query := fmt.Sprintf("%s %s", taskTitle, taskDescription)
	results, err := m.memory.Search(ctx, query, SearchOptions{
		Limit:         3,
		MinScore:      0.70,
		DocumentTypes: []DocumentType{TypeSpecification, TypeSolution},
	})

	if err != nil || len(results) == 0 {
		return "", fmt.Errorf("no similar tasks found: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("## Relevant Context from Similar Tasks\n\n")
	sb.WriteString("The following are past tasks and solutions that may be relevant to this request:\n\n")

	for _, result := range results {
		doc := result.Document
		sb.WriteString(fmt.Sprintf("### Task %s (Similarity: %.0f%%)\n", doc.TaskID, result.Score*100))
		sb.WriteString(fmt.Sprintf("**Type**: %s\n", doc.Type))

		// Add preview
		preview := doc.Content
		if len(preview) > 300 {
			preview = preview[:300] + "..."
		}
		sb.WriteString(fmt.Sprintf("**Content**:\n%s\n\n", preview))
	}

	sb.WriteString("Use this context to inform your approach. These are past solutions that worked for similar problems.\n")

	return sb.String(), nil
}

// GetCodeExamples retrieves relevant code examples from memory.
func (m *MemoryTool) GetCodeExamples(ctx context.Context, language, topic string) ([]string, error) {
	// Search for code changes related to the topic
	query := fmt.Sprintf("%s %s implementation code", language, topic)
	results, err := m.memory.Search(ctx, query, SearchOptions{
		Limit:         5,
		MinScore:      0.65,
		DocumentTypes: []DocumentType{TypeCodeChange},
	})

	if err != nil || len(results) == 0 {
		return nil, fmt.Errorf("no code examples found: %w", err)
	}

	var examples []string
	for _, result := range results {
		examples = append(examples, result.Document.Content)
	}

	return examples, nil
}

// LearnFromCorrection stores a correction/fix as a solution for future reference.
func (m *MemoryTool) LearnFromCorrection(ctx context.Context, taskID, problem, solution string) error {
	doc := &Document{
		ID:      fmt.Sprintf("solution:%s:%s", taskID, generateID()),
		TaskID:  taskID,
		Type:    TypeSolution,
		Content: fmt.Sprintf("Problem: %s\n\nSolution: %s", problem, solution),
		Metadata: map[string]interface{}{
			"problem": problem,
		},
		Tags: []string{"solution", "fix", "learned"},
	}

	return m.memory.Store(ctx, doc)
}

// formatResult formats a search result for display.
func formatResult(result *SearchResult) string {
	doc := result.Document
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## %s (%.2f similarity)\n", doc.TaskID, result.Score))
	sb.WriteString(fmt.Sprintf("Type: %s\n", doc.Type))

	// Add metadata
	if len(doc.Metadata) > 0 {
		sb.WriteString("Metadata:\n")
		for k, v := range doc.Metadata {
			sb.WriteString(fmt.Sprintf("  %s: %v\n", k, v))
		}
	}

	// Add content preview
	preview := doc.Content
	if len(preview) > 200 {
		preview = preview[:200] + "..."
	}
	sb.WriteString(fmt.Sprintf("\nContent:\n%s\n\n", preview))

	return sb.String()
}

// generateID generates a unique ID using timestamp.
func generateID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 10)
}

// GetMemoryStats returns statistics about the memory system.
func (m *MemoryTool) GetMemoryStats(ctx context.Context) (*MemoryStats, error) {
	if m.indexer == nil {
		return nil, errors.New("indexer not available")
	}

	return m.indexer.GetStats(ctx)
}

// ClearMemory clears all stored memory.
func (m *MemoryTool) ClearMemory(ctx context.Context) error {
	return m.memory.Clear(ctx)
}
