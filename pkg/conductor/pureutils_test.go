package conductor

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/valksor/kvelmo/pkg/settings"
)

// ─── readPackageJSONScripts ───────────────────────────────────────────────────

func TestReadPackageJSONScripts_NoFile(t *testing.T) {
	_, err := readPackageJSONScripts(t.TempDir())
	if err == nil {
		t.Error("readPackageJSONScripts() with no file should return error")
	}
}

func TestReadPackageJSONScripts_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte("{invalid}"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := readPackageJSONScripts(dir)
	if err == nil {
		t.Error("readPackageJSONScripts() with invalid JSON should return error")
	}
}

func TestReadPackageJSONScripts_WithScripts(t *testing.T) {
	dir := t.TempDir()
	pkg := map[string]any{
		"name": "test",
		"scripts": map[string]string{
			"build":     "go build ./...",
			"lint":      "golangci-lint run",
			"typecheck": "tsc --noEmit",
		},
	}
	data, err := json.Marshal(pkg)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	scripts, err := readPackageJSONScripts(dir)
	if err != nil {
		t.Fatalf("readPackageJSONScripts() error = %v", err)
	}
	if scripts["lint"] != "golangci-lint run" {
		t.Errorf("scripts[lint] = %q, want golangci-lint run", scripts["lint"])
	}
}

func TestReadPackageJSONScripts_NoScripts(t *testing.T) {
	dir := t.TempDir()
	pkg := map[string]any{"name": "test"}
	data, err := json.Marshal(pkg)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	scripts, err := readPackageJSONScripts(dir)
	if err != nil {
		t.Fatalf("readPackageJSONScripts() error = %v", err)
	}
	if len(scripts) != 0 {
		t.Errorf("readPackageJSONScripts() with no scripts = %v, want empty", scripts)
	}
}

// ─── collectPyFiles ───────────────────────────────────────────────────────────

func TestCollectPyFiles_NonExistentDir(t *testing.T) {
	_, err := collectPyFiles("/tmp/kvelmo-test-nonexistent-dir-xyz-abc")
	if err == nil {
		t.Error("collectPyFiles() with non-existent dir should return error")
	}
}

func TestCollectPyFiles_Empty(t *testing.T) {
	dir := t.TempDir()
	files, err := collectPyFiles(dir)
	if err != nil {
		t.Fatalf("collectPyFiles() error = %v", err)
	}
	if len(files) != 0 {
		t.Errorf("collectPyFiles() on empty dir = %v, want empty", files)
	}
}

func TestCollectPyFiles_MixedFiles(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"main.py", "helper.py", "main.go", "README.md"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("content"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	files, err := collectPyFiles(dir)
	if err != nil {
		t.Fatalf("collectPyFiles() error = %v", err)
	}
	if len(files) != 2 {
		t.Errorf("collectPyFiles() = %v (len=%d), want 2 .py files", files, len(files))
	}
	for _, f := range files {
		if filepath.Ext(f) != ".py" {
			t.Errorf("collectPyFiles() returned non-.py file: %q", f)
		}
	}
}

// ─── qualityCtx ──────────────────────────────────────────────────────────────

func TestQualityCtx(t *testing.T) {
	ctx, cancel := qualityCtx()
	defer cancel()
	if ctx == nil {
		t.Error("qualityCtx() returned nil context")
	}
	select {
	case <-ctx.Done():
		t.Error("qualityCtx() context should not be cancelled immediately")
	default:
	}
}

// ─── runQualityGate ───────────────────────────────────────────────────────────

// noExternalReviewSettings returns settings with external review disabled, for use in
// quality gate tests that should not block on a user prompt.
func noExternalReviewSettings() *settings.Settings {
	s := settings.DefaultSettings()
	s.Workflow.ExternalReview.Mode = settings.ExternalReviewNever

	return s
}

func TestRunQualityGate_NoProjectFiles(t *testing.T) {
	// Empty temp dir — no go.mod, package.json, setup.py, or pyproject.toml
	c, _ := New(WithWorkDir(t.TempDir()), WithSettings(noExternalReviewSettings()))
	if err := c.runQualityGate(context.Background()); err != nil {
		t.Errorf("runQualityGate() on unknown project type = %v, want nil (should skip)", err)
	}
}

func TestRunQualityGate_NodeProjectNoScripts(t *testing.T) {
	dir := t.TempDir()
	// Create package.json with no lint/typecheck scripts
	pkg := map[string]any{
		"name":    "test",
		"scripts": map[string]string{"start": "node index.js"},
	}
	data, err := json.Marshal(pkg)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	c, _ := New(WithWorkDir(dir), WithSettings(noExternalReviewSettings()))
	// qualityGateNode finds no lint/typecheck scripts → returns nil without running npm
	if err := c.runQualityGate(context.Background()); err != nil {
		t.Errorf("runQualityGate() with no lint/typecheck scripts = %v, want nil", err)
	}
}

// ─── Submit / SaveSpecification / GenerateDeltaSpecification ─────────────────

func TestSubmit_NoWorkUnit(t *testing.T) {
	c, _ := New()
	err := c.Submit(context.Background(), false)
	if err == nil {
		t.Error("Submit() with no work unit should return error")
	}
}

func TestSaveSpecification_NoWorkUnit(t *testing.T) {
	c, _ := New()
	_, err := c.SaveSpecification("content")
	if err == nil {
		t.Error("SaveSpecification() with no work unit should return error")
	}
}

func TestGenerateDeltaSpecification_NoWorkUnit(t *testing.T) {
	c, _ := New()
	_, err := c.GenerateDeltaSpecification(context.Background(), "old", "new")
	if err == nil {
		t.Error("GenerateDeltaSpecification() with no work unit should return error")
	}
}

// ─── watchJob / saveJobSession ────────────────────────────────────────────────

func TestWatchJob_NoPool(t *testing.T) {
	// New() without WithPool → c.pool is nil → watchJob returns immediately
	c, _ := New()
	ctx := context.Background()
	// Should return without panic or blocking
	done := make(chan struct{})
	go func() {
		c.watchJob(ctx, "job-1", EventPlanDone)
		close(done)
	}()
	select {
	case <-done:
		// expected
	case <-ctx.Done():
		t.Error("watchJob() with nil pool timed out unexpectedly")
	}
}

func TestSaveJobSession_NoStore(t *testing.T) {
	// New() without store → c.store is nil → saveJobSession returns immediately
	c, _ := New()
	c.saveJobSession("job-1", "plan", "claude") // Should not panic
}
