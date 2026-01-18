package orchestration

import (
	"time"

	"github.com/valksor/go-mehrhof/internal/agent"
	"github.com/valksor/go-mehrhof/internal/storage"
)

// TaskContext provides context for agent execution.
type TaskContext struct {
	TaskID  string   `yaml:"task_id"`
	Title   string   `yaml:"title"`
	State   string   `yaml:"state"`
	Inputs  []string `yaml:"inputs,omitempty"`
	WorkDir string   `yaml:"work_dir,omitempty"`
}

// OrchestratorMode defines how multiple agents execute.
type OrchestratorMode string

const (
	ModeSingle     OrchestratorMode = "single"     // Single agent (default behavior)
	ModeSequential OrchestratorMode = "sequential" // Execute agents one after another
	ModeParallel   OrchestratorMode = "parallel"   // Execute agents concurrently
	ModeConsensus  OrchestratorMode = "consensus"  // Execute multiple agents and build consensus
)

// AgentStep defines a single step in multi-agent orchestration.
type AgentStep struct {
	Name    string            `yaml:"name"`              // Step identifier
	Agent   string            `yaml:"agent"`             // Agent name to use
	Model   string            `yaml:"model,omitempty"`   // Optional model override
	Role    string            `yaml:"role"`              // Role/purpose for this agent
	Input   []string          `yaml:"input,omitempty"`   // Input artifact names from previous steps
	Output  string            `yaml:"output,omitempty"`  // Output artifact name
	Depends []string          `yaml:"depends,omitempty"` // Dependencies on other steps
	Env     map[string]string `yaml:"env,omitempty"`     // Step-specific environment variables
	Args    []string          `yaml:"args,omitempty"`    // Step-specific CLI arguments
	Timeout int               `yaml:"timeout,omitempty"` // Step-specific timeout in seconds
}

// OrchestratorConfig defines multi-agent orchestration configuration.
type OrchestratorConfig struct {
	Mode      OrchestratorMode `yaml:"mode"`                // Execution mode
	Agents    []AgentStep      `yaml:"agents"`              // Agent steps to execute
	Consensus ConsensusConfig  `yaml:"consensus,omitempty"` // Consensus settings
}

// ConsensusConfig defines how to build consensus from multiple agent outputs.
type ConsensusConfig struct {
	Mode          string `yaml:"mode"`                     // "majority", "unanimous", "any"
	MinVotes      int    `yaml:"min_votes,omitempty"`      // Minimum votes required (0 = auto)
	Synthesizer   string `yaml:"synthesizer,omitempty"`    // Agent to use for synthesizing results
	Similarity    string `yaml:"similarity,omitempty"`     // "jaccard" (default) or "semantic"
	SemanticModel string `yaml:"semantic_model,omitempty"` // Embedding model for semantic similarity (default: "local")
}

// Orchestrator coordinates multi-agent execution.
type Orchestrator struct {
	config   *OrchestratorConfig
	storage  *storage.Workspace
	registry AgentRegistry
}

// AgentRegistry is the interface for accessing agents.
type AgentRegistry interface {
	Get(name string) (agent.Agent, error)
}

// PipelineResult represents the outcome of multi-agent execution.
type PipelineResult struct {
	StepResults map[string]*StepResult `yaml:"step_results"`
	FinalOutput string                 `yaml:"final_output"`
	Consensus   float32                `yaml:"consensus"` // Agreement percentage (0-1)
	Errors      []error                `yaml:"errors,omitempty"`
	Duration    time.Duration          `yaml:"duration"`
}

// StepResult represents the result of a single agent step.
type StepResult struct {
	StepName   string            `yaml:"step_name"`
	AgentName  string            `yaml:"agent_name"`
	Output     string            `yaml:"output"`
	Artifacts  map[string]string `yaml:"artifacts"` // Generated artifacts by name
	Error      error             `yaml:"error,omitempty"`
	Duration   time.Duration     `yaml:"duration"`
	TokenUsage int               `yaml:"token_usage"`
	CostUSD    float64           `yaml:"cost_usd"`
}

// ConsensusResult represents the outcome of consensus building.
type ConsensusResult struct {
	Agreement   float32           `yaml:"agreement"`   // 0-1 scale
	Votes       map[string]string `yaml:"votes"`       // Step name -> output
	Synthesized string            `yaml:"synthesized"` // Final synthesized result
	Reasoning   string            `yaml:"reasoning"`   // Explanation of consensus
}
