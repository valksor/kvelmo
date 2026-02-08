package storage

import "strings"

// NewDefaultWorkspaceConfig creates a WorkspaceConfig with default values.
func NewDefaultWorkspaceConfig() *WorkspaceConfig {
	return &WorkspaceConfig{
		Git: GitSettings{
			AutoCommit:    true,
			CommitPrefix:  "[{key}]",
			BranchPattern: "{type}/{key}--{slug}",
			SignCommits:   false,
			StashOnStart:  false, // Default off, require explicit --stash or config
			AutoPopStash:  true,  // Default on for better UX when stashing
		},
		Agent: AgentSettings{
			Default:    "claude",
			Timeout:    300,
			MaxRetries: 3,
		},
		Workflow: WorkflowSettings{
			AutoInit:             true,
			SessionRetentionDays: 30,
			DeleteWorkOnFinish:   false, // Keep work dirs by default on finish
			DeleteWorkOnAbandon:  true,  // Delete work dirs by default on abandon
		},
		Budget: BudgetSettings{
			PerTask: BudgetConfig{
				MaxTokens: 100000,
				MaxCost:   10.00,
				Currency:  "USD",
				OnLimit:   "warn",
				WarningAt: 0.8,
			},
			Monthly: MonthlyBudgetSettings{
				MaxCost:   100.00,
				Currency:  "USD",
				WarningAt: 0.8,
			},
		},
		Providers: ProvidersSettings{
			Default: "file",
		},
		Update: UpdateSettings{
			Enabled:       true,
			CheckInterval: 24,
		},
		Storage: StorageSettings{
			SaveInProject: false, // Default: global storage (~/.valksor/mehrhof/workspaces/<name>/work/)
			ProjectDir:    "",    // Default: ".mehrhof/work" when save_in_project=true
		},
		Specification: SpecificationSettings{
			FilenamePattern: "specification-{n}.md", // Default: specification-1.md, specification-2.md, etc.
		},
		Review: ReviewSettings{
			FilenamePattern: "review-{n}.txt", // Default: review-1.txt, review-2.txt, etc.
		},
		Labels: &LabelSettings{
			Enabled: true,
			Defined: []LabelDefinition{
				{Name: "priority:critical"},
				{Name: "priority:high"},
				{Name: "priority:medium"},
				{Name: "priority:low"},
				{Name: "type:bug"},
				{Name: "type:feature"},
				{Name: "type:refactor"},
				{Name: "type:docs"},
				{Name: "type:test"},
				{Name: "team:frontend"},
				{Name: "team:backend"},
				{Name: "team:devops"},
				{Name: "status:blocked"},
				{Name: "status:in-review"},
			},
			Suggestions: []string{
				"priority:critical", "priority:high", "priority:medium", "priority:low",
				"type:bug", "type:feature", "type:refactor", "type:docs", "type:test",
				"team:frontend", "team:backend", "team:devops",
				"status:blocked", "status:in-review",
			},
		},
		Quality: &QualitySettings{
			Enabled:     true,
			UseDefaults: false, // Safer default: requires explicit linter configuration
		},
		Links: &LinksSettings{
			Enabled:          true,
			AutoIndex:        true,
			CaseSensitive:    false,
			MaxContextLength: 200,
		},
		Context: &ContextSettings{
			IncludeParent:    true,
			IncludeSiblings:  true,
			MaxSiblings:      5,
			DescriptionLimit: 500,
		},
		Stack: &StackSettings{
			AutoRebase:       "disabled", // Opt-in: "disabled" | "on_finish"
			BlockOnConflicts: true,       // Safe default: always block on conflicts
		},
		Display: &DisplaySettings{
			Timezone: "UTC",
		},
		Env: make(map[string]string),
	}
}

// GetEnvForAgent returns env vars for a specific agent, stripping the prefix.
// E.g., for agent "claude": CLAUDE_FOO=bar → FOO=bar.
func (cfg *WorkspaceConfig) GetEnvForAgent(agentName string) map[string]string {
	prefix := strings.ToUpper(agentName) + "_"
	result := make(map[string]string)
	for k, v := range cfg.Env {
		if strings.HasPrefix(k, prefix) {
			stripped := strings.TrimPrefix(k, prefix)
			result[stripped] = v
		}
	}

	return result
}
