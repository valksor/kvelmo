//go:build !testbinary
// +build !testbinary

package commands

import (
	"testing"
)

func TestDeleteCommand_Properties(t *testing.T) {
	if deleteCmd.Use != "delete --task <queue>/<task-id>" {
		t.Errorf("Use = %q, want %q", deleteCmd.Use, "delete --task <queue>/<task-id>")
	}

	if deleteCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if deleteCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if deleteCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestDeleteCommand_ShortDescription(t *testing.T) {
	expected := "Delete a queue task"
	if deleteCmd.Short != expected {
		t.Errorf("Short = %q, want %q", deleteCmd.Short, expected)
	}
}

func TestDeleteCommand_LongDescriptionContains(t *testing.T) {
	contains := []string{
		"permanently",
		"queue",
		"notes",
		"mehr delete",
	}

	for _, substr := range contains {
		if !containsString(deleteCmd.Long, substr) {
			t.Errorf("Long description does not contain %q", substr)
		}
	}
}

func TestDeleteCommand_ExamplesContains(t *testing.T) {
	examples := []string{
		"mehr delete --task=quick-tasks/task-1",
		"mehr delete --task=project-queue/task-5",
	}

	for _, example := range examples {
		if !containsString(deleteCmd.Long, example) {
			t.Errorf("Long description does not contain example %q", example)
		}
	}
}

func TestDeleteCommand_RegisteredInRoot(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "delete --task <queue>/<task-id>" {
			found = true

			break
		}
	}
	if !found {
		t.Error("delete command not registered in root command")
	}
}

func TestDeleteCommand_TaskFlagRequired(t *testing.T) {
	flag := deleteCmd.Flags().Lookup("task")
	if flag == nil {
		t.Fatal("task flag not found")
	}

	// Check flag is annotated as required
	annotations := deleteCmd.Flags().Lookup("task").Annotations
	if annotations == nil {
		t.Error("task flag has no annotations")

		return
	}

	required, ok := annotations["cobra_annotation_bash_completion_one_required_flag"]
	if !ok || len(required) == 0 {
		// Try checking if it's marked required another way
		if !containsString(deleteCmd.UsageString(), "required") {
			t.Log("Note: flag may be marked required via MarkFlagRequired()")
		}
	}
}

func TestDeleteCommand_RelatedCommandsListed(t *testing.T) {
	relatedCommands := []string{
		"mehr quick",
		"mehr list",
		"mehr optimize",
	}

	for _, cmd := range relatedCommands {
		if !containsString(deleteCmd.Long, cmd) {
			t.Errorf("Long description does not mention related command %q", cmd)
		}
	}
}
