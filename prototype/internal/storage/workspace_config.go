package storage

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

// WorkspaceConfig holds workspace-specific configuration that users can customize.
type WorkspaceConfig struct {
	Git           GitSettings                 `yaml:"git" json:"git"`
	Agent         AgentSettings               `yaml:"agent" json:"agent"`
	Workflow      WorkflowSettings            `yaml:"workflow" json:"workflow"`
	Budget        BudgetSettings              `yaml:"budget,omitempty" json:"budget,omitempty"`
	Providers     ProvidersSettings           `yaml:"providers,omitempty" json:"providers,omitempty"`
	Env           map[string]string           `yaml:"env,omitempty" json:"env,omitempty"`
	Agents        map[string]AgentAliasConfig `yaml:"agents,omitempty" json:"agents,omitempty"`
	GitHub        *GitHubSettings             `yaml:"github,omitempty" json:"github,omitempty"`
	GitLab        *GitLabSettings             `yaml:"gitlab,omitempty" json:"gitlab,omitempty"`
	Notion        *NotionSettings             `yaml:"notion,omitempty" json:"notion,omitempty"`
	Jira          *JiraSettings               `yaml:"jira,omitempty" json:"jira,omitempty"`
	Linear        *LinearSettings             `yaml:"linear,omitempty" json:"linear,omitempty"`
	Wrike         *WrikeSettings              `yaml:"wrike,omitempty" json:"wrike,omitempty"`
	YouTrack      *YouTrackSettings           `yaml:"youtrack,omitempty" json:"youtrack,omitempty"`
	Bitbucket     *BitbucketSettings          `yaml:"bitbucket,omitempty" json:"bitbucket,omitempty"`
	Asana         *AsanaSettings              `yaml:"asana,omitempty" json:"asana,omitempty"`
	ClickUp       *ClickUpSettings            `yaml:"clickup,omitempty" json:"clickup,omitempty"`
	AzureDevOps   *AzureDevOpsSettings        `yaml:"azure_devops,omitempty" json:"azure_devops,omitempty"`
	Trello        *TrelloSettings             `yaml:"trello,omitempty" json:"trello,omitempty"`
	Plugins       PluginsConfig               `yaml:"plugins,omitempty" json:"plugins,omitempty"`
	Update        UpdateSettings              `yaml:"update,omitempty" json:"update,omitempty"`
	Storage       StorageSettings             `yaml:"storage,omitempty" json:"storage,omitempty"`
	Browser       *BrowserSettings            `yaml:"browser,omitempty" json:"browser,omitempty"`
	MCP           *MCPSettings                `yaml:"mcp,omitempty" json:"mcp,omitempty"`
	Specification SpecificationSettings       `yaml:"specification,omitempty" json:"specification,omitempty"`
	Review        ReviewSettings              `yaml:"review,omitempty" json:"review,omitempty"`
	Security      *SecuritySettings           `yaml:"security,omitempty" json:"security,omitempty"`
	Memory        *MemorySettings             `yaml:"memory,omitempty" json:"memory,omitempty"`
	Library       *LibrarySettings            `yaml:"library,omitempty" json:"library,omitempty"`
	Orchestration *OrchestrationSettings      `yaml:"orchestration,omitempty" json:"orchestration,omitempty"`
	ML            *MLSettings                 `yaml:"ml,omitempty" json:"ml,omitempty"`
	Sandbox       *SandboxSettings            `yaml:"sandbox,omitempty" json:"sandbox,omitempty"`
	Labels        *LabelSettings              `yaml:"labels,omitempty" json:"labels,omitempty"`
	Quality       *QualitySettings            `yaml:"quality,omitempty" json:"quality,omitempty"`
	Links         *LinksSettings              `yaml:"links,omitempty" json:"links,omitempty"`
	Context       *ContextSettings            `yaml:"context,omitempty" json:"context,omitempty"`
	Automation    *AutomationSettings         `yaml:"automation,omitempty" json:"automation,omitempty"`
	Project       ProjectSettings             `yaml:"project,omitempty" json:"project,omitempty"`
	Stack         *StackSettings              `yaml:"stack,omitempty" json:"stack,omitempty"`
}

// PluginsConfig holds plugin-related configuration.
type PluginsConfig struct {
	// Enabled lists the plugin names that should be loaded
	// Only plugins in this list will be activated
	Enabled []string `yaml:"enabled,omitempty" json:"enabled,omitempty"`

	// Config holds plugin-specific configuration keyed by plugin name
	// Each plugin receives its configuration during initialization
	Config map[string]map[string]any `yaml:"config,omitempty" json:"config,omitempty"`
}

// GitHubSettings holds GitHub provider configuration.
type GitHubSettings struct {
	Token         string                  `yaml:"token,omitempty" json:"token,omitempty"`                   // GitHub token (env vars take priority)
	Owner         string                  `yaml:"owner,omitempty" json:"owner,omitempty"`                   // Repository owner (auto-detected from git remote)
	Repo          string                  `yaml:"repo,omitempty" json:"repo,omitempty"`                     // Repository name
	BranchPattern string                  `yaml:"branch_pattern,omitempty" json:"branch_pattern,omitempty"` // Default: "issue/{key}-{slug}"
	CommitPrefix  string                  `yaml:"commit_prefix,omitempty" json:"commit_prefix,omitempty"`   // Default: "[#{key}]"
	TargetBranch  string                  `yaml:"target_branch,omitempty" json:"target_branch,omitempty"`   // Default: detected from repo
	DraftPR       bool                    `yaml:"draft_pr,omitempty" json:"draft_pr,omitempty"`             // Create PRs as draft
	Comments      *GitHubCommentsSettings `yaml:"comments,omitempty" json:"comments,omitempty"`
}

// GitHubCommentsSettings controls automated GitHub issue commenting.
type GitHubCommentsSettings struct {
	Enabled         bool `yaml:"enabled" json:"enabled"`                     // Master switch (default: false)
	OnBranchCreated bool `yaml:"on_branch_created" json:"on_branch_created"` // Post when branch is created
	OnPlanDone      bool `yaml:"on_plan_done" json:"on_plan_done"`           // Post summary of planned implementation
	OnImplementDone bool `yaml:"on_implement_done" json:"on_implement_done"` // Post changelog with files changed
	OnPRCreated     bool `yaml:"on_pr_created" json:"on_pr_created"`         // Post PR link
}

// WrikeSettings holds Wrike provider configuration.
type WrikeSettings struct {
	Token   string `yaml:"token,omitempty" json:"token,omitempty"`     // Wrike API token (env vars take priority)
	Host    string `yaml:"host,omitempty" json:"host,omitempty"`       // API base URL override (default: https://www.wrike.com/api/v4)
	Space   string `yaml:"space,omitempty" json:"space,omitempty"`     // Space ID (for listing tasks across space)
	Folder  string `yaml:"folder,omitempty" json:"folder,omitempty"`   // Folder ID (for task lookup/creation if no project)
	Project string `yaml:"project,omitempty" json:"project,omitempty"` // Project ID (primary target for task creation)
}

// GitLabSettings holds GitLab provider configuration.
type GitLabSettings struct {
	Token         string `yaml:"token,omitempty" json:"token,omitempty"`                   // GitLab token (env vars take priority)
	Host          string `yaml:"host,omitempty" json:"host,omitempty"`                     // GitLab host (default: https://gitlab.com)
	ProjectPath   string `yaml:"project_path,omitempty" json:"project_path,omitempty"`     // Default project path (e.g., group/project)
	BranchPattern string `yaml:"branch_pattern,omitempty" json:"branch_pattern,omitempty"` // Default: "issue/{key}-{slug}"
	CommitPrefix  string `yaml:"commit_prefix,omitempty" json:"commit_prefix,omitempty"`   // Default: "[#{key}]"
}

// NotionSettings holds Notion provider configuration.
type NotionSettings struct {
	Token               string `yaml:"token,omitempty" json:"token,omitempty"`                               // Notion token (env vars take priority)
	DatabaseID          string `yaml:"database_id,omitempty" json:"database_id,omitempty"`                   // Default database ID
	StatusProperty      string `yaml:"status_property,omitempty" json:"status_property,omitempty"`           // Property name for status (default: Status)
	DescriptionProperty string `yaml:"description_property,omitempty" json:"description_property,omitempty"` // Property name for description
	LabelsProperty      string `yaml:"labels_property,omitempty" json:"labels_property,omitempty"`           // Property name for labels (default: Tags)
}

// JiraSettings holds Jira provider configuration.
type JiraSettings struct {
	Token   string `yaml:"token,omitempty" json:"token,omitempty"`       // Jira API token (env vars take priority)
	Email   string `yaml:"email,omitempty" json:"email,omitempty"`       // Email for Cloud auth
	BaseURL string `yaml:"base_url,omitempty" json:"base_url,omitempty"` // Base URL (optional, auto-detected)
	Project string `yaml:"project,omitempty" json:"project,omitempty"`   // Default project key
}

// LinearSettings holds Linear provider configuration.
type LinearSettings struct {
	Token string `yaml:"token,omitempty" json:"token,omitempty"` // Linear API key (env vars take priority)
	Team  string `yaml:"team,omitempty" json:"team,omitempty"`   // Default team key
}

// YouTrackSettings holds YouTrack provider configuration.
type YouTrackSettings struct {
	Token string `yaml:"token,omitempty" json:"token,omitempty"` // YouTrack token (env vars take priority)
	Host  string `yaml:"host,omitempty" json:"host,omitempty"`   // YouTrack host
}

// BitbucketSettings holds Bitbucket provider configuration.
type BitbucketSettings struct {
	Username          string `yaml:"username,omitempty" json:"username,omitempty"`                       // Bitbucket username
	AppPassword       string `yaml:"app_password,omitempty" json:"app_password,omitempty"`               // Bitbucket app password (env vars take priority)
	Workspace         string `yaml:"workspace,omitempty" json:"workspace,omitempty"`                     // Bitbucket workspace
	RepoSlug          string `yaml:"repo,omitempty" json:"repo,omitempty"`                               // Repository slug
	BranchPattern     string `yaml:"branch_pattern,omitempty" json:"branch_pattern,omitempty"`           // Git branch template
	CommitPrefix      string `yaml:"commit_prefix,omitempty" json:"commit_prefix,omitempty"`             // Commit message prefix
	TargetBranch      string `yaml:"target_branch,omitempty" json:"target_branch,omitempty"`             // Target branch for PRs
	CloseSourceBranch bool   `yaml:"close_source_branch,omitempty" json:"close_source_branch,omitempty"` // Delete source branch when PR is merged
}

// AsanaSettings holds Asana provider configuration.
type AsanaSettings struct {
	Token          string `yaml:"token,omitempty" json:"token,omitempty"`                     // Asana token (env vars take priority)
	WorkspaceGID   string `yaml:"workspace_gid,omitempty" json:"workspace_gid,omitempty"`     // Asana workspace GID
	DefaultProject string `yaml:"default_project,omitempty" json:"default_project,omitempty"` // Default project GID for list operations
	BranchPattern  string `yaml:"branch_pattern,omitempty" json:"branch_pattern,omitempty"`   // Git branch template
	CommitPrefix   string `yaml:"commit_prefix,omitempty" json:"commit_prefix,omitempty"`     // Commit message prefix
}

// ClickUpSettings holds ClickUp provider configuration.
type ClickUpSettings struct {
	Token         string `yaml:"token,omitempty" json:"token,omitempty"`                   // ClickUp API token (env vars take priority)
	TeamID        string `yaml:"team_id,omitempty" json:"team_id,omitempty"`               // Team/Workspace ID
	DefaultList   string `yaml:"default_list,omitempty" json:"default_list,omitempty"`     // Default list ID for list operations
	BranchPattern string `yaml:"branch_pattern,omitempty" json:"branch_pattern,omitempty"` // Git branch template
	CommitPrefix  string `yaml:"commit_prefix,omitempty" json:"commit_prefix,omitempty"`   // Commit message prefix
}

// AzureDevOpsSettings holds Azure DevOps provider configuration.
type AzureDevOpsSettings struct {
	Token         string `yaml:"token,omitempty" json:"token,omitempty"`                   // Azure DevOps PAT (env vars take priority)
	Organization  string `yaml:"organization,omitempty" json:"organization,omitempty"`     // Azure DevOps organization
	Project       string `yaml:"project,omitempty" json:"project,omitempty"`               // Project name
	AreaPath      string `yaml:"area_path,omitempty" json:"area_path,omitempty"`           // Filter by area path
	IterationPath string `yaml:"iteration_path,omitempty" json:"iteration_path,omitempty"` // Filter by iteration
	RepoName      string `yaml:"repo_name,omitempty" json:"repo_name,omitempty"`           // Default repository for PR creation
	TargetBranch  string `yaml:"target_branch,omitempty" json:"target_branch,omitempty"`   // Default target branch for PRs
	BranchPattern string `yaml:"branch_pattern,omitempty" json:"branch_pattern,omitempty"` // Git branch template
	CommitPrefix  string `yaml:"commit_prefix,omitempty" json:"commit_prefix,omitempty"`   // Commit message prefix
}

// TrelloSettings holds Trello provider configuration.
type TrelloSettings struct {
	APIKey string `yaml:"api_key,omitempty" json:"api_key,omitempty"` // Trello API key (env vars take priority)
	Token  string `yaml:"token,omitempty" json:"token,omitempty"`     // Trello token (env vars take priority)
	Board  string `yaml:"board,omitempty" json:"board,omitempty"`     // Default board ID
}

// BrowserSettings holds browser automation configuration.
type BrowserSettings struct {
	Enabled          bool   `yaml:"enabled,omitempty" json:"enabled,omitempty"`                       // Enable browser automation (default: false)
	Host             string `yaml:"host,omitempty" json:"host,omitempty"`                             // CDP host (default: localhost)
	Port             int    `yaml:"port,omitempty" json:"port,omitempty"`                             // CDP port: 0 = random (default), 9222 = existing Chrome
	Headless         bool   `yaml:"headless,omitempty" json:"headless,omitempty"`                     // Launch headless browser (default: false)
	IgnoreCertErrors bool   `yaml:"ignore_cert_errors,omitempty" json:"ignore_cert_errors,omitempty"` // Ignore SSL certificate errors (default: true for local dev)
	Timeout          int    `yaml:"timeout,omitempty" json:"timeout,omitempty"`                       // Operation timeout in seconds (default: 30)
	ScreenshotDir    string `yaml:"screenshot_dir,omitempty" json:"screenshot_dir,omitempty"`         // Directory for screenshots (default: .mehrhof/screenshots)
	CookieProfile    string `yaml:"cookie_profile,omitempty" json:"cookie_profile,omitempty"`         // Which cookie profile to use (default: "default")
	CookieAutoLoad   bool   `yaml:"cookie_auto_load,omitempty" json:"cookie_auto_load,omitempty"`     // Auto-load cookies on connect (default: true)
	CookieAutoSave   bool   `yaml:"cookie_auto_save,omitempty" json:"cookie_auto_save,omitempty"`     // Auto-save cookies on disconnect (default: true)
	CookieDir        string `yaml:"cookie_dir,omitempty" json:"cookie_dir,omitempty"`                 // Custom cookie directory (default: ~/.mehrhof/)
}

// MCPSettings holds MCP (Model Context Protocol) server configuration.
type MCPSettings struct {
	Enabled   bool               `yaml:"enabled,omitempty" json:"enabled,omitempty"`       // Enable MCP server (default: false)
	ToolList  []string           `yaml:"tools,omitempty" json:"tools,omitempty"`           // Allowlist of tools to expose (empty = all safe tools)
	RateLimit *RateLimitSettings `yaml:"rate_limit,omitempty" json:"rate_limit,omitempty"` // Rate limiting for tool calls
}

// RateLimitSettings holds rate limiter configuration for MCP server.
type RateLimitSettings struct {
	Rate  float64 `yaml:"rate,omitempty" json:"rate,omitempty"`   // Requests per second (default: 10)
	Burst int     `yaml:"burst,omitempty" json:"burst,omitempty"` // Burst size (default: 20)
}

// SecuritySettings holds security scanning configuration.
type SecuritySettings struct {
	Enabled  bool                   `yaml:"enabled,omitempty" json:"enabled,omitempty"`   // Enable security scanning (default: false)
	RunOn    SecurityRunOnConfig    `yaml:"run_on,omitempty" json:"run_on,omitempty"`     // When to run scans
	FailOn   SecurityFailOnConfig   `yaml:"fail_on,omitempty" json:"fail_on,omitempty"`   // Failure policy
	Scanners SecurityScannersConfig `yaml:"scanners,omitempty" json:"scanners,omitempty"` // Scanner configuration
	Output   SecurityOutputConfig   `yaml:"output,omitempty" json:"output,omitempty"`     // Reporting settings
	Tools    *SecurityToolsConfig   `yaml:"tools,omitempty" json:"tools,omitempty"`       // Tool management
}

// SecurityRunOnConfig controls when security scans run.
type SecurityRunOnConfig struct {
	Planning     bool `yaml:"planning,omitempty" json:"planning,omitempty"`         // Run during planning (default: false)
	Implementing bool `yaml:"implementing,omitempty" json:"implementing,omitempty"` // Run after implementation (default: true)
	Reviewing    bool `yaml:"reviewing,omitempty" json:"reviewing,omitempty"`       // Run during review (default: true)
}

// SecurityFailOnConfig controls failure behavior.
type SecurityFailOnConfig struct {
	Level       string `yaml:"level,omitempty" json:"level,omitempty"`               // Minimum severity to fail: "critical", "high", "medium", "low", "any" (default: "critical")
	BlockFinish bool   `yaml:"block_finish,omitempty" json:"block_finish,omitempty"` // Block task completion on failures (default: true)
}

// SecurityScannersConfig configures individual scanners.
type SecurityScannersConfig struct {
	SAST         *SASTScannerConfig       `yaml:"sast,omitempty" json:"sast,omitempty"`
	Secrets      *SecretScannerConfig     `yaml:"secrets,omitempty" json:"secrets,omitempty"`
	Dependencies *DependencyScannerConfig `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
	License      *LicenseScannerConfig    `yaml:"license,omitempty" json:"license,omitempty"`
}

// SASTScannerConfig configures static analysis scanners.
type SASTScannerConfig struct {
	Enabled bool                     `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	Tools   []map[string]interface{} `yaml:"tools,omitempty" json:"tools,omitempty"` // Tool-specific config
}

// SecretScannerConfig configures secret detection scanners.
type SecretScannerConfig struct {
	Enabled bool                     `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	Tools   []map[string]interface{} `yaml:"tools,omitempty" json:"tools,omitempty"` // Tool-specific config
}

// DependencyScannerConfig configures vulnerability scanners.
type DependencyScannerConfig struct {
	Enabled bool                     `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	Tools   []map[string]interface{} `yaml:"tools,omitempty" json:"tools,omitempty"` // Tool-specific config
}

// LicenseScannerConfig configures license compliance checking.
type LicenseScannerConfig struct {
	Enabled   bool     `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	Allowlist []string `yaml:"allowlist,omitempty" json:"allowlist,omitempty"` // Allowed licenses (e.g., "MIT", "Apache-2.0")
}

// SecurityOutputConfig controls report generation.
type SecurityOutputConfig struct {
	Format             string `yaml:"format,omitempty" json:"format,omitempty"`                           // "sarif", "json", "text" (default: "sarif")
	File               string `yaml:"file,omitempty" json:"file,omitempty"`                               // Report file path (default: ".mehrhof/security-report.json")
	IncludeSuggestions bool   `yaml:"include_suggestions,omitempty" json:"include_suggestions,omitempty"` // Include fix suggestions (default: true)
}

// SecurityToolsConfig controls security tool management.
type SecurityToolsConfig struct {
	AutoDownload bool   `yaml:"auto_download,omitempty" json:"auto_download,omitempty"` // Auto-download missing tools (default: true)
	CacheDir     string `yaml:"cache_dir,omitempty" json:"cache_dir,omitempty"`         // Override default cache directory (default: ~/.valksor/mehrhof/tools)
	Timeout      int    `yaml:"timeout,omitempty" json:"timeout,omitempty"`             // Download timeout in seconds (default: 60)
}

// MemorySettings holds memory system configuration.
type MemorySettings struct {
	Enabled   bool                  `yaml:"enabled,omitempty" json:"enabled,omitempty"`     // Enable memory system (default: false)
	VectorDB  VectorDBSettings      `yaml:"vector_db,omitempty" json:"vector_db,omitempty"` // Vector database configuration
	Retention MemoryRetentionConfig `yaml:"retention,omitempty" json:"retention,omitempty"` // Retention policy
	Search    MemorySearchConfig    `yaml:"search,omitempty" json:"search,omitempty"`       // Search settings
	Learning  MemoryLearningConfig  `yaml:"learning,omitempty" json:"learning,omitempty"`   // Learning settings
}

// VectorDBSettings configures vector database backend.
type VectorDBSettings struct {
	Backend          string       `yaml:"backend,omitempty" json:"backend,omitempty"`                     // "chromadb", "pinecone", "weaviate", "qdrant" (default: "chromadb")
	ConnectionString string       `yaml:"connection_string,omitempty" json:"connection_string,omitempty"` // Path or URL to vector DB (default: "./.mehrhof/vectors")
	Collection       string       `yaml:"collection,omitempty" json:"collection,omitempty"`               // Collection name (default: "mehr_task_memory")
	EmbeddingModel   string       `yaml:"embedding_model,omitempty" json:"embedding_model,omitempty"`     // Embedding model name: "default" (hash) or "onnx" (semantic)
	ONNX             ONNXSettings `yaml:"onnx,omitempty" json:"onnx,omitempty"`                           // ONNX embedding model settings
}

// ONNXSettings configures the ONNX embedding model.
type ONNXSettings struct {
	Model     string `yaml:"model,omitempty" json:"model,omitempty"`           // ONNX model name (default: "all-MiniLM-L6-v2")
	CachePath string `yaml:"cache_path,omitempty" json:"cache_path,omitempty"` // Custom model cache path (default: ~/.valksor/mehrhof/models/)
	MaxLength int    `yaml:"max_length,omitempty" json:"max_length,omitempty"` // Maximum sequence length (default: 256)
}

// MemoryRetentionConfig controls data retention.
type MemoryRetentionConfig struct {
	MaxDays  int `yaml:"max_days,omitempty" json:"max_days,omitempty"`   // Maximum days to keep documents (default: 90)
	MaxTasks int `yaml:"max_tasks,omitempty" json:"max_tasks,omitempty"` // Maximum number of tasks to store (default: 1000)
}

// MemorySearchConfig controls search behavior.
type MemorySearchConfig struct {
	SimilarityThreshold float32 `yaml:"similarity_threshold,omitempty" json:"similarity_threshold,omitempty"` // Minimum similarity score (default: 0.8)
	MaxResults          int     `yaml:"max_results,omitempty" json:"max_results,omitempty"`                   // Maximum results to return (default: 5)
	IncludeCode         bool    `yaml:"include_code,omitempty" json:"include_code,omitempty"`                 // Include code changes (default: true)
	IncludeSpecs        bool    `yaml:"include_specs,omitempty" json:"include_specs,omitempty"`               // Include specifications (default: true)
	IncludeSessions     bool    `yaml:"include_sessions,omitempty" json:"include_sessions,omitempty"`         // Include session logs (default: true)
}

// MemoryLearningConfig controls automatic learning.
type MemoryLearningConfig struct {
	AutoStore            bool `yaml:"auto_store,omitempty" json:"auto_store,omitempty"`                         // Automatically store task data (default: true)
	LearnFromCorrections bool `yaml:"learn_from_corrections,omitempty" json:"learn_from_corrections,omitempty"` // Learn from user corrections (default: true)
	SuggestSimilar       bool `yaml:"suggest_similar,omitempty" json:"suggest_similar,omitempty"`               // Auto-suggest similar tasks (default: true)
}

// LibrarySettings holds library documentation collection configuration.
type LibrarySettings struct {
	AutoIncludeMax    int    `yaml:"auto_include_max,omitempty" json:"auto_include_max,omitempty"`         // Max collections to auto-include (default: 3)
	MaxPagesPerPrompt int    `yaml:"max_pages_per_prompt,omitempty" json:"max_pages_per_prompt,omitempty"` // Max pages from a single collection (default: 20)
	MaxCrawlPages     int    `yaml:"max_crawl_pages,omitempty" json:"max_crawl_pages,omitempty"`           // Default max pages per crawl (default: 100)
	MaxCrawlDepth     int    `yaml:"max_crawl_depth,omitempty" json:"max_crawl_depth,omitempty"`           // Default max crawl depth (default: 3)
	MaxPageSizeBytes  int64  `yaml:"max_page_size_bytes,omitempty" json:"max_page_size_bytes,omitempty"`   // Max size per page (default: 1MB)
	LockTimeout       string `yaml:"lock_timeout,omitempty" json:"lock_timeout,omitempty"`                 // File lock timeout (default: "10s")
	MaxTokenBudget    int    `yaml:"max_token_budget,omitempty" json:"max_token_budget,omitempty"`         // Total token budget for library context (default: 8000)

	// Crawl filtering options
	DomainScope   string `yaml:"domain_scope,omitempty" json:"domain_scope,omitempty"`     // "same-host" (default) or "same-domain"
	VersionFilter bool   `yaml:"version_filter,omitempty" json:"version_filter,omitempty"` // Auto-detect version from URL path
	VersionPath   string `yaml:"version_path,omitempty" json:"version_path,omitempty"`     // Explicit version path segment (e.g., "v24", "v1.2.3")
}

// OrchestrationSettings holds multi-agent orchestration configuration.
type OrchestrationSettings struct {
	Enabled bool                              `yaml:"enabled,omitempty" json:"enabled,omitempty"` // Enable multi-agent orchestration (default: false)
	Steps   map[string]StepOrchestratorConfig `yaml:"steps,omitempty" json:"steps,omitempty"`     // Per-step orchestration config
}

// StepOrchestratorConfig defines orchestration for a workflow step.
type StepOrchestratorConfig struct {
	Mode      string                     `yaml:"mode,omitempty" json:"mode,omitempty"`           // "single", "sequential", "parallel", "consensus"
	Agents    []OrchestrationAgentConfig `yaml:"agents,omitempty" json:"agents,omitempty"`       // Agent steps
	Consensus StepConsensusConfig        `yaml:"consensus,omitempty" json:"consensus,omitempty"` // Consensus settings
}

// OrchestrationAgentConfig defines an agent step in orchestration.
type OrchestrationAgentConfig struct {
	Name    string            `yaml:"name" json:"name"`                           // Step identifier
	Agent   string            `yaml:"agent" json:"agent"`                         // Agent name to use
	Model   string            `yaml:"model,omitempty" json:"model,omitempty"`     // Optional model override
	Role    string            `yaml:"role" json:"role"`                           // Role/purpose for this agent
	Input   []string          `yaml:"input,omitempty" json:"input,omitempty"`     // Input artifact names
	Output  string            `yaml:"output,omitempty" json:"output,omitempty"`   // Output artifact name
	Depends []string          `yaml:"depends,omitempty" json:"depends,omitempty"` // Dependencies on other steps
	Env     map[string]string `yaml:"env,omitempty" json:"env,omitempty"`         // Environment variables
	Args    []string          `yaml:"args,omitempty" json:"args,omitempty"`       // CLI arguments
	Timeout int               `yaml:"timeout,omitempty" json:"timeout,omitempty"` // Timeout in seconds
}

// StepConsensusConfig defines consensus building for a step.
type StepConsensusConfig struct {
	Mode        string `yaml:"mode,omitempty" json:"mode,omitempty"`               // "majority", "unanimous", "any"
	MinVotes    int    `yaml:"min_votes,omitempty" json:"min_votes,omitempty"`     // Minimum votes required
	Synthesizer string `yaml:"synthesizer,omitempty" json:"synthesizer,omitempty"` // Agent to use for synthesis
}

// MLSettings holds ML prediction system configuration.
type MLSettings struct {
	Enabled     bool                `yaml:"enabled,omitempty" json:"enabled,omitempty"`         // Enable ML predictions (default: false)
	Telemetry   MLTelemetryConfig   `yaml:"telemetry,omitempty" json:"telemetry,omitempty"`     // Telemetry settings
	Model       MLModelConfig       `yaml:"model,omitempty" json:"model,omitempty"`             // Model configuration
	Predictions MLPredictionsConfig `yaml:"predictions,omitempty" json:"predictions,omitempty"` // Prediction settings
}

// MLTelemetryConfig controls telemetry collection.
type MLTelemetryConfig struct {
	Enabled    bool    `yaml:"enabled,omitempty" json:"enabled,omitempty"`         // Enable telemetry collection (default: true)
	Anonymize  bool    `yaml:"anonymize,omitempty" json:"anonymize,omitempty"`     // Anonymize task IDs (default: true)
	SampleRate float32 `yaml:"sample_rate,omitempty" json:"sample_rate,omitempty"` // Sampling rate 0-1 (default: 1.0)
	Storage    string  `yaml:"storage,omitempty" json:"storage,omitempty"`         // Storage path (default: ".mehrhof/telemetry")
}

// MLModelConfig controls ML model configuration.
type MLModelConfig struct {
	Type            string `yaml:"type,omitempty" json:"type,omitempty"`                         // Model type (default: "heuristic")
	RetrainInterval string `yaml:"retrain_interval,omitempty" json:"retrain_interval,omitempty"` // Retrain interval (default: "7d")
	MinSamples      int    `yaml:"min_samples,omitempty" json:"min_samples,omitempty"`           // Minimum samples for training (default: 100)
}

// MLPredictionsConfig controls which predictions are enabled.
type MLPredictionsConfig struct {
	NextAction     bool `yaml:"next_action,omitempty" json:"next_action,omitempty"`         // Predict next action (default: true)
	Duration       bool `yaml:"duration,omitempty" json:"duration,omitempty"`               // Predict duration (default: true)
	Complexity     bool `yaml:"complexity,omitempty" json:"complexity,omitempty"`           // Predict complexity (default: true)
	AgentSelection bool `yaml:"agent_selection,omitempty" json:"agent_selection,omitempty"` // Predict agent selection (default: true)
	RiskAssessment bool `yaml:"risk_assessment,omitempty" json:"risk_assessment,omitempty"` // Predict risks (default: true)
}

// AgentAliasConfig defines a user-defined agent alias that wraps an existing agent
// with custom environment variables and CLI arguments.
type AgentAliasConfig struct {
	Extends     string            `yaml:"extends" json:"extends"`                             // Base agent name to wrap
	Description string            `yaml:"description,omitempty" json:"description,omitempty"` // Human-readable description
	Components  []string          `yaml:"components,omitempty" json:"components,omitempty"`   // Components this agent handles (e.g., backend, frontend, tests)
	Env         map[string]string `yaml:"env,omitempty" json:"env,omitempty"`                 // Environment variables to pass
	Args        []string          `yaml:"args,omitempty" json:"args,omitempty"`               // CLI arguments to pass
}

// GitSettings holds git-related configuration.
type GitSettings struct {
	CommitPrefix  string `yaml:"commit_prefix" json:"commit_prefix"`
	BranchPattern string `yaml:"branch_pattern" json:"branch_pattern"`
	AutoCommit    bool   `yaml:"auto_commit" json:"auto_commit"`
	SignCommits   bool   `yaml:"sign_commits" json:"sign_commits"`
	StashOnStart  bool   `yaml:"stash_on_start" json:"stash_on_start"`                     // Auto-stash changes before creating task branch
	AutoPopStash  bool   `yaml:"auto_pop_stash" json:"auto_pop_stash"`                     // Auto-pop stash after branch creation (if stashed)
	DefaultBranch string `yaml:"default_branch,omitempty" json:"default_branch,omitempty"` // Override default branch detection (e.g., "main", "develop")
}

// StepAgentConfig holds agent configuration for a specific workflow step.
type StepAgentConfig struct {
	Name            string            `yaml:"name,omitempty" json:"name,omitempty"`                         // Agent name or alias
	Env             map[string]string `yaml:"env,omitempty" json:"env,omitempty"`                           // Step-specific env vars
	Args            []string          `yaml:"args,omitempty" json:"args,omitempty"`                         // Step-specific CLI args
	Instructions    string            `yaml:"instructions,omitempty" json:"instructions,omitempty"`         // Custom instructions for this step
	OptimizePrompts bool              `yaml:"optimize_prompts,omitempty" json:"optimize_prompts,omitempty"` // Optimize prompts for this step
}

// AgentSettings holds agent-related configuration.
type AgentSettings struct {
	Default         string                     `yaml:"default" json:"default"`
	Timeout         int                        `yaml:"timeout" json:"timeout"`
	MaxRetries      int                        `yaml:"max_retries" json:"max_retries"`
	Instructions    string                     `yaml:"instructions,omitempty" json:"instructions,omitempty"`         // Global instructions for all steps
	OptimizePrompts bool                       `yaml:"optimize_prompts,omitempty" json:"optimize_prompts,omitempty"` // Optimize prompts for all steps
	Steps           map[string]StepAgentConfig `yaml:"steps,omitempty" json:"steps,omitempty"`                       // Per-step agent configuration
	PRReview        *PRReviewConfig            `yaml:"pr_review,omitempty" json:"pr_review,omitempty"`               // PR review configuration
}

// PRReviewConfig holds PR review configuration.
type PRReviewConfig struct {
	Enabled          bool     `yaml:"enabled,omitempty" json:"enabled,omitempty"`                     // Enable PR review (default: false)
	Format           string   `yaml:"format,omitempty" json:"format,omitempty"`                       // Comment format: summary, line-comments
	Scope            string   `yaml:"scope,omitempty" json:"scope,omitempty"`                         // Review scope: full, compact, files-changed
	FailOnIssues     bool     `yaml:"fail_on_issues,omitempty" json:"fail_on_issues,omitempty"`       // Exit with error on issues
	MaxComments      int      `yaml:"max_comments,omitempty" json:"max_comments,omitempty"`           // Cap to avoid spam
	ExcludePatterns  []string `yaml:"exclude_patterns,omitempty" json:"exclude_patterns,omitempty"`   // File patterns to exclude
	AcknowledgeFixes bool     `yaml:"acknowledge_fixes,omitempty" json:"acknowledge_fixes,omitempty"` // Post "✓ Fixed" comments when issues are resolved
	UpdateExisting   bool     `yaml:"update_existing,omitempty" json:"update_existing,omitempty"`     // Edit existing comment vs post new ones
}

// SimplifySettings holds configuration for the simplify command.
type SimplifySettings struct {
	Instructions string `yaml:"instructions,omitempty" json:"instructions,omitempty"` // Custom instructions for all simplification steps
}

// WorkflowSettings holds workflow-related configuration.
type WorkflowSettings struct {
	AutoInit             bool             `yaml:"auto_init" json:"auto_init"`
	SessionRetentionDays int              `yaml:"session_retention_days" json:"session_retention_days"`
	DeleteWorkOnFinish   bool             `yaml:"delete_work_on_finish" json:"delete_work_on_finish"`   // Delete work dirs on finish (default: false)
	DeleteWorkOnAbandon  bool             `yaml:"delete_work_on_abandon" json:"delete_work_on_abandon"` // Delete work dirs on abandon (default: true)
	Simplify             SimplifySettings `yaml:"simplify,omitempty" json:"simplify,omitempty"`         // Simplification command settings
}

// BudgetSettings holds budget configuration for costs and tokens.
type BudgetSettings struct {
	PerTask       BudgetConfig          `yaml:"per_task,omitempty" json:"per_task,omitempty"`             // Default budget for tasks
	Monthly       MonthlyBudgetSettings `yaml:"monthly,omitempty" json:"monthly,omitempty"`               // Monthly workspace budget
	ExchangeRates map[string]float64    `yaml:"exchange_rates,omitempty" json:"exchange_rates,omitempty"` // Currency conversion rates (to USD)
}

// MonthlyBudgetSettings defines a workspace monthly budget.
type MonthlyBudgetSettings struct {
	Enabled   bool    `yaml:"enabled,omitempty" json:"enabled,omitempty"` // Enable monthly budget tracking (default: false)
	MaxCost   float64 `yaml:"max_cost,omitempty" json:"max_cost,omitempty"`
	Currency  string  `yaml:"currency,omitempty" json:"currency,omitempty"`
	WarningAt float64 `yaml:"warning_at,omitempty" json:"warning_at,omitempty"` // 0-1 (e.g., 0.8)
}

// UpdateSettings holds update-related configuration.
type UpdateSettings struct {
	Enabled       bool `yaml:"enabled" json:"enabled"`               // Enable automatic update checks
	CheckInterval int  `yaml:"check_interval" json:"check_interval"` // Hours between checks (default: 24)
}

// StorageSettings holds storage-related configuration.
type StorageSettings struct {
	HomeDir       string `yaml:"home_dir,omitempty" json:"home_dir,omitempty"`               // Override for mehrhof home directory (default: ~/.mehrhof)
	SaveInProject bool   `yaml:"save_in_project,omitempty" json:"save_in_project,omitempty"` // Store work in project directory (default: false = global)
	ProjectDir    string `yaml:"project_dir,omitempty" json:"project_dir,omitempty"`         // Project dir for work (default: ".mehrhof/work" when save_in_project=true)
}

// ProjectSettings holds project-level settings for decoupled hub/code workflows.
type ProjectSettings struct {
	CodeDir string `yaml:"code_dir,omitempty" json:"code_dir,omitempty"` // Separate code directory (relative to project root or absolute)
}

// StackSettings holds stacked feature branch configuration.
type StackSettings struct {
	AutoRebase       string `yaml:"auto_rebase,omitempty" json:"auto_rebase,omitempty"`               // When to auto-rebase children: "disabled" (default) | "on_finish"
	BlockOnConflicts bool   `yaml:"block_on_conflicts,omitempty" json:"block_on_conflicts,omitempty"` // Block auto-rebase if conflicts detected (default: true)
}

// SpecificationSettings holds specification-related configuration.
type SpecificationSettings struct {
	FilenamePattern string `yaml:"filename_pattern" json:"filename_pattern"` // Spec filename pattern (default: "specification-{n}.md")
}

// ReviewSettings holds code review output configuration.
type ReviewSettings struct {
	FilenamePattern string `yaml:"filename_pattern" json:"filename_pattern"` // Review filename pattern (default: "review-{n}.txt")
}

// SandboxSettings holds agent sandboxing configuration.
type SandboxSettings struct {
	Enabled bool     `yaml:"enabled,omitempty" json:"enabled,omitempty"` // Enable sandboxing (default: false)
	Network bool     `yaml:"network,omitempty" json:"network,omitempty"` // Allow network access (default: true - LLM APIs need this)
	TmpDir  string   `yaml:"tmp_dir,omitempty" json:"tmp_dir,omitempty"` // Tmpfs mount path (default: auto)
	Tools   []string `yaml:"tools,omitempty" json:"tools,omitempty"`     // Extra binary paths to allow (beyond defaults)
}

// ProvidersSettings holds provider-related configuration.
type ProvidersSettings struct {
	Default        string `yaml:"default,omitempty" json:"default,omitempty"`                 // Default provider for bare references (e.g., "file", "directory", "github")
	DefaultMention string `yaml:"default_mention,omitempty" json:"default_mention,omitempty"` // Default mention text when submitting tasks (e.g., "@manager please review")
}

// LabelDefinition defines a label with optional color.
type LabelDefinition struct {
	Name  string `yaml:"name" json:"name"`                       // Label name (e.g., "priority:high")
	Color string `yaml:"color,omitempty" json:"color,omitempty"` // Optional CSS color class (overrides hash-based color)
}

// LabelSettings holds label-related configuration.
type LabelSettings struct {
	Enabled     bool              `yaml:"enabled,omitempty" json:"enabled,omitempty"`         // Enable label system (default: true)
	Defined     []LabelDefinition `yaml:"defined,omitempty" json:"defined,omitempty"`         // Predefined labels with colors
	Suggestions []string          `yaml:"suggestions,omitempty" json:"suggestions,omitempty"` // Suggested labels for autocomplete
}

// QualitySettings holds code quality and linter configuration.
type QualitySettings struct {
	Enabled     bool                    `yaml:"enabled,omitempty" json:"enabled,omitempty"`           // Enable quality checks (default: true)
	UseDefaults bool                    `yaml:"use_defaults,omitempty" json:"use_defaults,omitempty"` // Auto-enable default linters (default: false - safer)
	Linters     map[string]LinterConfig `yaml:"linters,omitempty" json:"linters,omitempty"`           // Linter-specific config by name
}

// LinterConfig defines configuration for a single linter.
type LinterConfig struct {
	Enabled    bool     `yaml:"enabled,omitempty" json:"enabled,omitempty"`       // Enable/disable this linter (default: true for built-ins)
	Command    []string `yaml:"command,omitempty" json:"command,omitempty"`       // Custom command (e.g., ["vendor/bin/phpstan", "analyse"])
	Args       []string `yaml:"args,omitempty" json:"args,omitempty"`             // Additional arguments
	Extensions []string `yaml:"extensions,omitempty" json:"extensions,omitempty"` // File extensions to lint (default: auto-detected)
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
		Automation: &AutomationSettings{
			Enabled: false,
			Providers: map[string]ProviderAutoConfig{
				"github": {
					Enabled:       false,
					CommandPrefix: "@mehrhof",
					UseWorktrees:  true,
					TriggerOn: AutomationTriggerConfig{
						IssueOpened:     true,
						PROpened:        true,
						CommentCommands: true,
					},
				},
				"gitlab": {
					Enabled:       false,
					CommandPrefix: "@mehrhof",
					UseWorktrees:  true,
					TriggerOn: AutomationTriggerConfig{
						IssueOpened:     true,
						MROpened:        true,
						CommentCommands: true,
					},
				},
			},
			AccessControl: AutomationAccessControlConfig{
				Mode: "all",
			},
			Queue: AutomationQueueConfig{
				MaxConcurrent: 1,
				JobTimeout:    "30m",
			},
			Labels: AutomationLabelConfig{
				MehrhofGenerated: "mehrhof-generated",
				InProgress:       "mehrhof-processing",
				Failed:           "mehrhof-failed",
				SkipReview:       "mehrhof-skip-review",
			},
		},
		Stack: &StackSettings{
			AutoRebase:       "disabled", // Opt-in: "disabled" | "on_finish"
			BlockOnConflicts: true,       // Safe default: always block on conflicts
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
