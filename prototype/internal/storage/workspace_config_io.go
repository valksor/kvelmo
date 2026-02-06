package storage

import (
	"fmt"
	"os"
	"reflect"

	"gopkg.in/yaml.v3"
)

// SaveConfig saves the workspace configuration to .mehrhof/config.yaml.
func (w *Workspace) SaveConfig(cfg *WorkspaceConfig) error {
	// Ensure .mehrhof directory exists
	if err := os.MkdirAll(w.taskRoot, 0o755); err != nil {
		return fmt.Errorf("create task directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	// Add header comment
	header := `# Task workspace configuration
# Edit this file to customize task behavior
# Run 'task init' to regenerate with defaults

`
	// Add env section comment if env is empty (to show users how to use it)
	content := header + string(data)
	if len(cfg.Env) == 0 {
		content += `
# Environment variables passed to agents (filtered by agent name prefix)
# Prefix is stripped when passed: CLAUDE_FOO=bar -> FOO=bar
# Example:
# env:
#     CLAUDE_ANTHROPIC_API_KEY: your-key # passed to claude as ANTHROPIC_API_KEY
`
	}

	// Add providers section comment if providers.default is empty
	if cfg.Providers.Default == "" {
		content += `
# Provider settings
# Set a default provider for bare task references (without scheme prefix)
# Example:
# providers:
#     default: file    # "task.md" becomes "file:task.md"
`
	}

	// Add agents section comment if agents is empty
	if len(cfg.Agents) == 0 {
		content += `
# User-defined agent aliases
# Aliases wrap existing agents with custom environment variables and CLI arguments
# Use 'mehr agents list' to see all available agents
# Example:
# agents:
#     opus:
#         extends: claude                       # base agent to wrap
#         description: "Claude Opus model"      # shown in 'mehr agents list'
#         args: ["--model", "claude-opus-4-20250514"]  # CLI flags to pass
#     claude-fast:
#         extends: claude
#         description: "Claude with limited turns"
#         args: ["--max-turns", "3"]
#     glm:
#         extends: claude
#         description: "Claude with GLM key"
#         env:
#             ANTHROPIC_API_KEY: "${GLM_API_KEY}" # ${VAR} references system env
`
	}

	// Add plugins section comment if plugins is empty
	if len(cfg.Plugins.Enabled) == 0 {
		content += `
# Plugin configuration
# Plugins must be explicitly enabled to be loaded
# Use 'mehr plugins list' to see all discovered plugins
# Example:
# plugins:
#     enabled:
#         - jira                           # Enable the jira plugin
#         - youtrack                       # Enable the youtrack plugin
#     config:                              # Plugin-specific configuration
#         jira:
#             url: "https://company.atlassian.net"
#             project: "PROJ"
#         youtrack:
#             url: "https://youtrack.company.com"
`
	}

	// Add project section comment if code_dir is empty (default)
	if cfg.Project.CodeDir == "" {
		content += `
# Project settings
# Decouple the project hub (tasks, specs, config) from the code target directory
# Useful when tasks/research live separately from the implementation codebase
# Example:
# project:
#     code_dir: "../reporting-engine"   # Relative to project root, or absolute path
#     code_dir: "/workspace/my-code"    # Absolute path to code directory
`
	}

	// Add stack section comment if stack is nil or using defaults
	if cfg.Stack == nil || cfg.Stack.AutoRebase == "" || cfg.Stack.AutoRebase == "disabled" {
		content += `
# Stack settings
# Configure auto-rebase behavior for stacked feature branches
# Example:
# stack:
#     auto_rebase: disabled     # "disabled" (default) | "on_finish"
#     block_on_conflicts: true  # Block auto-rebase if conflicts detected (default: true)
`
	}

	// Add storage section comment if storage save_in_project is disabled (default)
	if !cfg.Storage.SaveInProject {
		content += `
# Storage settings
# Control where work files (specs, reviews) are stored
# Example:
# storage:
#     save_in_project: false   # Default: global (~/.valksor/mehrhof/workspaces/<name>/work/<taskid>/)
#     save_in_project: true    # Project: .mehrhof/work/<taskid>/
#     project_dir: "tickets"   # Custom: tickets/<taskid>/
`
	}

	// Add workflow cleanup settings comment
	if !cfg.Workflow.DeleteWorkOnFinish && cfg.Workflow.DeleteWorkOnAbandon {
		content += `
# Workflow cleanup settings
# Control whether work directories are deleted when tasks finish/abandon
# Example:
# workflow:
#     delete_work_on_finish: false   # Keep work dirs after finish (default)
#     delete_work_on_abandon: true   # Delete work dirs on abandon (default)
`
	}

	// Add browser section comment if browser is nil or disabled
	if cfg.Browser == nil || !cfg.Browser.Enabled {
		content += `
# Browser automation settings
# Enable AI agent browser access for web-based tasks (login, testing, scraping)
# Example:
# browser:
#     enabled: true                  # Enable browser automation
#     headless: false                # Show browser window (false = visible, true = background)
#     port: 0                        # 0 = random isolated browser, 9222 = existing Chrome
#     timeout: 30                    # Operation timeout in seconds
#     screenshot_dir: ".mehrhof/screenshots"
#     cookie_profile: "default"      # Which cookie profile to use
#     cookie_auto_load: true         # Auto-load cookies on connect
#     cookie_auto_save: true         # Auto-save cookies on disconnect
`
	}

	// Add MCP section comment if mcp is nil or disabled
	if cfg.MCP == nil || !cfg.MCP.Enabled {
		content += `
# MCP (Model Context Protocol) server settings
# Allow AI agents to call Mehrhof commands via MCP protocol
# Example:
# mcp:
#     enabled: true                  # Enable MCP server
#     tools:                         # Optional: specific tools to expose (empty = all safe tools)
#         - mehr_status
#         - mehr_browser_goto
#     rate_limit:                    # Optional: rate limiting for tool calls
#         rate: 10                   # Requests per second (default: 10)
#         burst: 20                  # Burst size (default: 20)
`
	}

	// Add security section comment if security is nil or disabled
	if cfg.Security == nil || !cfg.Security.Enabled {
		content += `
# Security scanning settings
# Automatically scan code for vulnerabilities, secrets, and compliance issues
# Example:
# security:
#     enabled: true                  # Enable security scanning
#     run_on:
#         implementing: true         # Run after implementation
#         reviewing: true            # Run during review
#     fail_on:
#         level: critical            # Block on critical findings
#         block_finish: true         # Block task completion
#     scanners:
#         sast:
#             enabled: true
#         secrets:
#             enabled: true
#         dependencies:
#             enabled: true
#     output:
#         format: sarif              # Report format (sarif, json, text)
#         file: ".mehrhof/security-report.json"
`
	}

	// Add memory section comment if memory is nil or disabled
	if cfg.Memory == nil || !cfg.Memory.Enabled {
		content += `
# Memory system settings
# Enable semantic search and learning from past tasks
# Example:
# memory:
#     enabled: true                  # Enable memory system
#     vector_db:
#         backend: chromadb          # Vector database backend
#         connection_string: "./.mehrhof/vectors"  # Storage path
#         collection: "mehr_task_memory"  # Collection name
#         embedding_model: "default"   # Embedding model name
#     retention:
#         max_days: 90               # Keep documents for 90 days
#         max_tasks: 1000            # Keep max 1000 tasks
#     search:
#         similarity_threshold: 0.8  # Minimum similarity score
#         max_results: 5             # Max results to return
#         include_code: true         # Include code changes
#         include_specs: true        # Include specifications
#         include_sessions: true     # Include session logs
#     learning:
#         auto_store: true           # Automatically store task data
#         learn_from_corrections: true  # Learn from user corrections
#         suggest_similar: true      # Auto-suggest similar tasks
`
	}

	// Add orchestration section comment if orchestration is nil or disabled
	if cfg.Orchestration == nil || !cfg.Orchestration.Enabled {
		content += `
# Multi-agent orchestration settings
# Enable multiple agents to work together on workflow steps
# Example:
# orchestration:
#     enabled: true                  # Enable multi-agent orchestration
#     steps:
#         planning:
#             mode: sequential       # Execute agents in sequence
#             agents:
#                 - name: architect
#                   agent: claude
#                   role: "Design system architecture"
#                   output: "architecture.md"
#                 - name: security-analyst
#                   agent: claude
#                   role: "Review architecture for security"
#                   input: ["architecture.md"]
#         implementing:
#             mode: single           # Use single agent (default)
#         reviewing:
#             mode: consensus        # Use multiple agents and build consensus
#             agents:
#                 - name: code-reviewer
#                   agent: claude
#                   role: "Review code quality"
#                 - name: security-reviewer
#                   agent: claude
#                   role: "Review for security"
#             consensus:
#                 mode: majority      # Require majority agreement
#                 synthesizer: claude # Agent to synthesize results
`
	}

	// Add ML section comment if ML is nil or disabled
	if cfg.ML == nil || !cfg.ML.Enabled {
		content += `
# ML prediction system settings
# Enable machine learning predictions for workflow guidance
# Example:
# ml:
#     enabled: true                  # Enable ML predictions
#     telemetry:
#         enabled: true              # Collect telemetry data
#         anonymize: true            # Anonymize task IDs
#         storage: ".mehrhof/telemetry"  # Telemetry storage path
#     model:
#         type: heuristic            # Model type (heuristic, xgboost, neural)
#         retrain_interval: "7d"     # How often to retrain models
#         min_samples: 100           # Minimum samples for training
#     predictions:
#         next_action: true          # Predict next workflow action
#         duration: true             # Predict task duration
#         complexity: true           # Predict task complexity
#         risk_assessment: true      # Predict potential risks
`
	}

	// Add specification section comment if using default pattern
	if cfg.Specification.FilenamePattern == "" || cfg.Specification.FilenamePattern == "specification-{n}.md" {
		content += `
# Specification settings
# Customize specification filenames (location controlled by storage.save_in_project)
# Example:
# specification:
#     filename_pattern: "SPEC-{n}.md"  # Filename pattern (default: "specification-{n}.md")
`
	}

	// Add review section comment if using default pattern
	if cfg.Review.FilenamePattern == "" || cfg.Review.FilenamePattern == "review-{n}.txt" {
		content += `
# Review settings
# Customize review filenames (location controlled by storage.save_in_project)
# Example:
# review:
#     filename_pattern: "CODERABBIT-{n}.txt" # Filename pattern (default: "review-{n}.txt")
`
	}

	// Add sandbox section comment if sandbox is nil or disabled
	if cfg.Sandbox == nil || !cfg.Sandbox.Enabled {
		content += `
# Sandbox settings
# Isolate agent execution for security (Linux: user namespaces, macOS: sandbox-exec)
# Example:
# sandbox:
#     enabled: true                  # Enable sandboxing
#     network: true                  # Allow network access (required for LLM APIs)
#     tmp_dir: "/tmp/mehrhof-sandbox"  # Custom tmpfs mount path (optional)
#     tools:                         # Additional tool paths to allow (optional)
#         - /usr/local/bin/node
#         - /usr/local/bin/python3
`
	}

	// Add simplify section comment if simplify is empty
	if cfg.Workflow.Simplify.Instructions == "" {
		content += `
# Simplification settings
# Customize how the 'mehr simplify' command refines your work
# Example:
# workflow:
#     simplify:
#         instructions: |
#             Follow our project standards:
#             - Use descriptive names (no abbreviations)
#             - Keep functions under 50 lines
#             - Prefer composition over inheritance
`
	}

	// Add labels section comment if labels is nil or default
	if cfg.Labels == nil || (len(cfg.Labels.Defined) == 0 && len(cfg.Labels.Suggestions) == 0) {
		content += `
# Label settings
# Configure predefined labels and suggestions for task organization
# Example:
# labels:
#     enabled: true                  # Enable label system
#     defined:                       # Predefined labels with custom colors
#         - name: priority:critical
#           color: bg-red-100 text-red-800
#         - name: priority:high
#         - name: type:bug
#         - name: team:frontend
#     suggestions:                   # Suggested labels for autocomplete
#         - priority:critical
#         - priority:high
#         - priority:medium
#         - priority:low
#         - type:bug
#         - type:feature
#         - type:refactor
#         - type:docs
#         - type:test
#         - team:frontend
#         - team:backend
#         - team:devops
#         - status:blocked
#         - status:in-review
`
	}

	// Add quality section comment if quality is nil or default
	if cfg.Quality == nil || !cfg.Quality.Enabled || len(cfg.Quality.Linters) == 0 {
		content += `
# Quality and linter settings
# Configure which linters run during review phase
# Example:
# quality:
#     enabled: true                  # Enable quality checks (default: true)
#     linters:
#         golangci-lint:
#             enabled: true          # Run Go linter
#         eslint:
#             enabled: true          # Run JS/TS linter
#         ruff:
#             enabled: true          # Run Python linter
#         php-cs-fixer:
#             enabled: false         # Disable built-in PHP linter
#         phpstan:                   # Use custom linter instead
#             enabled: true
#             command: ["vendor/bin/phpstan", "analyse", "--error-format=json"]
#             extensions: [".php"]
`
	}

	// Add links section comment if links is nil or default
	if cfg.Links == nil || !cfg.Links.Enabled {
		content += `
# Links settings
# Enable Logseq-style bidirectional linking between specs, notes, and sessions
# Example:
# links:
#     enabled: true                  # Enable link system (default: true)
#     auto_index: true               # Automatically index on save (default: true)
#     case_sensitive: false          # Case-sensitive name matching (default: false)
#     max_context_length: 200        # Context characters for links (default: 200)
`
	}

	if err := os.WriteFile(w.ConfigPath(), []byte(content), 0o644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}

// expandEnvInString expands ${VAR} and $VAR environment variable references in a string.
func expandEnvInString(s string) string {
	if s == "" {
		return s
	}

	return os.ExpandEnv(s)
}

// expandEnvInMap recursively expands env vars in map[string]string.
func expandEnvInMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = expandEnvInString(v)
	}

	return result
}

// expandEnvInStringSlice expands env vars in []string.
func expandEnvInStringSlice(s []string) []string {
	if s == nil {
		return nil
	}
	result := make([]string, len(s))
	for i, v := range s {
		result[i] = expandEnvInString(v)
	}

	return result
}

// expandEnvInStruct uses reflection to expand environment variables in all string fields of a struct.
// It returns a new copy of the struct with expanded values. If cfg is nil, it returns nil.
func expandEnvInStruct[T any](cfg *T) *T {
	if cfg == nil {
		return nil
	}

	val := reflect.ValueOf(cfg).Elem()
	typ := val.Type()

	result := reflect.New(typ).Elem()
	for i := range val.NumField() {
		field := val.Field(i)
		resultField := result.Field(i)

		switch field.Kind() {
		case reflect.String:
			resultField.SetString(expandEnvInString(field.String()))
		case reflect.Struct:
			// Handle nested structs (like SecuritySettings.Output)
			if field.CanAddr() && field.Addr().IsValid() {
				// For structs, recursively expand their string fields
				nestedResult := reflect.New(field.Type()).Elem()
				for j := range field.NumField() {
					nestedField := field.Field(j)
					nestedResultField := nestedResult.Field(j)
					if nestedField.Kind() == reflect.String {
						nestedResultField.SetString(expandEnvInString(nestedField.String()))
					} else {
						nestedResultField.Set(nestedField)
					}
				}
				resultField.Set(nestedResult)
			} else {
				resultField.Set(field)
			}
		case reflect.Invalid, reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
			reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128, reflect.Array,
			reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer,
			reflect.Slice, reflect.UnsafePointer:
			// Unsupported types - just copy the value
			resultField.Set(field)
		}
	}

	// Type assertion is safe here because we created the result from the same type
	resultTyped, ok := result.Addr().Interface().(*T)
	if !ok {
		// This should never happen, but handle it gracefully
		return nil
	}

	return resultTyped
}

// expandEnvInAgentAliasConfig expands env vars in agent alias config.
func expandEnvInAgentAliasConfig(cfg AgentAliasConfig) AgentAliasConfig {
	return AgentAliasConfig{
		Extends:     expandEnvInString(cfg.Extends),
		Description: expandEnvInString(cfg.Description),
		Components:  cfg.Components, // Components list doesn't need env expansion
		Env:         expandEnvInMap(cfg.Env),
		Args:        expandEnvInStringSlice(cfg.Args),
	}
}

// expandEnvInGitHubSettings expands env vars in GitHub config.
func expandEnvInGitHubSettings(cfg *GitHubSettings) *GitHubSettings {
	return expandEnvInStruct(cfg)
}

// expandEnvInWrikeSettings expands env vars in Wrike config.
func expandEnvInWrikeSettings(cfg *WrikeSettings) *WrikeSettings {
	return expandEnvInStruct(cfg)
}

// expandEnvInGitLabSettings expands env vars in GitLab config.
func expandEnvInGitLabSettings(cfg *GitLabSettings) *GitLabSettings {
	return expandEnvInStruct(cfg)
}

// expandEnvInNotionSettings expands env vars in Notion config.
func expandEnvInNotionSettings(cfg *NotionSettings) *NotionSettings {
	return expandEnvInStruct(cfg)
}

// expandEnvInJiraSettings expands env vars in Jira config.
func expandEnvInJiraSettings(cfg *JiraSettings) *JiraSettings {
	return expandEnvInStruct(cfg)
}

// expandEnvInLinearSettings expands env vars in Linear config.
func expandEnvInLinearSettings(cfg *LinearSettings) *LinearSettings {
	return expandEnvInStruct(cfg)
}

// expandEnvInYouTrackSettings expands env vars in YouTrack config.
func expandEnvInYouTrackSettings(cfg *YouTrackSettings) *YouTrackSettings {
	return expandEnvInStruct(cfg)
}

// expandEnvInBitbucketSettings expands env vars in Bitbucket config.
func expandEnvInBitbucketSettings(cfg *BitbucketSettings) *BitbucketSettings {
	return expandEnvInStruct(cfg)
}

// expandEnvInAsanaSettings expands env vars in Asana config.
func expandEnvInAsanaSettings(cfg *AsanaSettings) *AsanaSettings {
	return expandEnvInStruct(cfg)
}

// expandEnvInClickUpSettings expands env vars in ClickUp config.
func expandEnvInClickUpSettings(cfg *ClickUpSettings) *ClickUpSettings {
	return expandEnvInStruct(cfg)
}

// expandEnvInAzureDevOpsSettings expands env vars in Azure DevOps config.
func expandEnvInAzureDevOpsSettings(cfg *AzureDevOpsSettings) *AzureDevOpsSettings {
	return expandEnvInStruct(cfg)
}

// expandEnvInTrelloSettings expands env vars in Trello config.
func expandEnvInTrelloSettings(cfg *TrelloSettings) *TrelloSettings {
	return expandEnvInStruct(cfg)
}

// expandEnvInSecuritySettings expands env vars in Security config.
func expandEnvInSecuritySettings(cfg *SecuritySettings) *SecuritySettings {
	return expandEnvInStruct(cfg)
}

// expandEnvInMemorySettings expands env vars in Memory config.
func expandEnvInMemorySettings(cfg *MemorySettings) *MemorySettings {
	result := expandEnvInStruct(cfg)
	if result != nil && result.VectorDB.ConnectionString == "" {
		result.VectorDB.ConnectionString = "./.mehrhof/vectors"
	}

	return result
}

// LoadConfig loads the workspace configuration from .mehrhof/config.yaml.
// Environment variable references like ${VAR} and $VAR are expanded in all string values.
func (w *Workspace) LoadConfig() (*WorkspaceConfig, error) {
	data, err := os.ReadFile(w.ConfigPath())
	if err != nil {
		if os.IsNotExist(err) {
			// Return defaults if config doesn't exist
			return NewDefaultWorkspaceConfig(), nil
		}

		return nil, fmt.Errorf("read config file: %w", err)
	}

	cfg := NewDefaultWorkspaceConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}

	// Expand environment variable references
	cfg.Env = expandEnvInMap(cfg.Env)

	// Expand provider settings
	cfg.GitHub = expandEnvInGitHubSettings(cfg.GitHub)
	cfg.GitLab = expandEnvInGitLabSettings(cfg.GitLab)
	cfg.Notion = expandEnvInNotionSettings(cfg.Notion)
	cfg.Jira = expandEnvInJiraSettings(cfg.Jira)
	cfg.Linear = expandEnvInLinearSettings(cfg.Linear)
	cfg.Wrike = expandEnvInWrikeSettings(cfg.Wrike)
	cfg.YouTrack = expandEnvInYouTrackSettings(cfg.YouTrack)
	cfg.Bitbucket = expandEnvInBitbucketSettings(cfg.Bitbucket)
	cfg.Asana = expandEnvInAsanaSettings(cfg.Asana)
	cfg.ClickUp = expandEnvInClickUpSettings(cfg.ClickUp)
	cfg.AzureDevOps = expandEnvInAzureDevOpsSettings(cfg.AzureDevOps)
	cfg.Trello = expandEnvInTrelloSettings(cfg.Trello)

	// Expand security settings
	cfg.Security = expandEnvInSecuritySettings(cfg.Security)

	// Expand memory settings
	cfg.Memory = expandEnvInMemorySettings(cfg.Memory)

	// Expand project settings
	cfg.Project.CodeDir = expandEnvInString(cfg.Project.CodeDir)

	// Expand agent aliases
	if cfg.Agents != nil {
		expanded := make(map[string]AgentAliasConfig, len(cfg.Agents))
		for k, v := range cfg.Agents {
			expanded[k] = expandEnvInAgentAliasConfig(v)
		}
		cfg.Agents = expanded
	}

	return cfg, nil
}
