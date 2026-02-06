package storage

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
	BinaryPath  string            `yaml:"binary_path,omitempty" json:"binary_path,omitempty"` // Custom binary path (overrides base agent's default)
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
	Enabled       bool                  `yaml:"enabled,omitempty" json:"enabled,omitempty"`               // Enable budget tracking (default: false)
	PerTask       BudgetConfig          `yaml:"per_task,omitempty" json:"per_task,omitempty"`             // Default budget for tasks
	Monthly       MonthlyBudgetSettings `yaml:"monthly,omitempty" json:"monthly,omitempty"`               // Monthly workspace budget
	ExchangeRates map[string]float64    `yaml:"exchange_rates,omitempty" json:"exchange_rates,omitempty"` // Currency conversion rates (to USD)
}

// MonthlyBudgetSettings defines a workspace monthly budget.
type MonthlyBudgetSettings struct {
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
