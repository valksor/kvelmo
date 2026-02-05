// Package commands provides a unified command router for interactive modes.
// All command logic lives here; CLI and Web become thin presentation layers.
package commands

// ResultType categorizes the kind of response a command returns.
type ResultType string

const (
	ResultMessage        ResultType = "message"        // Simple text response
	ResultStatus         ResultType = "status"         // Workflow/task status
	ResultList           ResultType = "list"           // List of items (tasks, specs, etc.)
	ResultCost           ResultType = "cost"           // Token usage and costs
	ResultSpecifications ResultType = "specifications" // Specification details
	ResultBudget         ResultType = "budget"         // Budget status
	ResultError          ResultType = "error"          // Error response
	ResultQuestion       ResultType = "question"       // Agent asking a question
	ResultChat           ResultType = "chat"           // Chat/agent response
	ResultHelp           ResultType = "help"           // Help text
	ResultExit           ResultType = "exit"           // Signal to exit REPL
)

// Result is the unified response for all commands.
// Clients render this based on their interface (CLI, Web, IDE).
type Result struct {
	Type    ResultType `json:"type"`
	Message string     `json:"message"`           // Human-readable summary
	Data    any        `json:"data,omitempty"`    // Typed payload for structured data
	State   string     `json:"state,omitempty"`   // Current workflow state after command
	TaskID  string     `json:"task_id,omitempty"` // Active task ID if applicable
}

// NewResult creates a basic message result.
func NewResult(message string) *Result {
	return &Result{
		Type:    ResultMessage,
		Message: message,
	}
}

// NewErrorResult creates an error result.
func NewErrorResult(err error) *Result {
	return &Result{
		Type:    ResultError,
		Message: err.Error(),
	}
}

// NewStatusResult creates a status result with state information.
func NewStatusResult(message, state, taskID string, data any) *Result {
	return &Result{
		Type:    ResultStatus,
		Message: message,
		State:   state,
		TaskID:  taskID,
		Data:    data,
	}
}

// NewListResult creates a list result.
func NewListResult(message string, data any) *Result {
	return &Result{
		Type:    ResultList,
		Message: message,
		Data:    data,
	}
}

// NewCostResult creates a cost/token usage result.
func NewCostResult(message string, data any) *Result {
	return &Result{
		Type:    ResultCost,
		Message: message,
		Data:    data,
	}
}

// NewBudgetResult creates a budget status result.
func NewBudgetResult(message string, data any) *Result {
	return &Result{
		Type:    ResultBudget,
		Message: message,
		Data:    data,
	}
}

// NewChatResult creates a chat response result.
func NewChatResult(message string, data any) *Result {
	return &Result{
		Type:    ResultChat,
		Message: message,
		Data:    data,
	}
}

// NewHelpResult creates a help text result.
func NewHelpResult(commands []CommandInfo) *Result {
	return &Result{
		Type:    ResultHelp,
		Message: "Available commands",
		Data:    commands,
	}
}

// ExitResult signals the REPL to exit.
var ExitResult = &Result{Type: ResultExit}

// WithState adds state to a result.
func (r *Result) WithState(state string) *Result {
	r.State = state

	return r
}

// WithTaskID adds task ID to a result.
func (r *Result) WithTaskID(taskID string) *Result {
	r.TaskID = taskID

	return r
}

// WithData adds data to a result.
func (r *Result) WithData(data any) *Result {
	r.Data = data

	return r
}

// StatusData contains structured status information.
type StatusData struct {
	TaskID    string `json:"task_id,omitempty"`
	Title     string `json:"title,omitempty"`
	State     string `json:"state"`
	Ref       string `json:"ref,omitempty"`
	Branch    string `json:"branch,omitempty"`
	SpecCount int    `json:"spec_count,omitempty"`
	Phase     string `json:"phase,omitempty"`
}

// CostData contains token usage and cost information.
type CostData struct {
	TotalTokens   int     `json:"total_tokens"`
	InputTokens   int     `json:"input_tokens"`
	OutputTokens  int     `json:"output_tokens"`
	CachedTokens  int     `json:"cached_tokens"`
	CachedPercent float64 `json:"cached_percent,omitempty"`
	TotalCostUSD  float64 `json:"total_cost_usd"`
}

// BudgetData contains budget status information.
type BudgetData struct {
	Type       string  `json:"type"` // "cost" or "token"
	Used       string  `json:"used"`
	Max        string  `json:"max"`
	Percentage float64 `json:"percentage"`
	Warned     bool    `json:"warned"`
	LimitHit   bool    `json:"limit_hit"`
}

// TaskListItem represents a task in a list.
type TaskListItem struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	State     string `json:"state"`
	Ref       string `json:"ref,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

// SpecificationItem represents a specification in a list.
type SpecificationItem struct {
	Number      int    `json:"number"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status"`
	Component   string `json:"component,omitempty"`
}
