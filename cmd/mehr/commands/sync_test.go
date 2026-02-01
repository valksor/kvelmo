//go:build !testbinary
// +build !testbinary

package commands

import (
	"strings"
	"testing"

	"github.com/valksor/go-mehrhof/internal/provider"
)

func TestSyncCmd(t *testing.T) {
	// Test that the sync command is registered
	cmd, _, err := rootCmd.Find([]string{"sync"})
	if err != nil {
		t.Fatalf("sync command not found: %v", err)
	}

	if cmd == nil {
		t.Fatal("sync command is nil")

		return
	}

	// Verify command properties
	if cmd.Use != "sync <task-id>" {
		t.Errorf("expected Use 'sync <task-id>', got %q", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Short description is empty")
	}

	if cmd.Long == "" {
		t.Error("Long description is empty")
	}

	// Verify args
	if cmd.Args == nil {
		t.Error("Args validator is nil")
	}

	// Verify command is added to the root
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "sync" {
			found = true

			break
		}
	}

	if !found {
		t.Error("sync command not added to root command")
	}
}

func TestExtractContent(t *testing.T) {
	t.Run("full work unit", func(t *testing.T) {
		wu := &provider.WorkUnit{
			Title:       "Fix login bug",
			Description: "Users can't log in with SSO.",
			Comments: []provider.Comment{
				{Author: provider.Person{ID: "alice"}, Body: "Tried clearing cookies."},
			},
		}
		result := extractContent(wu)
		for _, substr := range []string{"# Fix login bug", "Users can't log in", "## Comments", "alice", "Tried clearing"} {
			if !strings.Contains(result, substr) {
				t.Errorf("output missing %q\nGot:\n%s", substr, result)
			}
		}
	})

	t.Run("title only", func(t *testing.T) {
		wu := &provider.WorkUnit{Title: "Quick fix"}
		result := extractContent(wu)
		if !strings.Contains(result, "# Quick fix") {
			t.Errorf("output missing '# Quick fix'\nGot:\n%s", result)
		}
		if strings.Contains(result, "## Comments") {
			t.Errorf("should NOT contain '## Comments'\nGot:\n%s", result)
		}
	})

	t.Run("empty", func(t *testing.T) {
		wu := &provider.WorkUnit{}
		result := extractContent(wu)
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})
}
