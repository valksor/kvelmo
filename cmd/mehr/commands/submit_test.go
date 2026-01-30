//go:build !testbinary
// +build !testbinary

package commands

import (
	"testing"
)

func TestSubmitCommand_Properties(t *testing.T) {
	if submitCmd.Use != "submit --provider <name> [--task <queue>/<task-id> | --source <path>]" {
		t.Errorf("Use = %q, want %q", submitCmd.Use, "submit --provider <name> [--task <queue>/<task-id> | --source <path>]")
	}

	if submitCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if submitCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if submitCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestSubmitCommand_ShortDescription(t *testing.T) {
	expected := "Submit a task to an external provider"
	if submitCmd.Short != expected {
		t.Errorf("Short = %q, want %q", submitCmd.Short, expected)
	}
}

func TestSubmitCommand_LongDescriptionContains(t *testing.T) {
	contains := []string{
		"GitHub",
		"Jira",
		"Wrike",
		"external ID",
		"external provider",
	}

	for _, substr := range contains {
		if !containsString(submitCmd.Long, substr) {
			t.Errorf("Long description does not contain %q", substr)
		}
	}
}

func TestSubmitCommand_ProvidersListed(t *testing.T) {
	// Check that major providers are documented
	providers := []string{
		"github",
		"gitlab",
		"jira",
		"linear",
		"asana",
		"notion",
		"trello",
		"wrike",
	}

	providersText := submitCmd.Long
	for _, provider := range providers {
		if !containsString(providersText, provider) {
			t.Errorf("Provider %s not listed in documentation", provider)
		}
	}
}

func TestSubmitCommand_ExamplesContains(t *testing.T) {
	examples := []string{
		`mehr submit --task=quick-tasks/task-1 --provider github`,
		`--labels urgent`,
		`--dry-run`,
		`--source ./specs/overview.md`,
	}

	for _, example := range examples {
		if !containsString(submitCmd.Long, example) {
			t.Errorf("Long description does not contain example %q", example)
		}
	}
}

func TestSubmitCommand_RegisteredInRoot(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "submit --provider <name> [--task <queue>/<task-id> | --source <path>]" {
			found = true

			break
		}
	}
	if !found {
		t.Error("submit command not registered in root command")
	}
}

func TestSubmitCommand_HasRequiredFlags(t *testing.T) {
	flags := submitCmd.Flags()

	requiredFlags := []string{
		"provider",
		"labels",
		"dry-run",
		"source",
		"note",
		"title",
		"instructions",
		"queue",
		"optimize",
	}

	for _, flagName := range requiredFlags {
		if flags.Lookup(flagName) == nil {
			t.Errorf("Missing required flag: %s", flagName)
		}
	}
}

func TestSubmitCommand_SeeAlsoReferences(t *testing.T) {
	seeAlso := []string{
		"mehr quick",
		"mehr optimize",
		"mehr export",
	}

	for _, ref := range seeAlso {
		if !containsString(submitCmd.Long, ref) {
			t.Errorf("Long description should reference %s", ref)
		}
	}
}
