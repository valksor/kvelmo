//go:build !testbinary
// +build !testbinary

package commands

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/valksor/go-toolkit/paths"
)

// Note: TestPlanCommand_Aliases and TestPlanCommand_StandaloneFlag are in common_test.go

func TestPlanCommand_Properties(t *testing.T) {
	if planCmd.Use != "plan [topic]" {
		t.Errorf("Use = %q, want %q", planCmd.Use, "plan [topic]")
	}

	if planCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if planCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if planCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestPlanCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		shorthand    string
		defaultValue string
	}{
		{
			name:         "standalone flag",
			flagName:     "standalone",
			shorthand:    "",
			defaultValue: "false",
		},
		{
			name:         "seed flag",
			flagName:     "seed",
			shorthand:    "s",
			defaultValue: "",
		},
		{
			name:         "full-context flag",
			flagName:     "full-context",
			shorthand:    "",
			defaultValue: "false",
		},
		{
			name:         "agent-plan flag",
			flagName:     "agent-plan",
			shorthand:    "",
			defaultValue: "",
		},
		{
			name:         "force flag",
			flagName:     "force",
			shorthand:    "",
			defaultValue: "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := planCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("flag %q not found", tt.flagName)

				return
			}

			if flag.DefValue != tt.defaultValue {
				t.Errorf("flag %q default value = %q, want %q", tt.flagName, flag.DefValue, tt.defaultValue)
			}

			if tt.shorthand != "" {
				shorthand := planCmd.Flags().ShorthandLookup(tt.shorthand)
				if shorthand == nil {
					t.Errorf("shorthand %q not found for flag %q", tt.shorthand, tt.flagName)
				}
			}
		})
	}
}

func TestPlanCommand_ShortDescription(t *testing.T) {
	expected := "Create implementation specifications for the active task"
	if planCmd.Short != expected {
		t.Errorf("Short = %q, want %q", planCmd.Short, expected)
	}
}

func TestPlanCommand_LongDescriptionContains(t *testing.T) {
	contains := []string{
		"planning phase",
		"specification files",
		"work directory",
	}

	for _, substr := range contains {
		if !containsString(planCmd.Long, substr) {
			t.Errorf("Long description does not contain %q", substr)
		}
	}
}

func TestPlanCommand_DocumentsStandaloneMode(t *testing.T) {
	if !containsString(planCmd.Long, "STANDALONE MODE") {
		t.Error("Long description does not document STANDALONE MODE section")
	}

	if !containsString(planCmd.Long, "--standalone") {
		t.Error("Long description does not mention --standalone flag")
	}
}

func TestPlanCommand_DocumentsSeedTopic(t *testing.T) {
	if !containsString(planCmd.Long, "SEED TOPIC") {
		t.Error("Long description does not document SEED TOPIC section")
	}

	if !containsString(planCmd.Long, "--seed") {
		t.Error("Long description does not mention --seed flag")
	}
}

func TestPlanCommand_NoAliases(t *testing.T) {
	// Aliases removed in favor of prefix matching
	if len(planCmd.Aliases) > 0 {
		t.Errorf("plan command should have no aliases, got %v", planCmd.Aliases)
	}
}

func TestPlanCommand_RegisteredInRoot(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "plan [topic]" {
			found = true

			break
		}
	}
	if !found {
		t.Error("plan command not registered in root command")
	}
}

func TestPlanCommand_Examples(t *testing.T) {
	examples := []string{
		"mehr plan",
		"--verbose",
		"--standalone",
		"--full-context",
	}

	for _, example := range examples {
		if !containsString(planCmd.Long, example) {
			t.Errorf("Long description does not contain example %q", example)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Behavioral tests for runStandalonePlan
// ─────────────────────────────────────────────────────────────────────────────

// savePlanFlags saves and defers restore of plan-related package vars.
func savePlanFlags(t *testing.T) {
	t.Helper()
	origSeed := planSeed
	origStandalone := planStandalone
	t.Cleanup(func() {
		planSeed = origSeed
		planStandalone = origStandalone
	})
	planSeed = ""
	planStandalone = false
}

// runStandalonePlanCapture pipes stdin input and captures stdout.
func runStandalonePlanCapture(t *testing.T, stdinInput string) (string, error) {
	t.Helper()

	// Pipe stdin
	oldStdin := os.Stdin
	stdinR, stdinW, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatalf("os.Pipe (stdin): %v", pipeErr)
	}
	os.Stdin = stdinR
	_, _ = stdinW.WriteString(stdinInput)
	_ = stdinW.Close()
	t.Cleanup(func() { os.Stdin = oldStdin })

	// Capture stdout
	r, w, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatalf("os.Pipe (stdout): %v", pipeErr)
	}
	oldStdout := os.Stdout
	os.Stdout = w

	err := runStandalonePlan()

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	return buf.String(), err
}

func TestRunStandalonePlan_WithSeed(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := t.TempDir()
	t.Cleanup(paths.SetHomeDirForTesting(homeDir))
	t.Chdir(tmpDir)
	savePlanFlags(t)
	planSeed = "build a REST API"

	output, err := runStandalonePlanCapture(t, "quit\n")
	if err != nil {
		t.Fatalf("runStandalonePlan() error = %v", err)
	}

	for _, substr := range []string{"planning session started", "Seed topic:", "build a REST API", "Seed recorded:"} {
		if !strings.Contains(output, substr) {
			t.Errorf("output missing %q\nGot:\n%s", substr, output)
		}
	}
}

func TestRunStandalonePlan_NoSeed(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := t.TempDir()
	t.Cleanup(paths.SetHomeDirForTesting(homeDir))
	t.Chdir(tmpDir)
	savePlanFlags(t)

	output, err := runStandalonePlanCapture(t, "quit\n")
	if err != nil {
		t.Fatalf("runStandalonePlan() error = %v", err)
	}

	if !strings.Contains(output, "planning session started") {
		t.Errorf("output missing 'planning session started'\nGot:\n%s", output)
	}
	if !strings.Contains(output, "Planning session ended") {
		t.Errorf("output missing 'Planning session ended'\nGot:\n%s", output)
	}
	if strings.Contains(output, "Seed topic:") {
		t.Errorf("output should NOT contain 'Seed topic:' when no seed\nGot:\n%s", output)
	}
}
