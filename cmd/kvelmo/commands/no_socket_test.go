package commands

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/valksor/kvelmo/pkg/meta"
)

// noSocketWorktreeTest verifies that a command returns an error when no worktree socket is running.
func noSocketWorktreeTest(t *testing.T, name string, fn func(*cobra.Command, []string) error, cmd *cobra.Command) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		t.Setenv(meta.EnvPrefix+"_HOME", t.TempDir())
		err := fn(cmd, nil)
		if err == nil {
			t.Errorf("%s() expected error (no socket), got nil", name)
		}
	})
}

// noSocketGlobalTest verifies that a command returns an error when no global socket is running.
func noSocketGlobalTest(t *testing.T, name string, fn func(*cobra.Command, []string) error, cmd *cobra.Command) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		t.Setenv(meta.EnvPrefix+"_HOME", t.TempDir())
		err := fn(cmd, nil)
		if err == nil {
			t.Errorf("%s() expected error (no global socket), got nil", name)
		}
	})
}

func TestWorktreeCommandsNoSocket(t *testing.T) {
	noSocketWorktreeTest(t, "abandon", runAbandon, AbandonCmd)
	noSocketWorktreeTest(t, "abort", runAbort, AbortCmd)
	noSocketWorktreeTest(t, "plan", runPlan, PlanCmd)
	noSocketWorktreeTest(t, "implement", runImplement, ImplementCmd)
	noSocketWorktreeTest(t, "optimize", runOptimize, OptimizeCmd)
	noSocketWorktreeTest(t, "submit", runSubmit, SubmitCmd)
	noSocketWorktreeTest(t, "undo", runUndo, UndoCmd)
	noSocketWorktreeTest(t, "redo", runRedo, RedoCmd)
	noSocketWorktreeTest(t, "reset", runReset, ResetCmd)
	noSocketWorktreeTest(t, "simplify", runSimplify, SimplifyCmd)
	noSocketWorktreeTest(t, "update", runUpdate, UpdateCmd)
	noSocketWorktreeTest(t, "checkpoints", runCheckpoints, CheckpointsCmd)
	noSocketWorktreeTest(t, "delete", runDelete, DeleteCmd)
	// review, review-list connect directly (no SocketExists check), still return connection error
	noSocketWorktreeTest(t, "review", runReview, ReviewCmd)
	noSocketWorktreeTest(t, "review_list", runReviewList, reviewListCmd)
}

func TestGlobalCommandsNoSocket(t *testing.T) {
	noSocketGlobalTest(t, "workers", runWorkers, WorkersCmd)
	noSocketGlobalTest(t, "jobs_list", runJobsList, jobsListCmd)
	noSocketGlobalTest(t, "projects", runProjects, ProjectsCmd)
}

// TestStopCommand_NoSocket verifies stop prints "No worktree socket running" and returns nil.
func TestStopCommand_NoSocket(t *testing.T) {
	t.Setenv(meta.EnvPrefix+"_HOME", t.TempDir())
	// runStop with no socket returns nil (not an error)
	if err := runStop(StopCmd, nil); err != nil {
		t.Errorf("runStop() no socket error = %v, want nil", err)
	}
}
