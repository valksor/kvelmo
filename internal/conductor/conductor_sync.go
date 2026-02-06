package conductor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-mehrhof/internal/workflow"
)

// SyncTaskResult describes task sync results for CLI/API consumers.
type SyncTaskResult struct {
	Success              bool
	HasChanges           bool
	TaskID               string
	Provider             string
	ChangesSummary       string
	SpecGenerated        string
	SourceUpdated        bool
	PreviousSnapshotPath string
	DiffPath             string
	Warnings             []string
}

// SyncTask syncs a task from its source provider and generates a delta specification.
func (c *Conductor) SyncTask(ctx context.Context, taskID string) (*SyncTaskResult, error) {
	c.mu.RLock()
	ws := c.workspace
	registry := c.providers
	c.mu.RUnlock()

	if ws == nil {
		return nil, errors.New("workspace not initialized")
	}
	if strings.TrimSpace(taskID) == "" {
		return nil, errors.New("task id is required")
	}

	work, err := ws.LoadWork(taskID)
	if err != nil {
		return nil, fmt.Errorf("load task work: %w", err)
	}

	taskDir := ws.WorkPath(taskID)
	if taskDir == "" {
		return nil, errors.New("task directory is empty")
	}

	sourcePath, err := resolveTaskSourcePath(taskDir, work.Source.Type, work.Source.Files)
	if err != nil {
		return nil, err
	}

	sourceContent, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("read source file: %w", err)
	}

	workUnit := &provider.WorkUnit{
		ID:          work.Metadata.ID,
		ExternalID:  work.Source.Ref,
		Provider:    work.Source.Type,
		Title:       work.Metadata.Title,
		Description: string(sourceContent),
	}

	updated, err := fetchUpdatedFromProvider(ctx, registry, workUnit)
	if err != nil {
		return nil, fmt.Errorf("fetch updated task: %w", err)
	}

	changes := provider.DetectChanges(workUnit, updated)
	result := &SyncTaskResult{
		Success:        true,
		HasChanges:     changes.HasChanges,
		TaskID:         taskID,
		Provider:       workUnit.Provider,
		ChangesSummary: changes.Summary(),
	}

	if !changes.HasChanges {
		return result, nil
	}

	gen := workflow.NewGenerator(taskDir)
	if err := gen.BackupSourceFile(sourcePath); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("backup source file: %v", err))
	}
	if err := gen.WriteDiffFile(changes); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("write changes summary: %v", err))
	}

	oldContent := extractSyncContent(workUnit)
	newContent := extractSyncContent(updated)

	genCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	specificationPath, err := gen.GenerateDeltaSpecification(genCtx, changes, oldContent, newContent)
	if err != nil {
		return nil, fmt.Errorf("generate delta specification: %w", err)
	}
	result.SpecGenerated = specificationPath

	sourceUpdated, previousPath, diffPath, err := persistSyncedSourceArtifacts(
		sourcePath,
		string(sourceContent),
		newContent,
		workUnit.Provider,
		changes,
	)
	result.SourceUpdated = sourceUpdated
	result.PreviousSnapshotPath = previousPath
	result.DiffPath = diffPath
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("persist synced source artifacts: %v", err))
	}

	return result, nil
}

func resolveTaskSourcePath(taskDir, sourceType string, sourceFiles []string) (string, error) {
	if len(sourceFiles) > 0 {
		sourcePath := sourceFiles[0]
		if filepath.IsAbs(sourcePath) {
			return "", fmt.Errorf("source file path is absolute, expected relative: %s", sourcePath)
		}

		return filepath.Join(taskDir, sourcePath), nil
	}

	return filepath.Join(taskDir, "source", sourceType+".txt"), nil
}

func fetchUpdatedFromProvider(
	ctx context.Context,
	registry *provider.Registry,
	old *provider.WorkUnit,
) (*provider.WorkUnit, error) {
	providerInstance, id, err := registry.Resolve(ctx, old.ExternalID, provider.NewConfig(), provider.ResolveOptions{})
	if err != nil {
		return nil, fmt.Errorf("resolve provider: %w", err)
	}

	reader, ok := providerInstance.(provider.Reader)
	if !ok {
		return nil, fmt.Errorf("provider %s does not support reading", old.Provider)
	}

	updated, err := reader.Fetch(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetch from provider: %w", err)
	}

	return updated, nil
}

func extractSyncContent(wu *provider.WorkUnit) string {
	var content strings.Builder

	if wu.Title != "" {
		content.WriteString("# ")
		content.WriteString(wu.Title)
		content.WriteString("\n\n")
	}

	if wu.Description != "" {
		content.WriteString(wu.Description)
		content.WriteString("\n")
	}

	if len(wu.Comments) > 0 {
		content.WriteString("\n## Comments\n\n")
		for _, comment := range wu.Comments {
			author := provider.ResolveAuthor(comment)
			if author == "" {
				author = comment.Author.ID
			}
			content.WriteString("### ")
			content.WriteString(author)
			content.WriteString("\n\n")
			content.WriteString(comment.Body)
			content.WriteString("\n\n")
		}
	}

	return content.String()
}

func persistSyncedSourceArtifacts(
	sourcePath, oldContent, newContent, providerType string,
	changes provider.ChangeSet,
) (bool, string, string, error) {
	if sourcePath == "" {
		return false, "", "", errors.New("source path is empty")
	}

	if writeErr := os.WriteFile(sourcePath, []byte(newContent), 0o644); writeErr != nil {
		return false, "", "", fmt.Errorf("write updated source: %w", writeErr)
	}
	sourceUpdated := true

	// Wrike-specific update flow artifacts for parity with previous behavior.
	if strings.EqualFold(providerType, "wrike") {
		sourceDir := filepath.Dir(sourcePath)
		previousPath := filepath.Join(sourceDir, "wrike_previous.txt")
		diffPath := filepath.Join(sourceDir, "wrike_diff.txt")

		if writeErr := os.WriteFile(previousPath, []byte(oldContent), 0o644); writeErr != nil {
			return sourceUpdated, previousPath, diffPath, fmt.Errorf("write wrike previous snapshot: %w", writeErr)
		}
		if writeErr := os.WriteFile(diffPath, []byte(changes.FormatDiff()), 0o644); writeErr != nil {
			return sourceUpdated, previousPath, diffPath, fmt.Errorf("write wrike diff summary: %w", writeErr)
		}

		return sourceUpdated, previousPath, diffPath, nil
	}

	return sourceUpdated, "", "", nil
}
