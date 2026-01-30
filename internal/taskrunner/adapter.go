package taskrunner

import (
	"context"
)

// ConductorAdapter wraps a concrete conductor to satisfy TaskConductor.
// This adapter pattern avoids circular imports between taskrunner and conductor.
type ConductorAdapter struct {
	// These fields are interface{} to avoid circular imports.
	// The actual types are *conductor.Conductor and func() error.
	conductor interface {
		Start(ctx context.Context, ref string) error
		Plan(ctx context.Context) error
		Implement(ctx context.Context) error
		AddNote(ctx context.Context, message string) error
		GetTaskID() string
		GetWorktreePath() string
		Close() error
	}
}

// NewConductorAdapter creates an adapter from a conductor instance.
// The conductor must implement the required methods.
func NewConductorAdapter(cond interface {
	Start(ctx context.Context, ref string) error
	Plan(ctx context.Context) error
	Implement(ctx context.Context) error
	AddNote(ctx context.Context, message string) error
	GetTaskID() string
	GetWorktreePath() string
	Close() error
},
) *ConductorAdapter {
	return &ConductorAdapter{conductor: cond}
}

// Start begins the task from the given reference.
func (a *ConductorAdapter) Start(ctx context.Context, ref string) error {
	return a.conductor.Start(ctx, ref)
}

// Plan runs the planning phase.
func (a *ConductorAdapter) Plan(ctx context.Context) error {
	return a.conductor.Plan(ctx)
}

// Implement runs the implementation phase.
func (a *ConductorAdapter) Implement(ctx context.Context) error {
	return a.conductor.Implement(ctx)
}

// AddNote adds a note to the task.
func (a *ConductorAdapter) AddNote(ctx context.Context, message string) error {
	return a.conductor.AddNote(ctx, message)
}

// GetTaskID returns the active task ID.
func (a *ConductorAdapter) GetTaskID() string {
	return a.conductor.GetTaskID()
}

// GetWorktreePath returns the worktree path.
func (a *ConductorAdapter) GetWorktreePath() string {
	return a.conductor.GetWorktreePath()
}

// Close performs cleanup.
func (a *ConductorAdapter) Close() error {
	return a.conductor.Close()
}

// Verify interface compliance at compile time.
var _ TaskConductor = (*ConductorAdapter)(nil)
