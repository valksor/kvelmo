package server

// Request/response types for API endpoints.

// continueRequest is the request body for POST /api/v1/workflow/continue.
type continueRequest struct {
	Auto bool `json:"auto"` // Auto-execute next logical step
}

// continueResponse is the response for POST /api/v1/workflow/continue.
type continueResponse struct {
	Success     bool     `json:"success"`
	State       string   `json:"state"`
	Action      string   `json:"action,omitempty"`
	NextActions []string `json:"next_actions"`
	Message     string   `json:"message"`
}

// autoRequest is the request body for POST /api/v1/workflow/auto.
type autoRequest struct {
	Ref           string `json:"ref"`
	Agent         string `json:"agent,omitempty"`
	MaxRetries    int    `json:"max_retries"`
	NoPush        bool   `json:"no_push"`
	NoDelete      bool   `json:"no_delete"`
	NoSquash      bool   `json:"no_squash"`
	TargetBranch  string `json:"target_branch,omitempty"`
	QualityTarget string `json:"quality_target"`
	NoQuality     bool   `json:"no_quality"`
}

// autoResponse is the response for POST /api/v1/workflow/auto.
type autoResponse struct {
	Success         bool   `json:"success"`
	PlanningDone    bool   `json:"planning_done"`
	ImplementDone   bool   `json:"implement_done"`
	QualityAttempts int    `json:"quality_attempts"`
	QualityPassed   bool   `json:"quality_passed"`
	FinishDone      bool   `json:"finish_done"`
	FailedAt        string `json:"failed_at,omitempty"`
	Error           string `json:"error,omitempty"`
}

// addNoteRequest is the request body for POST /api/v1/tasks/{id}/notes.
type addNoteRequest struct {
	Content string `json:"content,omitempty"` // Legacy field name
	Note    string `json:"note,omitempty"`    // New field name (preferred)
}

// getContent returns the note content, checking both fields for backward compatibility.
func (r *addNoteRequest) getContent() string {
	if r.Note != "" {
		return r.Note
	}

	return r.Content
}

// noteResponse is the response for POST /api/v1/tasks/{id}/notes.
type noteResponse struct {
	Success   bool   `json:"success"`
	WasAnswer bool   `json:"was_answer"`
	Message   string `json:"message"`
}

// notesListResponse is the response for GET /api/v1/tasks/{id}/notes.
type notesListResponse struct {
	TaskID  string `json:"task_id"`
	Content string `json:"content"`
}

// taskCostResponse is the response for GET /api/v1/tasks/{id}/costs.
type taskCostResponse struct {
	TaskID        string              `json:"task_id"`
	Title         string              `json:"title,omitempty"`
	TotalTokens   int                 `json:"total_tokens"`
	InputTokens   int                 `json:"input_tokens"`
	OutputTokens  int                 `json:"output_tokens"`
	CachedTokens  int                 `json:"cached_tokens"`
	CachedPercent float64             `json:"cached_percent,omitempty"`
	TotalCostUSD  float64             `json:"total_cost_usd"`
	ByStep        map[string]stepCost `json:"by_step,omitempty"`
	Budget        *budgetInfo         `json:"budget,omitempty"`
}

// stepCost represents cost data for a workflow step.
type stepCost struct {
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	CachedTokens int     `json:"cached_tokens"`
	TotalTokens  int     `json:"total_tokens"`
	CostUSD      float64 `json:"cost_usd"`
	Calls        int     `json:"calls"`
}

// grandTotal represents aggregated cost totals.
type grandTotal struct {
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	TotalTokens  int     `json:"total_tokens"`
	CachedTokens int     `json:"cached_tokens"`
	CostUSD      float64 `json:"cost_usd"`
}

// allCostsResponse is the response for GET /api/v1/costs.
type allCostsResponse struct {
	Tasks      []taskCostResponse `json:"tasks"`
	GrandTotal grandTotal         `json:"grand_total"`
	Monthly    *monthlyBudgetInfo `json:"monthly,omitempty"`
}

// budgetInfo exposes budget settings and status for API responses.
type budgetInfo struct {
	MaxTokens int     `json:"max_tokens,omitempty"`
	MaxCost   float64 `json:"max_cost,omitempty"`
	Currency  string  `json:"currency,omitempty"`
	OnLimit   string  `json:"on_limit,omitempty"`
	WarningAt float64 `json:"warning_at,omitempty"`
	Warned    bool    `json:"warned,omitempty"`
	LimitHit  bool    `json:"limit_hit,omitempty"`
}

// monthlyBudgetInfo exposes monthly budget status.
type monthlyBudgetInfo struct {
	Month       string  `json:"month"`
	Spent       float64 `json:"spent"`
	MaxCost     float64 `json:"max_cost,omitempty"`
	WarningAt   float64 `json:"warning_at,omitempty"`
	WarningSent bool    `json:"warning_sent,omitempty"`
}

// guideResponse is the response for GET /api/v1/guide.
type guideResponse struct {
	HasTask         bool                 `json:"has_task"`
	TaskID          string               `json:"task_id,omitempty"`
	Title           string               `json:"title,omitempty"`
	State           string               `json:"state,omitempty"`
	Specifications  int                  `json:"specifications"`
	PendingQuestion *pendingQuestionInfo `json:"pending_question,omitempty"`
	NextActions     []guideAction        `json:"next_actions"`
}

// pendingQuestionInfo contains pending agent question details.
type pendingQuestionInfo struct {
	Question string   `json:"question"`
	Options  []string `json:"options,omitempty"`
}

// guideAction represents a suggested next action.
type guideAction struct {
	Command     string `json:"command"`
	Description string `json:"description"`
	Endpoint    string `json:"endpoint,omitempty"`
}

// agentInfo represents information about an agent.
type agentInfo struct {
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	Extends      string                 `json:"extends,omitempty"`
	Description  string                 `json:"description,omitempty"`
	Version      string                 `json:"version,omitempty"`
	Available    bool                   `json:"available"`
	Capabilities *agentCapabilitiesInfo `json:"capabilities,omitempty"`
	Models       []agentModelInfo       `json:"models,omitempty"`
}

// agentCapabilitiesInfo represents agent capabilities.
type agentCapabilitiesInfo struct {
	Streaming      bool     `json:"streaming"`
	ToolUse        bool     `json:"tool_use"`
	FileOperations bool     `json:"file_operations"`
	CodeExecution  bool     `json:"code_execution"`
	MultiTurn      bool     `json:"multi_turn"`
	SystemPrompt   bool     `json:"system_prompt"`
	AllowedTools   []string `json:"allowed_tools,omitempty"`
}

// agentModelInfo represents an available model for an agent.
type agentModelInfo struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Default    bool    `json:"default"`
	MaxTokens  int     `json:"max_tokens,omitempty"`
	InputCost  float64 `json:"input_cost_usd,omitempty"`
	OutputCost float64 `json:"output_cost_usd,omitempty"`
}

// agentsListResponse is the response for GET /api/v1/agents.
type agentsListResponse struct {
	Agents []agentInfo `json:"agents"`
	Count  int         `json:"count"`
}

// providerInfo represents information about a provider.
type providerInfo struct {
	Scheme      string   `json:"scheme"`
	Shorthand   string   `json:"shorthand,omitempty"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	EnvVars     []string `json:"env_vars,omitempty"`
}

// providersListResponse is the response for GET /api/v1/providers.
type providersListResponse struct {
	Providers []providerInfo `json:"providers"`
	Count     int            `json:"count"`
}

// licenseInfo represents information about a license.
type licenseInfo struct {
	Path    string `json:"path"`
	License string `json:"license"`
	Unknown bool   `json:"unknown"`
}

// licensesListResponse is the response for GET /api/v1/license/info.
type licensesListResponse struct {
	Licenses []licenseInfo `json:"licenses"`
	Count    int           `json:"count"`
}

// browserGotoRequest is the request body for POST /api/v1/browser/goto.
type browserGotoRequest struct {
	URL string `json:"url"`
}

// browserNavigateRequest is the request body for POST /api/v1/browser/navigate.
type browserNavigateRequest struct {
	TabID string `json:"tab_id,omitempty"`
	URL   string `json:"url"`
}

// browserClickRequest is the request body for POST /api/v1/browser/click.
type browserClickRequest struct {
	TabID    string `json:"tab_id,omitempty"`
	Selector string `json:"selector"`
}

// browserTypeRequest is the request body for POST /api/v1/browser/type.
type browserTypeRequest struct {
	TabID    string `json:"tab_id,omitempty"`
	Selector string `json:"selector"`
	Text     string `json:"text"`
	Clear    bool   `json:"clear"`
}

// browserEvalRequest is the request body for POST /api/v1/browser/eval.
type browserEvalRequest struct {
	TabID      string `json:"tab_id,omitempty"`
	Expression string `json:"expression"`
}

// browserDOMRequest is the request body for POST /api/v1/browser/dom.
type browserDOMRequest struct {
	TabID    string `json:"tab_id,omitempty"`
	Selector string `json:"selector"`
	All      bool   `json:"all"`
	HTML     bool   `json:"html"`
	Limit    int    `json:"limit"`
}

// browserScreenshotRequest is the request body for POST /api/v1/browser/screenshot.
type browserScreenshotRequest struct {
	TabID    string `json:"tab_id,omitempty"`
	Format   string `json:"format"`
	Quality  int    `json:"quality"`
	FullPage bool   `json:"full_page"`
}

// browserReloadRequest is the request body for POST /api/v1/browser/reload.
type browserReloadRequest struct {
	TabID string `json:"tab_id,omitempty"`
	Hard  bool   `json:"hard"`
}

// browserCloseRequest is the request body for POST /api/v1/browser/close.
type browserCloseRequest struct {
	TabID string `json:"tab_id"`
}

// browserTabResponse represents a browser tab.
type browserTabResponse struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

// browserStatusResponse is the response for GET /api/v1/browser/status.
type browserStatusResponse struct {
	Connected bool                 `json:"connected"`
	Host      string               `json:"host,omitempty"`
	Port      int                  `json:"port,omitempty"`
	Tabs      []browserTabResponse `json:"tabs,omitempty"`
	Error     string               `json:"error,omitempty"`
}

// browserDOMElement represents a DOM element.
type browserDOMElement struct {
	TagName     string `json:"tag_name"`
	TextContent string `json:"text_content,omitempty"`
	OuterHTML   string `json:"outer_html,omitempty"`
	Visible     bool   `json:"visible"`
}

// scanRequest is the request body for POST /api/v1/scan.
type scanRequest struct {
	Dir       string   `json:"dir,omitempty"`
	Scanners  []string `json:"scanners,omitempty"`
	FailLevel string   `json:"fail_level,omitempty"`
	Format    string   `json:"format,omitempty"`
}

// scanFinding represents a security finding.
type scanFinding struct {
	Scanner  string `json:"scanner"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
	RuleID   string `json:"rule_id,omitempty"`
}

// scanResponse is the response for POST /api/v1/scan.
type scanResponse struct {
	Findings      []scanFinding `json:"findings"`
	TotalCount    int           `json:"total_count"`
	BlockingCount int           `json:"blocking_count"`
	Passed        bool          `json:"passed"`
}

// memoryResult represents a memory search result.
type memoryResult struct {
	TaskID   string         `json:"task_id"`
	Type     string         `json:"type"`
	Score    float64        `json:"score"`
	Content  string         `json:"content"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// memorySearchResponse is the response for GET /api/v1/memory/search.
type memorySearchResponse struct {
	Results []memoryResult `json:"results"`
	Count   int            `json:"count"`
}

// memoryIndexRequest is the request body for POST /api/v1/memory/index.
type memoryIndexRequest struct {
	TaskID string `json:"task_id"`
}

// memoryStatsResponse is the response for GET /api/v1/memory/stats.
type memoryStatsResponse struct {
	TotalDocuments int            `json:"total_documents"`
	ByType         map[string]int `json:"by_type"`
	Enabled        bool           `json:"enabled"`
}

// syncRequest is the request body for POST /api/v1/workflow/sync.
type syncRequest struct {
	TaskID string `json:"task_id"` // Required - task to sync
}

// syncResponse is the response for POST /api/v1/workflow/sync.
type syncResponse struct {
	Success        bool   `json:"success"`
	HasChanges     bool   `json:"has_changes"`
	ChangesSummary string `json:"changes_summary,omitempty"`
	SpecGenerated  string `json:"spec_generated,omitempty"`
	Message        string `json:"message"`
}

// simplifyRequest is the request body for POST /api/v1/workflow/simplify.
type simplifyRequest struct {
	NoCheckpoint bool   `json:"no_checkpoint"`
	Agent        string `json:"agent,omitempty"`
}

// simplifyResponse is the response for POST /api/v1/workflow/simplify.
type simplifyResponse struct {
	Success    bool   `json:"success"`
	Simplified string `json:"simplified"` // "task_input", "specifications", or "code"
	Message    string `json:"message"`
}

// templateInfo represents information about a template.
type templateInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// templatesListResponse is the response for GET /api/v1/templates.
type templatesListResponse struct {
	Templates []templateInfo `json:"templates"`
	Count     int            `json:"count"`
}

// templateShowResponse is the response for GET /api/v1/templates/{name}.
type templateShowResponse struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Frontmatter map[string]any    `json:"frontmatter,omitempty"`
	Agent       string            `json:"agent,omitempty"`
	AgentSteps  map[string]any    `json:"agent_steps,omitempty"`
	Git         map[string]string `json:"git,omitempty"`
	Workflow    map[string]any    `json:"workflow,omitempty"`
}

// templateApplyRequest is the request body for POST /api/v1/templates/apply.
type templateApplyRequest struct {
	TemplateName string `json:"template_name"` // Required
	FilePath     string `json:"file_path"`     // Required
}

// templateApplyResponse is the response for POST /api/v1/templates/apply.
type templateApplyResponse struct {
	Success     bool           `json:"success"`
	Frontmatter map[string]any `json:"frontmatter,omitempty"`
	Message     string         `json:"message"`
}

// optimizeRequest is the request body for POST /api/v1/quick/{id}/optimize.
type optimizeRequest struct {
	Agent string `json:"agent,omitempty"` // Agent override for this operation
}

// exportRequest is the request body for POST /api/v1/quick/{id}/export.
type exportRequest struct {
	Output string `json:"output,omitempty"` // Output file path (empty = download)
}

// submitRequest is the request body for POST /api/v1/quick/{id}/submit.
type submitRequest struct {
	Provider string   `json:"provider"`          // Required: target provider
	Labels   []string `json:"labels,omitempty"`  // Additional labels to apply
	DryRun   bool     `json:"dry_run,omitempty"` // Preview without submitting
}

// questionRequest is the request body for POST /api/v1/workflow/question.
type questionRequest struct {
	Question string `json:"question"` // The question to ask the agent
}

// standaloneReviewRequest is the request body for POST /api/v1/workflow/review/standalone.
type standaloneReviewRequest struct {
	Mode             string   `json:"mode"`                  // "uncommitted", "branch", "range", "files"
	BaseBranch       string   `json:"base_branch,omitempty"` // For branch mode
	Range            string   `json:"range,omitempty"`       // For range mode (e.g., "HEAD~3..HEAD")
	Files            []string `json:"files,omitempty"`       // For files mode
	Context          int      `json:"context,omitempty"`     // Lines of context (default: 3)
	Agent            string   `json:"agent,omitempty"`       // Agent override
	ApplyFixes       bool     `json:"apply_fixes,omitempty"` // If true, apply suggested fixes
	CreateCheckpoint bool     `json:"create_checkpoint"`     // Create checkpoint before changes (defaults to true)
}

// standaloneReviewResponse is the response for POST /api/v1/workflow/review/standalone.
type standaloneReviewResponse struct {
	Success bool                    `json:"success"`
	Verdict string                  `json:"verdict"` // "APPROVED", "NEEDS_CHANGES", "COMMENT"
	Summary string                  `json:"summary"`
	Issues  []standaloneReviewIssue `json:"issues,omitempty"`
	Changes []standaloneFileChange  `json:"changes,omitempty"` // File changes applied (only populated if apply_fixes was true)
	Usage   *standaloneUsageInfo    `json:"usage,omitempty"`
	Error   string                  `json:"error,omitempty"`
}

// standaloneReviewIssue represents an issue found during review.
type standaloneReviewIssue struct {
	Severity    string `json:"severity"`    // "critical", "high", "medium", "low"
	Category    string `json:"category"`    // "security", "correctness", "performance", "style"
	File        string `json:"file"`        // File path
	Line        int    `json:"line"`        // Line number
	Description string `json:"description"` // Issue description
	Suggestion  string `json:"suggestion"`  // Suggested fix
}

// standaloneSimplifyRequest is the request body for POST /api/v1/workflow/simplify/standalone.
type standaloneSimplifyRequest struct {
	Mode             string   `json:"mode"`                  // "uncommitted", "branch", "range", "files"
	BaseBranch       string   `json:"base_branch,omitempty"` // For branch mode
	Range            string   `json:"range,omitempty"`       // For range mode
	Files            []string `json:"files,omitempty"`       // For files mode
	Context          int      `json:"context,omitempty"`     // Lines of context (default: 3)
	Agent            string   `json:"agent,omitempty"`       // Agent override
	CreateCheckpoint bool     `json:"create_checkpoint"`     // Create checkpoint before changes
}

// standaloneSimplifyResponse is the response for POST /api/v1/workflow/simplify/standalone.
type standaloneSimplifyResponse struct {
	Success bool                   `json:"success"`
	Summary string                 `json:"summary"`
	Changes []standaloneFileChange `json:"changes,omitempty"`
	Usage   *standaloneUsageInfo   `json:"usage,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// standaloneFileChange represents a file change from simplification.
type standaloneFileChange struct {
	Path      string `json:"path"`
	Operation string `json:"operation"` // "create", "update", "delete"
}

// standaloneUsageInfo contains token usage information.
type standaloneUsageInfo struct {
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	CachedTokens int     `json:"cached_tokens,omitempty"`
	CostUSD      float64 `json:"cost_usd,omitempty"`
}
