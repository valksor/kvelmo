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

func TestListCommand_Properties(t *testing.T) {
	if listCmd.Use != "list" {
		t.Errorf("Use = %q, want %q", listCmd.Use, "list")
	}

	if listCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if listCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if listCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestListCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		shorthand    string
		defaultValue string
	}{
		{
			name:         "worktrees flag",
			flagName:     "worktrees",
			shorthand:    "w",
			defaultValue: "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := listCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("flag %q not found", tt.flagName)

				return
			}

			if flag.DefValue != tt.defaultValue {
				t.Errorf("flag %q default value = %q, want %q", tt.flagName, flag.DefValue, tt.defaultValue)
			}

			if tt.shorthand != "" {
				shorthand := listCmd.Flags().ShorthandLookup(tt.shorthand)
				if shorthand == nil {
					t.Errorf("shorthand %q not found for flag %q", tt.shorthand, tt.flagName)
				}
			}
		})
	}
}

func TestListCommand_ShortDescription(t *testing.T) {
	expected := "List all tasks in workspace"
	if listCmd.Short != expected {
		t.Errorf("Short = %q, want %q", listCmd.Short, expected)
	}
}

func TestListCommand_LongDescriptionContains(t *testing.T) {
	contains := []string{
		"List all tasks",
		"worktree paths",
		"states",
		"parallel tasks",
	}

	for _, substr := range contains {
		if !containsString(listCmd.Long, substr) {
			t.Errorf("Long description does not contain %q", substr)
		}
	}
}

func TestListCommand_Examples(t *testing.T) {
	examples := []string{
		"mehr list",
		"--worktrees",
	}

	for _, example := range examples {
		if !containsString(listCmd.Long, example) {
			t.Errorf("Long description does not contain example %q", example)
		}
	}
}

func TestListCommand_RegisteredInRoot(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "list" {
			found = true

			break
		}
	}
	if !found {
		t.Error("list command not registered in root command")
	}
}

func TestListCommand_WorktreesFlagShorthand(t *testing.T) {
	flag := listCmd.Flags().Lookup("worktrees")
	if flag == nil {
		t.Fatal("worktrees flag not found")

		return
	}
	if flag.Shorthand != "w" {
		t.Errorf("worktrees flag shorthand = %q, want 'w'", flag.Shorthand)
	}
}

func TestListCommand_NoAliases(t *testing.T) {
	// List command doesn't have aliases currently
	if len(listCmd.Aliases) > 0 {
		// If aliases are added in the future, document them here
		t.Logf("Note: list command has aliases: %v", listCmd.Aliases)
	}
}

func TestListCommand_DocumentsWorktrees(t *testing.T) {
	// Should explain worktree functionality
	if !containsString(listCmd.Long, "worktree") {
		t.Error("Long description does not mention worktrees")
	}

	if !containsString(listCmd.Long, "separate terminals") || !containsString(listCmd.Long, "independent") {
		t.Error("Long description does not explain worktree usage")
	}
}

func TestRunList_RunningEmpty(t *testing.T) {
	// Save and restore package-level vars
	oldFormat := listFormat
	oldRunning := listRunning

	defer func() {
		listFormat = oldFormat
		listRunning = oldRunning
	}()

	listFormat = "table"
	listRunning = true

	// Capture stdout
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	err := runListRunning(context.Background())

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("runListRunning() returned error: %v", err)
	}

	if !strings.Contains(output, "No running parallel tasks.") {
		t.Errorf("output = %q, want it to contain %q", output, "No running parallel tasks.")
	}
}

func TestFormatLabels(t *testing.T) {
	tests := []struct {
		name   string
		labels []string
		want   string
	}{
		{
			name:   "empty slice",
			labels: []string{},
			want:   "-",
		},
		{
			name:   "single label",
			labels: []string{"priority:high"},
			want:   "priority:high",
		},
		{
			name:   "multiple labels",
			labels: []string{"priority:high", "type:bug"},
			want:   "priority:high, type:bug",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatLabels(tt.labels)
			if got != tt.want {
				t.Errorf("formatLabels(%v) = %q, want %q", tt.labels, got, tt.want)
			}
		})
	}
}

func TestRunList_EmptyWorkspace(t *testing.T) {
	tc := NewTestContext(t)
	_ = tc

	// Save/restore all list flags
	origFormat := listFormat
	origRunning := listRunning
	origWorktrees := listWorktreesOnly
	origSearch := listSearch
	origFilter := listFilter
	origSort := listSort
	origLabelFilter := listLabelFilter
	origLabelAny := listLabelAny
	origNoLabel := listNoLabel

	defer func() {
		listFormat = origFormat
		listRunning = origRunning
		listWorktreesOnly = origWorktrees
		listSearch = origSearch
		listFilter = origFilter
		listSort = origSort
		listLabelFilter = origLabelFilter
		listLabelAny = origLabelAny
		listNoLabel = origNoLabel
	}()

	listFormat = "table"
	listRunning = false
	listWorktreesOnly = false
	listSearch = ""
	listFilter = ""
	listSort = ""
	listLabelFilter = ""
	listLabelAny = nil
	listNoLabel = false

	// Capture stdout
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := runList(cmd, nil)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("runList() returned error: %v", err)
	}

	if !strings.Contains(output, "No tasks found") {
		t.Errorf("output = %q, want it to contain %q", output, "No tasks found")
	}
}

func TestRunList_WithTasks(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := t.TempDir()

	// Override the global home directory so runList's internal OpenWorkspace
	// resolves to the same data directory as our test workspace.
	restoreHome := paths.SetHomeDirForTesting(homeDir)
	defer restoreHome()

	// Open workspace with the same homeDir
	cfg := storage.NewDefaultWorkspaceConfig()
	cfg.Storage.HomeDir = homeDir

	ws, err := storage.OpenWorkspace(context.Background(), tmpDir, cfg)
	if err != nil {
		t.Fatalf("OpenWorkspace: %v", err)
	}
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	// Create tasks
	activeTask := storage.NewActiveTask("task-1", "file:task1.md", ws.WorkPath("task-1"))
	if err := ws.SaveActiveTask(activeTask); err != nil {
		t.Fatalf("SaveActiveTask: %v", err)
	}

	work1, err := ws.CreateWork("task-1", storage.SourceInfo{
		Type: "file",
		Ref:  "task.md",
	})
	if err != nil {
		t.Fatalf("CreateWork task-1: %v", err)
	}
	work1.Metadata.Title = "First Task"
	if err := ws.SaveWork(work1); err != nil {
		t.Fatalf("SaveWork task-1: %v", err)
	}

	work2, err := ws.CreateWork("task-2", storage.SourceInfo{
		Type: "file",
		Ref:  "task.md",
	})
	if err != nil {
		t.Fatalf("CreateWork task-2: %v", err)
	}
	work2.Metadata.Title = "Second Task"
	if err := ws.SaveWork(work2); err != nil {
		t.Fatalf("SaveWork task-2: %v", err)
	}

	// Set working directory to tmpDir
	t.Chdir(tmpDir)

	// Save/restore all list flags
	origFormat := listFormat
	origRunning := listRunning
	origWorktrees := listWorktreesOnly
	origSearch := listSearch
	origFilter := listFilter
	origSort := listSort
	origLabelFilter := listLabelFilter
	origLabelAny := listLabelAny
	origNoLabel := listNoLabel

	defer func() {
		listFormat = origFormat
		listRunning = origRunning
		listWorktreesOnly = origWorktrees
		listSearch = origSearch
		listFilter = origFilter
		listSort = origSort
		listLabelFilter = origLabelFilter
		listLabelAny = origLabelAny
		listNoLabel = origNoLabel
	}()

	listFormat = "table"
	listRunning = false
	listWorktreesOnly = false
	listSearch = ""
	listFilter = ""
	listSort = ""
	listLabelFilter = ""
	listLabelAny = nil
	listNoLabel = false

	// Capture stdout
	r, w, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatalf("os.Pipe: %v", pipeErr)
	}
	oldStdout := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	runErr := runList(cmd, nil)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if runErr != nil {
		t.Fatalf("runList() returned error: %v", runErr)
	}

	expectedSubstrings := []string{
		"TASK ID",
		"First Task",
		"Second Task",
		"Legend:",
	}

	for _, substr := range expectedSubstrings {
		if !strings.Contains(output, substr) {
			t.Errorf("output does not contain %q\nGot:\n%s", substr, output)
		}
	}
}

func TestRunList_RunningEmptyJSON(t *testing.T) {
	// Save and restore package-level vars
	oldFormat := listFormat
	oldRunning := listRunning

	defer func() {
		listFormat = oldFormat
		listRunning = oldRunning
	}()

	listFormat = "json"
	listRunning = true

	// Capture stdout
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	err := runListRunning(context.Background())

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("runListRunning() returned error: %v", err)
	}

	// Should output empty JSON array
	if !strings.Contains(output, "[]") {
		t.Errorf("output = %q, want it to contain empty JSON array %q", output, "[]")
	}
}

func TestRunList_CSVFormat(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := t.TempDir()

	restoreHome := paths.SetHomeDirForTesting(homeDir)
	defer restoreHome()

	cfg := storage.NewDefaultWorkspaceConfig()
	cfg.Storage.HomeDir = homeDir

	ws, err := storage.OpenWorkspace(context.Background(), tmpDir, cfg)
	if err != nil {
		t.Fatalf("OpenWorkspace: %v", err)
	}
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	// Create an active task
	activeTask := storage.NewActiveTask("task-1", "file:task1.md", ws.WorkPath("task-1"))
	if err := ws.SaveActiveTask(activeTask); err != nil {
		t.Fatalf("SaveActiveTask: %v", err)
	}

	work1, err := ws.CreateWork("task-1", storage.SourceInfo{
		Type: "file",
		Ref:  "task.md",
	})
	if err != nil {
		t.Fatalf("CreateWork task-1: %v", err)
	}
	work1.Metadata.Title = "CSV Test Task"
	if err := ws.SaveWork(work1); err != nil {
		t.Fatalf("SaveWork task-1: %v", err)
	}

	t.Chdir(tmpDir)

	// Save/restore all list flags
	origFormat := listFormat
	origRunning := listRunning
	origWorktrees := listWorktreesOnly
	origSearch := listSearch
	origFilter := listFilter
	origSort := listSort
	origLabelFilter := listLabelFilter
	origLabelAny := listLabelAny
	origNoLabel := listNoLabel

	defer func() {
		listFormat = origFormat
		listRunning = origRunning
		listWorktreesOnly = origWorktrees
		listSearch = origSearch
		listFilter = origFilter
		listSort = origSort
		listLabelFilter = origLabelFilter
		listLabelAny = origLabelAny
		listNoLabel = origNoLabel
	}()

	listFormat = "csv"
	listRunning = false
	listWorktreesOnly = false
	listSearch = ""
	listFilter = ""
	listSort = ""
	listLabelFilter = ""
	listLabelAny = nil
	listNoLabel = false

	// Capture stdout
	r, w, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatalf("os.Pipe: %v", pipeErr)
	}
	oldStdout := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	runErr := runList(cmd, nil)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if runErr != nil {
		t.Fatalf("runList() returned error: %v", runErr)
	}

	// CSV should have header and data row
	expectedSubstrings := []string{
		"Task ID,State,Title,Worktree,Active,Cost",
		"task-1",
		"CSV Test Task",
	}

	for _, substr := range expectedSubstrings {
		if !strings.Contains(output, substr) {
			t.Errorf("CSV output does not contain %q\nGot:\n%s", substr, output)
		}
	}
}

func TestRunList_JSONFormat(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := t.TempDir()

	restoreHome := paths.SetHomeDirForTesting(homeDir)
	defer restoreHome()

	cfg := storage.NewDefaultWorkspaceConfig()
	cfg.Storage.HomeDir = homeDir

	ws, err := storage.OpenWorkspace(context.Background(), tmpDir, cfg)
	if err != nil {
		t.Fatalf("OpenWorkspace: %v", err)
	}
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	// Create an active task
	activeTask := storage.NewActiveTask("task-1", "file:task1.md", ws.WorkPath("task-1"))
	if err := ws.SaveActiveTask(activeTask); err != nil {
		t.Fatalf("SaveActiveTask: %v", err)
	}

	work1, err := ws.CreateWork("task-1", storage.SourceInfo{
		Type: "file",
		Ref:  "task.md",
	})
	if err != nil {
		t.Fatalf("CreateWork task-1: %v", err)
	}
	work1.Metadata.Title = "First Task"
	if err := ws.SaveWork(work1); err != nil {
		t.Fatalf("SaveWork task-1: %v", err)
	}

	t.Chdir(tmpDir)

	// Save/restore all list flags
	origFormat := listFormat
	origRunning := listRunning
	origWorktrees := listWorktreesOnly
	origSearch := listSearch
	origFilter := listFilter
	origSort := listSort
	origLabelFilter := listLabelFilter
	origLabelAny := listLabelAny
	origNoLabel := listNoLabel

	defer func() {
		listFormat = origFormat
		listRunning = origRunning
		listWorktreesOnly = origWorktrees
		listSearch = origSearch
		listFilter = origFilter
		listSort = origSort
		listLabelFilter = origLabelFilter
		listLabelAny = origLabelAny
		listNoLabel = origNoLabel
	}()

	listFormat = "json"
	listRunning = false
	listWorktreesOnly = false
	listSearch = ""
	listFilter = ""
	listSort = ""
	listLabelFilter = ""
	listLabelAny = nil
	listNoLabel = false

	// Capture stdout
	r, w, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatalf("os.Pipe: %v", pipeErr)
	}
	oldStdout := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	runErr := runList(cmd, nil)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if runErr != nil {
		t.Fatalf("runList() returned error: %v", runErr)
	}

	expectedSubstrings := []string{
		`"task_id"`,
		`"task-1"`,
		`"First Task"`,
	}

	for _, substr := range expectedSubstrings {
		if !strings.Contains(output, substr) {
			t.Errorf("JSON output does not contain %q\nGot:\n%s", substr, output)
		}
	}
}
