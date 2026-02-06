package views

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/helper_test"
	"github.com/valksor/go-mehrhof/internal/provider/file"
	"github.com/valksor/go-mehrhof/internal/storage"
)

func TestComputePageData(t *testing.T) {
	tests := []struct {
		name             string
		mode             string
		isGlobalMode     bool
		authEnabled      bool
		canSwitchProject bool
		isViewer         bool
		currentUser      string
		want             PageData
	}{
		{
			name:             "project mode with auth",
			mode:             "project",
			isGlobalMode:     false,
			authEnabled:      true,
			canSwitchProject: true,
			isViewer:         false,
			currentUser:      "user@example.com",
			want: PageData{
				Mode:             "project",
				IsGlobalMode:     false,
				IsProjectMode:    true,
				AuthEnabled:      true,
				CanSwitchProject: true,
				IsViewer:         false,
				CurrentUser:      "user@example.com",
			},
		},
		{
			name:             "global mode without auth",
			mode:             "global",
			isGlobalMode:     true,
			authEnabled:      false,
			canSwitchProject: false,
			isViewer:         false,
			currentUser:      "",
			want: PageData{
				Mode:             "global",
				IsGlobalMode:     true,
				IsProjectMode:    false,
				AuthEnabled:      false,
				CanSwitchProject: false,
				IsViewer:         false,
				CurrentUser:      "",
			},
		},
		{
			name:             "viewer mode",
			mode:             "project",
			isGlobalMode:     false,
			authEnabled:      true,
			canSwitchProject: false,
			isViewer:         true,
			currentUser:      "viewer@example.com",
			want: PageData{
				Mode:             "project",
				IsGlobalMode:     false,
				IsProjectMode:    true,
				AuthEnabled:      true,
				CanSwitchProject: false,
				IsViewer:         true,
				CurrentUser:      "viewer@example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ComputePageData(tt.mode, tt.isGlobalMode, tt.authEnabled, tt.canSwitchProject, tt.isViewer, tt.currentUser)

			assert.Equal(t, tt.want.Mode, result.Mode)
			assert.Equal(t, tt.want.IsGlobalMode, result.IsGlobalMode)
			assert.Equal(t, tt.want.IsProjectMode, result.IsProjectMode)
			assert.Equal(t, tt.want.AuthEnabled, result.AuthEnabled)
			assert.Equal(t, tt.want.CanSwitchProject, result.CanSwitchProject)
			assert.Equal(t, tt.want.IsViewer, result.IsViewer)
			assert.Equal(t, tt.want.CurrentUser, result.CurrentUser)
			assert.NotNil(t, result.Events) // Events should always be initialized
		})
	}
}

func TestComputeStats_NilWorkspace(t *testing.T) {
	result := ComputeStats(nil)

	assert.NotNil(t, result)
	assert.Equal(t, 0, result.TotalTasks)
	assert.NotNil(t, result.StateLines)
	assert.Empty(t, result.StateLines)
}

func TestComputeStats_WithTasks(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	wsCfg := storage.NewDefaultWorkspaceConfig()
	wsCfg.Storage.HomeDir = t.TempDir()
	ws, err := storage.OpenWorkspace(ctx, tmpDir, wsCfg)
	require.NoError(t, err)
	require.NoError(t, ws.EnsureInitialized())

	// Create task works using CreateWork to set up directories
	now := time.Now()
	source := storage.SourceInfo{Type: "file", Ref: "test.md"}

	work1, err := ws.CreateWork("task-1", source)
	require.NoError(t, err)
	work1.Metadata.Title = "Task 1"
	work1.Metadata.State = "done"
	work1.Metadata.CreatedAt = now
	work1.Metadata.UpdatedAt = now
	work1.Costs = storage.CostStats{
		TotalInputTokens:  1000,
		TotalOutputTokens: 500,
		TotalCachedTokens: 100,
		TotalCostUSD:      10.50,
	}
	require.NoError(t, ws.SaveWork(work1))

	work2, err := ws.CreateWork("task-2", source)
	require.NoError(t, err)
	work2.Metadata.Title = "Task 2"
	work2.Metadata.State = "idle"
	work2.Metadata.CreatedAt = now
	work2.Metadata.UpdatedAt = now
	work2.Costs = storage.CostStats{
		TotalInputTokens:  500,
		TotalOutputTokens: 250,
		TotalCachedTokens: 50,
		TotalCostUSD:      5.25,
	}
	require.NoError(t, ws.SaveWork(work2))

	result := ComputeStats(ws)

	assert.NotNil(t, result)
	assert.Equal(t, 2, result.TotalTasks)
	assert.NotEmpty(t, result.StateLines)
	assert.Contains(t, result.TotalCost, "15.75") // Total cost of both tasks (10.50 + 5.25)
}

func TestComputeStats_WithMonthlyBudget(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	wsCfg := storage.NewDefaultWorkspaceConfig()
	wsCfg.Storage.HomeDir = t.TempDir()
	ws, err := storage.OpenWorkspace(ctx, tmpDir, wsCfg)
	require.NoError(t, err)
	require.NoError(t, ws.EnsureInitialized())

	// Set up budget config
	cfg := storage.NewDefaultWorkspaceConfig()
	cfg.Budget.Monthly.MaxCost = 100.0
	cfg.Budget.Monthly.WarningAt = 0.8
	require.NoError(t, ws.SaveConfig(cfg))

	// Set monthly budget state
	state := &storage.MonthlyBudgetState{
		Month:       time.Now().Format("2006-01"),
		Spent:       45.50,
		WarningSent: false,
	}
	err = ws.SaveMonthlyBudgetState(state)
	require.NoError(t, err)

	// Add a task using CreateWork
	now := time.Now()
	source := storage.SourceInfo{Type: "file", Ref: "test.md"}
	work1, err := ws.CreateWork("task-1", source)
	require.NoError(t, err)
	work1.Metadata.State = "done"
	work1.Metadata.CreatedAt = now
	work1.Metadata.UpdatedAt = now
	work1.Costs = storage.CostStats{
		TotalInputTokens:  100,
		TotalOutputTokens: 50,
		TotalCostUSD:      1.0,
	}
	require.NoError(t, ws.SaveWork(work1))

	result := ComputeStats(ws)

	assert.True(t, result.HasMonthly)
	assert.NotEmpty(t, result.MonthlySpent)
	assert.NotEmpty(t, result.MonthlyMax)
	assert.Greater(t, result.MonthlyPct, float64(0))
	assert.NotEmpty(t, result.MonthlyColor)
}

func TestComputeActiveWork_NilConductor(t *testing.T) {
	result := ComputeActiveWork(nil, nil)

	assert.Nil(t, result)
}

func TestComputeActiveWork_NoActiveTask(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	wsCfg := storage.NewDefaultWorkspaceConfig()
	wsCfg.Storage.HomeDir = t.TempDir()
	ws, err := storage.OpenWorkspace(ctx, tmpDir, wsCfg)
	require.NoError(t, err)

	c := helper_test.NewTestConductor(t,
		helper_test.TestConductorOptions(tmpDir)...,
	)

	result := ComputeActiveWork(c, ws)

	// No active task, should return nil
	assert.Nil(t, result)
}

func TestClearStaleTask(t *testing.T) {
	// This test verifies that ClearStaleTask detects and clears stale task state
	// when the work directory has been deleted externally (e.g., .mehrhof/ removed).
	// In production, ClearStaleTask is called by HTTP handlers before ComputeActiveWork
	// to ensure state mutations happen at the handler level, not in compute functions.
	tmpDir := t.TempDir()
	homeDir := t.TempDir()
	ctx := context.Background()

	// Create task file
	taskPath := filepath.Join(tmpDir, "task.md")
	require.NoError(t, os.WriteFile(taskPath, []byte("# Test Task\n\nTest description"), 0o644))

	// Create conductor with task
	c, err := conductor.New(
		conductor.WithWorkDir(tmpDir),
		conductor.WithHomeDir(homeDir),
		conductor.WithAutoInit(true),
		conductor.WithNoBranch(true),
		conductor.WithAgent("mock"), // Use mock agent
	)
	require.NoError(t, err)

	// Register file provider
	file.Register(c.GetProviderRegistry())

	// Register mock agent
	mockAgent := helper_test.NewMockAgent("mock")
	require.NoError(t, c.GetAgentRegistry().Register(mockAgent))

	// Initialize and start task
	require.NoError(t, c.Initialize(ctx))
	require.NoError(t, c.Start(ctx, "file:"+taskPath))

	// Verify task is active
	activeTask := c.GetActiveTask()
	require.NotNil(t, activeTask, "task should be active after Start")
	require.NotNil(t, c.GetTaskWork(), "work should exist after Start")

	// Get workspace for testing
	ws := c.GetWorkspace()
	require.NotNil(t, ws)

	// Delete the work directory to simulate external deletion
	// Work is stored in the workspace data directory (from ActiveTask.WorkDir)
	workDir := activeTask.WorkDir
	require.NoError(t, os.RemoveAll(workDir))

	// ClearStaleTask should detect and clear the stale task
	// (This is called by handlers before ComputeActiveWork)
	cleared := c.ClearStaleTask()
	assert.True(t, cleared, "should return true when stale task was cleared")

	// Active task should be cleared
	assert.Nil(t, c.GetActiveTask(), "stale task should be cleared")

	// ComputeActiveWork should return nil since there's no active task
	result := ComputeActiveWork(c, ws)
	assert.Nil(t, result, "should return nil after stale task was cleared")
}

func TestComputeActions_NoActiveWork(t *testing.T) {
	result := ComputeActions(nil, nil)

	assert.NotNil(t, result)
	assert.Len(t, result, 2) // Start Task and Quick Task buttons

	// First action should be "start"
	assert.Equal(t, "start", result[0].Command)
	assert.Equal(t, "Start Task", result[0].Label)

	// Second action should be "quick"
	assert.Equal(t, "quick", result[1].Command)
	assert.Equal(t, "Quick Task", result[1].Label)
}

func TestComputeActions_WithStates(t *testing.T) {
	tests := []struct {
		name         string
		state        string
		hasSpecs     bool
		wantCommands []string
	}{
		{
			name:         "idle state with specs",
			state:        StateIdle,
			hasSpecs:     true,
			wantCommands: []string{"implement", "sync", "simplify", "abandon"},
		},
		{
			name:         "idle state without specs",
			state:        StateIdle,
			hasSpecs:     false,
			wantCommands: []string{"plan", "abandon"},
		},
		{
			name:         "planning state",
			state:        StatePlanning,
			hasSpecs:     false,
			wantCommands: []string{"undo", "reset"},
		},
		{
			name:         "implementing state",
			state:        StateImplementing,
			hasSpecs:     false,
			wantCommands: []string{"undo", "reset"},
		},
		{
			name:         "reviewing state",
			state:        StateReviewing,
			hasSpecs:     false,
			wantCommands: []string{"undo", "reset"},
		},
		{
			name:         "done state",
			state:        StateDone,
			hasSpecs:     false,
			wantCommands: []string{"finish"},
		},
		{
			name:         "waiting state",
			state:        StateWaiting,
			hasSpecs:     false,
			wantCommands: []string{"continue", "undo"},
		},
		{
			name:         "paused state",
			state:        StatePaused,
			hasSpecs:     false,
			wantCommands: []string{"budget", "resume"},
		},
		{
			name:         "failed state",
			state:        StateFailed,
			hasSpecs:     false,
			wantCommands: []string{"undo", "abandon"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			active := &ActiveWorkData{
				State:    tt.state,
				HasSpecs: tt.hasSpecs,
			}

			result := ComputeActions(active, nil)

			assert.NotNil(t, result)
			assert.Len(t, result, len(tt.wantCommands))

			commands := make([]string, len(result))
			for i, action := range result {
				commands[i] = action.Command
			}
			assert.Equal(t, tt.wantCommands, commands)
		})
	}
}

func TestComputeActions_DangerousActions(t *testing.T) {
	// Test abandon action in idle state without specs
	active := &ActiveWorkData{
		State:    StateIdle,
		HasSpecs: false,
	}

	result := ComputeActions(active, nil)

	// Second action should be abandon (dangerous)
	require.Len(t, result, 2)
	abandonAction := result[1]
	assert.Equal(t, "abandon", abandonAction.Command)
	assert.Equal(t, BtnDanger, abandonAction.ButtonClass)
	assert.True(t, abandonAction.Dangerous)
	assert.NotEmpty(t, abandonAction.Confirm)
}

func TestComputeRecentTasks_NilWorkspace(t *testing.T) {
	result := ComputeRecentTasks(nil, 5)

	assert.Nil(t, result)
}

func TestComputeRecentTasks_WithTasks(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	wsCfg := storage.NewDefaultWorkspaceConfig()
	wsCfg.Storage.HomeDir = t.TempDir()
	ws, err := storage.OpenWorkspace(ctx, tmpDir, wsCfg)
	require.NoError(t, err)
	require.NoError(t, ws.EnsureInitialized())

	now := time.Now()

	// Add tasks with different timestamps using CreateWork
	work1, err := ws.CreateWork("task-1", storage.SourceInfo{Ref: "file:task1.md"})
	require.NoError(t, err)
	work1.Metadata.Title = "Task 1"
	work1.Metadata.State = "done"
	work1.Metadata.CreatedAt = now.Add(-2 * time.Hour)
	work1.Metadata.UpdatedAt = now.Add(-2 * time.Hour)
	require.NoError(t, ws.SaveWork(work1))

	work2, err := ws.CreateWork("task-2", storage.SourceInfo{Ref: "file:task2.md"})
	require.NoError(t, err)
	work2.Metadata.Title = "Task 2"
	work2.Metadata.State = "implementing"
	work2.Metadata.CreatedAt = now.Add(-1 * time.Hour)
	work2.Metadata.UpdatedAt = now.Add(-30 * time.Minute)
	require.NoError(t, ws.SaveWork(work2))

	work3, err := ws.CreateWork("task-3", storage.SourceInfo{Ref: "file:task3.md"})
	require.NoError(t, err)
	work3.Metadata.Title = "Task 3"
	work3.Metadata.State = "idle"
	work3.Metadata.CreatedAt = now
	work3.Metadata.UpdatedAt = now
	require.NoError(t, ws.SaveWork(work3))

	result := ComputeRecentTasks(ws, 5)

	assert.NotNil(t, result)
	assert.Len(t, result, 3) // All 3 tasks

	// Most recent should be task-3 (updated now)
	assert.Equal(t, "task-3", result[0].ID)
	assert.Equal(t, "Task 3", result[0].Title)

	// Last should be task-1 (oldest)
	assert.Equal(t, "task-1", result[2].ID)
}

func TestComputeRecentTasks_WithLimit(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	wsCfg := storage.NewDefaultWorkspaceConfig()
	wsCfg.Storage.HomeDir = t.TempDir()
	ws, err := storage.OpenWorkspace(ctx, tmpDir, wsCfg)
	require.NoError(t, err)
	require.NoError(t, ws.EnsureInitialized())

	now := time.Now()

	// Add 5 tasks using CreateWork
	for i := 1; i <= 5; i++ {
		id := fmt.Sprintf("task-%d", i)
		work, err := ws.CreateWork(id, storage.SourceInfo{Type: "file"})
		require.NoError(t, err)
		work.Metadata.Title = fmt.Sprintf("Task %d", i)
		work.Metadata.State = "idle"
		work.Metadata.CreatedAt = now.Add(-time.Duration(i) * time.Hour)
		work.Metadata.UpdatedAt = now.Add(-time.Duration(i) * time.Hour)
		require.NoError(t, ws.SaveWork(work))
	}

	result := ComputeRecentTasks(ws, 3)

	assert.Len(t, result, 3) // Limited to 3
}

func TestComputeRecentTasks_SortsByUpdateTime(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	wsCfg := storage.NewDefaultWorkspaceConfig()
	wsCfg.Storage.HomeDir = t.TempDir()
	ws, err := storage.OpenWorkspace(ctx, tmpDir, wsCfg)
	require.NoError(t, err)
	require.NoError(t, ws.EnsureInitialized())

	now := time.Now()

	// Task 1: created earlier, updated recently
	work1, err := ws.CreateWork("task-1", storage.SourceInfo{Type: "file"})
	require.NoError(t, err)
	work1.Metadata.Title = "Task 1"
	work1.Metadata.State = "idle"
	work1.Metadata.CreatedAt = now.Add(-24 * time.Hour)
	work1.Metadata.UpdatedAt = now.Add(-1 * time.Hour)
	require.NoError(t, ws.SaveWork(work1))

	// Task 2: created recently, not updated
	work2, err := ws.CreateWork("task-2", storage.SourceInfo{Type: "file"})
	require.NoError(t, err)
	work2.Metadata.Title = "Task 2"
	work2.Metadata.State = "idle"
	work2.Metadata.CreatedAt = now.Add(-30 * time.Minute)
	work2.Metadata.UpdatedAt = time.Time{} // Zero
	require.NoError(t, ws.SaveWork(work2))

	result := ComputeRecentTasks(ws, 5)

	// task-2 should be first (30 min ago is more recent than 1 hour ago)
	// task-2 has zero UpdatedAt, so it uses CreatedAt (30 min ago)
	// task-1 has UpdatedAt of 1 hour ago
	assert.Equal(t, "task-2", result[0].ID)
	assert.Equal(t, "task-1", result[1].ID)
}

func TestComputeSettingsProjects(t *testing.T) {
	projects := []storage.ProjectMetadata{
		{
			ID:         "proj-1",
			Name:       "Project 1",
			Path:       "/path/to/proj1",
			RemoteURL:  "https://github.com/user/repo1",
			LastAccess: time.Now().Add(-1 * time.Hour),
		},
		{
			ID:         "proj-2",
			Name:       "Project 2",
			Path:       "/path/to/proj2",
			RemoteURL:  "https://github.com/user/repo2",
			LastAccess: time.Now().Add(-24 * time.Hour),
		},
	}

	result := ComputeSettingsProjects(projects)

	assert.Len(t, result, 2)

	assert.Equal(t, "proj-1", result[0].ID)
	assert.Equal(t, "Project 1", result[0].Name)
	assert.Equal(t, "/path/to/proj1", result[0].Path)
	assert.Equal(t, "https://github.com/user/repo1", result[0].RemoteURL)
	assert.NotEmpty(t, result[0].LastAccess)

	assert.Equal(t, "proj-2", result[1].ID)
	assert.Equal(t, "Project 2", result[1].Name)
}

func TestComputeSettingsProjects_Empty(t *testing.T) {
	result := ComputeSettingsProjects([]storage.ProjectMetadata{})

	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestComputeSpecifications_NilWorkspace(t *testing.T) {
	result := ComputeSpecifications(nil, "task-1")

	assert.Nil(t, result)
}

func TestComputeQuestion_NilWorkspace(t *testing.T) {
	result := ComputeQuestion(nil, "task-1")

	assert.Nil(t, result)
}

func TestComputeQuestion_NoPendingQuestion(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	wsCfg := storage.NewDefaultWorkspaceConfig()
	wsCfg.Storage.HomeDir = t.TempDir()
	ws, err := storage.OpenWorkspace(ctx, tmpDir, wsCfg)
	require.NoError(t, err)

	result := ComputeQuestion(ws, "task-1")

	assert.Nil(t, result)
}

func TestComputeCosts_NilWorkspace(t *testing.T) {
	result := ComputeCosts(nil, "task-1")

	assert.Nil(t, result)
}

func TestComputeCosts_NoWork(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	wsCfg := storage.NewDefaultWorkspaceConfig()
	wsCfg.Storage.HomeDir = t.TempDir()
	ws, err := storage.OpenWorkspace(ctx, tmpDir, wsCfg)
	require.NoError(t, err)

	result := ComputeCosts(ws, "task-1")

	assert.Nil(t, result)
}

func TestComputeDashboard_GlobalMode(t *testing.T) {
	pageData := PageData{
		Mode:          "global",
		IsGlobalMode:  true,
		IsProjectMode: false,
	}

	result := ComputeDashboard(nil, nil, pageData)

	assert.NotNil(t, result)
	assert.Equal(t, pageData, result.PageData)
	assert.NotNil(t, result.Projects)
	// In global mode, these should be empty/nil
	assert.Nil(t, result.ActiveWork)
	assert.Nil(t, result.Stats)
}

func TestComputeDashboard_ProjectMode(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	wsCfg := storage.NewDefaultWorkspaceConfig()
	wsCfg.Storage.HomeDir = t.TempDir()
	ws, err := storage.OpenWorkspace(ctx, tmpDir, wsCfg)
	require.NoError(t, err)

	c := helper_test.NewTestConductor(t,
		helper_test.TestConductorOptions(tmpDir)...,
	)

	pageData := PageData{
		Mode:          "project",
		IsGlobalMode:  false,
		IsProjectMode: true,
	}

	result := ComputeDashboard(c, ws, pageData)

	assert.NotNil(t, result)
	assert.Equal(t, pageData, result.PageData)
	assert.NotNil(t, result.Stats)
	// No active task, so these should be nil
	assert.Nil(t, result.ActiveWork)
	assert.Nil(t, result.Specifications)
}

func TestComputeGuide_NilConductor(t *testing.T) {
	result := ComputeGuide(nil, nil)

	assert.NotNil(t, result)
	assert.False(t, result.HasTask)
	assert.Empty(t, result.NextActions)
}

func TestComputeGuide_NoActiveTask(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	wsCfg := storage.NewDefaultWorkspaceConfig()
	wsCfg.Storage.HomeDir = t.TempDir()
	ws, err := storage.OpenWorkspace(ctx, tmpDir, wsCfg)
	require.NoError(t, err)

	c := helper_test.NewTestConductor(t,
		helper_test.TestConductorOptions(tmpDir)...,
	)

	result := ComputeGuide(c, ws)

	assert.NotNil(t, result)
	assert.False(t, result.HasTask)
	assert.Empty(t, result.NextActions)
}

func TestComputeProjects(t *testing.T) {
	// This test requires the project registry to exist
	// Since we can't easily mock it, we'll just check it doesn't crash
	_ = ComputeProjects()
	// If we get here without panicking, the test passes
}

func TestComputeLabels(t *testing.T) {
	labels := []string{"bug", "enhancement", "feature"}

	result := computeLabels(labels)

	assert.Len(t, result, 3)
	assert.Equal(t, "bug", result[0].Text)
	assert.NotEmpty(t, result[0].Color) // Color should be computed
}

func TestComputeLabels_Empty(t *testing.T) {
	result := computeLabels([]string{})

	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestComputePageData_EventsPopulated(t *testing.T) {
	result := ComputePageData("project", false, true, false, false, "user@example.com")

	assert.NotNil(t, result.Events)
	// Events should contain common event names
	assert.NotEmpty(t, result.Events)
}
