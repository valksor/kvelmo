package server

// Request/response types for API endpoints.

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
