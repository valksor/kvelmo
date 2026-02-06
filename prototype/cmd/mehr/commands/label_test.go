//go:build !testbinary
// +build !testbinary

package commands

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-toolkit/paths"
)

func TestLabelCommand_Properties(t *testing.T) {
	if labelCmd.Use != "label" {
		t.Errorf("Use = %q, want %q", labelCmd.Use, "label")
	}

	if labelCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if labelCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if labelCmd.RunE != nil {
		t.Error("RunE should be nil for parent command")
	}
}

func TestLabelCommand_HasSubcommands(t *testing.T) {
	expectedSubcommands := []string{"add", "remove", "set", "list"}

	for _, sub := range expectedSubcommands {
		found := false
		for _, cmd := range labelCmd.Commands() {
			if cmd.Name() == sub {
				found = true

				break
			}
		}
		if !found {
			t.Errorf("Missing subcommand %q", sub)
		}
	}
}

func TestLabelAddCommand_Properties(t *testing.T) {
	if labelAddCmd.Use != "add <task-id> <label>..." {
		t.Errorf("Use = %q, want %q", labelAddCmd.Use, "add <task-id> <label>...")
	}

	if labelAddCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if labelAddCmd.Args == nil {
		t.Error("Args not set")
	}

	if labelAddCmd.RunE == nil {
		t.Error("RunE not set")
	}

	// Args should require at least 2 args (task-id + label)
	if err := labelAddCmd.Args(labelAddCmd, []string{}); err == nil {
		t.Error("Args validation should require at least 2 arguments")
	}
}

func TestLabelRemoveCommand_Properties(t *testing.T) {
	if labelRemoveCmd.Use != "remove <task-id> <label>..." {
		t.Errorf("Use = %q, want %q", labelRemoveCmd.Use, "remove <task-id> <label>...")
	}

	if labelRemoveCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if labelRemoveCmd.Args == nil {
		t.Error("Args not set")
	}

	if labelRemoveCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestLabelSetCommand_Properties(t *testing.T) {
	if labelSetCmd.Use != "set <task-id> <label>..." {
		t.Errorf("Use = %q, want %q", labelSetCmd.Use, "set <task-id> <label>...")
	}

	if labelSetCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if labelSetCmd.Args == nil {
		t.Error("Args not set")
	}

	if labelSetCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestLabelListCommand_Properties(t *testing.T) {
	if labelListCmd.Use != "list <task-id>" {
		t.Errorf("Use = %q, want %q", labelListCmd.Use, "list <task-id>")
	}

	if labelListCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if labelListCmd.Args == nil {
		t.Error("Args not set")
	}

	if labelListCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestLabelAddCommand_RegisteredInRoot(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "label" {
			found = true

			break
		}
	}
	if !found {
		t.Error("label command not registered in root command")
	}
}

func TestLabelAddCommand_HasValidArgsPattern(t *testing.T) {
	// Check that the Args function is cobra.MinimumNArgs(2)
	// by verifying it requires at least 2 arguments
	minArgs := 2

	if labelAddCmd.Args == nil {
		t.Fatal("Args validation not configured")
	}

	// Test with insufficient args
	args := []string{"task-id-only"}
	err := labelAddCmd.Args(labelAddCmd, args)
	if err == nil {
		t.Errorf("Expected error with %d args, got nil", len(args))
	}

	// Test with sufficient args
	args = []string{"task-id", "label1"}
	err = labelAddCmd.Args(labelAddCmd, args)
	if err != nil && minArgs <= len(args) {
		t.Errorf("Expected no error with %d args (minimum %d), got %v", len(args), minArgs, err)
	}
}

func TestLabelRemoveCommand_HasValidArgsPattern(t *testing.T) {
	minArgs := 2

	if labelRemoveCmd.Args == nil {
		t.Fatal("Args validation not configured")
	}

	args := []string{"task-id-only"}
	err := labelRemoveCmd.Args(labelRemoveCmd, args)
	if err == nil {
		t.Errorf("Expected error with %d args, got nil", len(args))
	}

	args = []string{"task-id", "label1"}
	err = labelRemoveCmd.Args(labelRemoveCmd, args)
	if err != nil && minArgs <= len(args) {
		t.Errorf("Expected no error with %d args (minimum %d), got %v", len(args), minArgs, err)
	}
}

func TestLabelSetCommand_HasValidArgsPattern(t *testing.T) {
	// set command can accept 1+ args (task-id is required, labels are optional)
	minArgs := 1

	if labelSetCmd.Args == nil {
		t.Fatal("Args validation not configured")
	}

	// Test with no args (should fail since min is 1)
	args := []string{}
	err := labelSetCmd.Args(labelSetCmd, args)
	if err == nil {
		t.Errorf("Expected error with %d args, got nil", len(args))
	}

	args = []string{"task-id"}
	err = labelSetCmd.Args(labelSetCmd, args)
	if err != nil && minArgs <= len(args) {
		t.Errorf("Expected no error with %d args (minimum %d), got %v", len(args), minArgs, err)
	}
}

func TestLabelListCommand_HasValidArgsPattern(t *testing.T) {
	// list command requires exactly 1 arg (task-id)
	if labelListCmd.Args == nil {
		t.Fatal("Args validation not configured")
	}

	// Test with no args (should fail)
	args := []string{}
	err := labelListCmd.Args(labelListCmd, args)
	if err == nil {
		t.Error("Expected error with 0 args, got nil")
	}

	// Test with 1 arg (should pass)
	args = []string{"task-id"}
	err = labelListCmd.Args(labelListCmd, args)
	if err != nil {
		t.Errorf("Expected no error with 1 arg, got %v", err)
	}

	// Test with 2 args (should fail - requires exactly 1)
	args = []string{"task-id", "extra"}
	err = labelListCmd.Args(labelListCmd, args)
	if err == nil {
		t.Error("Expected error with 2 args, got nil")
	}
}

func TestLabelCommand_ShortDescription(t *testing.T) {
	expected := "Manage task labels"
	if labelCmd.Short != expected {
		t.Errorf("Short = %q, want %q", labelCmd.Short, expected)
	}
}

func TestLabelAddCommand_ShortDescription(t *testing.T) {
	expected := "Add labels to a task"
	if labelAddCmd.Short != expected {
		t.Errorf("Short = %q, want %q", labelAddCmd.Short, expected)
	}
}

func TestLabelRemoveCommand_ShortDescription(t *testing.T) {
	expected := "Remove labels from a task"
	if labelRemoveCmd.Short != expected {
		t.Errorf("Short = %q, want %q", labelRemoveCmd.Short, expected)
	}
}

func TestLabelSetCommand_ShortDescription(t *testing.T) {
	expected := "Set task labels (replace all)"
	if labelSetCmd.Short != expected {
		t.Errorf("Short = %q, want %q", labelSetCmd.Short, expected)
	}
}

func TestLabelListCommand_ShortDescription(t *testing.T) {
	expected := "List labels for a task"
	if labelListCmd.Short != expected {
		t.Errorf("Short = %q, want %q", labelListCmd.Short, expected)
	}
}

func TestLabelCommand_LongDescriptionContains(t *testing.T) {
	contains := []string{
		"Add",
		"remove",
		"list",
		"manage",
	}

	for _, substr := range contains {
		if !containsString(labelCmd.Long, substr) {
			t.Errorf("Long description does not contain %q", substr)
		}
	}
}

func setupLabelWorkspace(t *testing.T) *storage.Workspace {
	t.Helper()
	tmpDir := t.TempDir()
	homeDir := t.TempDir()
	t.Cleanup(paths.SetHomeDirForTesting(homeDir))
	cfg := storage.NewDefaultWorkspaceConfig()
	cfg.Storage.HomeDir = homeDir
	ws, err := storage.OpenWorkspace(context.Background(), tmpDir, cfg)
	if err != nil {
		t.Fatalf("OpenWorkspace: %v", err)
	}
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}
	t.Chdir(tmpDir)

	return ws
}

func createLabelTestTask(t *testing.T, ws *storage.Workspace) {
	t.Helper()
	const taskID = "task-1"
	const title = "Label Test"
	activeTask := storage.NewActiveTask(taskID, "file:task.md", ws.WorkPath(taskID))
	if err := ws.SaveActiveTask(activeTask); err != nil {
		t.Fatalf("SaveActiveTask: %v", err)
	}
	work, err := ws.CreateWork(taskID, storage.SourceInfo{
		Type: "file", Ref: "task.md",
	})
	if err != nil {
		t.Fatalf("CreateWork: %v", err)
	}
	work.Metadata.Title = title
	if err := ws.SaveWork(work); err != nil {
		t.Fatalf("SaveWork: %v", err)
	}
}

func captureLabelStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	r, w, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatalf("os.Pipe: %v", pipeErr)
	}
	oldStdout := os.Stdout
	os.Stdout = w
	err := fn()
	_ = w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	return buf.String(), err
}

func TestRunLabelAdd(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ws := setupLabelWorkspace(t)
		createLabelTestTask(t, ws)
		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())

		output, err := captureLabelStdout(t, func() error {
			return runLabelAdd(cmd, []string{"task-1", "bug", "priority:high"})
		})
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if !strings.Contains(output, "Added 2 label(s)") {
			t.Errorf("output missing 'Added 2 label(s)'\nGot:\n%s", output)
		}
		labels, _ := ws.GetLabels("task-1")
		if len(labels) != 2 {
			t.Errorf("expected 2 labels, got %d: %v", len(labels), labels)
		}
	})

	t.Run("task not found", func(t *testing.T) {
		_ = setupLabelWorkspace(t)
		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())
		err := runLabelAdd(cmd, []string{"nonexistent", "bug"})
		if err == nil || !strings.Contains(err.Error(), "task not found") {
			t.Errorf("expected 'task not found' error, got %v", err)
		}
	})
}

func TestRunLabelRemove(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ws := setupLabelWorkspace(t)
		createLabelTestTask(t, ws)
		_ = ws.AddLabel("task-1", "bug")
		_ = ws.AddLabel("task-1", "priority:high")
		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())

		output, err := captureLabelStdout(t, func() error {
			return runLabelRemove(cmd, []string{"task-1", "bug"})
		})
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if !strings.Contains(output, "Removed 1 label(s)") {
			t.Errorf("output missing 'Removed 1 label(s)'\nGot:\n%s", output)
		}
		labels, _ := ws.GetLabels("task-1")
		if len(labels) != 1 {
			t.Errorf("expected 1 label, got %d: %v", len(labels), labels)
		}
	})

	t.Run("task not found", func(t *testing.T) {
		_ = setupLabelWorkspace(t)
		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())
		err := runLabelRemove(cmd, []string{"nonexistent", "bug"})
		if err == nil || !strings.Contains(err.Error(), "task not found") {
			t.Errorf("expected 'task not found' error, got %v", err)
		}
	})
}

func TestRunLabelSet(t *testing.T) {
	t.Run("set labels", func(t *testing.T) {
		ws := setupLabelWorkspace(t)
		createLabelTestTask(t, ws)
		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())

		output, err := captureLabelStdout(t, func() error {
			return runLabelSet(cmd, []string{"task-1", "new-label", "another"})
		})
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if !strings.Contains(output, "Set 2 label(s)") {
			t.Errorf("output missing 'Set 2 label(s)'\nGot:\n%s", output)
		}
	})

	t.Run("clear all", func(t *testing.T) {
		ws := setupLabelWorkspace(t)
		createLabelTestTask(t, ws)
		_ = ws.AddLabel("task-1", "bug")
		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())

		output, err := captureLabelStdout(t, func() error {
			return runLabelSet(cmd, []string{"task-1"})
		})
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if !strings.Contains(output, "Cleared all labels") {
			t.Errorf("output missing 'Cleared all labels'\nGot:\n%s", output)
		}
	})
}

func TestRunLabelList(t *testing.T) {
	t.Run("with labels", func(t *testing.T) {
		ws := setupLabelWorkspace(t)
		createLabelTestTask(t, ws)
		_ = ws.AddLabel("task-1", "bug")
		_ = ws.AddLabel("task-1", "priority:high")
		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())

		output, err := captureLabelStdout(t, func() error {
			return runLabelList(cmd, []string{"task-1"})
		})
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		for _, substr := range []string{"Labels for", "bug", "priority:high"} {
			if !strings.Contains(output, substr) {
				t.Errorf("output missing %q\nGot:\n%s", substr, output)
			}
		}
	})

	t.Run("no labels", func(t *testing.T) {
		ws := setupLabelWorkspace(t)
		createLabelTestTask(t, ws)
		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())

		output, err := captureLabelStdout(t, func() error {
			return runLabelList(cmd, []string{"task-1"})
		})
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if !strings.Contains(output, "(no labels)") {
			t.Errorf("output missing '(no labels)'\nGot:\n%s", output)
		}
	})
}
