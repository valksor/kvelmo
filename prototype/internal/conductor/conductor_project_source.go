package conductor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/valksor/go-toolkit/providerconfig"
	"github.com/valksor/go-toolkit/workunit"
)

// readResearchSource scans a directory and builds a research manifest.
// This does NOT read file contents - it builds metadata for agent exploration.
// Used for research: source type to avoid token bloat from large documentation bases.
func (c *Conductor) readResearchSource(dirPath string) (*ResearchManifest, error) {
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("stat path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", absPath)
	}

	manifest := &ResearchManifest{
		BasePath:    absPath,
		Structure:   make([]DirEntry, 0),
		EntryPoints: make([]string, 0),
		ByCategory:  make(map[string][]string),
	}

	// Entry point patterns to detect
	entryPointPatterns := []string{
		"tasks/README.md", "tasks/index.md",
		"README.md", "readme.md",
		"TODOS.md", "TODO.md", "ROADMAP.md",
	}

	// File extension categories
	docExts := map[string]bool{".md": true, ".txt": true, ".rst": true, ".adoc": true}
	codeExts := map[string]bool{
		".go": true, ".js": true, ".ts": true, ".tsx": true, ".jsx": true,
		".py": true, ".java": true, ".rs": true, ".rb": true, ".php": true, ".c": true, ".cpp": true,
	}
	configExts := map[string]bool{".yaml": true, ".yml": true, ".json": true, ".toml": true, ".xml": true}

	// Walk directory and collect metadata
	err = filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil //nolint:nilerr // Skip unreadable files
		}

		// Skip hidden files/directories
		if strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}

			return nil
		}

		// Skip common exclusions
		if info.IsDir() {
			switch info.Name() {
			case "node_modules", "vendor", "target", "build", "dist", ".git", "venv", "__pycache__":
				return filepath.SkipDir
			}
		}

		relPath, _ := filepath.Rel(absPath, path)

		entry := DirEntry{
			Path: relPath,
			Name: info.Name(),
			Type: map[bool]string{true: "dir", false: "file"}[info.IsDir()],
			Size: info.Size(),
		}

		if info.IsDir() {
			manifest.Structure = append(manifest.Structure, entry)

			return nil
		}

		// Categorize file
		ext := strings.ToLower(filepath.Ext(path))
		switch {
		case docExts[ext]:
			entry.Category = "docs"
		case codeExts[ext]:
			entry.Category = "code"
		case configExts[ext]:
			entry.Category = "config"
		default:
			entry.Category = "other"
		}

		manifest.Structure = append(manifest.Structure, entry)
		manifest.FileCount++

		// Track by category (store absolute paths)
		manifest.ByCategory[entry.Category] = append(manifest.ByCategory[entry.Category], path)

		// Check for entry points
		if entry.Category == "docs" {
			for _, pattern := range entryPointPatterns {
				if strings.EqualFold(relPath, pattern) ||
					strings.Contains(strings.ToLower(relPath), strings.ToLower(pattern)) {
					manifest.EntryPoints = append(manifest.EntryPoints, path)
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk directory: %w", err)
	}

	// Sort entry points by path length (shorter first = more likely to be root-level)
	sort.Slice(manifest.EntryPoints, func(i, j int) bool {
		return len(manifest.EntryPoints[i]) < len(manifest.EntryPoints[j])
	})

	return manifest, nil
}

// buildResearchPlanningPrompt creates the prompt for research-based planning.
// The prompt provides a file manifest and instructs the agent to use Read/Grep tools
// for selective exploration, rather than concatenating all file contents.
func buildResearchPlanningPrompt(title string, manifest *ResearchManifest, customInstructions string) string {
	currentTime := time.Now().Format("2006-01-02 15:04")

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(`You are an expert project manager and software architect. Your task is to research documentation and create a structured task breakdown.

Current timestamp: %s

## Project
%s

## Research Base Path
%s

## Documentation Structure
This directory contains %d files for you to research.
`, currentTime, title, manifest.BasePath, manifest.FileCount))

	// Entry points
	if len(manifest.EntryPoints) > 0 {
		sb.WriteString("## Detected Entry Points\n")
		sb.WriteString("The following files appear to be task/index files:\n\n")
		for i, ep := range manifest.EntryPoints {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, ep))
		}
		sb.WriteString("\nStart by reading these files to understand existing task structure.\n\n")
	}

	// Directory tree
	sb.WriteString("## Directory Structure\n\n")
	sb.WriteString("```\n")
	for _, entry := range manifest.Structure {
		indent := strings.Repeat("  ", strings.Count(entry.Path, string(filepath.Separator)))
		if entry.Type == "dir" {
			sb.WriteString(fmt.Sprintf("%s%s/\n", indent, entry.Name))
		} else {
			sb.WriteString(fmt.Sprintf("%s%s (%s, %d bytes)\n", indent, entry.Name, entry.Category, entry.Size))
		}
	}
	sb.WriteString("```\n\n")

	// Custom instructions
	if customInstructions != "" {
		sb.WriteString(fmt.Sprintf(`## Custom Instructions
%s

`, customInstructions))
	}

	sb.WriteString(`## Research Instructions

IMPORTANT: You have access to Read, Glob, and Grep tools to explore these files.

1. **Start with entry points** - Read the detected entry point files first to understand any existing task structure
2. **Explore selectively** - Use Glob to find relevant files, Grep to search content, and Read to examine specific files
3. **Preserve existing structure** - If tasks/README.md or similar exists, incorporate those tasks rather than creating new ones
4. **Categorize intelligently** - Group related tasks based on the documentation structure you discover

## Output Format

Create a structured task breakdown in the following format:

## Tasks

For each task, use this format:

### task-N: Task Title
- **Priority**: N (1 = highest)
- **Status**: ready OR blocked
- **Labels**: comma, separated, labels
- **Depends on**: task-X, task-Y (if blocked)
- **Description**: Detailed description of what needs to be done

## Questions
List any questions that need to be resolved before implementation:
1. Question one?
2. Question two?

## Blockers
List any blockers that prevent progress:
- Blocker description

Do not include any other text or explanation. Only output the structured task breakdown.
`)

	return sb.String()
}

// readProjectSource reads content from various source types.
func (c *Conductor) readProjectSource(ctx context.Context, source string) (string, error) {
	// Parse source type
	if strings.HasPrefix(source, "dir:") {
		return c.readDirectorySource(source[4:])
	}
	if strings.HasPrefix(source, "file:") {
		return c.readFileSource(source[5:])
	}
	// Provider reference (github:123, jira:PROJ-123, etc.)
	return c.readProviderSource(ctx, source)
}

// readDirectorySource reads all relevant files from a directory.
func (c *Conductor) readDirectorySource(dirPath string) (string, error) {
	var content strings.Builder

	// Walk the directory and collect relevant files
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and hidden files
		if info.IsDir() || strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		// Read text files (markdown, txt, yaml, json, etc.)
		ext := strings.ToLower(filepath.Ext(path))
		textExts := map[string]bool{
			".md": true, ".txt": true, ".yaml": true, ".yml": true,
			".json": true, ".xml": true, ".html": true, ".css": true,
			".js": true, ".ts": true, ".go": true, ".py": true,
			".java": true, ".rs": true, ".rb": true, ".sh": true,
		}

		if !textExts[ext] {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil //nolint:nilerr // Skip unreadable files intentionally
		}

		relPath, _ := filepath.Rel(dirPath, path)
		content.WriteString(fmt.Sprintf("\n--- %s ---\n", relPath))
		content.Write(data)
		content.WriteString("\n")

		return nil
	})
	if err != nil {
		return "", fmt.Errorf("walk directory: %w", err)
	}

	return content.String(), nil
}

// readFileSource reads content from a single file.
func (c *Conductor) readFileSource(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	return string(data), nil
}

// readProviderSource fetches content from a provider reference.
func (c *Conductor) readProviderSource(ctx context.Context, reference string) (string, error) {
	// Parse provider:id format
	parts := strings.SplitN(reference, ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid reference format: %s (expected provider:id)", reference)
	}

	providerName := parts[0]
	taskID := parts[1]

	// Get the provider factory from registry
	_, factory, ok := c.providers.Get(providerName)
	if !ok {
		return "", fmt.Errorf("provider not found: %s", providerName)
	}

	// Create provider instance
	providerCfg := providerconfig.NewConfig()
	instance, err := factory(ctx, providerCfg)
	if err != nil {
		return "", fmt.Errorf("create provider: %w", err)
	}

	// Check if provider implements Reader interface
	reader, ok := instance.(workunit.Reader)
	if !ok {
		return "", fmt.Errorf("provider %s does not support fetching work units", providerName)
	}

	// Fetch the work unit
	workUnit, err := reader.Fetch(ctx, taskID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch from %s: %w", providerName, err)
	}

	// Format as planning input
	return formatWorkUnitAsSource(workUnit), nil
}

// formatWorkUnitAsSource converts a WorkUnit to a markdown-formatted planning source.
func formatWorkUnitAsSource(wu *workunit.WorkUnit) string {
	var sb strings.Builder

	sb.WriteString("# " + wu.Title + "\n\n")

	if wu.Description != "" {
		sb.WriteString(wu.Description + "\n\n")
	}

	if len(wu.Labels) > 0 {
		sb.WriteString("**Labels:** " + strings.Join(wu.Labels, ", ") + "\n")
	}

	if wu.Priority != 0 {
		sb.WriteString("**Priority:** " + wu.Priority.String() + "\n")
	}

	if wu.Status != "" {
		sb.WriteString("**Status:** " + string(wu.Status) + "\n")
	}

	if len(wu.Assignees) > 0 {
		var assigneeNames []string
		for _, a := range wu.Assignees {
			if a.Name != "" {
				assigneeNames = append(assigneeNames, a.Name)
			} else if a.ID != "" {
				assigneeNames = append(assigneeNames, a.ID)
			}
		}
		if len(assigneeNames) > 0 {
			sb.WriteString("**Assignees:** " + strings.Join(assigneeNames, ", ") + "\n")
		}
	}

	return sb.String()
}

// generateQueueID creates a queue ID from title or source.
func generateQueueID(title, source string) string {
	// Use title if provided
	base := title
	if base == "" {
		// Extract from source
		if strings.HasPrefix(source, "dir:") {
			base = filepath.Base(source[4:])
		} else if strings.HasPrefix(source, "file:") {
			base = strings.TrimSuffix(filepath.Base(source[5:]), filepath.Ext(source[5:]))
		} else {
			base = strings.ReplaceAll(source, ":", "-")
		}
	}

	// Normalize: lowercase, replace spaces with dashes, remove special chars
	id := strings.ToLower(base)
	id = strings.ReplaceAll(id, " ", "-")
	id = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}

		return -1
	}, id)

	// Add timestamp suffix for uniqueness
	timestamp := time.Now().Format("20060102-150405")

	return fmt.Sprintf("%s-%s", id, timestamp)
}

// buildProjectPlanningPrompt creates the prompt for project task breakdown.
func buildProjectPlanningPrompt(title, sourceContent, customInstructions string) string {
	currentTime := time.Now().Format("2006-01-02 15:04")

	prompt := fmt.Sprintf(`You are an expert project manager and software architect.

CRITICAL: Your response MUST start with "## Tasks" and follow the EXACT format below.
Do NOT include any preamble, explanation, or conversational text.
Do NOT ask questions in prose - put them in the "## Questions" section.

## Project
%s

## Source Content
%s

Current timestamp: %s

`, title, sourceContent, currentTime)

	if customInstructions != "" {
		prompt += fmt.Sprintf(`## Custom Instructions
%s

`, customInstructions)
	}

	prompt += `## Required Output Format

Your response MUST begin with "## Tasks" and use EXACTLY this structure:

## Tasks

### task-1: First Task Title
- **Priority**: 1
- **Status**: ready
- **Labels**: backend, setup
- **Description**: What needs to be done

### task-2: Second Task Title
- **Priority**: 2
- **Status**: blocked
- **Depends on**: task-1
- **Labels**: backend
- **Description**: What needs to be done

(continue for all tasks...)

## Questions
1. Any clarifying question goes here
2. Another question

## Blockers
- Any external blockers go here

## Rules

1. ALWAYS output "## Tasks" first - no preamble or explanation
2. Each task: "### task-N: Title" format (N starts at 1)
3. Each task MUST have: Priority, Status, Labels, Description
4. Status is ONLY "ready" or "blocked"
5. Use "Depends on" for blocking dependencies
6. Use "Parent" for hierarchical grouping (subtasks)
7. Tasks should be 1-4 hours of work each
8. Put questions in "## Questions" section, not as prose
9. If no questions/blockers, omit those sections

BEGIN YOUR RESPONSE WITH "## Tasks" NOW:
`

	return prompt
}
