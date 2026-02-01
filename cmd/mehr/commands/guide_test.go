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
	"github.com/valksor/go-mehrhof/internal/helper_test"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-toolkit/paths"
)

func TestGuideCommand_Structure(t *testing.T) {
	if guideCmd.Use != "guide" {
		t.Errorf("expected Use to be 'guide', got %q", guideCmd.Use)
	}
	if guideCmd.Short != "What should I do next?" {
		t.Errorf("expected Short to be 'What should I do next?', got %q", guideCmd.Short)
	}
	if guideCmd.Long == "" {
		t.Error("expected Long description to be set")
	}
}

func TestGuideCommand_HasParent(t *testing.T) {
	if !hasCommand(rootCmd, "guide") {
		t.Error("guide command not registered with rootCmd")
	}
}

func TestGuideCommand_NoArgsRequired(t *testing.T) {
	if guideCmd.Args != nil {
		err := guideCmd.Args(guideCmd, []string{})
		if err != nil {
			t.Errorf("expected no args validation, got: %v", err)
		}
	}
}

// setupGuideWorkspace creates a workspace for guide tests.
// Returns the workspace and a cleanup function.
func setupGuideWorkspace(t *testing.T) *storage.Workspace {
	t.Helper()

	tmpDir := t.TempDir()
	homeDir := t.TempDir()

	restoreHome := paths.SetHomeDirForTesting(homeDir)
	t.Cleanup(restoreHome)

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

// runGuideCapture calls runGuide and captures stdout.
func runGuideCapture(t *testing.T) (string, error) {
	t.Helper()

	r, w, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatalf("os.Pipe: %v", pipeErr)
	}

	oldStdout := os.Stdout
	os.Stdout = w

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := runGuide(cmd, nil)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	return buf.String(), err
}

func TestRunGuide_NoActiveTask(t *testing.T) {
	_ = setupGuideWorkspace(t)

	output, err := runGuideCapture(t)
	if err != nil {
		t.Fatalf("runGuide() error = %v", err)
	}

	if !strings.Contains(output, "No active task") {
		t.Errorf("output missing 'No active task'\nGot:\n%s", output)
	}

	if !strings.Contains(output, "mehr start") {
		t.Errorf("output missing 'mehr start'\nGot:\n%s", output)
	}
}

func TestRunGuide_IdleNoSpecs(t *testing.T) {
	ws := setupGuideWorkspace(t)

	activeTask := storage.NewActiveTask("task-1", "file:task1.md", ws.WorkPath("task-1"))
	activeTask.State = "idle"

	if err := ws.SaveActiveTask(activeTask); err != nil {
		t.Fatalf("SaveActiveTask: %v", err)
	}

	work, err := ws.CreateWork("task-1", storage.SourceInfo{
		Type:    "file",
		Ref:     "task.md",
		Content: helper_test.SampleTaskContent("Test Task"),
	})
	if err != nil {
		t.Fatalf("CreateWork: %v", err)
	}

	work.Metadata.Title = "Test Task"

	if err := ws.SaveWork(work); err != nil {
		t.Fatalf("SaveWork: %v", err)
	}

	output, runErr := runGuideCapture(t)
	if runErr != nil {
		t.Fatalf("runGuide() error = %v", runErr)
	}

	if !strings.Contains(output, "mehr plan") {
		t.Errorf("output missing 'mehr plan'\nGot:\n%s", output)
	}

	if !strings.Contains(output, "mehr note") {
		t.Errorf("output missing 'mehr note'\nGot:\n%s", output)
	}
}

func TestRunGuide_Planning(t *testing.T) {
	ws := setupGuideWorkspace(t)

	activeTask := storage.NewActiveTask("task-1", "file:task1.md", ws.WorkPath("task-1"))
	activeTask.State = "planning"

	if err := ws.SaveActiveTask(activeTask); err != nil {
		t.Fatalf("SaveActiveTask: %v", err)
	}

	work, err := ws.CreateWork("task-1", storage.SourceInfo{
		Type:    "file",
		Ref:     "task.md",
		Content: helper_test.SampleTaskContent("Planning Task"),
	})
	if err != nil {
		t.Fatalf("CreateWork: %v", err)
	}

	work.Metadata.Title = "Planning Task"

	if err := ws.SaveWork(work); err != nil {
		t.Fatalf("SaveWork: %v", err)
	}

	output, runErr := runGuideCapture(t)
	if runErr != nil {
		t.Fatalf("runGuide() error = %v", runErr)
	}

	if !strings.Contains(output, "mehr status") {
		t.Errorf("output missing 'mehr status'\nGot:\n%s", output)
	}

	if !strings.Contains(output, "mehr question") {
		t.Errorf("output missing 'mehr question'\nGot:\n%s", output)
	}
}

func TestRunGuide_Implementing(t *testing.T) {
	ws := setupGuideWorkspace(t)

	activeTask := storage.NewActiveTask("task-1", "file:task1.md", ws.WorkPath("task-1"))
	activeTask.State = "implementing"

	if err := ws.SaveActiveTask(activeTask); err != nil {
		t.Fatalf("SaveActiveTask: %v", err)
	}

	work, err := ws.CreateWork("task-1", storage.SourceInfo{
		Type:    "file",
		Ref:     "task.md",
		Content: helper_test.SampleTaskContent("Implementing Task"),
	})
	if err != nil {
		t.Fatalf("CreateWork: %v", err)
	}

	work.Metadata.Title = "Implementing Task"

	if err := ws.SaveWork(work); err != nil {
		t.Fatalf("SaveWork: %v", err)
	}

	output, runErr := runGuideCapture(t)
	if runErr != nil {
		t.Fatalf("runGuide() error = %v", runErr)
	}

	for _, substr := range []string{"mehr status", "mehr undo", "mehr finish"} {
		if !strings.Contains(output, substr) {
			t.Errorf("output missing %q\nGot:\n%s", substr, output)
		}
	}
}

func TestRunGuide_Reviewing(t *testing.T) {
	ws := setupGuideWorkspace(t)

	activeTask := storage.NewActiveTask("task-1", "file:task1.md", ws.WorkPath("task-1"))
	activeTask.State = "reviewing"

	if err := ws.SaveActiveTask(activeTask); err != nil {
		t.Fatalf("SaveActiveTask: %v", err)
	}

	work, err := ws.CreateWork("task-1", storage.SourceInfo{
		Type:    "file",
		Ref:     "task.md",
		Content: helper_test.SampleTaskContent("Reviewing Task"),
	})
	if err != nil {
		t.Fatalf("CreateWork: %v", err)
	}

	work.Metadata.Title = "Reviewing Task"

	if err := ws.SaveWork(work); err != nil {
		t.Fatalf("SaveWork: %v", err)
	}

	output, runErr := runGuideCapture(t)
	if runErr != nil {
		t.Fatalf("runGuide() error = %v", runErr)
	}

	for _, substr := range []string{"mehr status", "mehr finish"} {
		if !strings.Contains(output, substr) {
			t.Errorf("output missing %q\nGot:\n%s", substr, output)
		}
	}
}

func TestRunGuide_Done(t *testing.T) {
	ws := setupGuideWorkspace(t)

	activeTask := storage.NewActiveTask("task-1", "file:task1.md", ws.WorkPath("task-1"))
	activeTask.State = "done"

	if err := ws.SaveActiveTask(activeTask); err != nil {
		t.Fatalf("SaveActiveTask: %v", err)
	}

	work, err := ws.CreateWork("task-1", storage.SourceInfo{
		Type:    "file",
		Ref:     "task.md",
		Content: helper_test.SampleTaskContent("Done Task"),
	})
	if err != nil {
		t.Fatalf("CreateWork: %v", err)
	}

	work.Metadata.Title = "Done Task"

	if err := ws.SaveWork(work); err != nil {
		t.Fatalf("SaveWork: %v", err)
	}

	output, runErr := runGuideCapture(t)
	if runErr != nil {
		t.Fatalf("runGuide() error = %v", runErr)
	}

	if !strings.Contains(output, "Task is complete!") {
		t.Errorf("output missing 'Task is complete!'\nGot:\n%s", output)
	}
}

func TestRunGuide_Waiting(t *testing.T) {
	ws := setupGuideWorkspace(t)

	activeTask := storage.NewActiveTask("task-1", "file:task1.md", ws.WorkPath("task-1"))
	activeTask.State = "waiting"

	if err := ws.SaveActiveTask(activeTask); err != nil {
		t.Fatalf("SaveActiveTask: %v", err)
	}

	work, err := ws.CreateWork("task-1", storage.SourceInfo{
		Type:    "file",
		Ref:     "task.md",
		Content: helper_test.SampleTaskContent("Waiting Task"),
	})
	if err != nil {
		t.Fatalf("CreateWork: %v", err)
	}

	work.Metadata.Title = "Waiting Task"

	if err := ws.SaveWork(work); err != nil {
		t.Fatalf("SaveWork: %v", err)
	}

	output, runErr := runGuideCapture(t)
	if runErr != nil {
		t.Fatalf("runGuide() error = %v", runErr)
	}

	if !strings.Contains(output, "mehr answer") {
		t.Errorf("output missing 'mehr answer'\nGot:\n%s", output)
	}
}

func TestRunGuide_Paused(t *testing.T) {
	ws := setupGuideWorkspace(t)

	activeTask := storage.NewActiveTask("task-1", "file:task1.md", ws.WorkPath("task-1"))
	activeTask.State = "paused"

	if err := ws.SaveActiveTask(activeTask); err != nil {
		t.Fatalf("SaveActiveTask: %v", err)
	}

	work, err := ws.CreateWork("task-1", storage.SourceInfo{
		Type:    "file",
		Ref:     "task.md",
		Content: helper_test.SampleTaskContent("Paused Task"),
	})
	if err != nil {
		t.Fatalf("CreateWork: %v", err)
	}

	work.Metadata.Title = "Paused Task"

	if err := ws.SaveWork(work); err != nil {
		t.Fatalf("SaveWork: %v", err)
	}

	output, runErr := runGuideCapture(t)
	if runErr != nil {
		t.Fatalf("runGuide() error = %v", runErr)
	}

	if !strings.Contains(output, "mehr budget") {
		t.Errorf("output missing 'mehr budget'\nGot:\n%s", output)
	}
}

func TestRunGuide_Failed(t *testing.T) {
	ws := setupGuideWorkspace(t)

	activeTask := storage.NewActiveTask("task-1", "file:task1.md", ws.WorkPath("task-1"))
	activeTask.State = "failed"

	if err := ws.SaveActiveTask(activeTask); err != nil {
		t.Fatalf("SaveActiveTask: %v", err)
	}

	work, err := ws.CreateWork("task-1", storage.SourceInfo{
		Type:    "file",
		Ref:     "task.md",
		Content: helper_test.SampleTaskContent("Failed Task"),
	})
	if err != nil {
		t.Fatalf("CreateWork: %v", err)
	}

	work.Metadata.Title = "Failed Task"

	if err := ws.SaveWork(work); err != nil {
		t.Fatalf("SaveWork: %v", err)
	}

	output, runErr := runGuideCapture(t)
	if runErr != nil {
		t.Fatalf("runGuide() error = %v", runErr)
	}

	for _, substr := range []string{"mehr status", "mehr start"} {
		if !strings.Contains(output, substr) {
			t.Errorf("output missing %q\nGot:\n%s", substr, output)
		}
	}
}
