//go:build !testbinary
// +build !testbinary

package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// chdirToModuleRoot changes to the project root directory where go.mod lives,
// needed because template loading uses os.ReadFile with paths relative to CWD.
func chdirToModuleRoot(t *testing.T) {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			t.Chdir(dir)

			return
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find go.mod in any parent directory")
		}

		dir = parent
	}
}

func TestTemplatesCommand_Properties(t *testing.T) {
	if templatesCmd.Use != "templates" {
		t.Errorf("Use = %q, want %q", templatesCmd.Use, "templates")
	}

	if templatesCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if templatesCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if templatesCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestTemplateShowCommand_Properties(t *testing.T) {
	if templateShowCmd.Use != "show <name>" {
		t.Errorf("Use = %q, want %q", templateShowCmd.Use, "show <name>")
	}

	if templateShowCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if templateShowCmd.RunE == nil {
		t.Error("RunE not set")
	}

	if templateShowCmd.Args == nil {
		t.Error("Args validator not set")
	}
}

func TestTemplateApplyCommand_Properties(t *testing.T) {
	if templateApplyCmd.Use != "apply <name> <file>" {
		t.Errorf("Use = %q, want %q", templateApplyCmd.Use, "apply <name> <file>")
	}

	if templateApplyCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if templateApplyCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if templateApplyCmd.RunE == nil {
		t.Error("RunE not set")
	}

	if templateApplyCmd.Args == nil {
		t.Error("Args validator not set")
	}
}

func TestTemplatesCommand_ShortDescription(t *testing.T) {
	expected := "Manage task templates"
	if templatesCmd.Short != expected {
		t.Errorf("Short = %q, want %q", templatesCmd.Short, expected)
	}
}

func TestTemplatesCommand_LongDescriptionContains(t *testing.T) {
	contains := []string{
		"templates",
		"frontmatter",
		"workflow settings",
	}

	for _, substr := range contains {
		if !containsString(templatesCmd.Long, substr) {
			t.Errorf("Long description does not contain %q", substr)
		}
	}
}

func TestTemplatesCommand_DocumentsBuiltInTemplates(t *testing.T) {
	templates := []string{
		"bug-fix",
		"feature",
		"refactor",
		"docs",
		"test",
		"chore",
	}

	for _, tpl := range templates {
		if !containsString(templatesCmd.Long, tpl) {
			t.Errorf("Long description does not document template %q", tpl)
		}
	}
}

func TestTemplatesCommand_HasSubcommands(t *testing.T) {
	subcommands := templatesCmd.Commands()
	if len(subcommands) < 2 {
		t.Errorf("templates command has %d subcommands, want at least 2", len(subcommands))
	}

	expectedSubcommands := []string{"show <name>", "apply <name> <file>"}
	for _, exp := range expectedSubcommands {
		found := false
		for _, cmd := range subcommands {
			if cmd.Use == exp {
				found = true

				break
			}
		}
		if !found {
			t.Errorf("templates command missing subcommand %q", exp)
		}
	}
}

func TestTemplatesCommand_RegisteredInRoot(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "templates" {
			found = true

			break
		}
	}
	if !found {
		t.Error("templates command not registered in root command")
	}
}

func TestTemplateApplyCommand_Examples(t *testing.T) {
	examples := []string{
		"mehr templates apply bug-fix task.md",
		"mehr templates apply feature",
	}

	for _, example := range examples {
		if !containsString(templateApplyCmd.Long, example) {
			t.Errorf("Long description does not contain example %q", example)
		}
	}
}

func TestTemplateApplyCommand_DocumentsMerge(t *testing.T) {
	if !containsString(templateApplyCmd.Long, "merge") {
		t.Error("Long description does not mention merging with existing frontmatter")
	}
}

func TestRunTemplatesList(t *testing.T) {
	chdirToModuleRoot(t)

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	err := runTemplatesList(nil, nil)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("runTemplatesList() error = %v", err)
	}

	expectedSubstrings := []string{
		"Available templates:",
		"bug-fix",
		"feature",
		"refactor",
		"docs",
		"test",
		"chore",
		"mehr templates show",
		"mehr templates apply",
	}

	for _, substr := range expectedSubstrings {
		if !strings.Contains(output, substr) {
			t.Errorf("output does not contain %q\nGot:\n%s", substr, output)
		}
	}
}

func TestRunTemplateShow_ValidTemplate(t *testing.T) {
	chdirToModuleRoot(t)

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	err := runTemplateShow(nil, []string{"feature"})

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("runTemplateShow(feature) error = %v", err)
	}

	expectedSubstrings := []string{
		"Template:",
		"feature",
		"Description:",
		"Example usage:",
	}

	for _, substr := range expectedSubstrings {
		if !strings.Contains(output, substr) {
			t.Errorf("output does not contain %q\nGot:\n%s", substr, output)
		}
	}
}

func TestRunTemplateShow_InvalidTemplate(t *testing.T) {
	chdirToModuleRoot(t)

	err := runTemplateShow(nil, []string{"nonexistent"})
	if err == nil {
		t.Fatal("runTemplateShow(nonexistent) expected error, got nil")
	}

	if !strings.Contains(err.Error(), "load template") {
		t.Errorf("error = %q, want it to contain 'load template'", err.Error())
	}
}

func TestRunTemplateApply_NewFile(t *testing.T) {
	chdirToModuleRoot(t)

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "task.md")

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	err := runTemplateApply(cmd, []string{"feature", filePath})

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("runTemplateApply() error = %v", err)
	}

	// Verify stdout message
	if !strings.Contains(output, "Applied template") {
		t.Errorf("output does not contain 'Applied template'\nGot:\n%s", output)
	}

	// Verify a file was created
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", filePath, err)
	}

	content := string(data)
	if content == "" {
		t.Error("applied template produced empty file")
	}

	// New files get a placeholder title
	if !strings.Contains(content, "Task Title") {
		t.Errorf("file content does not contain placeholder title\nGot:\n%s", content)
	}
}
