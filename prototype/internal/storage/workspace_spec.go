package storage

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// validTaskIDRegex is pre-compiled to avoid re-compiling on every validation call.
// Valid task IDs contain only alphanumeric characters, hyphens, and underscores.
var validTaskIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// resolveFilenamePattern converts a pattern like "SPEC-{n}.md" with a number into a filename.
// Example: resolveFilenamePattern("SPEC-{n}.md", 1) returns "SPEC-1.md".
func resolveFilenamePattern(pattern string, number int) string {
	if pattern == "" {
		pattern = "specification-{n}.md" // fallback default
	}

	return strings.Replace(pattern, "{n}", strconv.Itoa(number), 1)
}

// SpecificationsDir returns the specifications directory path (internal storage).
func (w *Workspace) SpecificationsDir(taskID string) string {
	return filepath.Join(w.WorkPath(taskID), specsDirName)
}

// SpecificationPath returns the path for a specification file (internal storage).
func (w *Workspace) SpecificationPath(taskID string, number int, cfg *WorkspaceConfig) string {
	pattern := "specification-{n}.md"
	if cfg != nil && cfg.Specification.FilenamePattern != "" {
		pattern = cfg.Specification.FilenamePattern
	}

	return filepath.Join(w.SpecificationsDir(taskID), resolveFilenamePattern(pattern, number))
}

// ProjectSpecificationsDir returns the project-local specifications directory path.
// Uses storage.project_dir from config (e.g., "tickets" → <root>/tickets/<task-id>/)
// If project_dir is empty, returns .mehrhof/work/<task-id>/.
// Note: Project-local storage is flat (no /specifications/ subdirectory).
func (w *Workspace) ProjectSpecificationsDir(taskID string, cfg *WorkspaceConfig) string {
	projectDir := ".mehrhof/work"
	if cfg != nil && cfg.Storage.ProjectDir != "" {
		projectDir = cfg.Storage.ProjectDir
	}

	return filepath.Join(w.root, projectDir, taskID)
}

// ProjectSpecificationPath returns the project-local path for a specification file.
func (w *Workspace) ProjectSpecificationPath(taskID string, number int, cfg *WorkspaceConfig) string {
	pattern := "specification-{n}.md"
	if cfg != nil && cfg.Specification.FilenamePattern != "" {
		pattern = cfg.Specification.FilenamePattern
	}

	return filepath.Join(w.ProjectSpecificationsDir(taskID, cfg), resolveFilenamePattern(pattern, number))
}

// EffectiveSpecificationsDir returns the specifications directory based on storage config.
// If storage.save_in_project is true, returns project-local path; otherwise returns global storage path.
func (w *Workspace) EffectiveSpecificationsDir(taskID string, cfg *WorkspaceConfig) string {
	if cfg != nil && cfg.Storage.SaveInProject {
		return w.ProjectSpecificationsDir(taskID, cfg)
	}

	return w.SpecificationsDir(taskID)
}

// EffectiveSpecificationPath returns the specification path based on storage config.
// If storage.save_in_project is true, returns project-local path; otherwise returns global storage path.
func (w *Workspace) EffectiveSpecificationPath(taskID string, number int, cfg *WorkspaceConfig) string {
	if cfg != nil && cfg.Storage.SaveInProject {
		return w.ProjectSpecificationPath(taskID, number, cfg)
	}

	return w.SpecificationPath(taskID, number, cfg)
}

// isValidTaskID validates that a taskID is safe to use as a directory name.
// It rejects empty strings, path traversal attempts, and characters that are
// problematic on common filesystems. Valid task IDs contain only alphanumeric
// characters, hyphens, and underscores.
func isValidTaskID(taskID string) bool {
	if taskID == "" {
		return false
	}
	// Reject path traversal attempts and absolute paths
	if strings.Contains(taskID, "..") || strings.ContainsAny(taskID, "\\/") {
		return false
	}
	// Restrict to alphanumeric, hyphen, underscore - safe on all platforms
	return validTaskIDRegex.MatchString(taskID)
}

// SaveSpecification saves a specification file (markdown).
//
// Storage location is determined by storage config:
//   - storage.save_in_project: false → ~/.valksor/mehrhof/workspaces/<name>/work/<task-id>/specifications/
//   - storage.save_in_project: true → .mehrhof/work/<task-id>/
//   - storage.save_in_project: true + project_dir: "tickets" → tickets/<task-id>/
func (w *Workspace) SaveSpecification(taskID string, number int, content string) error {
	if !isValidTaskID(taskID) {
		return fmt.Errorf("invalid task ID %q: must contain only alphanumeric characters, hyphens, and underscores", taskID)
	}

	cfg, err := w.LoadConfig()
	if err != nil {
		slog.Warn("failed to load workspace config, using defaults",
			"task_id", taskID,
			"spec_number", number,
			"error", err,
		)
		cfg = NewDefaultWorkspaceConfig()
	}

	// Single storage location based on config
	specPath := w.EffectiveSpecificationPath(taskID, number, cfg)
	specDir := filepath.Dir(specPath)

	if err := os.MkdirAll(specDir, 0o755); err != nil {
		return fmt.Errorf("create specifications directory: %w", err)
	}
	if err := os.WriteFile(specPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("save specification: %w", err)
	}

	return nil
}

// LoadSpecification loads a specification file content.
func (w *Workspace) LoadSpecification(taskID string, number int) (string, error) {
	cfg, _ := w.LoadConfig() // ignore error, will use defaults
	specPath := w.EffectiveSpecificationPath(taskID, number, cfg)
	data, err := os.ReadFile(specPath)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// buildSpecPatternRegex builds a regex to match filenames based on the config pattern.
// E.g., "SPEC-{n}.md" becomes `^SPEC-(\d+)\.md$`.
func buildSpecPatternRegex(pattern string) *regexp.Regexp {
	if pattern == "" {
		pattern = "specification-{n}.md"
	}
	// Escape regex special chars except {n} placeholder
	escaped := regexp.QuoteMeta(pattern)
	// Replace escaped \{n\} with capturing group for digits
	regexStr := strings.Replace(escaped, `\{n\}`, `(\d+)`, 1)

	return regexp.MustCompile(`^` + regexStr + `$`)
}

// ListSpecifications returns all specification numbers for a task.
func (w *Workspace) ListSpecifications(taskID string) ([]int, error) {
	cfg, _ := w.LoadConfig() // ignore error, will use defaults
	specsDir := w.EffectiveSpecificationsDir(taskID, cfg)

	entries, err := os.ReadDir(specsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []int{}, nil
		}

		return nil, fmt.Errorf("read specifications directory: %w", err)
	}

	pattern := buildSpecPatternRegex(cfg.Specification.FilenamePattern)
	var numbers []int

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		matches := pattern.FindStringSubmatch(entry.Name())
		if matches != nil {
			num, _ := strconv.Atoi(matches[1])
			numbers = append(numbers, num)
		}
	}

	slices.Sort(numbers)

	return numbers, nil
}

// NextSpecificationNumber returns the next available specification number.
func (w *Workspace) NextSpecificationNumber(taskID string) (int, error) {
	specifications, err := w.ListSpecifications(taskID)
	if err != nil {
		return 0, err
	}

	if len(specifications) == 0 {
		return 1, nil
	}

	return specifications[len(specifications)-1] + 1, nil
}

// GatherSpecificationsContent reads all specifications and returns combined content.
func (w *Workspace) GatherSpecificationsContent(taskID string) (string, error) {
	specifications, err := w.ListSpecifications(taskID)
	if err != nil {
		return "", err
	}

	var parts []string
	for _, num := range specifications {
		content, err := w.LoadSpecification(taskID, num)
		if err != nil {
			continue
		}
		parts = append(parts, fmt.Sprintf("### Specification %d\n\n%s", num, content))
	}

	return strings.Join(parts, "\n\n---\n\n"), nil
}

// GetLatestSpecificationContent returns only the most recent specification content.
func (w *Workspace) GetLatestSpecificationContent(taskID string) (string, int, error) {
	specifications, err := w.ListSpecifications(taskID)
	if err != nil {
		return "", 0, err
	}

	if len(specifications) == 0 {
		return "", 0, nil
	}

	// specifications are sorted, so last one is the latest
	latestNum := specifications[len(specifications)-1]
	content, err := w.LoadSpecification(taskID, latestNum)
	if err != nil {
		return "", 0, err
	}

	return content, latestNum, nil
}

// ParseSpecification parses a specification file with optional YAML frontmatter.
func (w *Workspace) ParseSpecification(taskID string, number int) (*Specification, error) {
	content, err := w.LoadSpecification(taskID, number)
	if err != nil {
		return nil, err
	}

	spec := &Specification{
		Number: number,
		Status: SpecificationStatusDraft, // default status
	}

	// Check for YAML frontmatter (starts with ---)
	if strings.HasPrefix(content, "---\n") {
		// Find the closing ---
		endIdx := strings.Index(content[4:], "\n---")
		if endIdx > 0 {
			frontmatter := content[4 : 4+endIdx]
			spec.Content = strings.TrimSpace(content[4+endIdx+4:])

			// Parse frontmatter
			if err := yaml.Unmarshal([]byte(frontmatter), spec); err != nil {
				// Ignore frontmatter parsing errors, just use content
				spec.Content = content
			}
		} else {
			spec.Content = content
		}
	} else {
		spec.Content = content
	}

	// Extract title from first heading
	lines := strings.Split(spec.Content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "# ") {
			spec.Title = strings.TrimPrefix(line, "# ")

			break
		}
	}

	return spec, nil
}

// SaveSpecificationWithMeta saves a specification with YAML frontmatter.
func (w *Workspace) SaveSpecificationWithMeta(taskID string, spec *Specification) error {
	// Ensure timestamps
	now := time.Now()
	if spec.CreatedAt.IsZero() {
		spec.CreatedAt = now
	}
	spec.UpdatedAt = now

	// Build frontmatter
	frontmatter, err := yaml.Marshal(spec)
	if err != nil {
		return fmt.Errorf("marshal specification frontmatter: %w", err)
	}

	// Combine frontmatter and content
	var content strings.Builder
	content.WriteString("---\n")
	content.Write(frontmatter)
	content.WriteString("---\n\n")
	content.WriteString(spec.Content)

	return w.SaveSpecification(taskID, spec.Number, content.String())
}

// UpdateSpecificationStatus updates the status of a specification file.
func (w *Workspace) UpdateSpecificationStatus(taskID string, number int, status string) error {
	spec, err := w.ParseSpecification(taskID, number)
	if err != nil {
		return err
	}

	spec.Status = status
	spec.UpdatedAt = time.Now()

	// Set completion timestamp if done
	if status == SpecificationStatusDone && spec.CompletedAt.IsZero() {
		spec.CompletedAt = time.Now()
	}

	return w.SaveSpecificationWithMeta(taskID, spec)
}

// ListSpecificationsWithStatus returns all specifications with their parsed status.
func (w *Workspace) ListSpecificationsWithStatus(taskID string) ([]*Specification, error) {
	numbers, err := w.ListSpecifications(taskID)
	if err != nil {
		return nil, err
	}

	specifications := make([]*Specification, 0, len(numbers))
	for _, num := range numbers {
		specification, err := w.ParseSpecification(taskID, num)
		if err != nil {
			// Include specification with error status
			specifications = append(specifications, &Specification{Number: num, Status: "error"})

			continue
		}
		specifications = append(specifications, specification)
	}

	return specifications, nil
}

// GetSpecificationsSummary returns a summary of specification statuses.
func (w *Workspace) GetSpecificationsSummary(taskID string) (map[string]int, error) {
	specifications, err := w.ListSpecificationsWithStatus(taskID)
	if err != nil {
		return nil, err
	}

	summary := map[string]int{
		SpecificationStatusDraft:        0,
		SpecificationStatusReady:        0,
		SpecificationStatusImplementing: 0,
		SpecificationStatusDone:         0,
	}

	for _, specification := range specifications {
		summary[specification.Status]++
	}

	return summary, nil
}

// SpecValidationResult represents the result of validating a specification.
type SpecValidationResult struct {
	Number   int
	IsValid  bool
	Errors   []string
	Warnings []string
}

// ValidateSpecification validates a specification for completeness and quality.
// Returns a validation result with errors and warnings.
func (w *Workspace) ValidateSpecification(taskID string, number int) (*SpecValidationResult, error) {
	spec, err := w.ParseSpecification(taskID, number)
	if err != nil {
		return nil, err
	}

	result := &SpecValidationResult{
		Number:   number,
		IsValid:  true,
		Errors:   []string{},
		Warnings: []string{},
	}

	// Check required sections
	requiredSections := []struct {
		name     string
		pattern  string
		required bool
	}{
		// Use \s+ to allow flexible spacing between ## and section name
		{"## Request", `(?im)^##\s+Request\s*$`, true},
		{"## Plan", `(?im)^##\s+Plan\s*$`, true},
		{"## Context", `(?im)^##\s+Context\s*$`, true},
		{"## Unknowns", `(?im)^##\s+Unknowns\s*$`, true},
		{"## Complete Condition", `(?im)^##\s+Complete\s+Condition\s*$`, true},
		{"## Status", `(?im)^##\s+Status\s*$`, true},
	}

	for _, section := range requiredSections {
		if section.required {
			matched, _ := regexp.MatchString(section.pattern, spec.Content)
			if !matched {
				result.Errors = append(result.Errors, "Missing: "+section.name)
				result.IsValid = false
			}
		}
	}

	// Only run quality checks if all required sections are present
	if result.IsValid {
		// Quality checks
		planPattern := regexp.MustCompile(`(?im)^## Plan\s*$`)
		planMatch := planPattern.FindStringIndex(spec.Content)
		if planMatch != nil {
			// Count plan steps (numbered lines after "## Plan")
			afterPlan := spec.Content[planMatch[1]:]
			// Limit to just the Plan section (until next section)
			nextSection := regexp.MustCompile(`(?im)^## [A-Z]`)
			nextSectionMatch := nextSection.FindStringIndex(afterPlan)
			planText := afterPlan
			if nextSectionMatch != nil {
				planText = afterPlan[:nextSectionMatch[0]]
			}
			stepPattern := regexp.MustCompile(`(?m)^\d+\.\s+`)
			steps := stepPattern.FindAllString(planText, -1)
			if len(steps) < 2 {
				result.Warnings = append(result.Warnings, "Plan should have at least 2 steps")
			}
		}

		// Check if Unknowns has default answers
		unknownsPattern := regexp.MustCompile(`(?im)^## Unknowns\s*$`)
		unknownsMatch := unknownsPattern.FindStringIndex(spec.Content)
		if unknownsMatch != nil {
			afterUnknowns := spec.Content[unknownsMatch[1]:]
			// Get text until next section or end
			nextSection := regexp.MustCompile(`(?im)^## [A-Z]`)
			nextSectionMatch := nextSection.FindStringIndex(afterUnknowns)
			unknownsText := afterUnknowns
			if nextSectionMatch != nil {
				unknownsText = afterUnknowns[:nextSectionMatch[0]]
			}

			// Check for "user input required" pattern
			if strings.Contains(strings.ToLower(unknownsText), "user input required") {
				result.Warnings = append(result.Warnings, "Unknowns should have default answers, not 'user input required'")
			}
		}

		// Check Complete Condition has both manual and run validation
		completePattern := regexp.MustCompile(`(?im)^## Complete Condition\s*$`)
		completeMatch := completePattern.FindStringIndex(spec.Content)
		if completeMatch != nil {
			afterComplete := spec.Content[completeMatch[1]:]
			nextSection := regexp.MustCompile(`(?im)^## [A-Z]`)
			nextSectionMatch := nextSection.FindStringIndex(afterComplete)
			completeText := afterComplete
			if nextSectionMatch != nil {
				completeText = afterComplete[:nextSectionMatch[0]]
			}

			hasManual := regexp.MustCompile(`(?m)^-\s*manual:\s*`).MatchString(completeText)
			hasRun := regexp.MustCompile(`(?m)^-\s*run:\s*`).MatchString(completeText)

			if !hasManual {
				result.Warnings = append(result.Warnings, "Complete Condition should include manual validation step")
			}
			if !hasRun {
				result.Warnings = append(result.Warnings, "Complete Condition should include run validation step")
			}
		}
	}

	return result, nil
}
