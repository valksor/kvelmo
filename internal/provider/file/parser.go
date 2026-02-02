package file

import (
	"cmp"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// StepAgentFrontmatter holds step-specific agent config in frontmatter.
type StepAgentFrontmatter struct {
	Agent string            `yaml:"agent,omitempty"` // Agent name or alias
	Env   map[string]string `yaml:"env,omitempty"`   // Step-specific env vars
	Args  []string          `yaml:"args,omitempty"`  // Step-specific CLI args
}

// BudgetFrontmatter holds per-task budget configuration in frontmatter.
type BudgetFrontmatter struct {
	MaxTokens int     `yaml:"max_tokens,omitempty"`
	MaxCost   float64 `yaml:"max_cost,omitempty"`
	Currency  string  `yaml:"currency,omitempty"`
	OnLimit   string  `yaml:"on_limit,omitempty"`   // warn | pause | stop
	WarningAt float64 `yaml:"warning_at,omitempty"` // 0-1 (e.g., 0.8)
}

// Frontmatter represents YAML frontmatter in markdown.
type Frontmatter struct {
	Title       string   `yaml:"title"`
	Description string   `yaml:"description"`
	Status      string   `yaml:"status,omitempty"` // Task status (e.g., "open", "in_progress", "done")
	Priority    string   `yaml:"priority"`
	Labels      []string `yaml:"labels"`
	Assignees   []string `yaml:"assignees"`

	// Naming overrides for branch/commit customization
	Key  string `yaml:"key"`  // External key override (e.g., "FEATURE-123")
	Type string `yaml:"type"` // Task type override (e.g., "feature", "fix")
	Slug string `yaml:"slug"` // Branch slug override (e.g., "custom-slug")

	// Agent configuration
	Agent      string                          `yaml:"agent,omitempty"`       // Agent name or alias (e.g., "glm", "claude")
	AgentEnv   map[string]string               `yaml:"agent_env,omitempty"`   // Inline environment variables
	AgentArgs  []string                        `yaml:"agent_args,omitempty"`  // CLI arguments
	AgentSteps map[string]StepAgentFrontmatter `yaml:"agent_steps,omitempty"` // Per-step agent overrides

	// Budget configuration
	Budget *BudgetFrontmatter `yaml:"budget,omitempty"`

	// Extra holds custom frontmatter fields not mapped to struct fields.
	// Populated by a second parse pass to preserve arbitrary user metadata.
	Extra map[string]any `yaml:"-"`
}

// ParsedMarkdown contains parsed markdown file content.
type ParsedMarkdown struct {
	Frontmatter *Frontmatter
	Title       string // From first # heading if no frontmatter title
	Body        string // Everything after frontmatter/title
	Raw         string // Full file content
}

// ParseMarkdownFile reads and parses a markdown file.
func ParseMarkdownFile(path, fallbackTitle string) (*ParsedMarkdown, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return ParseMarkdown(string(content), fallbackTitle)
}

// ParseMarkdown parses markdown content.
func ParseMarkdown(content, fallbackTitle string) (*ParsedMarkdown, error) {
	result := &ParsedMarkdown{Raw: content}

	// Check for YAML frontmatter (--- delimited)
	if strings.HasPrefix(content, "---\n") {
		before, after, found := strings.Cut(content[4:], "\n---")
		if found {
			var fm Frontmatter
			if err := yaml.Unmarshal([]byte(before), &fm); err == nil {
				// Second pass: capture all fields as map to find extras
				var allFields map[string]any
				if err := yaml.Unmarshal([]byte(before), &allFields); err == nil {
					fm.Extra = extractExtraFrontmatter(allFields)
				}
				result.Frontmatter = &fm
				content = strings.TrimPrefix(after, "\n")
			}
		}
	}

	// Extract title from first # heading
	lines := strings.Split(content, "\n")
	bodyStart := 0
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			result.Title = strings.TrimPrefix(trimmed, "# ")
			bodyStart = i + 1

			break
		}
		// Stop looking if we hit non-empty, non-heading content
		if trimmed != "" && !strings.HasPrefix(trimmed, "---") {
			break
		}
	}

	// Use frontmatter title if available
	if result.Frontmatter != nil && result.Frontmatter.Title != "" {
		result.Title = result.Frontmatter.Title
	}

	// Fallback to provided title (filename) if no title found
	result.Title = cmp.Or(result.Title, fallbackTitle)

	// Body is everything after title
	if bodyStart > 0 && bodyStart < len(lines) {
		result.Body = strings.TrimSpace(strings.Join(lines[bodyStart:], "\n"))
	} else {
		result.Body = strings.TrimSpace(content)
	}

	return result, nil
}

// knownFrontmatterKeys lists all YAML keys that map to typed Frontmatter struct fields.
var knownFrontmatterKeys = map[string]bool{
	"title": true, "description": true, "status": true, "priority": true,
	"labels": true, "assignees": true, "key": true, "type": true, "slug": true,
	"agent": true, "agent_env": true, "agent_args": true, "agent_steps": true,
	"budget": true,
}

// extractExtraFrontmatter returns frontmatter fields not mapped to the typed struct.
func extractExtraFrontmatter(allFields map[string]any) map[string]any {
	extra := make(map[string]any)
	for k, v := range allFields {
		if !knownFrontmatterKeys[k] {
			extra[k] = v
		}
	}
	if len(extra) == 0 {
		return nil
	}

	return extra
}
