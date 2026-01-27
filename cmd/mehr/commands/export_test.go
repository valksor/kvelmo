//go:build !testbinary
// +build !testbinary

package commands

import (
	"testing"
)

func TestExportCommand_Properties(t *testing.T) {
	if exportCmd.Use != "export --task <queue>/<task-id> --output <file>" {
		t.Errorf("Use = %q, want %q", exportCmd.Use, "export --task <queue>/<task-id> --output <file>")
	}

	if exportCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if exportCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if exportCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestExportCommand_ShortDescription(t *testing.T) {
	expected := "Export a queue task to a markdown file"
	if exportCmd.Short != expected {
		t.Errorf("Short = %q, want %q", exportCmd.Short, expected)
	}
}

func TestExportCommand_LongDescriptionContains(t *testing.T) {
	contains := []string{
		"YAML frontmatter",
		"Task description",
		"accumulated notes",
		"mehr start file:",
	}

	for _, substr := range contains {
		if !containsString(exportCmd.Long, substr) {
			t.Errorf("Long description does not contain %q", substr)
		}
	}
}

func TestExportCommand_ExamplesContains(t *testing.T) {
	examples := []string{
		`mehr export --task=quick-tasks/task-1 --output task.md`,
		`mehr export --task=quick-tasks/task-1 --output tasks/feature.md`,
		`mehr start file:task.md`,
	}

	for _, example := range examples {
		if !containsString(exportCmd.Long, example) {
			t.Errorf("Long description does not contain example %q", example)
		}
	}
}

func TestExportCommand_RegisteredInRoot(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "export --task <queue>/<task-id> --output <file>" {
			found = true

			break
		}
	}
	if !found {
		t.Error("export command not registered in root command")
	}
}

func TestExportCommand_HasRequiredFlags(t *testing.T) {
	flags := exportCmd.Flags()

	requiredFlags := []string{
		"task",
		"output",
	}

	for _, flagName := range requiredFlags {
		if flags.Lookup(flagName) == nil {
			t.Errorf("Missing required flag: %s", flagName)
		}
	}
}

func TestExportCommand_SeeAlsoReferences(t *testing.T) {
	seeAlso := []string{
		"mehr quick",
		"mehr optimize",
		"mehr start",
	}

	for _, ref := range seeAlso {
		if !containsString(exportCmd.Long, ref) {
			t.Errorf("Long description should reference %s", ref)
		}
	}
}
