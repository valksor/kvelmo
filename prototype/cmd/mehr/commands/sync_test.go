package commands

import (
	"testing"
)

func TestSyncCmd(t *testing.T) {
	// Test that sync command is registered
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

	// Verify command is added to root
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
