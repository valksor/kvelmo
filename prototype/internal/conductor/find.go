package conductor

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/valksor/go-mehrhof/internal/agent"
	"github.com/valksor/go-mehrhof/internal/storage"
)

// FindOptions contains options for the find operation.
type FindOptions struct {
	Query     string // Search query from user
	Path      string // Restrict search to this path (relative to working dir)
	Pattern   string // Glob pattern for files to search
	Context   int    // Lines of context to include (default: 3)
	Workspace *storage.Workspace
}

// FindResult represents a single search result.
type FindResult struct {
	File    string   `json:"file"`
	Line    int      `json:"line"`
	Snippet string   `json:"snippet"`
	Context []string `json:"context,omitempty"`
	Reason  string   `json:"reason,omitempty"`
}

// Find performs an AI-powered code search with focused results.
// Returns a channel that streams results as they are found.
func (c *Conductor) Find(ctx context.Context, opts FindOptions) (<-chan FindResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if opts.Query == "" {
		return nil, errors.New("query is required")
	}

	// Set defaults
	if opts.Context <= 0 {
		opts.Context = 3
	}

	// Get agent for search
	searchAgent, err := c.getAgentForFind(ctx)
	if err != nil {
		return nil, fmt.Errorf("get search agent: %w", err)
	}

	// Determine working directory
	workingDir := c.opts.WorkDir
	if c.git != nil {
		workingDir = c.git.Root()
	}

	// Build the focused search prompt
	prompt := buildFindPrompt(opts.Query, workingDir, c.workspace, opts)

	// Create result channel
	resultChan := make(chan FindResult, 10)

	// Start search in goroutine
	go func() {
		defer close(resultChan)

		var responseBuilder strings.Builder
		var response *agent.Response

		// Run agent with streaming callback
		response, err = searchAgent.RunWithCallback(ctx, prompt, func(event agent.Event) error {
			if event.Text != "" {
				responseBuilder.WriteString(event.Text)
			}

			return nil
		})
		if err != nil {
			// Send error as a special result
			resultChan <- FindResult{
				File:    "__error__",
				Snippet: fmt.Sprintf("Search failed: %v", err),
			}

			return
		}

		// Parse the agent's response to extract results
		results := parseFindResults(responseBuilder.String())
		for _, result := range results {
			select {
			case resultChan <- result:
			case <-ctx.Done():
				return
			}
		}

		// Record usage stats if available
		if response != nil && response.Usage != nil && c.workspace != nil {
			// Try to get task ID for recording, but don't fail if none exists
			var taskID string
			if c.activeTask != nil {
				taskID = c.activeTask.ID
			}
			if taskID != "" {
				_ = c.workspace.AddUsage(taskID, "find",
					response.Usage.InputTokens,
					response.Usage.OutputTokens,
					response.Usage.CachedTokens,
					response.Usage.CostUSD,
				)
			}
		}
	}()

	return resultChan, nil
}

// getAgentForFind returns the agent to use for find operations.
// Uses step-specific agent for "finding" if configured, otherwise uses default agent.
func (c *Conductor) getAgentForFind(ctx context.Context) (agent.Agent, error) {
	// Try to get agent for "finding" step first
	stepAgent, err := c.GetAgentForStep(ctx, "finding")
	if err == nil && stepAgent != nil {
		return stepAgent, nil
	}

	// Fall back to default agent
	if c.activeAgent != nil {
		return c.activeAgent, nil
	}

	// Last resort: get default agent
	return c.agents.GetDefault()
}

// parseFindResults parses the agent's response to extract FindResults.
// Looks for the structured output format:
// --- FIND ---
// file: <path>
// line: <number>
// ...
// --- END ---.
func parseFindResults(response string) []FindResult {
	var results []FindResult

	// Pattern to match result blocks
	findPattern := regexp.MustCompile(`---\s*FIND\s*---([\s\S]*?)---\s*END\s*---`)
	matches := findPattern.FindAllStringSubmatch(response, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		block := strings.TrimSpace(match[1])
		result := parseFindResultBlock(block)
		if result.File != "" && result.File != "__error__" {
			results = append(results, result)
		}
	}

	// If no structured results found, try to extract file:line patterns
	// as a fallback for more conversational responses
	if len(results) == 0 {
		results = extractFallbackResults(response)
	}

	return results
}

// parseFindResultBlock parses a single result block into a FindResult.
func parseFindResultBlock(block string) FindResult {
	result := FindResult{}

	scanner := bufio.NewScanner(strings.NewReader(block))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "file:") {
			result.File = strings.TrimSpace(strings.TrimPrefix(line, "file:"))
		} else if strings.HasPrefix(line, "line:") {
			lineStr := strings.TrimSpace(strings.TrimPrefix(line, "line:"))
			lineNum, err := strconv.Atoi(lineStr)
			if err == nil {
				result.Line = lineNum
			}
		} else if strings.HasPrefix(line, "snippet:") {
			result.Snippet = strings.TrimSpace(strings.TrimPrefix(line, "snippet:"))
		} else if strings.HasPrefix(line, "reason:") {
			result.Reason = strings.TrimSpace(strings.TrimPrefix(line, "reason:"))
		} else if strings.HasPrefix(line, "context:") {
			ctx := strings.TrimSpace(strings.TrimPrefix(line, "context:"))
			if ctx != "" {
				result.Context = strings.Split(ctx, "\n")
			}
		}
	}

	return result
}

// extractFallbackResults extracts results from conversational responses.
// Looks for patterns like "path/to/file.go:42" or "path/to/file.go:42: snippet".
func extractFallbackResults(response string) []FindResult {
	var results []FindResult

	// Pattern for file:line or file:line: snippet
	// This handles formats like:
	// - internal/file.go:42
	// - internal/file.go:42: function call
	linePattern := regexp.MustCompile(`([^\s:]+\.[a-z]+):(\d+)(?::\s*)?(.*)?`)

	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		matches := linePattern.FindStringSubmatch(line)
		if len(matches) >= 3 {
			result := FindResult{
				File:    matches[1],
				Snippet: strings.TrimSpace(matches[3]),
			}
			lineNum, err := strconv.Atoi(matches[2])
			if err == nil {
				result.Line = lineNum
			}
			results = append(results, result)
		}
	}

	return results
}

// FindInFiles performs a local file-based search (fallback when agent unavailable).
// Uses grep patterns to search files directly without AI.
func (c *Conductor) FindInFiles(ctx context.Context, opts FindOptions) ([]FindResult, error) {
	if opts.Query == "" {
		return nil, errors.New("query is required")
	}

	// Determine search path
	searchPath := c.opts.WorkDir
	if c.git != nil {
		searchPath = c.git.Root()
	}
	if opts.Path != "" {
		searchPath = filepath.Join(searchPath, opts.Path)
	}

	var results []FindResult
	var mu sync.Mutex

	// Walk the directory tree
	walkErr := filepath.WalkDir(searchPath, func(path string, d os.DirEntry, walkErr error) error {
		// For individual file/directory errors, continue walking other entries
		// We intentionally ignore errors on individual files during the walk
		_ = walkErr
		if d.IsDir() {
			// Skip common directories to ignore
			base := filepath.Base(path)
			if base == ".git" || base == "node_modules" || base == "vendor" || base == ".mehrhof" {
				return filepath.SkipDir
			}

			return nil
		}

		// Check file pattern if specified
		if opts.Pattern != "" {
			matched, matchErr := filepath.Match(opts.Pattern, filepath.Base(path))
			_ = matchErr // Pattern match errors result in no match (file skipped)
			if matchErr == nil && !matched {
				return nil
			}
		}

		// Search in file
		fileResults, searchErr := searchInFile(path, opts.Query, opts.Context, searchPath)
		_ = searchErr // File read errors result in skipping the file
		mu.Lock()
		results = append(results, fileResults...)
		mu.Unlock()

		return nil
	})

	return results, walkErr
}

// searchInFile searches for query string in a single file.
func searchInFile(filePath, query string, contextLines int, basePath string) ([]FindResult, error) {
	var results []FindResult

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()

	relPath, err := filepath.Rel(basePath, filePath)
	if err != nil {
		relPath = filePath
	}

	scanner := bufio.NewScanner(file)
	lineNum := 0
	var buffer []string

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		buffer = append(buffer, line)

		// Check if query matches (case-insensitive)
		if strings.Contains(strings.ToLower(line), strings.ToLower(query)) {
			// Extract context
			contextStart := findMax(0, len(buffer)-contextLines-1)

			var context []string
			if contextStart < len(buffer) {
				context = buffer[contextStart:]
			}

			results = append(results, FindResult{
				File:    relPath,
				Line:    lineNum,
				Snippet: strings.TrimSpace(line),
				Context: context,
				Reason:  "contains search term",
			})
		}

		// Keep buffer manageable
		if len(buffer) > contextLines*2 {
			buffer = buffer[len(buffer)-contextLines:]
		}
	}

	if err := scanner.Err(); err != nil {
		return results, err
	}

	return results, nil
}

// findMax returns the maximum of two integers.
// Named to avoid shadowing the built-in max function.
func findMax(a, b int) int {
	if a > b {
		return a
	}

	return b
}
