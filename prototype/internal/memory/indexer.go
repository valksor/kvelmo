package memory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/vcs"
)

// Indexer automatically indexes tasks into vector memory.
type Indexer struct {
	memory    Memory
	workspace *storage.Workspace
	git       *vcs.Git
}

// NewIndexer creates a new task indexer.
func NewIndexer(memory Memory, workspace *storage.Workspace, git *vcs.Git) *Indexer {
	return &Indexer{
		memory:    memory,
		workspace: workspace,
		git:       git,
	}
}

// IndexTask indexes a completed task into memory.
func (i *Indexer) IndexTask(ctx context.Context, taskID string) error {
	// Get task data from work
	work, err := i.workspace.LoadWork(taskID)
	if err != nil {
		return fmt.Errorf("load work: %w", err)
	}

	// Index specification if present
	specs, _ := i.workspace.ListSpecifications(taskID)
	for _, specNum := range specs {
		specContent, err := i.workspace.LoadSpecification(taskID, specNum)
		if err != nil {
			slog.Warn("Failed to load specification for indexing", "task_id", taskID, "spec_num", specNum, "error", err)

			continue // Skip if can't read
		}

		doc := &Document{
			ID:      fmt.Sprintf("spec:%s:%d", taskID, specNum),
			TaskID:  taskID,
			Type:    TypeSpecification,
			Content: specContent,
			Metadata: map[string]interface{}{
				"title":   work.Metadata.Title,
				"specNum": specNum,
				"status":  "completed",
			},
			Tags: []string{"specification", "completed"},
		}

		if err := i.memory.Store(ctx, doc); err != nil {
			return fmt.Errorf("store specification: %w", err)
		}
	}

	// Index code changes (from git diff)
	if i.git != nil && work.Git.Branch != "" {
		diff, err := i.getDiff(ctx, taskID, work.Git.Branch, work.Git.BaseBranch)
		if err != nil {
			slog.Warn("Failed to get git diff for indexing", "task_id", taskID, "branch", work.Git.Branch, "error", err)
		} else if diff != "" {
			doc := &Document{
				ID:      "code:" + taskID,
				TaskID:  taskID,
				Type:    TypeCodeChange,
				Content: diff,
				Metadata: map[string]interface{}{
					"title": work.Metadata.Title,
				},
				Tags: []string{"code", "diff", "completed"},
			}

			if err := i.memory.Store(ctx, doc); err != nil {
				return fmt.Errorf("store code diff: %w", err)
			}
		}
	}

	// Index session logs
	sessions, _ := i.workspace.ListSessions(taskID)
	for _, session := range sessions {
		// Build session content from exchanges
		var transcript strings.Builder
		for _, ex := range session.Exchanges {
			transcript.WriteString(fmt.Sprintf("[%s] %s:\n%s\n\n", ex.Timestamp.Format("2006-01-02 15:04:05"), ex.Role, ex.Content))
		}

		if transcript.Len() > 0 {
			doc := &Document{
				ID:      fmt.Sprintf("session:%s:%s", taskID, session.Metadata.StartedAt.Format("20060102-150405")),
				TaskID:  taskID,
				Type:    TypeSession,
				Content: transcript.String(),
				Metadata: map[string]interface{}{
					"type":  session.Metadata.Type,
					"agent": session.Metadata.Agent,
					"state": session.Metadata.State,
				},
				Tags: []string{"session", "completed"},
			}

			if err := i.memory.Store(ctx, doc); err != nil {
				return fmt.Errorf("store session: %w", err)
			}
		}
	}

	return nil
}

// getDiff gets the git diff for a task.
func (i *Indexer) getDiff(ctx context.Context, _, branch, baseBranch string) (string, error) {
	if i.git == nil {
		return "", errors.New("git not available")
	}

	if branch == "" {
		return "", errors.New("task has no branch")
	}

	// Get base branch
	if baseBranch == "" {
		var err error
		baseBranch, err = i.git.GetBaseBranch(ctx)
		if err != nil {
			return "", err
		}
	}

	diff, err := i.git.Diff(ctx, baseBranch, branch)
	if err != nil {
		return "", err
	}

	return diff, nil
}

// SearchSimilar finds similar past tasks.
func (i *Indexer) SearchSimilar(ctx context.Context, query string, limit int) ([]*SearchResult, error) {
	opts := SearchOptions{
		Limit:         limit,
		MinScore:      0.7, // Similarity threshold
		DocumentTypes: []DocumentType{TypeSpecification, TypeCodeChange, TypeSolution},
	}

	return i.memory.Search(ctx, query, opts)
}

// FormatResults formats search results for display.
func (i *Indexer) FormatResults(results []*SearchResult) string {
	if len(results) == 0 {
		return "No similar tasks found."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d similar task(s):\n\n", len(results)))

	for _, result := range results {
		doc := result.Document
		sb.WriteString(fmt.Sprintf("## Task %s (Similarity: %.2f)\n", doc.TaskID, result.Score))

		// Add content preview
		preview := doc.Content
		if len(preview) > 500 {
			preview = preview[:500] + "..."
		}
		sb.WriteString(preview + "\n\n")
	}

	return sb.String()
}

// SaveToFile saves memory data to a file for backup.
func (i *Indexer) SaveToFile(ctx context.Context, filepath string) error {
	// Search for all documents with a high limit
	opts := SearchOptions{
		Limit: 10000, // High limit to get all documents
	}

	results, err := i.memory.Search(ctx, "", opts)
	if err != nil {
		return fmt.Errorf("search memory for export: %w", err)
	}

	// Extract documents from results
	docs := make([]*Document, len(results))
	for idx, result := range results {
		docs[idx] = result.Document
	}

	// Serialize to JSON
	data, err := json.MarshalIndent(docs, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal documents: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filepath, data, 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	slog.Info("Saved memory backup", "path", filepath, "documents", len(docs))

	return nil
}

// LoadFromFile loads memory data from a backup file.
func (i *Indexer) LoadFromFile(ctx context.Context, filepath string) error {
	// Read file
	data, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	// Deserialize JSON
	var docs []*Document
	if err := json.Unmarshal(data, &docs); err != nil {
		return fmt.Errorf("unmarshal documents: %w", err)
	}

	// Clear existing memory
	if err := i.memory.Clear(ctx); err != nil {
		return fmt.Errorf("clear existing memory: %w", err)
	}

	// Store each document
	for _, doc := range docs {
		if err := i.memory.Store(ctx, doc); err != nil {
			slog.Warn("Failed to restore document", "id", doc.ID, "error", err)
			// Continue with other documents
		}
	}

	slog.Info("Loaded memory backup", "path", filepath, "documents", len(docs))

	return nil
}

// ClearForTask removes all memory data for a specific task.
func (i *Indexer) ClearForTask(ctx context.Context, taskID string) error {
	// Search for all documents belonging to this task
	opts := SearchOptions{
		Limit: 1000, // High limit to find all
	}

	results, err := i.memory.Search(ctx, taskID, opts)
	if err != nil {
		return err
	}

	// Delete each document
	for _, result := range results {
		if err := i.memory.Delete(ctx, result.Document.ID); err != nil {
			return err
		}
	}

	return nil
}

// GetStats returns statistics about stored memory.
func (i *Indexer) GetStats(ctx context.Context) (*MemoryStats, error) {
	// Count documents by type
	stats := &MemoryStats{
		ByType: make(map[string]int),
	}

	// Search with a simple query to count results
	opts := SearchOptions{
		Limit: 10000, // High limit to get all
	}

	results, err := i.memory.Search(ctx, "", opts)
	if err != nil {
		return nil, err
	}

	stats.TotalDocuments = len(results)
	for _, result := range results {
		stats.ByType[string(result.Document.Type)]++
	}

	return stats, nil
}

// MemoryStats holds statistics about stored memory.
type MemoryStats struct {
	TotalDocuments int
	ByType         map[string]int
}
