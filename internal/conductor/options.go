package conductor

import (
	"io"
	"os"
	"time"

	"github.com/valksor/go-mehrhof/internal/browser"
)

// Options configures the Conductor.
type Options struct {
	// Agent configuration
	AgentName  string            // Which agent to use for all steps (default: auto-detect)
	StepAgents map[string]string // Per-step agent overrides (e.g., {"planning": "glm", "implementing": "claude"})
	Timeout    time.Duration     // Agent execution timeout

	// Behavior
	DryRun       bool // If true, don't apply file changes
	Verbose      bool // Enable verbose output
	CreateBranch bool // Create git branch for task
	UseWorktree  bool // Create git worktree for task
	StashChanges bool // Stash uncommitted changes before creating branch
	AutoPopStash bool // Automatically pop stash after branch creation (default: true)
	AutoInit     bool // Auto-initialize workspace if needed

	// Auto mode (full automation)
	AutoMode           bool // Enable full automation mode
	SkipAgentQuestions bool // Skip agent questions, proceed with best guess
	MaxQualityRetries  int  // Max retries for quality loop (default: 3)

	// Context preservation
	IncludeFullContext bool // Include full exploration context from pending question (default: summary only)

	// Planning behavior
	UseDefaults bool // Use default answers for unknowns without asking user (default: false, ask user)

	// Prompt optimization
	OptimizePrompts bool // Optimize prompts before sending to working agent

	// Component filtering
	OnlyComponent string // Only implement this component (e.g., "backend", "frontend", "tests")
	ParallelCount string // Parallel execution: count (N) or comma-separated agents

	// Output
	Stdout io.Writer // Where to write output (default: os.Stdout)
	Stderr io.Writer // Where to write errors (default: os.Stderr)

	// Paths
	WorkDir string // Working directory (default: current dir)
	HomeDir string // Override for mehrhof home directory (default: user home, for testing)

	// Provider configuration
	DefaultProvider string // Default provider for bare references (e.g., "file")

	// Browser configuration
	BrowserConfig *browser.Config // Browser automation configuration (nil = disabled)

	// Sandbox configuration
	SandboxEnabled bool // Enable sandbox for agent execution

	// Hierarchical context (overrides workspace config)
	WithParent   *bool // Include parent task context (nil = use config)
	WithSiblings *bool // Include sibling subtask context (nil = use config)
	MaxSiblings  *int  // Maximum sibling tasks to include (nil = use config)

	// Library documentation auto-include
	LibraryAutoInclude bool // Automatically include relevant library docs based on file paths

	// Naming overrides (CLI flags)
	ExternalKey           string // Override external key (e.g., "FEATURE-123")
	TitleOverride         string // Override task title
	SlugOverride          string // Override branch slug
	CommitPrefixTemplate  string // Override commit prefix template (e.g., "[{key}]")
	BranchPatternTemplate string // Override branch pattern template (e.g., "{type}/{key}--{slug}")

	// Stacked features
	DependsOn string // Parent task ID for stacked features (branch from parent's branch)

	// Callbacks
	OnStateChange func(from, to string)
	OnProgress    func(message string, percent int)
	OnError       func(err error)
}

// Option is a functional option for configuring Conductor.
type Option func(*Options)

// DefaultOptions returns default options.
func DefaultOptions() Options {
	return Options{
		AgentName:         "", // Auto-detect
		Timeout:           30 * time.Minute,
		DryRun:            false,
		Verbose:           false,
		Stdout:            os.Stdout,
		Stderr:            os.Stderr,
		WorkDir:           ".",
		MaxQualityRetries: 3,
		AutoPopStash:      true, // Default to true for better UX when stashing
	}
}

// WithAgent sets the agent name for all steps.
func WithAgent(name string) Option {
	return func(o *Options) {
		o.AgentName = name
	}
}

// WithStepAgent sets a specific agent for a workflow step.
func WithStepAgent(step, agentName string) Option {
	return func(o *Options) {
		if o.StepAgents == nil {
			o.StepAgents = make(map[string]string)
		}
		o.StepAgents[step] = agentName
	}
}

// WithTimeout sets the execution timeout.
func WithTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.Timeout = d
	}
}

// WithDryRun enables dry-run mode.
func WithDryRun(enabled bool) Option {
	return func(o *Options) {
		o.DryRun = enabled
	}
}

// WithVerbose enables verbose output.
func WithVerbose(enabled bool) Option {
	return func(o *Options) {
		o.Verbose = enabled
	}
}

// WithCreateBranch enables git branch creation.
func WithCreateBranch(enabled bool) Option {
	return func(o *Options) {
		o.CreateBranch = enabled
	}
}

// WithUseWorktree enables git worktree creation.
func WithUseWorktree(enabled bool) Option {
	return func(o *Options) {
		o.UseWorktree = enabled
		// Worktree implies branch creation
		if enabled {
			o.CreateBranch = true
		}
	}
}

// WithStashChanges enables stashing uncommitted changes before branch creation.
func WithStashChanges(enabled bool) Option {
	return func(o *Options) {
		o.StashChanges = enabled
	}
}

// WithAutoPopStash configures whether to automatically pop stash after branch creation.
func WithAutoPopStash(enabled bool) Option {
	return func(o *Options) {
		o.AutoPopStash = enabled
	}
}

// WithAutoInit enables auto-initialization of workspace.
func WithAutoInit(enabled bool) Option {
	return func(o *Options) {
		o.AutoInit = enabled
	}
}

// WithAutoMode enables full automation mode.
func WithAutoMode(enabled bool) Option {
	return func(o *Options) {
		o.AutoMode = enabled
		if enabled {
			o.SkipAgentQuestions = true
		}
	}
}

// WithSkipAgentQuestions skips pending questions from agents.
func WithSkipAgentQuestions(enabled bool) Option {
	return func(o *Options) {
		o.SkipAgentQuestions = enabled
	}
}

// WithMaxQualityRetries sets max retries for quality loop.
func WithMaxQualityRetries(n int) Option {
	return func(o *Options) {
		o.MaxQualityRetries = n
	}
}

// WithIncludeFullContext enables including full exploration context from pending question.
func WithIncludeFullContext(enabled bool) Option {
	return func(o *Options) {
		o.IncludeFullContext = enabled
	}
}

// WithUseDefaults enables using default answers for unknowns without asking user.
func WithUseDefaults(enabled bool) Option {
	return func(o *Options) {
		o.UseDefaults = enabled
	}
}

// WithOptimizePrompts enables prompt optimization before execution.
func WithOptimizePrompts(enabled bool) Option {
	return func(o *Options) {
		o.OptimizePrompts = enabled
	}
}

// WithStdout sets the stdout writer.
func WithStdout(w io.Writer) Option {
	return func(o *Options) {
		o.Stdout = w
	}
}

// WithStderr sets the stderr writer.
func WithStderr(w io.Writer) Option {
	return func(o *Options) {
		o.Stderr = w
	}
}

// WithWorkDir sets the working directory.
func WithWorkDir(dir string) Option {
	return func(o *Options) {
		o.WorkDir = dir
	}
}

// WithDefaultProvider sets the default provider for bare references.
func WithDefaultProvider(provider string) Option {
	return func(o *Options) {
		o.DefaultProvider = provider
	}
}

// WithExternalKey sets the external key override for branch/commit naming.
func WithExternalKey(key string) Option {
	return func(o *Options) {
		o.ExternalKey = key
	}
}

// WithCommitPrefixTemplate sets the commit prefix template override.
func WithCommitPrefixTemplate(template string) Option {
	return func(o *Options) {
		o.CommitPrefixTemplate = template
	}
}

// WithBranchPatternTemplate sets the branch pattern template override.
func WithBranchPatternTemplate(template string) Option {
	return func(o *Options) {
		o.BranchPatternTemplate = template
	}
}

// WithTitleOverride sets the task title override.
func WithTitleOverride(title string) Option {
	return func(o *Options) {
		o.TitleOverride = title
	}
}

// WithSlugOverride sets the branch slug override.
func WithSlugOverride(slug string) Option {
	return func(o *Options) {
		o.SlugOverride = slug
	}
}

// WithDependsOn sets the parent task ID for stacked features.
// The new task will branch from the parent's branch instead of the target branch.
func WithDependsOn(taskID string) Option {
	return func(o *Options) {
		o.DependsOn = taskID
	}
}

// WithStateChangeCallback sets the state change callback.
func WithStateChangeCallback(fn func(from, to string)) Option {
	return func(o *Options) {
		o.OnStateChange = fn
	}
}

// WithProgressCallback sets the progress callback.
func WithProgressCallback(fn func(message string, percent int)) Option {
	return func(o *Options) {
		o.OnProgress = fn
	}
}

// WithErrorCallback sets the error callback.
func WithErrorCallback(fn func(err error)) Option {
	return func(o *Options) {
		o.OnError = fn
	}
}

// WithHomeDir sets the mehrhof home directory override (for testing).
func WithHomeDir(dir string) Option {
	return func(o *Options) {
		o.HomeDir = dir
	}
}

// WithBrowserConfig sets the browser configuration.
func WithBrowserConfig(cfg browser.Config) Option {
	return func(o *Options) {
		o.BrowserConfig = &cfg
	}
}

// WithSandbox enables sandbox for agent execution.
func WithSandbox(enabled bool) Option {
	return func(o *Options) {
		o.SandboxEnabled = enabled
	}
}

// WithOnlyComponent sets component filtering to only implement one component.
func WithOnlyComponent(component string) Option {
	return func(o *Options) {
		o.OnlyComponent = component
	}
}

// WithParallel sets parallel execution mode (count or comma-separated agents).
func WithParallel(parallel string) Option {
	return func(o *Options) {
		o.ParallelCount = parallel
	}
}

// WithParent sets whether to include parent task context (overrides workspace config).
func WithParent(enabled bool) Option {
	return func(o *Options) {
		o.WithParent = &enabled
	}
}

// WithoutParent disables parent task context (overrides workspace config).
func WithoutParent() Option {
	return func(o *Options) {
		disabled := false
		o.WithParent = &disabled
	}
}

// WithSiblings sets whether to include sibling subtask context (overrides workspace config).
func WithSiblings(enabled bool) Option {
	return func(o *Options) {
		o.WithSiblings = &enabled
	}
}

// WithoutSiblings disables sibling subtask context (overrides workspace config).
func WithoutSiblings() Option {
	return func(o *Options) {
		disabled := false
		o.WithSiblings = &disabled
	}
}

// WithMaxSiblings sets the maximum number of sibling tasks to include (overrides workspace config).
func WithMaxSiblings(maxSiblings int) Option {
	return func(o *Options) {
		o.MaxSiblings = &maxSiblings
	}
}

// WithLibraryAutoInclude enables automatic inclusion of relevant library documentation.
// When enabled, library docs matching the current file paths are automatically added to prompts.
func WithLibraryAutoInclude(enabled bool) Option {
	return func(o *Options) {
		o.LibraryAutoInclude = enabled
	}
}

// Apply applies options to the Options struct.
func (o *Options) Apply(opts ...Option) {
	for _, opt := range opts {
		opt(o)
	}
}

// FinishOptions configures the finish operation.
type FinishOptions struct {
	SquashMerge  bool   // Use squash merge
	DeleteBranch bool   // Delete branch after merge
	TargetBranch string // Branch to merge into
	PushAfter    bool   // Push after merge
	DeleteWork   *bool  // Delete work directory: nil=defer to config, true=delete, false=keep

	// PR-related options (for GitHub provider)
	ForceMerge bool   // Force local merge instead of PR creation
	DraftPR    bool   // Create PR as draft
	PRTitle    string // Custom PR title (defaults to task title)
	PRBody     string // Custom PR body

	// Commit message
	CommitMessage string // Optional pre-generated commit message
}

// DefaultFinishOptions returns default finish options.
func DefaultFinishOptions() FinishOptions {
	return FinishOptions{
		SquashMerge:  false,
		DeleteBranch: false, // Don't delete by default
		TargetBranch: "",    // Auto-detect base branch
		PushAfter:    false, // Don't push by default
		DeleteWork:   nil,   // Defer to config (default: keep)
		ForceMerge:   false, // Create PR by default if supported
		DraftPR:      false,
	}
}

// DeleteOptions configures the delete (abandon) operation.
type DeleteOptions struct {
	Force      bool  // Skip confirmation prompt
	KeepBranch bool  // Keep the git branch (only delete workspace)
	DeleteWork *bool // Delete work directory: nil=defer to config, true=delete, false=keep
}

// DefaultDeleteOptions returns default delete options.
func DefaultDeleteOptions() DeleteOptions {
	return DeleteOptions{
		Force:      false,
		KeepBranch: false,
		DeleteWork: nil, // Defer to config (default: delete)
	}
}

// BoolPtr returns a pointer to the given bool value.
// Useful for setting DeleteWork in FinishOptions/DeleteOptions.
func BoolPtr(b bool) *bool {
	return &b
}
