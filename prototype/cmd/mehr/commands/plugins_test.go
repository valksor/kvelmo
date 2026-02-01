//go:build !testbinary
// +build !testbinary

package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valksor/go-toolkit/paths"
)

func TestPluginsCommand_Properties(t *testing.T) {
	if pluginsCmd.Use != "plugins" {
		t.Errorf("Use = %q, want %q", pluginsCmd.Use, "plugins")
	}

	if pluginsCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if pluginsCmd.Long == "" {
		t.Error("Long description is empty")
	}
}

func TestPluginsListCommand_Properties(t *testing.T) {
	if pluginsListCmd.Use != "list" {
		t.Errorf("Use = %q, want %q", pluginsListCmd.Use, "list")
	}

	if pluginsListCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if pluginsListCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if pluginsListCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestPluginsInstallCommand_Properties(t *testing.T) {
	if pluginsInstallCmd.Use != "install <source>" {
		t.Errorf("Use = %q, want %q", pluginsInstallCmd.Use, "install <source>")
	}

	if pluginsInstallCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if pluginsInstallCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if pluginsInstallCmd.RunE == nil {
		t.Error("RunE not set")
	}

	if pluginsInstallCmd.Args == nil {
		t.Error("Args validator not set")
	}
}

func TestPluginsRemoveCommand_Properties(t *testing.T) {
	if pluginsRemoveCmd.Use != "remove <name>" {
		t.Errorf("Use = %q, want %q", pluginsRemoveCmd.Use, "remove <name>")
	}

	if pluginsRemoveCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if pluginsRemoveCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if pluginsRemoveCmd.RunE == nil {
		t.Error("RunE not set")
	}

	if pluginsRemoveCmd.Args == nil {
		t.Error("Args validator not set")
	}
}

func TestPluginsValidateCommand_Properties(t *testing.T) {
	if pluginsValidateCmd.Use != "validate [name]" {
		t.Errorf("Use = %q, want %q", pluginsValidateCmd.Use, "validate [name]")
	}

	if pluginsValidateCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if pluginsValidateCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if pluginsValidateCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestPluginsInfoCommand_Properties(t *testing.T) {
	if pluginsInfoCmd.Use != "info <name>" {
		t.Errorf("Use = %q, want %q", pluginsInfoCmd.Use, "info <name>")
	}

	if pluginsInfoCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if pluginsInfoCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if pluginsInfoCmd.RunE == nil {
		t.Error("RunE not set")
	}

	if pluginsInfoCmd.Args == nil {
		t.Error("Args validator not set")
	}
}

func TestPluginsCommand_ShortDescription(t *testing.T) {
	expected := "Manage plugins"
	if pluginsCmd.Short != expected {
		t.Errorf("Short = %q, want %q", pluginsCmd.Short, expected)
	}
}

func TestPluginsCommand_LongDescriptionContains(t *testing.T) {
	contains := []string{
		"provider",
		"agent",
		"workflow",
		"enabled:",
	}

	for _, substr := range contains {
		if !containsString(pluginsCmd.Long, substr) {
			t.Errorf("Long description does not contain %q", substr)
		}
	}
}

func TestPluginsListCommand_DocumentsLocations(t *testing.T) {
	locations := []string{
		"~/.valksor/mehrhof/plugins/",
		".mehrhof/plugins/",
	}

	for _, loc := range locations {
		if !containsString(pluginsListCmd.Long, loc) {
			t.Errorf("Long description does not document location %q", loc)
		}
	}
}

func TestPluginsCommand_HasSubcommands(t *testing.T) {
	subcommands := pluginsCmd.Commands()
	if len(subcommands) < 5 {
		t.Errorf("plugins command has %d subcommands, want at least 5", len(subcommands))
	}

	expectedSubcommands := []string{"list", "install <source>", "remove <name>", "validate [name]", "info <name>"}
	for _, exp := range expectedSubcommands {
		found := false
		for _, cmd := range subcommands {
			if cmd.Use == exp {
				found = true

				break
			}
		}
		if !found {
			t.Errorf("plugins command missing subcommand %q", exp)
		}
	}
}

func TestPluginsCommand_RegisteredInRoot(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "plugins" {
			found = true

			break
		}
	}
	if !found {
		t.Error("plugins command not registered in root command")
	}
}

func TestPluginsInstallCommand_GlobalFlag(t *testing.T) {
	flag := pluginsInstallCmd.Flags().Lookup("global")
	if flag == nil {
		t.Fatal("global flag not found")

		return
	}
	if flag.DefValue != "false" {
		t.Errorf("global flag default = %q, want 'false'", flag.DefValue)
	}
}

func TestPluginsRemoveCommand_GlobalFlag(t *testing.T) {
	flag := pluginsRemoveCmd.Flags().Lookup("global")
	if flag == nil {
		t.Fatal("global flag not found")

		return
	}
	if flag.DefValue != "false" {
		t.Errorf("global flag default = %q, want 'false'", flag.DefValue)
	}
}

func TestPluginsInstallCommand_Examples(t *testing.T) {
	examples := []string{
		"mehr plugins install",
		"--global",
	}

	for _, example := range examples {
		if !containsString(pluginsInstallCmd.Long, example) {
			t.Errorf("Long description does not contain example %q", example)
		}
	}
}

func TestPluginsValidateCommand_Examples(t *testing.T) {
	examples := []string{
		"mehr plugins validate jira",
		"mehr plugins validate",
	}

	for _, example := range examples {
		if !containsString(pluginsValidateCmd.Long, example) {
			t.Errorf("Long description does not contain example %q", example)
		}
	}
}

func TestRunPluginsList_NoPlugins(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := t.TempDir()
	t.Cleanup(paths.SetHomeDirForTesting(homeDir))
	t.Chdir(tmpDir)

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	err := runPluginsList(nil, nil)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("runPluginsList() error = %v", err)
	}

	if !strings.Contains(output, "No plugins discovered") {
		t.Errorf("output missing 'No plugins discovered'\nGot:\n%s", output)
	}
}

func TestCopyDir(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := filepath.Join(t.TempDir(), "dest")

	// Create source files
	if err := os.MkdirAll(filepath.Join(srcDir, "sub"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	if err := os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := os.WriteFile(filepath.Join(srcDir, "sub", "file2.txt"), []byte("world"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := copyDir(srcDir, dstDir); err != nil {
		t.Fatalf("copyDir() error = %v", err)
	}

	// Verify files were copied
	data, err := os.ReadFile(filepath.Join(dstDir, "file1.txt"))
	if err != nil {
		t.Fatalf("ReadFile file1.txt: %v", err)
	}

	if string(data) != "hello" {
		t.Errorf("file1.txt content = %q, want 'hello'", string(data))
	}

	data, err = os.ReadFile(filepath.Join(dstDir, "sub", "file2.txt"))
	if err != nil {
		t.Fatalf("ReadFile sub/file2.txt: %v", err)
	}

	if string(data) != "world" {
		t.Errorf("sub/file2.txt content = %q, want 'world'", string(data))
	}
}
