package file

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-toolkit/naming"
	"github.com/valksor/go-toolkit/slug"
)

// ProviderName is the registered name for this provider.
const ProviderName = "file"

// Provider handles markdown file tasks.
type Provider struct {
	basePath string
}

// Info returns provider metadata.
func Info() provider.ProviderInfo {
	return provider.ProviderInfo{
		Name:        ProviderName,
		Description: "Local file task source",
		Schemes:     []string{"file"},
		Priority:    10,
		Capabilities: provider.CapabilitySet{
			provider.CapRead:     true,
			provider.CapSnapshot: true,
		},
	}
}

// New creates a file provider.
func New(ctx context.Context, cfg provider.Config) (any, error) {
	basePath := cfg.GetString("base_path")
	if basePath == "" {
		basePath = "."
	}

	return &Provider{basePath: basePath}, nil
}

// Match checks if input has the file: scheme prefix.
func (p *Provider) Match(input string) bool {
	return strings.HasPrefix(input, "file:")
}

// Parse extracts the file path from input.
func (p *Provider) Parse(input string) (string, error) {
	// Remove file: prefix if present
	path := strings.TrimPrefix(input, "file:")

	// Resolve to absolute path
	resolved := p.resolvePath(path)

	// Verify file exists
	info, err := os.Stat(resolved)
	if err != nil {
		return "", fmt.Errorf("file not found: %s", resolved)
	}
	if info.IsDir() {
		return "", fmt.Errorf("path is a directory, not a file: %s", resolved)
	}

	return resolved, nil
}

// Fetch reads the file and creates a WorkUnit.
func (p *Provider) Fetch(ctx context.Context, id string) (*provider.WorkUnit, error) {
	// Extract filename without extension for fallback title
	filename := strings.TrimSuffix(filepath.Base(id), filepath.Ext(id))

	parsed, err := ParseMarkdownFile(id, filename)
	if err != nil {
		return nil, fmt.Errorf("parse file: %w", err)
	}

	info, _ := os.Stat(id)
	modTime := time.Now()
	if info != nil {
		modTime = info.ModTime()
	}

	// Extract naming info from filename
	externalKey := naming.KeyFromFilename(filename)
	taskType := naming.TaskTypeFromFilename(filename)

	wu := &provider.WorkUnit{
		ID:          p.generateID(id),
		ExternalID:  id,
		Provider:    ProviderName,
		Title:       parsed.Title,
		Description: parsed.Body,
		Status:      provider.StatusOpen,
		Priority:    provider.PriorityNormal,
		Labels:      []string{},
		Metadata:    make(map[string]any),
		CreatedAt:   modTime,
		UpdatedAt:   modTime,
		Source: provider.SourceInfo{
			Type:      ProviderName,
			Reference: id,
			SyncedAt:  time.Now(),
		},
		// Naming fields
		ExternalKey: externalKey,
		TaskType:    taskType,
		Slug:        slug.Slugify(parsed.Title, 50),
	}

	// Extract attachment references from markdown
	wu.Attachments = ExtractAttachmentReferences(parsed.Body)

	// Apply frontmatter metadata if present
	if parsed.Frontmatter != nil {
		if parsed.Frontmatter.Status != "" {
			wu.Status = parseStatus(parsed.Frontmatter.Status)
		}
		if parsed.Frontmatter.Priority != "" {
			wu.Priority = parsePriority(parsed.Frontmatter.Priority)
		}
		if len(parsed.Frontmatter.Labels) > 0 {
			wu.Labels = parsed.Frontmatter.Labels
		}
		// Frontmatter overrides filename-derived naming
		if parsed.Frontmatter.Key != "" {
			wu.ExternalKey = parsed.Frontmatter.Key
		}
		if parsed.Frontmatter.Type != "" {
			wu.TaskType = parsed.Frontmatter.Type
		}
		if parsed.Frontmatter.Slug != "" {
			wu.Slug = parsed.Frontmatter.Slug
		}
		// Agent configuration from frontmatter
		if parsed.Frontmatter.Agent != "" || len(parsed.Frontmatter.AgentEnv) > 0 || len(parsed.Frontmatter.AgentArgs) > 0 || len(parsed.Frontmatter.AgentSteps) > 0 {
			wu.AgentConfig = &provider.AgentConfig{
				Name: parsed.Frontmatter.Agent,
				Env:  parsed.Frontmatter.AgentEnv,
				Args: parsed.Frontmatter.AgentArgs,
			}
			// Map per-step agent configuration
			if len(parsed.Frontmatter.AgentSteps) > 0 {
				wu.AgentConfig.Steps = make(map[string]provider.StepAgentConfig)
				for step, stepCfg := range parsed.Frontmatter.AgentSteps {
					wu.AgentConfig.Steps[step] = provider.StepAgentConfig{
						Name: stepCfg.Agent,
						Env:  stepCfg.Env,
						Args: stepCfg.Args,
					}
				}
			}
		}
		// Budget configuration from frontmatter
		if parsed.Frontmatter.Budget != nil {
			wu.Budget = &provider.BudgetConfig{
				MaxTokens: parsed.Frontmatter.Budget.MaxTokens,
				MaxCost:   parsed.Frontmatter.Budget.MaxCost,
				Currency:  parsed.Frontmatter.Budget.Currency,
				OnLimit:   parsed.Frontmatter.Budget.OnLimit,
				WarningAt: parsed.Frontmatter.Budget.WarningAt,
			}
		}
		// Preserve extra frontmatter fields as metadata
		for k, v := range parsed.Frontmatter.Extra {
			wu.Metadata[k] = v
		}
	}

	return wu, nil
}

func (p *Provider) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}

	return filepath.Join(p.basePath, path)
}

func (p *Provider) generateID(path string) string {
	// Use filename without extension as ID
	base := filepath.Base(path)
	ext := filepath.Ext(base)

	return strings.TrimSuffix(base, ext)
}

func parsePriority(s string) provider.Priority {
	switch strings.ToLower(s) {
	case "critical", "urgent":
		return provider.PriorityCritical
	case "high":
		return provider.PriorityHigh
	case "low":
		return provider.PriorityLow
	default:
		return provider.PriorityNormal
	}
}

func parseStatus(s string) provider.Status {
	// Normalize: replace hyphens with underscores, split on whitespace, join with underscores, lowercase
	// Handles all Unicode whitespace (spaces, tabs, non-breaking spaces, etc.)
	normalized := strings.ReplaceAll(s, "-", "_")
	fields := strings.Fields(normalized)
	normalized = strings.ToLower(strings.Join(fields, "_"))

	switch normalized {
	case "open", "todo", "backlog":
		return provider.StatusOpen
	case "in_progress", "inprogress", "doing", "active":
		return provider.StatusInProgress
	case "review", "in_review", "inreview", "code_review", "codereview":
		return provider.StatusReview
	case "done", "closed", "complete", "completed", "finished":
		return provider.StatusDone
	default:
		// Log warning for unrecognized status values - this helps catch typos in frontmatter
		// Default to StatusOpen for backward compatibility
		if s != "" {
			slog.Warn("Unrecognized status value, defaulting to 'open'", "value", s, "normalized", normalized)
		}

		return provider.StatusOpen
	}
}

// Snapshot captures the file content for storage.
func (p *Provider) Snapshot(ctx context.Context, id string) (*provider.Snapshot, error) {
	content, err := os.ReadFile(id)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	return &provider.Snapshot{
		Type:    "file",
		Ref:     id,
		Content: string(content),
	}, nil
}

// Register adds file provider to registry.
func Register(r *provider.Registry) {
	_ = r.Register(Info(), New)
}
