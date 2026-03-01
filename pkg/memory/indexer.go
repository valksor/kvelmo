package memory

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Indexer auto-indexes task artefacts into the VectorStore when a task
// completes.  It extracts specification files, the git diff, and session
// log files from the project directory.
type Indexer struct {
	store      *VectorStore
	projectDir string // root of the project (contains .kvelmo/)
}

// NewIndexer creates an Indexer for the given project directory.
func NewIndexer(store *VectorStore, projectDir string) *Indexer {
	return &Indexer{
		store:      store,
		projectDir: projectDir,
	}
}

// IndexTask indexes artefacts for a completed task.
//
// taskID is the internal task ID; title and description are used as metadata.
// branch is the git branch containing the implementation; baseBranch is the
// branch it diverged from (used to compute the diff).
func (idx *Indexer) IndexTask(ctx context.Context, taskID, title, description, branch, baseBranch string) error {
	if err := idx.indexSpecifications(ctx, taskID, title); err != nil {
		slog.Warn("index specifications failed", "task_id", taskID, "error", err)
	}

	if err := idx.indexCodeChange(ctx, taskID, title, branch, baseBranch); err != nil {
		slog.Warn("index code change failed", "task_id", taskID, "error", err)
	}

	if err := idx.indexSession(ctx, taskID, title, description); err != nil {
		slog.Warn("index session failed", "task_id", taskID, "error", err)
	}

	return nil
}

// SearchSimilar searches the store for documents similar to query.
func (idx *Indexer) SearchSimilar(ctx context.Context, query string, limit int) ([]*SearchResult, error) {
	return idx.store.Search(ctx, query, SearchOptions{
		Limit:         limit,
		MinScore:      0.7,
		DocumentTypes: []DocumentType{TypeSpecification, TypeCodeChange, TypeSolution},
	})
}

// Stats returns current store statistics.
func (idx *Indexer) Stats() Stats {
	return idx.store.Stats()
}

// --- private helpers ---

func (idx *Indexer) indexSpecifications(ctx context.Context, taskID, title string) error {
	specDir := filepath.Join(idx.projectDir, ".kvelmo", "specifications")
	entries, err := os.ReadDir(specDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("read spec dir: %w", err)
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(specDir, e.Name()))
		if err != nil {
			slog.Warn("read specification file", "file", e.Name(), "error", err)

			continue
		}
		doc := &Document{
			ID:      fmt.Sprintf("specification:%s:%s", taskID, e.Name()),
			TaskID:  taskID,
			Type:    TypeSpecification,
			Content: string(data),
			Metadata: map[string]interface{}{
				"title": title,
				"file":  e.Name(),
			},
			Tags:      []string{"specification", "completed"},
			CreatedAt: time.Now(),
		}
		if err := idx.store.Store(ctx, doc); err != nil {
			slog.Warn("store specification document", "task_id", taskID, "error", err)
		}
	}

	return nil
}

func (idx *Indexer) indexCodeChange(ctx context.Context, taskID, title, branch, baseBranch string) error {
	if branch == "" {
		return nil
	}
	diff := gitDiff(ctx, idx.projectDir, baseBranch, branch)
	if strings.TrimSpace(diff) == "" {
		return nil
	}

	doc := &Document{
		ID:      "code_change:" + taskID,
		TaskID:  taskID,
		Type:    TypeCodeChange,
		Content: diff,
		Metadata: map[string]interface{}{
			"title":       title,
			"branch":      branch,
			"base_branch": baseBranch,
		},
		Tags:      []string{"code", "diff", "completed"},
		CreatedAt: time.Now(),
	}

	return idx.store.Store(ctx, doc)
}

func (idx *Indexer) indexSession(ctx context.Context, taskID, title, description string) error {
	logDir := filepath.Join(idx.projectDir, ".kvelmo", "sessions")
	entries, err := os.ReadDir(logDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("read session dir: %w", err)
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), taskID) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(logDir, e.Name()))
		if err != nil {
			continue
		}
		doc := &Document{
			ID:      fmt.Sprintf("session:%s:%s", taskID, e.Name()),
			TaskID:  taskID,
			Type:    TypeSession,
			Content: string(data),
			Metadata: map[string]interface{}{
				"title":       title,
				"description": description,
				"file":        e.Name(),
			},
			Tags:      []string{"session", "completed"},
			CreatedAt: time.Now(),
		}
		if err := idx.store.Store(ctx, doc); err != nil {
			slog.Warn("store session document", "task_id", taskID, "error", err)
		}
	}

	return nil
}

// gitDiff runs git diff between two refs in dir.
func gitDiff(ctx context.Context, dir, from, to string) string {
	if from == "" {
		from = "HEAD"
	}
	cmd := exec.CommandContext(ctx, "git", "diff", from+"..."+to)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		// Non-zero exit is common when refs don't exist; treat as empty diff.
		return ""
	}

	return string(out)
}
