package commands

import (
	"testing"

	"github.com/valksor/go-mehrhof/internal/vcs"
)

func TestCommitCommand_HasFlags(t *testing.T) {
	t.Parallel()

	if commitCmd.Flags().Lookup("push") == nil {
		t.Error("commitCmd missing --push flag")
	}
	if commitCmd.Flags().Lookup("all") == nil {
		t.Error("commitCmd missing --all flag")
	}
	if commitCmd.Flags().Lookup("dry-run") == nil {
		t.Error("commitCmd missing --dry-run flag")
	}
	if commitCmd.Flags().Lookup("note") == nil {
		t.Error("commitCmd missing --note flag")
	}
	if commitCmd.Flags().Lookup("agent-commit") == nil {
		t.Error("commitCmd missing --agent-commit flag")
	}
}

func TestAgentAdapter_ImplementsVCSAgent(t *testing.T) {
	t.Parallel()

	// Verify the adapter implements the vcs.Agent interface
	var _ vcs.Agent = (*agentAdapter)(nil)
}

func TestAgentAdapter_Run(t *testing.T) {
	t.Parallel()

	// Verify the adapter type is correctly defined
	// Full integration tests would require mocking the agent
	var _ *agentAdapter
}
