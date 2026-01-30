package conductor

import (
	"context"
	"fmt"
)

// Factory creates conductor instances for parallel task execution.
// Each task needs its own conductor to maintain isolated state.
//
// The factory stores base options that apply to all conductors,
// allowing per-task customization through the Create() method.
type Factory struct {
	baseOpts     []Option
	registerFunc func(*Conductor) error
}

// FactoryOption configures the factory.
type FactoryOption func(*Factory)

// NewFactory creates a new conductor factory.
// Base options are applied to all conductors created by this factory.
func NewFactory(opts ...FactoryOption) *Factory {
	f := &Factory{
		baseOpts: make([]Option, 0),
	}

	for _, opt := range opts {
		opt(f)
	}

	return f
}

// WithBaseOptions sets the base conductor options applied to all instances.
func WithBaseOptions(opts ...Option) FactoryOption {
	return func(f *Factory) {
		f.baseOpts = append(f.baseOpts, opts...)
	}
}

// WithRegistrationFunc sets a function that registers providers and agents.
// This should call registration.RegisterStandardProviders and RegisterStandardAgents.
func WithRegistrationFunc(fn func(*Conductor) error) FactoryOption {
	return func(f *Factory) {
		f.registerFunc = fn
	}
}

// Create creates a new conductor instance for a task.
// Additional options can be provided to customize this specific conductor.
//
// The conductor is fully initialized and ready to use, but no task is started yet.
// Call conductor.Start(ctx, ref) to begin the task.
func (f *Factory) Create(ctx context.Context, additionalOpts ...Option) (*Conductor, error) {
	// Combine base options with additional options
	opts := make([]Option, 0, len(f.baseOpts)+len(additionalOpts))
	opts = append(opts, f.baseOpts...)
	opts = append(opts, additionalOpts...)

	// Create conductor
	cond, err := New(opts...)
	if err != nil {
		return nil, fmt.Errorf("create conductor: %w", err)
	}

	// Register providers and agents
	if f.registerFunc != nil {
		if err := f.registerFunc(cond); err != nil {
			return nil, fmt.Errorf("register providers/agents: %w", err)
		}
	}

	// Initialize the conductor
	if err := cond.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("initialize conductor: %w", err)
	}

	return cond, nil
}

// CreateForWorktree creates a conductor configured for a specific worktree.
// This is a convenience method that adds worktree-specific options.
func (f *Factory) CreateForWorktree(ctx context.Context, workDir string, additionalOpts ...Option) (*Conductor, error) {
	opts := []Option{
		WithWorkDir(workDir),
		WithUseWorktree(true),
		WithAutoInit(true),
	}
	opts = append(opts, additionalOpts...)

	return f.Create(ctx, opts...)
}

// CreateForParallel creates a conductor configured for parallel execution.
// This ensures worktrees are used to prevent file conflicts.
func (f *Factory) CreateForParallel(ctx context.Context, additionalOpts ...Option) (*Conductor, error) {
	opts := []Option{
		WithUseWorktree(true),
		WithAutoInit(true),
		WithCreateBranch(true),
	}
	opts = append(opts, additionalOpts...)

	return f.Create(ctx, opts...)
}
