// Package helper_test provides shared testing utilities for go-mehrhof tests.
package helper_test

import (
	"context"
	"io"
	"sync"

	"github.com/valksor/go-mehrhof/internal/agent"
	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/vcs"
	"github.com/valksor/go-mehrhof/internal/workflow"
	"github.com/valksor/go-toolkit/eventbus"
)

// MockConductor provides a testable conductor implementation for command tests.
// It tracks all method calls and allows injecting errors and custom responses.
type MockConductor struct {
	mu sync.Mutex

	// State - configurable return values
	ActiveTask     *storage.ActiveTask
	TaskWork       *storage.TaskWork
	Workspace      *storage.Workspace
	Git            *vcs.Git
	Machine        *workflow.Machine
	ActiveAgent    agent.Agent
	StatusResult   *conductor.TaskStatus
	AgentRegistry  *agent.Registry
	ProviderReg    *provider.Registry
	EventBus       *eventbus.Bus
	Stdout         io.Writer
	Stderr         io.Writer
	CodeDirectory  string
	TasksDirectory string
	TaskID         string
	WorktreePath   string

	// Call tracking
	InitializeCalls              int
	CloseCalls                   int
	StartCalls                   []string
	PlanCalls                    int
	ImplementCalls               int
	RunImplementationCalls       int
	ReviewCalls                  int
	ImplementReviewCalls         []int
	RunReviewImplementationCalls []int
	FinishCalls                  []conductor.FinishOptions
	StatusCalls                  int
	AnswerQuestionCalls          []string
	ResetStateCalls              int
	AddNoteCalls                 []string
	SetAgentCalls                []string
	ClearAgentCalls              int
	SetImplementationCalls       []implementationOptions
	ClearImplementationCalls     int
	ClearStaleTaskCalls          int
	DeleteCalls                  []conductor.DeleteOptions

	// Error injection
	InitializeErr              error
	CloseErr                   error
	StartErr                   error
	PlanErr                    error
	ImplementErr               error
	RunImplementationErr       error
	ReviewErr                  error
	ImplementReviewErr         error
	RunReviewImplementationErr error
	FinishErr                  error
	StatusErr                  error
	AnswerQuestionErr          error
	ResetStateErr              error
	AddNoteErr                 error
	DeleteErr                  error

	// Behavior flags
	ClearStaleTaskResult bool

	// Quality operations
	RunQualityCalls           []conductor.QualityOptions
	RunQualityResult          *conductor.QualityResult
	RunQualityErr             error
	CommitMessagePreview      string
	CommitMessagePreviewCalls int
	CommitMessagePreviewErr   error

	// Specification operations
	SpecificationFileDiffCalls []specFileDiffCall
	SpecificationFileDiff      string
	SpecificationFileDiffErr   error
}

// specFileDiffCall tracks calls to GetSpecificationFileDiff.
type specFileDiffCall struct {
	TaskID       string
	SpecNumber   int
	FilePath     string
	ContextLines int
}

// implementationOptions tracks SetImplementationOptions calls.
type implementationOptions struct {
	Component string
	Parallel  string
}

// NewMockConductor creates a new MockConductor with default values.
func NewMockConductor() *MockConductor {
	return &MockConductor{
		StartCalls:                   make([]string, 0),
		ImplementReviewCalls:         make([]int, 0),
		RunReviewImplementationCalls: make([]int, 0),
		FinishCalls:                  make([]conductor.FinishOptions, 0),
		AnswerQuestionCalls:          make([]string, 0),
		AddNoteCalls:                 make([]string, 0),
		SetAgentCalls:                make([]string, 0),
		SetImplementationCalls:       make([]implementationOptions, 0),
		DeleteCalls:                  make([]conductor.DeleteOptions, 0),
		RunQualityCalls:              make([]conductor.QualityOptions, 0),
		SpecificationFileDiffCalls:   make([]specFileDiffCall, 0),
		EventBus:                     eventbus.NewBus(),
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Lifecycle methods
// ──────────────────────────────────────────────────────────────────────────────

// Initialize initializes the conductor.
func (m *MockConductor) Initialize(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.InitializeCalls++

	return m.InitializeErr
}

// Close closes the conductor.
func (m *MockConductor) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CloseCalls++

	return m.CloseErr
}

// ──────────────────────────────────────────────────────────────────────────────
// Workflow operations
// ──────────────────────────────────────────────────────────────────────────────

// Start starts a new task from a reference.
func (m *MockConductor) Start(ctx context.Context, reference string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.StartCalls = append(m.StartCalls, reference)

	return m.StartErr
}

// Plan runs the planning phase.
func (m *MockConductor) Plan(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.PlanCalls++

	return m.PlanErr
}

// Implement runs the implementation phase.
func (m *MockConductor) Implement(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ImplementCalls++

	return m.ImplementErr
}

// RunImplementation executes the implementation agent.
func (m *MockConductor) RunImplementation(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RunImplementationCalls++

	return m.RunImplementationErr
}

// Review runs the review phase.
func (m *MockConductor) Review(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ReviewCalls++

	return m.ReviewErr
}

// ImplementReview implements a specific review.
func (m *MockConductor) ImplementReview(ctx context.Context, reviewNumber int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ImplementReviewCalls = append(m.ImplementReviewCalls, reviewNumber)

	return m.ImplementReviewErr
}

// RunReviewImplementation executes the review implementation agent.
func (m *MockConductor) RunReviewImplementation(ctx context.Context, reviewNumber int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RunReviewImplementationCalls = append(m.RunReviewImplementationCalls, reviewNumber)

	return m.RunReviewImplementationErr
}

// Finish finishes the current task.
func (m *MockConductor) Finish(ctx context.Context, opts conductor.FinishOptions) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.FinishCalls = append(m.FinishCalls, opts)

	return m.FinishErr
}

// Status returns the current task status.
func (m *MockConductor) Status(ctx context.Context) (*conductor.TaskStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.StatusCalls++
	if m.StatusErr != nil {
		return nil, m.StatusErr
	}

	return m.StatusResult, nil
}

// AnswerQuestion answers an agent question.
func (m *MockConductor) AnswerQuestion(ctx context.Context, answer string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.AnswerQuestionCalls = append(m.AnswerQuestionCalls, answer)

	return m.AnswerQuestionErr
}

// ResetState resets the task state.
func (m *MockConductor) ResetState(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ResetStateCalls++

	return m.ResetStateErr
}

// Delete deletes the current task.
func (m *MockConductor) Delete(ctx context.Context, opts conductor.DeleteOptions) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.DeleteCalls = append(m.DeleteCalls, opts)

	return m.DeleteErr
}

// ──────────────────────────────────────────────────────────────────────────────
// Note management
// ──────────────────────────────────────────────────────────────────────────────

// AddNote adds a note to the task.
func (m *MockConductor) AddNote(ctx context.Context, message string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.AddNoteCalls = append(m.AddNoteCalls, message)

	return m.AddNoteErr
}

// ──────────────────────────────────────────────────────────────────────────────
// Quality operations
// ──────────────────────────────────────────────────────────────────────────────

// RunQuality runs quality checks.
func (m *MockConductor) RunQuality(ctx context.Context, opts conductor.QualityOptions) (*conductor.QualityResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RunQualityCalls = append(m.RunQualityCalls, opts)

	if m.RunQualityErr != nil {
		return nil, m.RunQualityErr
	}

	return m.RunQualityResult, nil
}

// GenerateCommitMessagePreview generates a commit message preview.
func (m *MockConductor) GenerateCommitMessagePreview(ctx context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CommitMessagePreviewCalls++

	if m.CommitMessagePreviewErr != nil {
		return "", m.CommitMessagePreviewErr
	}

	return m.CommitMessagePreview, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Specification operations
// ──────────────────────────────────────────────────────────────────────────────

// GetSpecificationFileDiff returns a diff for a specification file.
func (m *MockConductor) GetSpecificationFileDiff(ctx context.Context, taskID string, specNumber int, filePath string, contextLines int) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SpecificationFileDiffCalls = append(m.SpecificationFileDiffCalls, specFileDiffCall{
		TaskID:       taskID,
		SpecNumber:   specNumber,
		FilePath:     filePath,
		ContextLines: contextLines,
	})

	if m.SpecificationFileDiffErr != nil {
		return "", m.SpecificationFileDiffErr
	}

	return m.SpecificationFileDiff, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// State access
// ──────────────────────────────────────────────────────────────────────────────

// GetActiveTask returns the active task.
func (m *MockConductor) GetActiveTask() *storage.ActiveTask {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.ActiveTask
}

// GetTaskWork returns the current task work.
func (m *MockConductor) GetTaskWork() *storage.TaskWork {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.TaskWork
}

// GetWorkspace returns the workspace.
func (m *MockConductor) GetWorkspace() *storage.Workspace {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.Workspace
}

// GetGit returns the git instance.
func (m *MockConductor) GetGit() *vcs.Git {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.Git
}

// GetMachine returns the workflow state machine.
func (m *MockConductor) GetMachine() *workflow.Machine {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.Machine
}

// GetActiveAgent returns the active agent.
func (m *MockConductor) GetActiveAgent() agent.Agent {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.ActiveAgent
}

// ──────────────────────────────────────────────────────────────────────────────
// I/O
// ──────────────────────────────────────────────────────────────────────────────

// GetStdout returns the stdout writer.
func (m *MockConductor) GetStdout() io.Writer {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.Stdout
}

// GetStderr returns the stderr writer.
func (m *MockConductor) GetStderr() io.Writer {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.Stderr
}

// GetEventBus returns the event bus.
func (m *MockConductor) GetEventBus() *eventbus.Bus {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.EventBus
}

// ──────────────────────────────────────────────────────────────────────────────
// Registries
// ──────────────────────────────────────────────────────────────────────────────

// GetAgentRegistry returns the agent registry.
func (m *MockConductor) GetAgentRegistry() *agent.Registry {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.AgentRegistry
}

// GetProviderRegistry returns the provider registry.
func (m *MockConductor) GetProviderRegistry() *provider.Registry {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.ProviderReg
}

// ──────────────────────────────────────────────────────────────────────────────
// Configuration
// ──────────────────────────────────────────────────────────────────────────────

// SetAgent sets the agent to use.
func (m *MockConductor) SetAgent(agentName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SetAgentCalls = append(m.SetAgentCalls, agentName)
}

// ClearAgent clears the agent override.
func (m *MockConductor) ClearAgent() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ClearAgentCalls++
}

// SetImplementationOptions sets implementation options.
func (m *MockConductor) SetImplementationOptions(component, parallel string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SetImplementationCalls = append(m.SetImplementationCalls, implementationOptions{
		Component: component,
		Parallel:  parallel,
	})
}

// ClearImplementationOptions clears implementation options.
func (m *MockConductor) ClearImplementationOptions() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ClearImplementationCalls++
}

// ──────────────────────────────────────────────────────────────────────────────
// Directories
// ──────────────────────────────────────────────────────────────────────────────

// CodeDir returns the code directory.
func (m *MockConductor) CodeDir() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.CodeDirectory
}

// TasksDir returns the tasks directory.
func (m *MockConductor) TasksDir() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.TasksDirectory
}

// GetTaskID returns the current task ID.
func (m *MockConductor) GetTaskID() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.TaskID
}

// GetWorktreePath returns the worktree path.
func (m *MockConductor) GetWorktreePath() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.WorktreePath
}

// ──────────────────────────────────────────────────────────────────────────────
// Utilities
// ──────────────────────────────────────────────────────────────────────────────

// ClearStaleTask clears stale task if present.
func (m *MockConductor) ClearStaleTask() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ClearStaleTaskCalls++

	return m.ClearStaleTaskResult
}

// ──────────────────────────────────────────────────────────────────────────────
// Fluent builders for test setup
// ──────────────────────────────────────────────────────────────────────────────

// WithActiveTask sets the active task.
func (m *MockConductor) WithActiveTask(task *storage.ActiveTask) *MockConductor {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ActiveTask = task

	return m
}

// WithTaskWork sets the task work.
func (m *MockConductor) WithTaskWork(work *storage.TaskWork) *MockConductor {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TaskWork = work

	return m
}

// WithStatus sets the status result.
func (m *MockConductor) WithStatus(status *conductor.TaskStatus) *MockConductor {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.StatusResult = status

	return m
}

// WithWorkspace sets the workspace.
func (m *MockConductor) WithWorkspace(ws *storage.Workspace) *MockConductor {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Workspace = ws

	return m
}

// WithGit sets the git instance.
func (m *MockConductor) WithGit(git *vcs.Git) *MockConductor {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Git = git

	return m
}

// WithMachine sets the workflow machine.
func (m *MockConductor) WithMachine(machine *workflow.Machine) *MockConductor {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Machine = machine

	return m
}

// WithStdout sets the stdout writer.
func (m *MockConductor) WithStdout(w io.Writer) *MockConductor {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Stdout = w

	return m
}

// WithStderr sets the stderr writer.
func (m *MockConductor) WithStderr(w io.Writer) *MockConductor {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Stderr = w

	return m
}

// WithCodeDir sets the code directory.
func (m *MockConductor) WithCodeDir(dir string) *MockConductor {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CodeDirectory = dir

	return m
}

// WithTaskID sets the task ID.
func (m *MockConductor) WithTaskID(id string) *MockConductor {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TaskID = id

	return m
}

// ──────────────────────────────────────────────────────────────────────────────
// Error injection builders
// ──────────────────────────────────────────────────────────────────────────────

// WithInitializeError sets the initialize error.
func (m *MockConductor) WithInitializeError(err error) *MockConductor {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.InitializeErr = err

	return m
}

// WithStartError sets the start error.
func (m *MockConductor) WithStartError(err error) *MockConductor {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.StartErr = err

	return m
}

// WithPlanError sets the plan error.
func (m *MockConductor) WithPlanError(err error) *MockConductor {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.PlanErr = err

	return m
}

// WithImplementError sets the implement error.
func (m *MockConductor) WithImplementError(err error) *MockConductor {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ImplementErr = err

	return m
}

// WithRunImplementationError sets the run implementation error.
func (m *MockConductor) WithRunImplementationError(err error) *MockConductor {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RunImplementationErr = err

	return m
}

// WithReviewError sets the review error.
func (m *MockConductor) WithReviewError(err error) *MockConductor {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ReviewErr = err

	return m
}

// WithImplementReviewError sets the implement review error.
func (m *MockConductor) WithImplementReviewError(err error) *MockConductor {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ImplementReviewErr = err

	return m
}

// WithRunReviewImplementationError sets the run review implementation error.
func (m *MockConductor) WithRunReviewImplementationError(err error) *MockConductor {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RunReviewImplementationErr = err

	return m
}

// WithFinishError sets the finish error.
func (m *MockConductor) WithFinishError(err error) *MockConductor {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.FinishErr = err

	return m
}

// WithStatusError sets the status error.
func (m *MockConductor) WithStatusError(err error) *MockConductor {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.StatusErr = err

	return m
}

// WithAnswerError sets the answer question error.
func (m *MockConductor) WithAnswerError(err error) *MockConductor {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.AnswerQuestionErr = err

	return m
}

// WithResetStateError sets the reset state error.
func (m *MockConductor) WithResetStateError(err error) *MockConductor {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ResetStateErr = err

	return m
}

// WithAddNoteError sets the add note error.
func (m *MockConductor) WithAddNoteError(err error) *MockConductor {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.AddNoteErr = err

	return m
}

// WithDeleteError sets the delete error.
func (m *MockConductor) WithDeleteError(err error) *MockConductor {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.DeleteErr = err

	return m
}

// WithClearStaleTaskResult sets the clear stale task result.
func (m *MockConductor) WithClearStaleTaskResult(result bool) *MockConductor {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ClearStaleTaskResult = result

	return m
}

// ──────────────────────────────────────────────────────────────────────────────
// Quality operation builders
// ──────────────────────────────────────────────────────────────────────────────

// WithRunQualityResult sets the run quality result.
func (m *MockConductor) WithRunQualityResult(result *conductor.QualityResult) *MockConductor {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RunQualityResult = result

	return m
}

// WithRunQualityError sets the run quality error.
func (m *MockConductor) WithRunQualityError(err error) *MockConductor {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RunQualityErr = err

	return m
}

// WithCommitMessagePreview sets the commit message preview.
func (m *MockConductor) WithCommitMessagePreview(msg string) *MockConductor {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CommitMessagePreview = msg

	return m
}

// WithCommitMessagePreviewError sets the commit message preview error.
func (m *MockConductor) WithCommitMessagePreviewError(err error) *MockConductor {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CommitMessagePreviewErr = err

	return m
}

// ──────────────────────────────────────────────────────────────────────────────
// Specification operation builders
// ──────────────────────────────────────────────────────────────────────────────

// WithSpecificationFileDiff sets the specification file diff result.
func (m *MockConductor) WithSpecificationFileDiff(diff string) *MockConductor {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SpecificationFileDiff = diff

	return m
}

// WithSpecificationFileDiffError sets the specification file diff error.
func (m *MockConductor) WithSpecificationFileDiffError(err error) *MockConductor {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SpecificationFileDiffErr = err

	return m
}

// ──────────────────────────────────────────────────────────────────────────────
// Test assertions
// ──────────────────────────────────────────────────────────────────────────────

// Reset clears all call tracking for reuse between tests.
func (m *MockConductor) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.InitializeCalls = 0
	m.CloseCalls = 0
	m.StartCalls = make([]string, 0)
	m.PlanCalls = 0
	m.ImplementCalls = 0
	m.RunImplementationCalls = 0
	m.ReviewCalls = 0
	m.ImplementReviewCalls = make([]int, 0)
	m.RunReviewImplementationCalls = make([]int, 0)
	m.FinishCalls = make([]conductor.FinishOptions, 0)
	m.StatusCalls = 0
	m.AnswerQuestionCalls = make([]string, 0)
	m.ResetStateCalls = 0
	m.AddNoteCalls = make([]string, 0)
	m.SetAgentCalls = make([]string, 0)
	m.ClearAgentCalls = 0
	m.SetImplementationCalls = make([]implementationOptions, 0)
	m.ClearImplementationCalls = 0
	m.ClearStaleTaskCalls = 0
	m.DeleteCalls = make([]conductor.DeleteOptions, 0)
	m.RunQualityCalls = make([]conductor.QualityOptions, 0)
	m.SpecificationFileDiffCalls = make([]specFileDiffCall, 0)
	m.CommitMessagePreviewCalls = 0
}

// WasPlanCalled returns true if Plan was called.
func (m *MockConductor) WasPlanCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.PlanCalls > 0
}

// WasImplementCalled returns true if Implement was called.
func (m *MockConductor) WasImplementCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.ImplementCalls > 0
}

// WasReviewCalled returns true if Review was called.
func (m *MockConductor) WasReviewCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.ReviewCalls > 0
}

// GetStartCalls returns all Start call arguments.
func (m *MockConductor) GetStartCalls() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, len(m.StartCalls))
	copy(result, m.StartCalls)

	return result
}

// GetNoteCalls returns all AddNote call arguments.
func (m *MockConductor) GetNoteCalls() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, len(m.AddNoteCalls))
	copy(result, m.AddNoteCalls)

	return result
}
