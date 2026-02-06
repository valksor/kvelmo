package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valksor/go-mehrhof/internal/server/views"
	"github.com/valksor/go-mehrhof/internal/storage"
)

func TestSpecItemData_Structure(t *testing.T) {
	// Test that SpecItemData has the correct fields
	spec := views.SpecItemData{
		Number:      1,
		Name:        "specification-1",
		Title:       "Test Specification",
		Status:      "draft",
		Description: "Test description content",
		Component:   "backend",
		CreatedAt:   "2026-01-26 15:00",
		CompletedAt: "",
	}

	assert.Equal(t, 1, spec.Number)
	assert.Equal(t, "specification-1", spec.Name)
	assert.Equal(t, "Test Specification", spec.Title)
	assert.Equal(t, "draft", spec.Status)
	assert.Equal(t, "Test description content", spec.Description)
	assert.Equal(t, "backend", spec.Component)
	assert.Equal(t, "2026-01-26 15:00", spec.CreatedAt)
	assert.Equal(t, "", spec.CompletedAt)
}

func TestSpecificationsData_Structure(t *testing.T) {
	// Test that SpecificationsData has the correct fields
	data := views.SpecificationsData{
		Items: []views.SpecItemData{
			{
				Number:      1,
				Name:        "specification-1",
				Title:       "Test",
				Status:      "draft",
				Description: "Content",
			},
		},
		Total:    1,
		Done:     0,
		Progress: 0.0,
	}

	assert.Equal(t, 1, data.Total)
	assert.Equal(t, 0, data.Done)
	assert.Equal(t, 0.0, data.Progress)
	assert.Len(t, data.Items, 1)
	assert.Equal(t, "specification-1", data.Items[0].Name)
}

func TestHandler_GetSpecifications_IncludesImplementedFiles(t *testing.T) {
	cond, tmpDir := createTestConductor(t)
	ws := cond.GetWorkspace()
	require.NotNil(t, ws)

	taskID := "spec-files-task"
	_, err := ws.CreateWork(taskID, storage.SourceInfo{
		Type:   "file",
		Ref:    "file:task.md",
		ReadAt: time.Now(),
	})
	require.NoError(t, err)

	err = ws.SaveSpecificationWithMeta(taskID, &storage.Specification{
		Number: 1,
		Title:  "Expose implementation details",
		Status: storage.SpecificationStatusDone,
		ImplementedFiles: []string{
			"internal/server/handlers.go",
			"web/src/components/task/SpecificationsList.tsx",
		},
	})
	require.NoError(t, err)

	srv, err := New(Config{
		Port:          0,
		Mode:          ModeProject,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+taskID+"/specs", nil)
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	body, err := io.ReadAll(rr.Result().Body)
	require.NoError(t, err)

	var result struct {
		Specifications []struct {
			Number           int      `json:"number"`
			ImplementedFiles []string `json:"implemented_files"`
		} `json:"specifications"`
	}
	require.NoError(t, json.Unmarshal(body, &result))
	require.Len(t, result.Specifications, 1)
	assert.Equal(t, 1, result.Specifications[0].Number)
	assert.Equal(t, []string{
		"internal/server/handlers.go",
		"web/src/components/task/SpecificationsList.tsx",
	}, result.Specifications[0].ImplementedFiles)
}

func TestHandler_GetSpecificationDiff(t *testing.T) {
	cond, tmpDir := createTestConductor(t)
	ws := cond.GetWorkspace()
	require.NotNil(t, ws)

	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "config", "user.email", "test@example.com")
	runGit(t, tmpDir, "config", "user.name", "Test User")

	taskID := "spec-diff-task"
	_, err := ws.CreateWork(taskID, storage.SourceInfo{
		Type:   "file",
		Ref:    "file:task.md",
		ReadAt: time.Now(),
	})
	require.NoError(t, err)

	filePath := "internal/server/sample.txt"
	absPath := filepath.Join(tmpDir, filepath.FromSlash(filePath))
	require.NoError(t, os.MkdirAll(filepath.Dir(absPath), 0o755))
	require.NoError(t, os.WriteFile(absPath, []byte("line 1\n"), 0o644))

	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "initial")

	require.NoError(t, ws.SaveSpecificationWithMeta(taskID, &storage.Specification{
		Number:           1,
		Title:            "Spec with implementation files",
		Status:           storage.SpecificationStatusDone,
		ImplementedFiles: []string{filePath},
	}))

	require.NoError(t, os.WriteFile(absPath, []byte("line 1\nline 2\n"), 0o644))

	srv, err := New(Config{
		Port:          0,
		Mode:          ModeProject,
		Conductor:     cond,
		WorkspaceRoot: tmpDir,
	})
	require.NoError(t, err)

	t.Run("requires file query parameter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+taskID+"/specs/1/diff", nil)
		rr := httptest.NewRecorder()
		srv.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("rejects files not listed in specification", func(t *testing.T) {
		req := httptest.NewRequest(
			http.MethodGet,
			"/api/v1/tasks/"+taskID+"/specs/1/diff?file="+filepath.ToSlash("other/file.txt"),
			nil,
		)
		rr := httptest.NewRecorder()
		srv.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("returns unified diff for implemented file", func(t *testing.T) {
		req := httptest.NewRequest(
			http.MethodGet,
			"/api/v1/tasks/"+taskID+"/specs/1/diff?file="+filePath,
			nil,
		)
		rr := httptest.NewRecorder()
		srv.router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		body, readErr := io.ReadAll(rr.Result().Body)
		require.NoError(t, readErr)

		var result struct {
			HasDiff bool   `json:"has_diff"`
			Diff    string `json:"diff"`
		}
		require.NoError(t, json.Unmarshal(body, &result))
		assert.True(t, result.HasDiff)
		assert.Contains(t, result.Diff, "diff --git")
		assert.Contains(t, result.Diff, filePath)
		assert.Contains(t, result.Diff, "+line 2")
	})
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.CommandContext(context.Background(), "git", args...)
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	if err == nil {
		return
	}

	t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(output))
}
