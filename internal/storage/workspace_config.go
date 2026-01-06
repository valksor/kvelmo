package storage

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// WorkspaceConfig holds workspace-specific configuration that users can customize.
type WorkspaceConfig struct {
	Git         GitSettings                 `yaml:"git"`
	Agent       AgentSettings               `yaml:"agent"`
	Workflow    WorkflowSettings            `yaml:"workflow"`
	Providers   ProvidersSettings           `yaml:"providers,omitempty"`
	Env         map[string]string           `yaml:"env,omitempty"`
	Agents      map[string]AgentAliasConfig `yaml:"agents,omitempty"`
	GitHub      *GitHubSettings             `yaml:"github,omitempty"`
	GitLab      *GitLabSettings             `yaml:"gitlab,omitempty"`
	Notion      *NotionSettings             `yaml:"notion,omitempty"`
	Jira        *JiraSettings               `yaml:"jira,omitempty"`
	Linear      *LinearSettings             `yaml:"linear,omitempty"`
	Wrike       *WrikeSettings              `yaml:"wrike,omitempty"`
	YouTrack    *YouTrackSettings           `yaml:"youtrack,omitempty"`
	Bitbucket   *BitbucketSettings          `yaml:"bitbucket,omitempty"`
	Asana       *AsanaSettings              `yaml:"asana,omitempty"`
	ClickUp     *ClickUpSettings            `yaml:"clickup,omitempty"`
	AzureDevOps *AzureDevOpsSettings        `yaml:"azure_devops,omitempty"`
	Trello      *TrelloSettings             `yaml:"trello,omitempty"`
	Plugins     PluginsConfig               `yaml:"plugins,omitempty"`
	Update      UpdateSettings              `yaml:"update,omitempty"`
	Storage     StorageSettings             `yaml:"storage,omitempty"`
	Browser     *BrowserSettings            `yaml:"browser,omitempty"`
}

// PluginsConfig holds plugin-related configuration.
type PluginsConfig struct {
	// Enabled lists the plugin names that should be loaded
	// Only plugins in this list will be activated
	Enabled []string `yaml:"enabled,omitempty"`

	// Config holds plugin-specific configuration keyed by plugin name
	// Each plugin receives its configuration during initialization
	Config map[string]map[string]any `yaml:"config,omitempty"`
}

// GitHubSettings holds GitHub provider configuration.
type GitHubSettings struct {
	Token         string                  `yaml:"token,omitempty"`          // GitHub token (env vars take priority)
	Owner         string                  `yaml:"owner,omitempty"`          // Repository owner (auto-detected from git remote)
	Repo          string                  `yaml:"repo,omitempty"`           // Repository name
	BranchPattern string                  `yaml:"branch_pattern,omitempty"` // Default: "issue/{key}-{slug}"
	CommitPrefix  string                  `yaml:"commit_prefix,omitempty"`  // Default: "[#{key}]"
	TargetBranch  string                  `yaml:"target_branch,omitempty"`  // Default: detected from repo
	DraftPR       bool                    `yaml:"draft_pr,omitempty"`       // Create PRs as draft
	Comments      *GitHubCommentsSettings `yaml:"comments,omitempty"`
}

// GitHubCommentsSettings controls automated GitHub issue commenting.
type GitHubCommentsSettings struct {
	Enabled         bool `yaml:"enabled"`           // Master switch (default: false)
	OnBranchCreated bool `yaml:"on_branch_created"` // Post when branch is created
	OnPlanDone      bool `yaml:"on_plan_done"`      // Post summary of planned implementation
	OnImplementDone bool `yaml:"on_implement_done"` // Post changelog with files changed
	OnPRCreated     bool `yaml:"on_pr_created"`     // Post PR link
}

// WrikeSettings holds Wrike provider configuration.
type WrikeSettings struct {
	Token  string `yaml:"token,omitempty"`  // Wrike API token (env vars take priority)
	Host   string `yaml:"host,omitempty"`   // API base URL override (default: https://www.wrike.com/api/v4)
	Folder string `yaml:"folder,omitempty"` // Default folder ID for task lookup
}

// GitLabSettings holds GitLab provider configuration.
type GitLabSettings struct {
	Token         string `yaml:"token,omitempty"`          // GitLab token (env vars take priority)
	Host          string `yaml:"host,omitempty"`           // GitLab host (default: https://gitlab.com)
	ProjectPath   string `yaml:"project_path,omitempty"`   // Default project path (e.g., group/project)
	BranchPattern string `yaml:"branch_pattern,omitempty"` // Default: "issue/{key}-{slug}"
	CommitPrefix  string `yaml:"commit_prefix,omitempty"`  // Default: "[#{key}]"
}

// NotionSettings holds Notion provider configuration.
type NotionSettings struct {
	Token               string `yaml:"token,omitempty"`                // Notion token (env vars take priority)
	DatabaseID          string `yaml:"database_id,omitempty"`          // Default database ID
	StatusProperty      string `yaml:"status_property,omitempty"`      // Property name for status (default: Status)
	DescriptionProperty string `yaml:"description_property,omitempty"` // Property name for description
	LabelsProperty      string `yaml:"labels_property,omitempty"`      // Property name for labels (default: Tags)
}

// JiraSettings holds Jira provider configuration.
type JiraSettings struct {
	Token   string `yaml:"token,omitempty"`    // Jira API token (env vars take priority)
	Email   string `yaml:"email,omitempty"`    // Email for Cloud auth
	BaseURL string `yaml:"base_url,omitempty"` // Base URL (optional, auto-detected)
	Project string `yaml:"project,omitempty"`  // Default project key
}

// LinearSettings holds Linear provider configuration.
type LinearSettings struct {
	Token string `yaml:"token,omitempty"` // Linear API key (env vars take priority)
	Team  string `yaml:"team,omitempty"`  // Default team key
}

// YouTrackSettings holds YouTrack provider configuration.
type YouTrackSettings struct {
	Token string `yaml:"token,omitempty"` // YouTrack token (env vars take priority)
	Host  string `yaml:"host,omitempty"`  // YouTrack host
}

// BitbucketSettings holds Bitbucket provider configuration.
type BitbucketSettings struct {
	Username          string `yaml:"username,omitempty"`            // Bitbucket username
	AppPassword       string `yaml:"app_password,omitempty"`        // Bitbucket app password (env vars take priority)
	Workspace         string `yaml:"workspace,omitempty"`           // Bitbucket workspace
	RepoSlug          string `yaml:"repo,omitempty"`                // Repository slug
	BranchPattern     string `yaml:"branch_pattern,omitempty"`      // Git branch template
	CommitPrefix      string `yaml:"commit_prefix,omitempty"`       // Commit message prefix
	TargetBranch      string `yaml:"target_branch,omitempty"`       // Target branch for PRs
	CloseSourceBranch bool   `yaml:"close_source_branch,omitempty"` // Delete source branch when PR is merged
}

// AsanaSettings holds Asana provider configuration.
type AsanaSettings struct {
	Token          string `yaml:"token,omitempty"`           // Asana token (env vars take priority)
	WorkspaceGID   string `yaml:"workspace_gid,omitempty"`   // Asana workspace GID
	DefaultProject string `yaml:"default_project,omitempty"` // Default project GID for list operations
	BranchPattern  string `yaml:"branch_pattern,omitempty"`  // Git branch template
	CommitPrefix   string `yaml:"commit_prefix,omitempty"`   // Commit message prefix
}

// ClickUpSettings holds ClickUp provider configuration.
type ClickUpSettings struct {
	Token         string `yaml:"token,omitempty"`          // ClickUp API token (env vars take priority)
	TeamID        string `yaml:"team_id,omitempty"`        // Team/Workspace ID
	DefaultList   string `yaml:"default_list,omitempty"`   // Default list ID for list operations
	BranchPattern string `yaml:"branch_pattern,omitempty"` // Git branch template
	CommitPrefix  string `yaml:"commit_prefix,omitempty"`  // Commit message prefix
}

// AzureDevOpsSettings holds Azure DevOps provider configuration.
type AzureDevOpsSettings struct {
	Token         string `yaml:"token,omitempty"`          // Azure DevOps PAT (env vars take priority)
	Organization  string `yaml:"organization,omitempty"`   // Azure DevOps organization
	Project       string `yaml:"project,omitempty"`        // Project name
	AreaPath      string `yaml:"area_path,omitempty"`      // Filter by area path
	IterationPath string `yaml:"iteration_path,omitempty"` // Filter by iteration
	RepoName      string `yaml:"repo_name,omitempty"`      // Default repository for PR creation
	TargetBranch  string `yaml:"target_branch,omitempty"`  // Default target branch for PRs
	BranchPattern string `yaml:"branch_pattern,omitempty"` // Git branch template
	CommitPrefix  string `yaml:"commit_prefix,omitempty"`  // Commit message prefix
}

// TrelloSettings holds Trello provider configuration.
type TrelloSettings struct {
	APIKey string `yaml:"api_key,omitempty"` // Trello API key (env vars take priority)
	Token  string `yaml:"token,omitempty"`   // Trello token (env vars take priority)
	Board  string `yaml:"board,omitempty"`   // Default board ID
}

// BrowserSettings holds browser automation configuration.
type BrowserSettings struct {
	Enabled       bool   `yaml:"enabled,omitempty"`        // Enable browser automation (default: false)
	Host          string `yaml:"host,omitempty"`           // CDP host (default: localhost)
	Port          int    `yaml:"port,omitempty"`           // CDP port: 0 = random (default), 9222 = existing Chrome
	Headless      bool   `yaml:"headless,omitempty"`       // Launch headless browser (default: false)
	Timeout       int    `yaml:"timeout,omitempty"`        // Operation timeout in seconds (default: 30)
	ScreenshotDir string `yaml:"screenshot_dir,omitempty"` // Directory for screenshots (default: .mehrhof/screenshots)
}

// AgentAliasConfig defines a user-defined agent alias that wraps an existing agent
// with custom environment variables and CLI arguments.
type AgentAliasConfig struct {
	Extends     string            `yaml:"extends"`               // Base agent name to wrap
	Description string            `yaml:"description,omitempty"` // Human-readable description
	Env         map[string]string `yaml:"env,omitempty"`         // Environment variables to pass
	Args        []string          `yaml:"args,omitempty"`        // CLI arguments to pass
}

// GitSettings holds git-related configuration.
type GitSettings struct {
	CommitPrefix  string `yaml:"commit_prefix"`
	BranchPattern string `yaml:"branch_pattern"`
	AutoCommit    bool   `yaml:"auto_commit"`
	SignCommits   bool   `yaml:"sign_commits"`
	StashOnStart  bool   `yaml:"stash_on_start"` // Auto-stash changes before creating task branch
	AutoPopStash  bool   `yaml:"auto_pop_stash"` // Auto-pop stash after branch creation (if stashed)
}

// StepAgentConfig holds agent configuration for a specific workflow step.
type StepAgentConfig struct {
	Name         string            `yaml:"name,omitempty"`         // Agent name or alias
	Env          map[string]string `yaml:"env,omitempty"`          // Step-specific env vars
	Args         []string          `yaml:"args,omitempty"`         // Step-specific CLI args
	Instructions string            `yaml:"instructions,omitempty"` // Custom instructions for this step
}

// AgentSettings holds agent-related configuration.
type AgentSettings struct {
	Default      string                     `yaml:"default"`
	Timeout      int                        `yaml:"timeout"`
	MaxRetries   int                        `yaml:"max_retries"`
	Instructions string                     `yaml:"instructions,omitempty"` // Global instructions for all steps
	Steps        map[string]StepAgentConfig `yaml:"steps,omitempty"`        // Per-step agent configuration
}

// WorkflowSettings holds workflow-related configuration.
type WorkflowSettings struct {
	AutoInit             bool `yaml:"auto_init"`
	SessionRetentionDays int  `yaml:"session_retention_days"`
	DeleteWorkOnFinish   bool `yaml:"delete_work_on_finish"`  // Delete work dirs on finish (default: false)
	DeleteWorkOnAbandon  bool `yaml:"delete_work_on_abandon"` // Delete work dirs on abandon (default: true)
}

// UpdateSettings holds update-related configuration.
type UpdateSettings struct {
	Enabled       bool `yaml:"enabled"`        // Enable automatic update checks
	CheckInterval int  `yaml:"check_interval"` // Hours between checks (default: 24)
}

// StorageSettings holds storage-related configuration.
type StorageSettings struct {
	HomeDir string `yaml:"home_dir,omitempty"` // Override for mehrhof home directory (default: ~/.mehrhof)
	WorkDir string `yaml:"work_dir,omitempty"` // Path to work directory (relative to project root)
}

// ProvidersSettings holds provider-related configuration.
type ProvidersSettings struct {
	Default string `yaml:"default,omitempty"` // Default provider for bare references (e.g., "file", "directory", "github")
}

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
		Providers: ProvidersSettings{
			Default: "file",
		},
		Update: UpdateSettings{
			Enabled:       true,
			CheckInterval: 24,
		},
		Storage: StorageSettings{
			WorkDir: "work", // Default: work/ (relative to global workspace location)
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

	// Add storage section comment if storage work_dir is default/empty
	if cfg.Storage.WorkDir == "" || cfg.Storage.WorkDir == "work" {
		content += `
# Storage settings
# Workspace is stored in: ~/.mehrhof/workspaces/<project-id>/
# Work directories are stored in: ~/.mehrhof/workspaces/<project-id>/work/
# You can customize the work directory path (relative to workspace):
# storage:
#     work_dir: work    # Default location
#     work_dir: tasks   # Alternative: store in tasks/ subdirectory
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

// expandEnvInAgentAliasConfig expands env vars in agent alias config.
func expandEnvInAgentAliasConfig(cfg AgentAliasConfig) AgentAliasConfig {
	return AgentAliasConfig{
		Extends:     expandEnvInString(cfg.Extends),
		Description: expandEnvInString(cfg.Description),
		Env:         expandEnvInMap(cfg.Env),
		Args:        expandEnvInStringSlice(cfg.Args),
	}
}

// expandEnvInGitHubSettings expands env vars in GitHub config.
func expandEnvInGitHubSettings(cfg *GitHubSettings) *GitHubSettings {
	if cfg == nil {
		return nil
	}
	result := *cfg // Copy
	result.Token = expandEnvInString(result.Token)
	result.Owner = expandEnvInString(result.Owner)
	result.Repo = expandEnvInString(result.Repo)
	result.BranchPattern = expandEnvInString(result.BranchPattern)
	result.CommitPrefix = expandEnvInString(result.CommitPrefix)
	result.TargetBranch = expandEnvInString(result.TargetBranch)

	return &result
}

// expandEnvInWrikeSettings expands env vars in Wrike config.
func expandEnvInWrikeSettings(cfg *WrikeSettings) *WrikeSettings {
	if cfg == nil {
		return nil
	}
	result := *cfg // Copy
	result.Token = expandEnvInString(result.Token)
	result.Host = expandEnvInString(result.Host)
	result.Folder = expandEnvInString(result.Folder)

	return &result
}

// expandEnvInGitLabSettings expands env vars in GitLab config.
func expandEnvInGitLabSettings(cfg *GitLabSettings) *GitLabSettings {
	if cfg == nil {
		return nil
	}
	result := *cfg // Copy
	result.Token = expandEnvInString(result.Token)
	result.Host = expandEnvInString(result.Host)
	result.ProjectPath = expandEnvInString(result.ProjectPath)
	result.BranchPattern = expandEnvInString(result.BranchPattern)
	result.CommitPrefix = expandEnvInString(result.CommitPrefix)

	return &result
}

// expandEnvInNotionSettings expands env vars in Notion config.
func expandEnvInNotionSettings(cfg *NotionSettings) *NotionSettings {
	if cfg == nil {
		return nil
	}
	result := *cfg // Copy
	result.Token = expandEnvInString(result.Token)
	result.DatabaseID = expandEnvInString(result.DatabaseID)
	result.StatusProperty = expandEnvInString(result.StatusProperty)
	result.DescriptionProperty = expandEnvInString(result.DescriptionProperty)
	result.LabelsProperty = expandEnvInString(result.LabelsProperty)

	return &result
}

// expandEnvInJiraSettings expands env vars in Jira config.
func expandEnvInJiraSettings(cfg *JiraSettings) *JiraSettings {
	if cfg == nil {
		return nil
	}
	result := *cfg // Copy
	result.Token = expandEnvInString(result.Token)
	result.Email = expandEnvInString(result.Email)
	result.BaseURL = expandEnvInString(result.BaseURL)
	result.Project = expandEnvInString(result.Project)

	return &result
}

// expandEnvInLinearSettings expands env vars in Linear config.
func expandEnvInLinearSettings(cfg *LinearSettings) *LinearSettings {
	if cfg == nil {
		return nil
	}
	result := *cfg // Copy
	result.Token = expandEnvInString(result.Token)
	result.Team = expandEnvInString(result.Team)

	return &result
}

// expandEnvInYouTrackSettings expands env vars in YouTrack config.
func expandEnvInYouTrackSettings(cfg *YouTrackSettings) *YouTrackSettings {
	if cfg == nil {
		return nil
	}
	result := *cfg // Copy
	result.Token = expandEnvInString(result.Token)
	result.Host = expandEnvInString(result.Host)

	return &result
}

// expandEnvInBitbucketSettings expands env vars in Bitbucket config.
func expandEnvInBitbucketSettings(cfg *BitbucketSettings) *BitbucketSettings {
	if cfg == nil {
		return nil
	}
	result := *cfg // Copy
	result.Username = expandEnvInString(result.Username)
	result.AppPassword = expandEnvInString(result.AppPassword)
	result.Workspace = expandEnvInString(result.Workspace)
	result.RepoSlug = expandEnvInString(result.RepoSlug)
	result.BranchPattern = expandEnvInString(result.BranchPattern)
	result.CommitPrefix = expandEnvInString(result.CommitPrefix)
	result.TargetBranch = expandEnvInString(result.TargetBranch)

	return &result
}

// expandEnvInAsanaSettings expands env vars in Asana config.
func expandEnvInAsanaSettings(cfg *AsanaSettings) *AsanaSettings {
	if cfg == nil {
		return nil
	}
	result := *cfg // Copy
	result.Token = expandEnvInString(result.Token)
	result.WorkspaceGID = expandEnvInString(result.WorkspaceGID)
	result.DefaultProject = expandEnvInString(result.DefaultProject)
	result.BranchPattern = expandEnvInString(result.BranchPattern)
	result.CommitPrefix = expandEnvInString(result.CommitPrefix)

	return &result
}

// expandEnvInClickUpSettings expands env vars in ClickUp config.
func expandEnvInClickUpSettings(cfg *ClickUpSettings) *ClickUpSettings {
	if cfg == nil {
		return nil
	}
	result := *cfg // Copy
	result.Token = expandEnvInString(result.Token)
	result.TeamID = expandEnvInString(result.TeamID)
	result.DefaultList = expandEnvInString(result.DefaultList)
	result.BranchPattern = expandEnvInString(result.BranchPattern)
	result.CommitPrefix = expandEnvInString(result.CommitPrefix)

	return &result
}

// expandEnvInAzureDevOpsSettings expands env vars in Azure DevOps config.
func expandEnvInAzureDevOpsSettings(cfg *AzureDevOpsSettings) *AzureDevOpsSettings {
	if cfg == nil {
		return nil
	}
	result := *cfg // Copy
	result.Token = expandEnvInString(result.Token)
	result.Organization = expandEnvInString(result.Organization)
	result.Project = expandEnvInString(result.Project)
	result.AreaPath = expandEnvInString(result.AreaPath)
	result.IterationPath = expandEnvInString(result.IterationPath)
	result.RepoName = expandEnvInString(result.RepoName)
	result.TargetBranch = expandEnvInString(result.TargetBranch)
	result.BranchPattern = expandEnvInString(result.BranchPattern)
	result.CommitPrefix = expandEnvInString(result.CommitPrefix)

	return &result
}

// expandEnvInTrelloSettings expands env vars in Trello config.
func expandEnvInTrelloSettings(cfg *TrelloSettings) *TrelloSettings {
	if cfg == nil {
		return nil
	}
	result := *cfg // Copy
	result.APIKey = expandEnvInString(result.APIKey)
	result.Token = expandEnvInString(result.Token)
	result.Board = expandEnvInString(result.Board)

	return &result
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
