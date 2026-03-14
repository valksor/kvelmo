// Package settings provides two-tier configuration management with schema-driven UI generation.
//
// Configuration is stored in two locations:
//   - Global: ~/.valksor/kvelmo/kvelmo.yaml (lower priority)
//   - Project: .valksor/kvelmo.yaml (higher priority, overrides global)
//
// Sensitive fields (tokens) are stored in .env files next to the config:
//   - Global: ~/.valksor/kvelmo/.env
//   - Project: .valksor/.env
//
// Schema tags on struct fields drive automatic React form generation.
package settings

// Settings represents the complete configuration for kvelmo.
// Project settings override global settings when both are present.
type Settings struct {
	Agent        AgentSettings          `yaml:"agent,omitempty" json:"agent,omitempty"`
	Providers    ProviderSettings       `yaml:"providers,omitempty" json:"providers,omitempty"`
	Git          GitSettings            `yaml:"git,omitempty" json:"git,omitempty"`
	Workers      WorkerSettings         `yaml:"workers,omitempty" json:"workers,omitempty"`
	Storage      StorageSettings        `yaml:"storage,omitempty" json:"storage,omitempty"`
	Workflow     WorkflowSettings       `yaml:"workflow,omitempty" json:"workflow,omitempty"`
	Watchdog     WatchdogSettings       `yaml:"watchdog,omitempty" json:"watchdog,omitempty"`
	UI           UISettings             `yaml:"ui,omitempty" json:"ui,omitempty"`
	Environment  string                 `yaml:"environment,omitempty" json:"environment,omitempty" schema:"label=Environment;desc=Deployment environment (dev, staging, prod);options=dev|staging|prod;default=dev"`
	CustomAgents map[string]CustomAgent `yaml:"custom_agents,omitempty" json:"custom_agents,omitempty"`
}

// UISettings configures UI state that persists across sessions.
type UISettings struct {
	OnboardingDismissed bool `yaml:"onboarding_dismissed,omitempty" json:"onboarding_dismissed,omitempty"`
}

// AgentSettings configures AI agent behavior.
type AgentSettings struct {
	Default string   `yaml:"default,omitempty" json:"default,omitempty" schema:"label=Default Agent;desc=Agent used when none specified;options=claude|codex"`
	Allowed []string `yaml:"allowed,omitempty" json:"allowed,omitempty" schema:"label=Allowed Agents;desc=Agents permitted for this project;type=multiselect;options=claude|codex"`
}

// CustomAgent defines a user-created agent that wraps a base agent.
// Custom agents are stored in settings.yaml under custom_agents map.
// They automatically appear in agent selection dropdowns in the UI.
type CustomAgent struct {
	Extends     string            `yaml:"extends" json:"extends" schema:"label=Base Agent;desc=Agent to wrap;options=claude|codex;required"`
	Description string            `yaml:"description,omitempty" json:"description,omitempty" schema:"label=Description;desc=Human-readable description"`
	Args        []string          `yaml:"args,omitempty" json:"args,omitempty" schema:"label=CLI Arguments;desc=Additional arguments passed to agent;type=tags"`
	Env         map[string]string `yaml:"env,omitempty" json:"env,omitempty" schema:"label=Environment;desc=Environment variables for this agent;type=keyvalue"`
}

// ProviderSettings configures task providers (GitHub, GitLab, etc.).
type ProviderSettings struct {
	Default string       `yaml:"default,omitempty" json:"default,omitempty" schema:"label=Default Provider;desc=Provider used when none specified;options=github|gitlab|wrike|linear|file"`
	GitHub  GitHubConfig `yaml:"github,omitempty" json:"github,omitempty"`
	GitLab  GitLabConfig `yaml:"gitlab,omitempty" json:"gitlab,omitempty"`
	Wrike   WrikeConfig  `yaml:"wrike,omitempty" json:"wrike,omitempty"`
	Linear  LinearConfig `yaml:"linear,omitempty" json:"linear,omitempty"`
}

// GitHubConfig configures the GitHub provider.
type GitHubConfig struct {
	// Token is stored in .env as GITHUB_TOKEN, not in settings.yaml.
	// The yaml:"-" tag prevents yaml serialization.
	// The env=GITHUB_TOKEN tells the system which env var to use.
	Token              string `yaml:"-" json:"token,omitempty" schema:"label=Token;desc=Personal access token (repo, workflow scopes);sensitive;env=GITHUB_TOKEN;helpUrl=https://github.com/settings/tokens"`
	Owner              string `yaml:"owner,omitempty" json:"owner,omitempty" schema:"label=Owner;desc=Default repository owner;placeholder=auto-detect from git remote"`
	AllowTicketComment bool   `yaml:"allow_ticket_comment,omitempty" json:"allow_ticket_comment,omitempty" schema:"label=Allow Ticket Comments;desc=Post status comments on GitHub issues"`
}

// GitLabConfig configures the GitLab provider.
type GitLabConfig struct {
	Token              string `yaml:"-" json:"token,omitempty" schema:"label=Token;desc=Personal access token (api scope);sensitive;env=GITLAB_TOKEN;helpUrl=https://gitlab.com/-/user_settings/personal_access_tokens"`
	BaseURL            string `yaml:"base_url,omitempty" json:"base_url,omitempty" schema:"label=Base URL;desc=GitLab instance URL;default=https://gitlab.com;placeholder=https://gitlab.com"`
	AllowTicketComment bool   `yaml:"allow_ticket_comment,omitempty" json:"allow_ticket_comment,omitempty" schema:"label=Allow Ticket Comments;desc=Post status comments on GitLab issues"`
}

// WrikeConfig configures the Wrike provider.
type WrikeConfig struct {
	Token string `yaml:"-" json:"token,omitempty" schema:"label=Token;desc=Wrike API token;sensitive;env=WRIKE_TOKEN;helpUrl=https://www.wrike.com/frontend/apps/index.html#/api"`
	// IncludeParentContext and IncludeSiblingContext have default=true,
	// so we must NOT use omitempty (it drops false values on serialize).
	IncludeParentContext  bool `yaml:"include_parent_context" json:"include_parent_context" schema:"label=Include Parent Context;desc=Fetch parent task and include its context in AI prompts;default=true"`
	IncludeSiblingContext bool `yaml:"include_sibling_context" json:"include_sibling_context" schema:"label=Include Sibling Context;desc=Fetch sibling tasks and include them in AI prompts to avoid duplication;default=true"`
	// AllowTicketComment controls whether status comments are posted to Wrike tasks.
	AllowTicketComment bool `yaml:"allow_ticket_comment,omitempty" json:"allow_ticket_comment,omitempty" schema:"label=Allow Ticket Comments;desc=Post status comments on Wrike tasks"`
}

// LinearConfig configures the Linear provider.
type LinearConfig struct {
	// Token is stored in .env as LINEAR_TOKEN, not in settings.yaml.
	Token string `yaml:"-" json:"token,omitempty" schema:"label=Token;desc=Linear API token;sensitive;env=LINEAR_TOKEN;helpUrl=https://linear.app/settings/api"`
	// Team is the default team key (e.g. "ENG") used when creating tasks or listing.
	Team string `yaml:"team,omitempty" json:"team,omitempty" schema:"label=Default Team;desc=Default team key (e.g. ENG);placeholder=auto-detect"`
	// IncludeParentContext controls whether the parent issue is fetched and
	// included in AI prompts when planning/implementing a Linear task.
	// Has default=true, so we must NOT use omitempty (it drops false values on serialize).
	IncludeParentContext bool `yaml:"include_parent_context" json:"include_parent_context" schema:"label=Include Parent Context;desc=Fetch parent issue and include its context in AI prompts;default=true"`
	// IncludeSiblingContext controls whether sibling tasks (sub-issues of the
	// same parent) are fetched and included in AI prompts. Up to 5 siblings
	// are included.
	// Has default=true, so we must NOT use omitempty (it drops false values on serialize).
	IncludeSiblingContext bool `yaml:"include_sibling_context" json:"include_sibling_context" schema:"label=Include Sibling Context;desc=Fetch sibling issues and include them in AI prompts;default=true"`
	// AllowTicketComment controls whether status comments are posted to Linear issues.
	AllowTicketComment bool `yaml:"allow_ticket_comment,omitempty" json:"allow_ticket_comment,omitempty" schema:"label=Allow Ticket Comments;desc=Post status comments on Linear issues"`
}

// GitSettings configures git behavior for the workflow.
// Pointer bools allow project-level false to override global-level true.
type GitSettings struct {
	BaseBranch     string `yaml:"base_branch,omitempty" json:"base_branch,omitempty" schema:"label=Base Branch;desc=Default branch for PRs (auto-detected from git remote if empty);placeholder=auto-detect"`
	BranchPattern  string `yaml:"branch_pattern,omitempty" json:"branch_pattern,omitempty" schema:"label=Branch Pattern;desc=Pattern for branch names. Variables: {key}, {type}, {slug};default=feature/{key}--{slug}"`
	CommitPrefix   string `yaml:"commit_prefix,omitempty" json:"commit_prefix,omitempty" schema:"label=Commit Prefix;desc=Pattern for commit messages. Variables: {key};default=[{key}]"`
	CreateBranch   *bool  `yaml:"create_branch,omitempty" json:"create_branch,omitempty" schema:"label=Create Branch;desc=Automatically create a branch when starting a task. If the branch already exists, switches to it;default=true"`
	AutoCommit     *bool  `yaml:"auto_commit,omitempty" json:"auto_commit,omitempty" schema:"label=Auto Commit;desc=Automatically commit after implementation;default=true"`
	SignCommits    *bool  `yaml:"sign_commits,omitempty" json:"sign_commits,omitempty" schema:"label=Sign Commits;desc=GPG sign commits;showWhen=git.auto_commit:true"`
	AllowPRComment *bool  `yaml:"allow_pr_comment,omitempty" json:"allow_pr_comment,omitempty" schema:"label=Allow PR Comments;desc=Post status comments on pull requests after submit"`
}

// WorkerSettings configures the worker pool.
type WorkerSettings struct {
	Max int `yaml:"max,omitempty" json:"max,omitempty" schema:"label=Max Workers;desc=Maximum concurrent workers;default=3;min=1;max=10"`
}

// StorageSettings configures where specs, plans, reviews, and chat are stored.
type StorageSettings struct {
	SaveInProject *bool             `yaml:"save_in_project,omitempty" json:"save_in_project,omitempty" schema:"label=Save in Project;desc=Store specs/plans/chat in .valksor/ instead of home (~/.valksor/kvelmo/);default=false"`
	Recording     RecordingSettings `yaml:"recording,omitempty" json:"recording,omitempty"`
}

// RecordingSettings configures agent interaction recording.
type RecordingSettings struct {
	Enabled bool   `yaml:"enabled,omitempty" json:"enabled,omitempty" schema:"label=Enable Recording;desc=Record agent interactions to JSONL files for debugging and replay;default=false"`
	Dir     string `yaml:"dir,omitempty" json:"dir,omitempty" schema:"label=Recording Directory;desc=Directory for recording files (default: ~/.valksor/kvelmo/recordings)"`
}

// CodeRabbitMode controls when the CodeRabbit CLI runs in the quality gate.
type CodeRabbitMode string

const (
	CodeRabbitModeAsk    CodeRabbitMode = "ask"    // Prompt user before running (default)
	CodeRabbitModeAlways CodeRabbitMode = "always" // Always run without prompting
	CodeRabbitModeNever  CodeRabbitMode = "never"  // Skip CodeRabbit entirely
)

// CodeRabbitConfig configures CodeRabbit CLI integration in the quality gate.
type CodeRabbitConfig struct {
	Mode CodeRabbitMode `yaml:"mode,omitempty" json:"mode,omitempty" schema:"label=Mode;desc=When to run CodeRabbit CLI review;options=ask|always|never;default=ask"`
}

// WorkflowSettings contains per-project workflow options.
// These are intentionally project-scoped and not meaningful at global level.
type WorkflowSettings struct {
	UseWorktreeIsolation *bool            `yaml:"use_worktree_isolation,omitempty" json:"use_worktree_isolation,omitempty" schema:"label=Use Worktree Isolation;desc=Create an isolated git worktree for each task, enabling parallel work without conflicts;default=true"`
	CodeRabbit           CodeRabbitConfig `yaml:"coderabbit,omitempty" json:"coderabbit,omitempty" schema:"label=CodeRabbit;desc=CodeRabbit CLI review integration"`
}

// WatchdogSettings configures the memory leak watchdog.
type WatchdogSettings struct {
	Enabled     bool `yaml:"enabled,omitempty" json:"enabled,omitempty" schema:"label=Enable Watchdog;desc=Monitor heap growth and trigger graceful shutdown on confirmed memory leak;default=true"`
	IntervalSec int  `yaml:"interval_sec,omitempty" json:"interval_sec,omitempty" schema:"label=Interval (sec);desc=How often to sample heap usage (min 10s — ReadMemStats stops the world briefly);default=30;min=10;max=300;advanced"`
	WindowSize  int  `yaml:"window_size,omitempty" json:"window_size,omitempty" schema:"label=Window Size;desc=Number of consecutive samples required to confirm a leak;default=10;min=5;max=60;advanced"`
	ThresholdMB int  `yaml:"threshold_mb,omitempty" json:"threshold_mb,omitempty" schema:"label=Threshold (MB);desc=Total heap growth over the window before triggering;default=200;min=50;advanced"`
}

// Scope represents where settings are stored/loaded from.
type Scope string

const (
	// ScopeGlobal refers to ~/.valksor/kvelmo/kvelmo.yaml.
	ScopeGlobal Scope = "global"
	// ScopeProject refers to .valksor/kvelmo.yaml.
	ScopeProject Scope = "project"
)

// SettingsResponse is returned by the settings.get socket handler.
type SettingsResponse struct {
	Schema    *Schema   `json:"schema"`    // Generated schema for UI rendering
	Effective *Settings `json:"effective"` // Merged global + project settings
	Global    *Settings `json:"global"`    // Global-only values
	Project   *Settings `json:"project"`   // Project-only overrides
}

// SettingsSetParams is the input for the settings.set socket handler.
type SettingsSetParams struct {
	Scope  Scope          `json:"scope"`  // "global" or "project"
	Values map[string]any `json:"values"` // Dot-notation paths to values
}

// Schema types for UI generation

// FieldType represents the type of a schema field for UI rendering.
type FieldType string

const (
	TypeString   FieldType = "string"
	TypeBoolean  FieldType = "boolean"
	TypeNumber   FieldType = "number"
	TypeSelect   FieldType = "select"
	TypeTextarea FieldType = "textarea"
	TypePassword FieldType = "password"
	TypeTags     FieldType = "tags"     // For []string fields
	TypeKeyValue FieldType = "keyvalue" // For map[string]string fields
	TypeList     FieldType = "list"     // For dynamic lists (e.g., custom_agents)
)

// SelectOption represents a choice in a select/dropdown field.
type SelectOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// ValidationRules defines validation constraints for a field.
type ValidationRules struct {
	Required       bool   `json:"required,omitempty"`
	Min            *int   `json:"min,omitempty"`
	Max            *int   `json:"max,omitempty"`
	MaxLength      *int   `json:"maxLength,omitempty"`
	Pattern        string `json:"pattern,omitempty"`
	PatternMessage string `json:"patternMessage,omitempty"`
}

// Condition defines when a field should be visible.
type Condition struct {
	Field     string `json:"field"`               // Path to the controlling field
	Equals    any    `json:"equals,omitempty"`    // Show when field equals this value
	NotEquals any    `json:"notEquals,omitempty"` // Show when field does not equal this value
}

// Field represents a single configuration field in the schema.
type Field struct {
	Path        string           `json:"path"`                  // Dot-notation path: "git.commit_prefix"
	Type        FieldType        `json:"type"`                  // UI field type
	Label       string           `json:"label"`                 // Human-readable label
	Description string           `json:"description,omitempty"` // Help text
	Placeholder string           `json:"placeholder,omitempty"` // Input placeholder
	Default     any              `json:"default,omitempty"`     // Default value
	Options     []SelectOption   `json:"options,omitempty"`     // For select/multiselect types
	Multiple    bool             `json:"multiple,omitempty"`    // True for multiselect fields (renders checkboxes)
	ItemSchema  []Field          `json:"itemSchema,omitempty"`  // For list type - schema of each list item
	Validation  *ValidationRules `json:"validation,omitempty"`  // Validation constraints
	Sensitive   bool             `json:"sensitive,omitempty"`   // Mask in UI, protect in API
	EnvVar      string           `json:"envVar,omitempty"`      // Environment variable name for sensitive fields
	HelpURL     string           `json:"helpUrl,omitempty"`     // Link to help page (e.g., token setup)
	ShowWhen    *Condition       `json:"showWhen,omitempty"`    // Conditional visibility
	Advanced    bool             `json:"advanced,omitempty"`    // Hide in simple mode
}

// Section groups related fields together in the UI.
type Section struct {
	ID          string  `json:"id"`                    // Unique section identifier
	Title       string  `json:"title"`                 // Display title
	Description string  `json:"description,omitempty"` // Section description
	Icon        string  `json:"icon,omitempty"`        // Icon name (lucide-react)
	Category    string  `json:"category"`              // "core" | "providers" | "features"
	Fields      []Field `json:"fields"`                // Fields in this section
}

// Schema represents the complete settings schema for UI generation.
type Schema struct {
	Version  string    `json:"version"`  // Schema version for compatibility
	Sections []Section `json:"sections"` // All settings sections
}

// SectionMeta holds metadata for a section that is not defined in struct tags.
type SectionMeta struct {
	Title       string
	Description string
	Icon        string
	Category    string // "core" | "providers" | "features"
}
