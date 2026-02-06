package views

import "time"

// PageData contains common data for all pages.
type PageData struct {
	// Mode information
	Mode             string
	IsGlobalMode     bool
	IsProjectMode    bool
	AuthEnabled      bool
	CanSwitchProject bool
	CurrentUser      string
	IsViewer         bool // True if current user has read-only access

	// Project info (when in project mode)
	ProjectName string
	ProjectPath string

	// SSE event names (for templates to reference)
	Events EventNames

	// Flash messages
	Success string
	Error   string
}

// EventNames provides SSE event name constants for templates.
type EventNames struct {
	WorkflowStateChanged string
	SpecUpdated          string
	QuestionAsked        string
	CostsUpdated         string
	TaskCreated          string
	TaskCompleted        string
	BudgetWarning        string
	BudgetLimit          string
	HierarchyUpdated     string
}

// DefaultEventNames returns the standard event names.
func DefaultEventNames() EventNames {
	return EventNames{
		WorkflowStateChanged: EventWorkflowStateChanged,
		SpecUpdated:          EventSpecUpdated,
		QuestionAsked:        EventQuestionAsked,
		CostsUpdated:         EventCostsUpdated,
		TaskCreated:          EventTaskCreated,
		TaskCompleted:        EventTaskCompleted,
		BudgetWarning:        EventBudgetWarning,
		BudgetLimit:          EventBudgetLimit,
		HierarchyUpdated:     EventHierarchyUpdated,
	}
}

// DashboardData contains all data for the main dashboard.
type DashboardData struct {
	PageData

	// Independent sections - nil means render empty state
	Stats          *StatsData
	ActiveWork     *ActiveWorkData
	Actions        []ActionData
	Specifications *SpecificationsData
	Reviews        *ReviewsData
	Question       *QuestionData
	Costs          *CostsData
	Notes          *NotesData
	RecentTasks    []RecentTaskData

	// Global mode specific
	Projects     []ProjectData
	SavedProject *ProjectData
}

// ActiveWorkData represents the currently active work item (task/quick/project).
type ActiveWorkData struct {
	// Core identification
	Type string // WorkTypeTask, WorkTypeQuick, WorkTypeProject
	ID   string
	Ref  string // Provider reference (e.g., "github:123")

	// Display
	Title       string
	Description string
	State       string
	Branch      string
	Worktree    string
	Started     string // Pre-formatted time ago
	StartedAt   time.Time
	Labels      []LabelData

	// Pre-computed display values
	StateIcon  string
	StateBadge string
	StateColor string
	BarColor   string

	// Type-specific flags
	SandboxActive bool // Only for tasks
	HasQuestion   bool
	HasSpecs      bool

	// Optional modifiers (detected from session history)
	IsOptimized  bool
	IsSimplified bool

	// Hierarchical context
	Hierarchy *HierarchyData `json:"hierarchy,omitempty"`
}

// StatsData contains workspace statistics.
type StatsData struct {
	TotalTasks   int
	TotalCost    string // Pre-formatted "$12.34"
	TotalTokens  string // Pre-formatted "1.2M"
	CachedTokens string // Pre-formatted "450K"
	CachedPct    string // Pre-formatted "12%"

	// State breakdown (flat, pre-computed)
	StateLines []StateLineData

	// Monthly budget
	HasMonthly   bool
	MonthlySpent string
	MonthlyMax   string
	MonthlyPct   float64
	MonthlyColor string
	MonthlyMonth string // "January 2025"
}

// StateLineData represents a single state in the stats breakdown.
type StateLineData struct {
	State    string
	Icon     string
	Badge    string
	Count    int
	Percent  string
	Color    string
	BarColor string
}

// ActionData represents a pre-computed action button.
type ActionData struct {
	Command     string
	Label       string
	Endpoint    string
	Method      string
	ButtonClass string
	Icon        string // Optional icon name

	// Behavior modifiers
	Dangerous  bool
	Confirm    string // Confirmation message if dangerous
	HasOptions bool   // Show dropdown instead of direct action
	Disabled   bool
	Tooltip    string
}

// SpecificationsData contains specification list information.
type SpecificationsData struct {
	Items    []SpecItemData
	Total    int
	Done     int
	Progress float64 // 0-100
}

// SpecItemData represents a single specification.
type SpecItemData struct {
	Number      int
	Name        string
	Title       string
	Description string
	Component   string
	Status      string
	CreatedAt   string
	CompletedAt string

	// Pre-computed display
	StatusIcon  string
	StatusColor string
	IsCompleted bool
	IsActive    bool
}

// ReviewsData contains review list information for the dashboard.
type ReviewsData struct {
	Items []ReviewItem
	Total int
}

// ReviewItem represents a single code review.
type ReviewItem struct {
	Number     int
	Status     string // "PASSED", "ISSUES", "PENDING"
	Summary    string
	CreatedAt  string
	IssueCount int

	// Pre-computed display
	StatusIcon  string
	StatusClass string // Tailwind badge class
	HasIssues   bool
}

// QuestionData contains pending agent question information.
type QuestionData struct {
	Question string
	Options  []OptionData
	TaskID   string
	IsViewer bool
}

// OptionData represents a question option.
type OptionData struct {
	Label       string
	Value       string
	Description string
}

// NotesData contains notes for the dashboard.
type NotesData struct {
	Notes []NoteItem
	Count int
}

// NoteItem represents a single note in the dashboard.
type NoteItem struct {
	Number    int
	Timestamp string
	State     string
	Content   string // HTML-rendered markdown
}

// CostsData contains cost and budget information.
type CostsData struct {
	TotalCost    string
	TotalTokens  string
	InputTokens  string
	OutputTokens string
	CachedTokens string
	CachedPct    string

	// Budget information
	HasBudget      bool
	BudgetType     string // "cost" or "tokens"
	BudgetUsed     string
	BudgetMax      string
	BudgetPct      float64
	BudgetColor    string
	BudgetWarned   bool
	BudgetLimitHit bool

	// Per-step breakdown
	Steps []StepCostData
}

// StepCostData represents costs for a workflow step.
type StepCostData struct {
	Name         string
	InputTokens  string
	OutputTokens string
	CachedTokens string
	TotalTokens  string
	Cost         string
	Calls        int
}

// LabelData represents a task label.
type LabelData struct {
	Text  string
	Color string // Full Tailwind class
}

// RecentTaskData represents a task in the recent tasks list.
type RecentTaskData struct {
	ID         string
	ShortID    string
	Title      string
	State      string
	StateIcon  string
	StateColor string
	TimeAgo    string
	Ref        string
}

// ProjectData represents a project for the project picker.
type ProjectData struct {
	ID         string
	Name       string
	Path       string
	RemoteURL  string
	LastAccess string // Pre-formatted time ago
}

// SettingsData contains all data for the settings page.
// This uses interface{} for Config and SandboxStatus to allow passing storage
// types directly, enabling templates to access .Config.Git.AutoCommit etc.
type SettingsData struct {
	PageData

	// Configuration
	ShowSensitive   bool          // True for Project mode, false for Global mode
	Config          interface{}   // *storage.WorkspaceConfig - for template form binding
	Agents          []string      // Available agent names for dropdown
	Projects        []ProjectData // Available projects (global mode) - reuse ProjectData
	SelectedProject string        // Currently selected project ID (global mode)
	SandboxStatus   interface{}   // sandbox.Status - for template binding

	// Project detection for security scanners
	ProjectInfo *ProjectInfoData // Detected project languages and markers

	// Validation errors
	ValidationErrors []ValidationErrorData

	// For future structured editing
	ConfigParsed *ParsedConfig
	AgentDetails []AgentData
	DefaultAgent string
	Providers    []ProviderStatusData
}

// ProjectInfoData contains detected project information for the UI.
type ProjectInfoData struct {
	// Detected languages
	Languages []string

	// Marker files detected
	HasGoMod           bool
	HasPackageJSON     bool
	HasPackageLockJSON bool
	HasYarnLock        bool
	HasTSConfig        bool
	HasPyProjectTOML   bool
	HasRequirementsTXT bool
	HasSetupPy         bool
	HasPipfile         bool
	HasComposerJSON    bool
	HasGemfile         bool
	HasCargoTOML       bool

	// Applicable scanners based on detected languages
	ApplicableScanners []ScannerInfoData
}

// ScannerInfoData describes a security scanner for the UI.
type ScannerInfoData struct {
	Name           string   // Scanner name (e.g., "gosec")
	DisplayName    string   // Human-readable name (e.g., "Gosec")
	Description    string   // Brief description
	Type           string   // "sast", "dependency", "secrets"
	Languages      []string // Languages this scanner supports (empty = all)
	InstallCommand string   // Command to install the scanner
	Requires       string   // What marker file is required (e.g., "package-lock.json")
	AlwaysShow     bool     // Show regardless of detected languages
}

// ParsedConfig represents structured configuration for form editing.
type ParsedConfig struct {
	Agent            AgentConfigData
	Budget           BudgetConfigData
	Git              GitConfigData
	Quality          QualityConfigData
	Providers        []ProviderConfigData
	DefaultProvider  string
	IntegrationToken string
}

// AgentConfigData represents agent configuration.
type AgentConfigData struct {
	Default        string
	PlanningAgent  string
	ImplementAgent string
	ReviewAgent    string
}

// BudgetConfigData represents budget configuration.
type BudgetConfigData struct {
	HasTask       bool
	TaskMaxCost   string
	TaskMaxTokens string
	TaskOnLimit   string
	TaskWarningAt string

	HasMonthly       bool
	MonthlyMaxCost   string
	MonthlyWarningAt string
}

// GitConfigData represents git configuration.
type GitConfigData struct {
	BranchPrefix         string
	CommitPrefix         string
	AutoCommit           bool
	SquashOnFinish       bool
	DeleteBranchOnFinish bool
}

// QualityConfigData represents quality configuration.
type QualityConfigData struct {
	Target      string
	MaxAttempts int
	FailOnError bool
}

// ProviderConfigData represents provider configuration.
type ProviderConfigData struct {
	Scheme    string
	Shorthand string
	Enabled   bool
}

// AgentData represents agent information.
type AgentData struct {
	Name        string
	Type        string
	Description string
	Available   bool
	IsDefault   bool
	IsAlias     bool
	Extends     string
}

// ProviderStatusData represents provider health status.
type ProviderStatusData struct {
	Name        string
	Scheme      string
	Healthy     bool
	Message     string
	LastChecked string
}

// SandboxStatusData represents sandbox environment status.
type SandboxStatusData struct {
	Enabled bool
	Active  bool
	Path    string
}

// ValidationErrorData represents a config validation error.
type ValidationErrorData struct {
	Field   string
	Message string
	Code    string
}

// BrowserData contains all data for the browser page.
type BrowserData struct {
	PageData

	Connected    bool
	Host         string
	Port         int
	Tabs         []BrowserTabData
	ActiveTab    *BrowserTabData
	ErrorMessage string
}

// BrowserTabData represents a browser tab.
type BrowserTabData struct {
	ID     string
	Title  string
	URL    string
	Active bool
}

// MemoryData contains all data for the memory page.
type MemoryData struct {
	PageData

	// Search results
	Results []MemoryResultData
	Query   string

	// Stats
	Stats *MemoryStatsData

	// Whether memory system is available
	Enabled bool
}

// MemoryResultData represents a search result from the memory system.
type MemoryResultData struct {
	TaskID   string
	Type     string
	Score    float64
	Content  string
	Metadata map[string]any
}

// MemoryStatsData represents memory system statistics.
type MemoryStatsData struct {
	TotalDocuments int
	ByType         map[string]int
	Enabled        bool
}

// HistoryData contains all data for the history page.
type HistoryData struct {
	PageData

	Sessions    []SessionData
	TotalCount  int
	CurrentPage int
	TotalPages  int
	HasPrev     bool
	HasNext     bool
}

// AutomationData contains data for the automation page.
type AutomationData struct {
	PageData

	// Status
	Enabled bool
	Running bool
	Workers int

	// Queue statistics
	PendingJobs   int
	RunningJobs   int
	CompletedJobs int
	FailedJobs    int
	CancelledJobs int

	// Job list
	Jobs []AutomationJobData

	// Configuration
	Config AutomationConfigData
}

// AutomationJobData represents a single automation job.
type AutomationJobData struct {
	ID           string
	Status       string
	StatusBadge  string
	StatusIcon   string
	WorkflowType string
	Provider     string
	Repository   string
	Reference    string // e.g., "#123"
	Sender       string
	Command      string
	Error        string
	CreatedAt    string
	StartedAt    string
	CompletedAt  string
	Duration     string
	Attempts     int
	MaxAttempts  int
	CanCancel    bool
	CanRetry     bool
}

// AutomationConfigData represents automation configuration for display.
type AutomationConfigData struct {
	Providers     []AutomationProviderData
	AccessControl AutomationAccessControlData
	Labels        AutomationLabelsData
}

// AutomationProviderData represents a provider configuration.
type AutomationProviderData struct {
	Name          string
	Enabled       bool
	CommandPrefix string
	TriggerOn     []string // Human-readable triggers
}

// AutomationAccessControlData represents access control settings.
type AutomationAccessControlData struct {
	Mode      string
	Allowlist []string
	Blocklist []string
	AllowBots bool
}

// AutomationLabelsData represents label configuration.
type AutomationLabelsData struct {
	MehrhofGenerated string
	InProgress       string
	Failed           string
}

// SessionData represents a task session.
type SessionData struct {
	ID         string
	TaskID     string
	TaskTitle  string
	State      string
	StateIcon  string
	StateColor string
	StartedAt  string
	EndedAt    string
	Duration   string
	Cost       string
	Tokens     string
}

// QuickTasksData contains all data for the quick tasks page.
type QuickTasksData struct {
	PageData

	Tasks      []QuickTaskItemData
	TotalCount int
}

// QuickTaskItemData represents a quick task item.
type QuickTaskItemData struct {
	ID          string
	ShortID     string
	Title       string
	Description string
	State       string
	StateIcon   string
	StateColor  string
	CreatedAt   string
	HasNotes    bool
	NoteCount   int
}

// ProjectPlanningData contains all data for the project planning page.
type ProjectPlanningData struct {
	PageData

	// Project info
	ProjectName   string
	ProjectSource string

	// Queues
	Queues []QueueData

	// Tasks
	Tasks      []ProjectTaskData
	TotalCount int

	// Upload state
	CanUpload bool
}

// QueueData represents a task queue.
type QueueData struct {
	ID       string
	Name     string
	Count    int
	Priority int
}

// ProjectTaskData represents a task in project planning.
type ProjectTaskData struct {
	ID          string
	Title       string
	Description string
	Priority    int
	Status      string
	Queue       string
	Assignee    string
	DueDate     string
	Labels      []LabelData
}

// LoginData contains data for the login page.
type LoginData struct {
	PageData

	Error    string
	Redirect string // URL to redirect to after successful login
}

// LicenseData contains data for the license page.
type LicenseData struct {
	PageData

	ProjectLicense string
	Licenses       []LicenseItemData
	Count          int
}

// LicenseItemData represents a license entry.
type LicenseItemData struct {
	Path    string
	License string
	Unknown bool
}

// StackData contains data for the stacks management page.
type StackData struct {
	PageData

	Stacks []StackViewData
}

// StackViewData represents a stack of dependent features.
type StackViewData struct {
	ID          string
	RootTask    string
	TaskCount   int
	Tasks       []StackTaskView
	CreatedAt   string
	UpdatedAt   string
	HasRebase   bool // True if any task needs rebase
	HasConflict bool // True if any task has conflict
}

// StackTaskView represents a task within a stack.
type StackTaskView struct {
	ID        string
	Branch    string
	State     string
	StateIcon string
	DependsOn string
	PRNumber  int
	PRURL     string
}

// RebasePreviewData contains data for the rebase preview partial.
type RebasePreviewData struct {
	Tasks             []RebaseTaskPreview
	HasConflicts      bool
	SafeCount         int
	ConflictCount     int
	Unavailable       bool
	UnavailableReason string
}

// RebaseTaskPreview represents a single task in the rebase preview.
type RebaseTaskPreview struct {
	TaskID           string
	Branch           string
	OntoBase         string
	Safe             bool
	WouldConflict    bool
	ConflictingFiles []string
	Unavailable      bool
}

// GuideData contains data for guide/help content.
type GuideData struct {
	HasTask         bool
	TaskID          string
	Title           string
	State           string
	Specifications  int
	PendingQuestion *QuestionData
	NextActions     []GuideActionData
}

// GuideActionData represents a suggested action.
type GuideActionData struct {
	Command     string
	Description string
	Endpoint    string
	Primary     bool
}

// LinksData contains all data for the links page.
type LinksData struct {
	PageData

	// Search results
	Entities []LinkedEntityData
	Query    string

	// Stats
	Stats *LinksStatsData

	// Whether links system is available
	Enabled bool
}

// CommitData contains all data for the commit page.
type CommitData struct {
	PageData

	// Whether git is available
	Enabled bool
}

// ScanData contains all data for the security scan page.
type ScanData struct {
	PageData

	// Whether scanning is available
	Enabled bool

	// Detected project info for scanner recommendations
	ProjectInfo *ProjectInfoData
}

// LinkedEntityData represents an entity with links.
type LinkedEntityData struct {
	EntityID   string
	Type       string // spec, session, decision, note
	Title      string // Human-readable name
	TaskID     string // Task ID (if applicable)
	ID         string // Entity-specific ID
	Outgoing   int    // Number of outgoing links
	Incoming   int    // Number of incoming links
	LastLinked string // Time ago (human-readable)
}

// LinksStatsData represents link graph statistics.
type LinksStatsData struct {
	TotalLinks     int
	TotalSources   int
	TotalTargets   int
	OrphanEntities int
	MostLinked     []LinkedEntityData
	Enabled        bool
}

// LinkData represents a single link.
type LinkData struct {
	Source    string
	Target    string
	Context   string
	CreatedAt string
}

// HierarchyData represents hierarchical task context for display.
type HierarchyData struct {
	Parent   *ParentTaskData    `json:"parent,omitempty"`
	Siblings []*SiblingTaskData `json:"siblings,omitempty"`
}

// LibraryData contains all data for the library page.
type LibraryData struct {
	PageData

	// Collections
	Collections []LibraryCollectionData
	Query       string

	// Stats
	TotalCollections int
	TotalPages       int
	TotalSize        string // Pre-formatted size

	// Whether library system is available
	Enabled bool
}

// LibraryCollectionData represents a documentation collection.
type LibraryCollectionData struct {
	ID          string
	Name        string
	Source      string
	SourceType  string // "url", "file", "git"
	IncludeMode string // "auto", "explicit", "always"
	PageCount   int
	TotalSize   string // Pre-formatted size
	Location    string // "project" or "shared"
	PulledAt    string // Pre-formatted time ago
	Tags        []string
	Paths       []string

	// Pre-computed display
	SourceIcon  string
	ModeIcon    string
	ModeBadge   string
	ModeColor   string
	LocationTag string
}

// LibraryPageData represents a page within a collection.
type LibraryPageData struct {
	Path    string
	Title   string
	Size    string // Pre-formatted size
	Snippet string // Preview snippet
}

// ParentTaskData represents the parent task.
type ParentTaskData struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	State       string `json:"state"`
	StateIcon   string `json:"state_icon"`
	StateColor  string `json:"state_color"`
	URL         string `json:"url,omitempty"`
}

// SiblingTaskData represents a sibling subtask.
type SiblingTaskData struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	State      string `json:"state"`
	StateIcon  string `json:"state_icon"`
	StateColor string `json:"state_color"`
}
