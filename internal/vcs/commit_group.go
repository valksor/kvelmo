package vcs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// ChangeAnalyzer analyzes changes and uses AI to determine logical groups.
type ChangeAnalyzer struct {
	git   *Git
	agent Agent // Agent interface for AI calls (injected for testing)
}

// Agent represents an AI agent that can process prompts.
type Agent interface {
	Run(ctx context.Context, prompt string) (*AgentResponse, error)
}

// AgentResponse is the response from an AI agent.
type AgentResponse struct {
	Messages []string
}

// NewChangeAnalyzer creates a new analyzer.
func NewChangeAnalyzer(git *Git) *ChangeAnalyzer {
	return &ChangeAnalyzer{git: git}
}

// SetAgent sets the AI agent for grouping (used in production).
func (a *ChangeAnalyzer) SetAgent(agent Agent) {
	a.agent = agent
}

// AnalyzeChanges uses AI to group uncommitted changes logically.
// This is GENERIC - works in ANY repo, no hardcoded patterns.
func (a *ChangeAnalyzer) AnalyzeChanges(ctx context.Context, includeUnstaged bool) ([]FileGroup, error) {
	// 1. Get file status - works in any repo
	status, err := a.git.Status(ctx)
	if err != nil {
		return nil, err
	}

	// 2. Filter and collect changed files
	var files []FileStatus
	for _, f := range status {
		if includeUnstaged {
			if f.Index != ' ' || f.WorkDir != ' ' {
				files = append(files, f)
			}
		} else {
			if f.Index != ' ' && f.Index != '?' {
				files = append(files, f)
			}
		}
	}

	if len(files) == 0 {
		return nil, nil
	}

	// 3. Get repo context for AI - GENERIC, works in any repo
	repoInfo, _ := a.git.GetRepoInfo(ctx)

	// 4. Use AI to group - NO hardcoded logic
	if a.agent != nil {
		return a.groupWithAI(ctx, files, repoInfo)
	}

	// Fallback: group by directory if no agent
	return a.groupByDirectory(files)
}

// FileGroup represents a group of files for commit message generation.
// This is the input type for GenerateCommitMessageForGroup.
type FileGroup struct {
	Files   []string `json:"files"`  // Files in this group
	Message string   `json:"-"`      // Generated commit message (not from AI)
	Reason  string   `json:"reason"` // Reason for the grouping (from AI)
}

// groupWithAI asks the agent to logically group files based on the actual repo context.
func (a *ChangeAnalyzer) groupWithAI(ctx context.Context, files []FileStatus, repoInfo RepoInfo) ([]FileGroup, error) {
	// Build prompt with REAL repo context
	prompt := a.buildGroupingPrompt(files, repoInfo)

	// Call agent to get groupings
	response, err := a.agent.Run(ctx, prompt)
	if err != nil {
		// Fallback: group by directory
		return a.groupByDirectory(files)
	}

	// Parse AI response into groups
	return a.parseGroupingResponse(response)
}

// buildGroupingPrompt creates a prompt that helps AI understand ANY repo.
func (a *ChangeAnalyzer) buildGroupingPrompt(files []FileStatus, repoInfo RepoInfo) string {
	var b strings.Builder

	// Repo context - AI figures out what kind of project this is
	b.WriteString("Analyze these file changes and group them into logical commits.\n\n")
	b.WriteString("Repository context:\n")
	if repoInfo.Language != "" {
		b.WriteString(fmt.Sprintf("  Language: %s\n", repoInfo.Language))
	}
	if len(repoInfo.RootDirs) > 0 {
		b.WriteString(fmt.Sprintf("  Root directories: %s\n", strings.Join(repoInfo.RootDirs, ", ")))
	}
	if len(repoInfo.BuildFiles) > 0 {
		b.WriteString(fmt.Sprintf("  Build files: %s\n", strings.Join(repoInfo.BuildFiles, ", ")))
	}
	b.WriteString("\n")

	// The files - AI figures out relationships from NAMES
	b.WriteString("Changed files:\n")
	for _, f := range files {
		status := formatFileStatus(f.Index, f.WorkDir)
		b.WriteString(fmt.Sprintf("  [%s] %s\n", status, f.Path))
	}

	b.WriteString(`
Group these files into logical commits. Return ONLY valid JSON:
[
  {"files": ["path1", "path2"], "reason": "why these go together"},
  ...
]

Rules:
- Group related changes together (same feature/bugfix/refactor)
- Keep tests with their source code (e.g., auth.go + auth_test.go together)
- Documentation changes that relate to code go with that code
- Standalone docs changes get their own group
- Config/dependency changes get their own group
- 3-10 groups typically, not 1 and not 50
- Use exact file paths as provided above
`)

	return b.String()
}

// formatFileStatus formats the git status characters.
func formatFileStatus(index, workDir byte) string {
	if index == '?' {
		return "?"
	}
	if workDir != ' ' {
		return "M"
	}
	if index == 'M' {
		return "M"
	}
	if index == 'A' {
		return "A"
	}
	if index == 'D' {
		return "D"
	}
	if index == 'R' {
		return "R"
	}
	if index == 'C' {
		return "C"
	}

	return string(index)
}

// parseGroupingResponse parses the AI response into groups.
func (a *ChangeAnalyzer) parseGroupingResponse(response *AgentResponse) ([]FileGroup, error) {
	if len(response.Messages) == 0 {
		return nil, errors.New("empty response from agent")
	}

	// Extract JSON from response (might be wrapped in markdown code blocks)
	content := response.Messages[0]
	content = strings.TrimSpace(content)

	// Remove markdown code blocks if present
	if strings.HasPrefix(content, "```") {
		parts := strings.SplitN(content, "\n", 2)
		if len(parts) > 1 {
			content = parts[1]
			// Remove closing code block
			if idx := strings.Index(content, "```"); idx != -1 {
				content = content[:idx]
			}
			content = strings.TrimSpace(content)
		}
	}

	var groups []FileGroup
	if err := json.Unmarshal([]byte(content), &groups); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if len(groups) == 0 {
		return nil, errors.New("no groups in response")
	}

	return groups, nil
}

// groupByDirectory is a fallback that groups by directory.
func (a *ChangeAnalyzer) groupByDirectory(files []FileStatus) ([]FileGroup, error) {
	dirMap := make(map[string][]string)

	for _, f := range files {
		dir := filepathDir(f.Path)
		dirMap[dir] = append(dirMap[dir], f.Path)
	}

	var groups []FileGroup
	for dir, files := range dirMap {
		groups = append(groups, FileGroup{
			Files:  files,
			Reason: "Changes in " + dir,
		})
	}

	return groups, nil
}

// filepathDir returns the directory part of a path.
// Similar to filepath.Dir but handles root directory specially.
func filepathDir(path string) string {
	dir := filepath.Dir(path)
	if dir == "." {
		return "(root)"
	}
	if dir == "/" {
		return "(root)"
	}
	// Get just the last directory component for grouping
	parts := strings.Split(path, string(filepath.Separator))
	if len(parts) > 1 {
		// For deeper paths, use first two components
		if len(parts) > 2 {
			return strings.Join(parts[:2], string(filepath.Separator))
		}

		return dir
	}

	return dir
}
