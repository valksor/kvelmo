package commands

import (
	"context"
	"io"

	"github.com/valksor/go-mehrhof/internal/agent"
	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/vcs"
	"github.com/valksor/go-mehrhof/internal/workflow"
	"github.com/valksor/go-toolkit/eventbus"
)

// ConductorAPI defines the conductor methods used by commands.
// This interface enables mocking for unit tests while keeping production
// code unchanged. The real *conductor.Conductor satisfies this interface.
//
//nolint:interfacebloat // Large interface mirrors real Conductor API for test mocking
type ConductorAPI interface {
	// Lifecycle
	Initialize(ctx context.Context) error
	Close() error

	// Workflow operations
	Start(ctx context.Context, reference string) error
	Plan(ctx context.Context) error
	Implement(ctx context.Context) error
	RunImplementation(ctx context.Context) error
	Review(ctx context.Context) error
	ImplementReview(ctx context.Context, reviewNumber int) error
	RunReviewImplementation(ctx context.Context, reviewNumber int) error
	Finish(ctx context.Context, opts conductor.FinishOptions) error
	Status(ctx context.Context) (*conductor.TaskStatus, error)
	AnswerQuestion(ctx context.Context, answer string) error
	ResetState(ctx context.Context) error
	Delete(ctx context.Context, opts conductor.DeleteOptions) error

	// Note management
	AddNote(ctx context.Context, message string) error

	// Quality operations
	RunQuality(ctx context.Context, opts conductor.QualityOptions) (*conductor.QualityResult, error)
	GenerateCommitMessagePreview(ctx context.Context) (string, error)

	// Specification operations
	GetSpecificationFileDiff(ctx context.Context, taskID string, specNumber int, filePath string, contextLines int) (string, error)

	// State access
	GetActiveTask() *storage.ActiveTask
	GetTaskWork() *storage.TaskWork
	GetWorkspace() *storage.Workspace
	GetGit() *vcs.Git
	GetMachine() *workflow.Machine
	GetActiveAgent() agent.Agent

	// I/O
	GetStdout() io.Writer
	GetStderr() io.Writer
	GetEventBus() *eventbus.Bus

	// Registries
	GetAgentRegistry() *agent.Registry
	GetProviderRegistry() *provider.Registry

	// Configuration
	SetAgent(agent string)
	ClearAgent()
	SetImplementationOptions(component, parallel string)
	ClearImplementationOptions()

	// Directories
	CodeDir() string
	TasksDir() string
	GetTaskID() string
	GetWorktreePath() string

	// Utilities
	ClearStaleTask() bool
}

// Compile-time check that *conductor.Conductor satisfies ConductorAPI.
var _ ConductorAPI = (*conductor.Conductor)(nil)
