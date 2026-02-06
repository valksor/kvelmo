package conductor

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/valksor/go-mehrhof/internal/vcs"
)

const defaultSpecDiffContextLines = 3

// GetSpecificationFileDiff returns a unified diff for a specification's implemented file.
// It combines committed branch changes and uncommitted changes when both are available.
func (c *Conductor) GetSpecificationFileDiff(
	ctx context.Context,
	taskID string,
	specNumber int,
	filePath string,
	contextLines int,
) (string, error) {
	if contextLines <= 0 {
		contextLines = defaultSpecDiffContextLines
	}

	if c.workspace == nil {
		return "", errors.New("workspace not initialized")
	}

	gitClient := c.git
	if gitClient == nil {
		var gitErr error
		gitClient, gitErr = vcs.New(ctx, c.CodeDir())
		if gitErr != nil {
			return "", fmt.Errorf("git not initialized: %w", gitErr)
		}
	}

	if taskID == "" {
		return "", errors.New("task ID is required")
	}

	if specNumber <= 0 {
		return "", fmt.Errorf("specification number must be positive, got %d", specNumber)
	}

	if filePath == "" {
		return "", errors.New("file path is required")
	}

	normalizedPath, err := normalizeSpecFilePath(filePath, c.CodeDir())
	if err != nil {
		return "", err
	}

	spec, err := c.workspace.ParseSpecification(taskID, specNumber)
	if err != nil {
		return "", fmt.Errorf("load specification: %w", err)
	}

	implementedFiles := make([]string, 0, len(spec.ImplementedFiles))
	for _, implemented := range spec.ImplementedFiles {
		normalizedImplemented, normalizeErr := normalizeSpecFilePath(implemented, c.CodeDir())
		if normalizeErr != nil {
			continue
		}
		implementedFiles = append(implementedFiles, normalizedImplemented)
	}
	if !slices.Contains(implementedFiles, normalizedPath) {
		return "", fmt.Errorf("file %q is not listed in specification-%d implemented files", filePath, specNumber)
	}

	branchDiff := ""
	baseBranch, baseErr := gitClient.DetectDefaultBranch(ctx)
	if baseErr == nil {
		diff, diffErr := gitClient.Diff(
			ctx,
			fmt.Sprintf("-U%d", contextLines),
			baseBranch+"...HEAD",
			"--",
			normalizedPath,
		)
		if diffErr == nil {
			branchDiff = diff
		}
	}

	uncommittedDiff := ""
	diff, diffErr := gitClient.Diff(ctx, fmt.Sprintf("-U%d", contextLines), "HEAD", "--", normalizedPath)
	if diffErr == nil {
		uncommittedDiff = diff
	}

	switch {
	case branchDiff != "" && uncommittedDiff != "":
		return fmt.Sprintf(
			"# Branch changes since %s\n\n%s\n\n# Uncommitted changes\n\n%s",
			baseBranch,
			branchDiff,
			uncommittedDiff,
		), nil
	case branchDiff != "":
		return branchDiff, nil
	default:
		return uncommittedDiff, nil
	}
}

func normalizeSpecFilePath(path, codeRoot string) (string, error) {
	cleaned := filepath.Clean(path)
	if cleaned == "." || cleaned == "" {
		return "", errors.New("file path is required")
	}

	if filepath.IsAbs(cleaned) {
		rootAbs, err := filepath.Abs(codeRoot)
		if err != nil {
			return "", fmt.Errorf("resolve code root: %w", err)
		}
		rel, err := filepath.Rel(rootAbs, cleaned)
		if err != nil {
			return "", fmt.Errorf("resolve absolute file path: %w", err)
		}
		if rel == "." {
			return "", errors.New("file path cannot be repository root")
		}
		if strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
			return "", errors.New("absolute file path must be inside repository")
		}
		cleaned = rel
	}

	return filepath.ToSlash(cleaned), nil
}
