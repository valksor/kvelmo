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
	Project       ProjectSettings             `yaml:"project,omitempty" json:"project,omitempty"`
	Stack         *StackSettings              `yaml:"stack,omitempty" json:"stack,omitempty"`
	Display       *DisplaySettings            `yaml:"display,omitempty" json:"display,omitempty"`
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
	Token         string                  `yaml:"token,omitempty" json:"token,omitempty" schema:"label=Token;desc=GitHub personal access token;sensitive"`
	Owner         string                  `yaml:"owner,omitempty" json:"owner,omitempty" schema:"label=Owner;desc=Repository owner (auto-detected from git remote)"`
	Repo          string                  `yaml:"repo,omitempty" json:"repo,omitempty" schema:"label=Repository;desc=Repository name"`
	BranchPattern string                  `yaml:"branch_pattern,omitempty" json:"branch_pattern,omitempty" schema:"label=Branch Pattern;desc=Pattern for branch names;default=issue/{key}-{slug}"`
	CommitPrefix  string                  `yaml:"commit_prefix,omitempty" json:"commit_prefix,omitempty" schema:"label=Commit Prefix;desc=Pattern for commit messages;default=[#{key}]"`
	TargetBranch  string                  `yaml:"target_branch,omitempty" json:"target_branch,omitempty" schema:"label=Target Branch;desc=Default target branch for PRs;placeholder=auto-detect"`
	DraftPR       bool                    `yaml:"draft_pr,omitempty" json:"draft_pr,omitempty" schema:"label=Draft PR;desc=Create PRs as draft;default=false"`
	Comments      *GitHubCommentsSettings `yaml:"comments,omitempty" json:"comments,omitempty" schema:"-"`
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
	Token   string `yaml:"token,omitempty" json:"token,omitempty" schema:"label=Token;desc=Wrike API token;sensitive"`
	Host    string `yaml:"host,omitempty" json:"host,omitempty" schema:"label=Host;desc=API base URL override;default=https://www.wrike.com/api/v4;advanced"`
	Space   string `yaml:"space,omitempty" json:"space,omitempty" schema:"label=Space;desc=Space ID"`
	Folder  string `yaml:"folder,omitempty" json:"folder,omitempty" schema:"label=Folder;desc=Folder ID"`
	Project string `yaml:"project,omitempty" json:"project,omitempty" schema:"label=Project;desc=Project ID"`
}

// GitLabSettings holds GitLab provider configuration.
type GitLabSettings struct {
	Token         string `yaml:"token,omitempty" json:"token,omitempty" schema:"label=Token;desc=GitLab personal access token;sensitive"`
	Host          string `yaml:"host,omitempty" json:"host,omitempty" schema:"label=Host;desc=GitLab host URL;default=https://gitlab.com;placeholder=https://gitlab.com"`
	ProjectPath   string `yaml:"project_path,omitempty" json:"project_path,omitempty" schema:"label=Project Path;desc=Project path (e.g., group/project)"`
	BranchPattern string `yaml:"branch_pattern,omitempty" json:"branch_pattern,omitempty" schema:"label=Branch Pattern;desc=Pattern for branch names;default=issue/{key}-{slug}"`
	CommitPrefix  string `yaml:"commit_prefix,omitempty" json:"commit_prefix,omitempty" schema:"label=Commit Prefix;desc=Pattern for commit messages;default=[#{key}]"`
}

// NotionSettings holds Notion provider configuration.
type NotionSettings struct {
	Token               string `yaml:"token,omitempty" json:"token,omitempty" schema:"label=Token;desc=Notion integration token;sensitive"`
	DatabaseID          string `yaml:"database_id,omitempty" json:"database_id,omitempty" schema:"label=Database ID;desc=Default database ID"`
	StatusProperty      string `yaml:"status_property,omitempty" json:"status_property,omitempty" schema:"label=Status Property;desc=Property name for status;default=Status"`
	DescriptionProperty string `yaml:"description_property,omitempty" json:"description_property,omitempty" schema:"label=Description Property;desc=Property name for description"`
	LabelsProperty      string `yaml:"labels_property,omitempty" json:"labels_property,omitempty" schema:"label=Labels Property;desc=Property name for labels;default=Tags"`
}

// JiraSettings holds Jira provider configuration.
type JiraSettings struct {
	Token   string `yaml:"token,omitempty" json:"token,omitempty" schema:"label=Token;desc=Jira API token;sensitive"`
	Email   string `yaml:"email,omitempty" json:"email,omitempty" schema:"label=Email;desc=Email for Cloud auth"`
	BaseURL string `yaml:"base_url,omitempty" json:"base_url,omitempty" schema:"label=Base URL;desc=Jira base URL;placeholder=https://your-domain.atlassian.net"`
	Project string `yaml:"project,omitempty" json:"project,omitempty" schema:"label=Project;desc=Default project key"`
}

// LinearSettings holds Linear provider configuration.
type LinearSettings struct {
	Token string `yaml:"token,omitempty" json:"token,omitempty" schema:"label=API Key;desc=Linear API key;sensitive"`
	Team  string `yaml:"team,omitempty" json:"team,omitempty" schema:"label=Team;desc=Default team key"`
}

// YouTrackSettings holds YouTrack provider configuration.
type YouTrackSettings struct {
	Token string `yaml:"token,omitempty" json:"token,omitempty" schema:"label=Token;desc=YouTrack token;sensitive"`
	Host  string `yaml:"host,omitempty" json:"host,omitempty" schema:"label=Host;desc=YouTrack host URL"`
}

// BitbucketSettings holds Bitbucket provider configuration.
type BitbucketSettings struct {
	Username          string `yaml:"username,omitempty" json:"username,omitempty" schema:"label=Username;desc=Bitbucket username"`
	AppPassword       string `yaml:"app_password,omitempty" json:"app_password,omitempty" schema:"label=App Password;desc=Bitbucket app password;sensitive"`
	Workspace         string `yaml:"workspace,omitempty" json:"workspace,omitempty" schema:"label=Workspace;desc=Bitbucket workspace"`
	RepoSlug          string `yaml:"repo,omitempty" json:"repo,omitempty" schema:"label=Repository;desc=Repository slug"`
	BranchPattern     string `yaml:"branch_pattern,omitempty" json:"branch_pattern,omitempty" schema:"label=Branch Pattern;desc=Git branch template"`
	CommitPrefix      string `yaml:"commit_prefix,omitempty" json:"commit_prefix,omitempty" schema:"label=Commit Prefix;desc=Commit message prefix"`
	TargetBranch      string `yaml:"target_branch,omitempty" json:"target_branch,omitempty" schema:"label=Target Branch;desc=Target branch for PRs"`
	CloseSourceBranch bool   `yaml:"close_source_branch,omitempty" json:"close_source_branch,omitempty" schema:"label=Close Source Branch;desc=Delete source branch when PR is merged;default=false"`
}

// AsanaSettings holds Asana provider configuration.
type AsanaSettings struct {
	Token          string `yaml:"token,omitempty" json:"token,omitempty" schema:"label=Token;desc=Asana access token;sensitive"`
	WorkspaceGID   string `yaml:"workspace_gid,omitempty" json:"workspace_gid,omitempty" schema:"label=Workspace GID;desc=Asana workspace GID"`
	DefaultProject string `yaml:"default_project,omitempty" json:"default_project,omitempty" schema:"label=Default Project;desc=Default project GID"`
	BranchPattern  string `yaml:"branch_pattern,omitempty" json:"branch_pattern,omitempty" schema:"label=Branch Pattern;desc=Git branch template"`
	CommitPrefix   string `yaml:"commit_prefix,omitempty" json:"commit_prefix,omitempty" schema:"label=Commit Prefix;desc=Commit message prefix"`
}

// ClickUpSettings holds ClickUp provider configuration.
type ClickUpSettings struct {
	Token         string `yaml:"token,omitempty" json:"token,omitempty" schema:"label=Token;desc=ClickUp API token;sensitive"`
	TeamID        string `yaml:"team_id,omitempty" json:"team_id,omitempty" schema:"label=Team ID;desc=Team/Workspace ID"`
	DefaultList   string `yaml:"default_list,omitempty" json:"default_list,omitempty" schema:"label=Default List;desc=Default list ID"`
	BranchPattern string `yaml:"branch_pattern,omitempty" json:"branch_pattern,omitempty" schema:"label=Branch Pattern;desc=Git branch template"`
	CommitPrefix  string `yaml:"commit_prefix,omitempty" json:"commit_prefix,omitempty" schema:"label=Commit Prefix;desc=Commit message prefix"`
}

// AzureDevOpsSettings holds Azure DevOps provider configuration.
type AzureDevOpsSettings struct {
	Token         string `yaml:"token,omitempty" json:"token,omitempty" schema:"label=Token;desc=Azure DevOps PAT;sensitive"`
	Organization  string `yaml:"organization,omitempty" json:"organization,omitempty" schema:"label=Organization;desc=Azure DevOps organization"`
	Project       string `yaml:"project,omitempty" json:"project,omitempty" schema:"label=Project;desc=Project name"`
	AreaPath      string `yaml:"area_path,omitempty" json:"area_path,omitempty" schema:"label=Area Path;desc=Filter by area path;advanced"`
	IterationPath string `yaml:"iteration_path,omitempty" json:"iteration_path,omitempty" schema:"label=Iteration Path;desc=Filter by iteration;advanced"`
	RepoName      string `yaml:"repo_name,omitempty" json:"repo_name,omitempty" schema:"label=Repository;desc=Default repository for PR creation"`
	TargetBranch  string `yaml:"target_branch,omitempty" json:"target_branch,omitempty" schema:"label=Target Branch;desc=Default target branch for PRs"`
	BranchPattern string `yaml:"branch_pattern,omitempty" json:"branch_pattern,omitempty" schema:"label=Branch Pattern;desc=Git branch template"`
	CommitPrefix  string `yaml:"commit_prefix,omitempty" json:"commit_prefix,omitempty" schema:"label=Commit Prefix;desc=Commit message prefix"`
}

// TrelloSettings holds Trello provider configuration.
type TrelloSettings struct {
	APIKey string `yaml:"api_key,omitempty" json:"api_key,omitempty" schema:"label=API Key;desc=Trello API key;sensitive"`
	Token  string `yaml:"token,omitempty" json:"token,omitempty" schema:"label=Token;desc=Trello token;sensitive"`
	Board  string `yaml:"board,omitempty" json:"board,omitempty" schema:"label=Board;desc=Default board ID"`
}

// BrowserSettings holds browser automation configuration.
type BrowserSettings struct {
	Enabled          bool   `yaml:"enabled,omitempty" json:"enabled,omitempty" schema:"label=Enable Browser;desc=Enable browser automation;default=false"`
	Host             string `yaml:"host,omitempty" json:"host,omitempty" schema:"label=Host;desc=CDP host;default=localhost;advanced"`
	Port             int    `yaml:"port,omitempty" json:"port,omitempty" schema:"label=Port;desc=CDP port (0=random, 9222=existing Chrome);default=0;advanced"`
	Headless         bool   `yaml:"headless,omitempty" json:"headless,omitempty" schema:"label=Headless;desc=Launch headless browser;default=false"`
	IgnoreCertErrors bool   `yaml:"ignore_cert_errors,omitempty" json:"ignore_cert_errors,omitempty" schema:"label=Ignore Cert Errors;desc=Ignore SSL certificate errors;default=true;advanced"`
	Timeout          int    `yaml:"timeout,omitempty" json:"timeout,omitempty" schema:"label=Timeout;desc=Operation timeout in seconds;default=30;min=5;max=300"`
	ScreenshotDir    string `yaml:"screenshot_dir,omitempty" json:"screenshot_dir,omitempty" schema:"label=Screenshot Dir;desc=Directory for screenshots;default=.mehrhof/screenshots;advanced"`
	CookieProfile    string `yaml:"cookie_profile,omitempty" json:"cookie_profile,omitempty" schema:"label=Cookie Profile;desc=Cookie profile name;default=default;advanced"`
	CookieAutoLoad   bool   `yaml:"cookie_auto_load,omitempty" json:"cookie_auto_load,omitempty" schema:"label=Auto Load Cookies;desc=Auto-load cookies on connect;default=true;advanced"`
	CookieAutoSave   bool   `yaml:"cookie_auto_save,omitempty" json:"cookie_auto_save,omitempty" schema:"label=Auto Save Cookies;desc=Auto-save cookies on disconnect;default=true;advanced"`
	CookieDir        string `yaml:"cookie_dir,omitempty" json:"cookie_dir,omitempty" schema:"label=Cookie Dir;desc=Custom cookie directory;placeholder=~/.mehrhof/;advanced"`
}

// MCPSettings holds MCP (Model Context Protocol) server configuration.
type MCPSettings struct {
	Enabled   bool               `yaml:"enabled,omitempty" json:"enabled,omitempty" schema:"label=Enable MCP;desc=Enable Model Context Protocol server;default=false"`
	ToolList  []string           `yaml:"tools,omitempty" json:"tools,omitempty" schema:"-"`
	RateLimit *RateLimitSettings `yaml:"rate_limit,omitempty" json:"rate_limit,omitempty" schema:"-"`
}

// RateLimitSettings holds rate limiter configuration for MCP server.
type RateLimitSettings struct {
	Rate  float64 `yaml:"rate,omitempty" json:"rate,omitempty"`   // Requests per second (default: 10)
	Burst int     `yaml:"burst,omitempty" json:"burst,omitempty"` // Burst size (default: 20)
}

// SecuritySettings holds security scanning configuration.
type SecuritySettings struct {
	Enabled  bool                   `yaml:"enabled,omitempty" json:"enabled,omitempty" schema:"label=Enable Security;desc=Enable security scanning;default=false"`
	RunOn    SecurityRunOnConfig    `yaml:"run_on,omitempty" json:"run_on,omitempty" schema:"-"`
	FailOn   SecurityFailOnConfig   `yaml:"fail_on,omitempty" json:"fail_on,omitempty" schema:"-"`
	Scanners SecurityScannersConfig `yaml:"scanners,omitempty" json:"scanners,omitempty" schema:"-"`
	Output   SecurityOutputConfig   `yaml:"output,omitempty" json:"output,omitempty" schema:"-"`
	Tools    *SecurityToolsConfig   `yaml:"tools,omitempty" json:"tools,omitempty" schema:"-"`
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
	Enabled   bool                  `yaml:"enabled,omitempty" json:"enabled,omitempty" schema:"label=Enable Memory;desc=Enable semantic memory system;default=false"`
	VectorDB  VectorDBSettings      `yaml:"vector_db,omitempty" json:"vector_db,omitempty" schema:"-"`
	Retention MemoryRetentionConfig `yaml:"retention,omitempty" json:"retention,omitempty" schema:"-"`
	Search    MemorySearchConfig    `yaml:"search,omitempty" json:"search,omitempty" schema:"-"`
	Learning  MemoryLearningConfig  `yaml:"learning,omitempty" json:"learning,omitempty" schema:"-"`
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
	AutoIncludeMax    int    `yaml:"auto_include_max,omitempty" json:"auto_include_max,omitempty" schema:"label=Auto Include Max;desc=Max collections to auto-include;default=3;min=0;max=10;advanced"`
	MaxPagesPerPrompt int    `yaml:"max_pages_per_prompt,omitempty" json:"max_pages_per_prompt,omitempty" schema:"label=Max Pages Per Prompt;desc=Max pages from a single collection;default=20;min=1;max=100;advanced"`
	MaxCrawlPages     int    `yaml:"max_crawl_pages,omitempty" json:"max_crawl_pages,omitempty" schema:"label=Max Crawl Pages;desc=Default max pages per crawl;default=100;min=1;max=1000;advanced"`
	MaxCrawlDepth     int    `yaml:"max_crawl_depth,omitempty" json:"max_crawl_depth,omitempty" schema:"label=Max Crawl Depth;desc=Default max crawl depth;default=3;min=1;max=10;advanced"`
	MaxPageSizeBytes  int64  `yaml:"max_page_size_bytes,omitempty" json:"max_page_size_bytes,omitempty" schema:"label=Max Page Size;desc=Max size per page in bytes;default=1048576;advanced"`
	LockTimeout       string `yaml:"lock_timeout,omitempty" json:"lock_timeout,omitempty" schema:"label=Lock Timeout;desc=File lock timeout;default=10s;advanced"`
	MaxTokenBudget    int    `yaml:"max_token_budget,omitempty" json:"max_token_budget,omitempty" schema:"label=Max Token Budget;desc=Total token budget for library context;default=8000;min=1000;max=50000;advanced"`

	// Crawl filtering options
	DomainScope   string `yaml:"domain_scope,omitempty" json:"domain_scope,omitempty" schema:"label=Domain Scope;desc=Crawl domain scope;type=select;options=same-host,same-domain;default=same-host;advanced"`
	VersionFilter bool   `yaml:"version_filter,omitempty" json:"version_filter,omitempty" schema:"label=Version Filter;desc=Auto-detect version from URL path;default=false;advanced"`
	VersionPath   string `yaml:"version_path,omitempty" json:"version_path,omitempty" schema:"label=Version Path;desc=Explicit version path segment;placeholder=v24;advanced"`
}

// OrchestrationSettings holds multi-agent orchestration configuration.
type OrchestrationSettings struct {
	Enabled bool                              `yaml:"enabled,omitempty" json:"enabled,omitempty" schema:"label=Enable Orchestration;desc=Enable multi-agent orchestration;default=false"`
	Steps   map[string]StepOrchestratorConfig `yaml:"steps,omitempty" json:"steps,omitempty" schema:"-"`
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
	Enabled     bool                `yaml:"enabled,omitempty" json:"enabled,omitempty" schema:"label=Enable ML;desc=Enable ML predictions;default=false"`
	Telemetry   MLTelemetryConfig   `yaml:"telemetry,omitempty" json:"telemetry,omitempty" schema:"-"`
	Model       MLModelConfig       `yaml:"model,omitempty" json:"model,omitempty" schema:"-"`
	Predictions MLPredictionsConfig `yaml:"predictions,omitempty" json:"predictions,omitempty" schema:"-"`
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
	CommitPrefix  string `yaml:"commit_prefix" json:"commit_prefix" schema:"label=Commit Prefix;desc=Pattern for commit messages. Use {key}, {type}, {slug};default=[{key}];maxlen=100;simple"`
	BranchPattern string `yaml:"branch_pattern" json:"branch_pattern" schema:"label=Branch Pattern;desc=Pattern for branch names. Use {key}, {type}, {slug};default={type}/{key}--{slug};simple"`
	AutoCommit    bool   `yaml:"auto_commit" json:"auto_commit" schema:"label=Auto Commit;desc=Automatically commit after implementation;default=true;simple"`
	SignCommits   bool   `yaml:"sign_commits" json:"sign_commits" schema:"label=Sign Commits;desc=GPG sign commits;default=false;showWhen=git.auto_commit:true"`
	StashOnStart  bool   `yaml:"stash_on_start" json:"stash_on_start" schema:"label=Stash on Start;desc=Auto-stash changes before creating task branch;default=false"`
	AutoPopStash  bool   `yaml:"auto_pop_stash" json:"auto_pop_stash" schema:"label=Auto Pop Stash;desc=Auto-pop stash after branch creation;default=true;showWhen=git.stash_on_start:true"`
	DefaultBranch string `yaml:"default_branch,omitempty" json:"default_branch,omitempty" schema:"label=Default Branch;desc=Override default branch detection (e.g., main, develop);placeholder=auto-detect"`
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
	Default         string                     `yaml:"default" json:"default" schema:"label=Default Agent;desc=Agent to use when not specified;default=claude;simple"`
	Timeout         int                        `yaml:"timeout" json:"timeout" schema:"label=Timeout;desc=Maximum time for agent execution in seconds;default=300;min=30;max=3600"`
	MaxRetries      int                        `yaml:"max_retries" json:"max_retries" schema:"label=Max Retries;desc=Retry count on transient failures;default=3;min=0;max=10"`
	Instructions    string                     `yaml:"instructions,omitempty" json:"instructions,omitempty" schema:"label=Instructions;desc=Global instructions included in all agent prompts;type=textarea"`
	OptimizePrompts bool                       `yaml:"optimize_prompts,omitempty" json:"optimize_prompts,omitempty" schema:"label=Optimize Prompts;desc=Optimize prompts for token efficiency;default=false"`
	Steps           map[string]StepAgentConfig `yaml:"steps,omitempty" json:"steps,omitempty" schema:"-"`
	PRReview        *PRReviewConfig            `yaml:"pr_review,omitempty" json:"pr_review,omitempty" schema:"-"`
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
	AutoInit             bool             `yaml:"auto_init" json:"auto_init" schema:"label=Auto Init;desc=Auto-initialize workspace;default=true;simple"`
	SessionRetentionDays int              `yaml:"session_retention_days" json:"session_retention_days" schema:"label=Session Retention;desc=How long to keep session logs in days;default=30;min=1;max=365"`
	DeleteWorkOnFinish   bool             `yaml:"delete_work_on_finish" json:"delete_work_on_finish" schema:"label=Delete Work on Finish;desc=Clean up work directory after finish;default=false"`
	DeleteWorkOnAbandon  bool             `yaml:"delete_work_on_abandon" json:"delete_work_on_abandon" schema:"label=Delete Work on Abandon;desc=Clean up work directory on abandon;default=true"`
	PreferLocalMerge     bool             `yaml:"prefer_local_merge" json:"prefer_local_merge" schema:"label=Prefer Local Merge;desc=Use local merge instead of creating PR on finish;default=false"`
	Simplify             SimplifySettings `yaml:"simplify,omitempty" json:"simplify,omitempty" schema:"-"`
}

// BudgetSettings holds budget configuration for costs and tokens.
type BudgetSettings struct {
	Enabled       bool                  `yaml:"enabled,omitempty" json:"enabled,omitempty" schema:"label=Enable Budget;desc=Track costs per task and monthly;default=false"`
	PerTask       BudgetConfig          `yaml:"per_task,omitempty" json:"per_task,omitempty" schema:"-"`
	Monthly       MonthlyBudgetSettings `yaml:"monthly,omitempty" json:"monthly,omitempty" schema:"-"`
	ExchangeRates map[string]float64    `yaml:"exchange_rates,omitempty" json:"exchange_rates,omitempty" schema:"-"`
}

// MonthlyBudgetSettings defines a workspace monthly budget.
type MonthlyBudgetSettings struct {
	MaxCost   float64 `yaml:"max_cost,omitempty" json:"max_cost,omitempty" schema:"label=Monthly Max Cost;desc=Maximum monthly spend in USD;min=0"`
	Currency  string  `yaml:"currency,omitempty" json:"currency,omitempty" schema:"label=Currency;desc=Currency code;default=USD;placeholder=USD"`
	WarningAt float64 `yaml:"warning_at,omitempty" json:"warning_at,omitempty" schema:"label=Warning At;desc=Percentage to warn at (0-1);default=0.8;min=0;max=1"`
}

// UpdateSettings holds update-related configuration.
type UpdateSettings struct {
	Enabled       bool `yaml:"enabled" json:"enabled" schema:"label=Enable Updates;desc=Enable automatic update checks;default=true;simple"`
	CheckInterval int  `yaml:"check_interval" json:"check_interval" schema:"label=Check Interval;desc=Hours between update checks;default=24;min=1;max=168"`
}

// StorageSettings holds storage-related configuration.
type StorageSettings struct {
	HomeDir       string `yaml:"home_dir,omitempty" json:"home_dir,omitempty" schema:"label=Home Directory;desc=Override mehrhof home directory;placeholder=~/.mehrhof;advanced"`
	SaveInProject bool   `yaml:"save_in_project,omitempty" json:"save_in_project,omitempty" schema:"label=Save in Project;desc=Store work in project directory instead of global;default=false"`
	ProjectDir    string `yaml:"project_dir,omitempty" json:"project_dir,omitempty" schema:"label=Project Directory;desc=Project dir for work files;default=.mehrhof/work;showWhen=storage.save_in_project:true"`
}

// ProjectSettings holds project-level settings for decoupled hub/code workflows.
type ProjectSettings struct {
	CodeDir string `yaml:"code_dir,omitempty" json:"code_dir,omitempty" schema:"label=Code Directory;desc=Separate code directory (relative or absolute);placeholder=../code"`
}

// StackSettings holds stacked feature branch configuration.
type StackSettings struct {
	AutoRebase       string `yaml:"auto_rebase,omitempty" json:"auto_rebase,omitempty" schema:"label=Auto Rebase;desc=When to auto-rebase children;type=select;options=disabled,on_finish;default=disabled;advanced"`
	BlockOnConflicts bool   `yaml:"block_on_conflicts,omitempty" json:"block_on_conflicts,omitempty" schema:"label=Block on Conflicts;desc=Block auto-rebase if conflicts detected;default=true;showWhen=stack.auto_rebase:on_finish;advanced"`
}

// SpecificationSettings holds specification-related configuration.
type SpecificationSettings struct {
	FilenamePattern string `yaml:"filename_pattern" json:"filename_pattern" schema:"label=Spec Filename Pattern;desc=Pattern for spec filenames;default=specification-{n}.md;advanced"`
}

// ReviewSettings holds code review output configuration.
type ReviewSettings struct {
	FilenamePattern string `yaml:"filename_pattern" json:"filename_pattern" schema:"label=Review Filename Pattern;desc=Pattern for review filenames;default=review-{n}.txt;advanced"`
}

// SandboxSettings holds agent sandboxing configuration.
type SandboxSettings struct {
	Enabled bool     `yaml:"enabled,omitempty" json:"enabled,omitempty" schema:"label=Enable Sandbox;desc=Enable agent sandboxing;default=false"`
	Network bool     `yaml:"network,omitempty" json:"network,omitempty" schema:"label=Allow Network;desc=Allow network access (LLM APIs need this);default=true"`
	TmpDir  string   `yaml:"tmp_dir,omitempty" json:"tmp_dir,omitempty" schema:"label=Temp Directory;desc=Tmpfs mount path;placeholder=auto;advanced"`
	Tools   []string `yaml:"tools,omitempty" json:"tools,omitempty" schema:"-"`
}

// ProvidersSettings holds provider-related configuration.
type ProvidersSettings struct {
	Default        string `yaml:"default,omitempty" json:"default,omitempty" schema:"label=Default Provider;desc=Default provider for bare references;type=select;options=file,directory,github,gitlab,jira,linear,notion"`
	DefaultMention string `yaml:"default_mention,omitempty" json:"default_mention,omitempty" schema:"label=Default Mention;desc=Default mention text when submitting tasks;placeholder=@manager please review"`
}

// LabelDefinition defines a label with optional color.
type LabelDefinition struct {
	Name  string `yaml:"name" json:"name"`                       // Label name (e.g., "priority:high")
	Color string `yaml:"color,omitempty" json:"color,omitempty"` // Optional CSS color class (overrides hash-based color)
}

// LabelSettings holds label-related configuration.
type LabelSettings struct {
	Enabled     bool              `yaml:"enabled,omitempty" json:"enabled,omitempty" schema:"label=Enable Labels;desc=Enable label system;default=true"`
	Defined     []LabelDefinition `yaml:"defined,omitempty" json:"defined,omitempty" schema:"-"`
	Suggestions []string          `yaml:"suggestions,omitempty" json:"suggestions,omitempty" schema:"-"`
}

// QualitySettings holds code quality and linter configuration.
type QualitySettings struct {
	Enabled     bool                    `yaml:"enabled,omitempty" json:"enabled,omitempty" schema:"label=Enable Quality;desc=Enable code quality checks;default=true"`
	UseDefaults bool                    `yaml:"use_defaults,omitempty" json:"use_defaults,omitempty" schema:"label=Use Defaults;desc=Auto-enable default linters;default=false"`
	Linters     map[string]LinterConfig `yaml:"linters,omitempty" json:"linters,omitempty" schema:"-"`
}

// LinterConfig defines configuration for a single linter.
type LinterConfig struct {
	Enabled    bool     `yaml:"enabled,omitempty" json:"enabled,omitempty"`       // Enable/disable this linter (default: true for built-ins)
	Command    []string `yaml:"command,omitempty" json:"command,omitempty"`       // Custom command (e.g., ["vendor/bin/phpstan", "analyse"])
	Args       []string `yaml:"args,omitempty" json:"args,omitempty"`             // Additional arguments
	Extensions []string `yaml:"extensions,omitempty" json:"extensions,omitempty"` // File extensions to lint (default: auto-detected)
}

// DisplaySettings holds display preferences for dates and times.
type DisplaySettings struct {
	Timezone string `yaml:"timezone,omitempty" json:"timezone,omitempty" schema:"label=Timezone;desc=IANA timezone for display (e.g., Europe/Riga);default=UTC;placeholder=UTC;simple"`
}
