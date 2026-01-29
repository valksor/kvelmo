package conductor

import (
	"context"
	"fmt"
	"strings"

	"github.com/valksor/go-mehrhof/internal/quality"
)

// runLinters executes available linters for the project and returns formatted results.
// Returns empty string if no linters are available or all pass with no issues.
func (c *Conductor) runLinters(ctx context.Context) string {
	workDir := c.opts.WorkDir
	if c.git != nil {
		workDir = c.git.Root()
	}

	// Load workspace config for quality settings
	workspaceCfg, _ := c.workspace.LoadConfig()

	// Check if quality checks are disabled
	if workspaceCfg.Quality != nil && !workspaceCfg.Quality.Enabled {
		c.logVerbosef("Quality checks disabled in config")

		return ""
	}

	// Create linter registry with config and detect applicable linters
	registry := quality.NewRegistry(workspaceCfg.Quality)
	linters := registry.DetectForProject(workDir)

	// Warning: quality enabled but no linters configured
	if len(linters) == 0 && workspaceCfg.Quality != nil && workspaceCfg.Quality.Enabled {
		if !workspaceCfg.Quality.UseDefaults {
			c.logVerbosef("Warning: Quality checks enabled but no linters configured. " +
				"Set use_defaults: true or explicitly configure linters in .mehrhof/config.yaml")
		} else {
			c.logVerbosef("No linters detected for this project")
		}

		return ""
	}

	if len(linters) == 0 {
		c.logVerbosef("No linters detected for this project")

		return ""
	}

	c.logVerbosef("Running %d linter(s): %s", len(linters), linterNames(linters))

	// Get changed files if git is available (only lint changed files for efficiency)
	var files []string
	if c.git != nil {
		changedFiles, err := c.git.Status(ctx)
		if err == nil {
			for _, f := range changedFiles {
				// Check if file is modified, staged, or untracked ('?' in index)
				if f.IsModified() || f.IsStaged() || f.Index == '?' {
					files = append(files, f.Path)
				}
			}
		}
	}

	// Run all detected linters
	results := registry.RunAll(ctx, workDir, files)

	// Format results for the agent prompt
	formatted := quality.FormatResults(results)

	// Log summary
	totalIssues := 0
	for _, r := range results {
		if r != nil && r.Issues != nil {
			totalIssues += len(r.Issues)
		}
	}
	if totalIssues > 0 {
		c.publishProgress(fmt.Sprintf("Linters found %d issues to address", totalIssues), 15)
	} else {
		c.publishProgress("Linters passed with no issues", 15)
	}

	return formatted
}

// linterNames returns a comma-separated list of linter names.
func linterNames(linters []quality.Linter) string {
	names := make([]string, len(linters))
	for i, l := range linters {
		names[i] = l.Name()
	}

	return strings.Join(names, ", ")
}
