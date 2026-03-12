package conductor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCollectPyFiles_SkipsDirs(t *testing.T) {
	dir := t.TempDir()
	// Create files that should be found
	if err := os.WriteFile(filepath.Join(dir, "main.py"), []byte("print('hi')"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# readme"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pkg", "util.py"), []byte("pass"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Create dirs that should be skipped
	for _, skipDir := range []string{".venv", "venv", "__pycache__", ".git", "node_modules"} {
		if err := os.MkdirAll(filepath.Join(dir, skipDir), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, skipDir, "skip.py"), []byte("pass"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	files, err := collectPyFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Should find main.py and pkg/util.py only
	if len(files) != 2 {
		t.Errorf("collectPyFiles() found %d files, want 2: %v", len(files), files)
	}
	for _, f := range files {
		if filepath.Ext(f) != ".py" {
			t.Errorf("unexpected non-.py file: %q", f)
		}
	}
}

func TestCollectPyFiles_NestedSubdirs(t *testing.T) {
	dir := t.TempDir()
	// Create nested .py files
	nested := filepath.Join(dir, "a", "b", "c")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "deep.py"), []byte("pass"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "top.py"), []byte("pass"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := collectPyFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Errorf("collectPyFiles() found %d files, want 2: %v", len(files), files)
	}
}

func TestReadPackageJSONScripts_MultipleScripts(t *testing.T) {
	dir := t.TempDir()

	content := `{"name":"test","scripts":{"lint":"eslint .","test":"vitest","build":"tsc"}}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	scripts, err := readPackageJSONScripts(dir)
	if err != nil {
		t.Fatalf("readPackageJSONScripts() error = %v", err)
	}
	if scripts["lint"] != "eslint ." {
		t.Errorf("scripts[lint] = %q, want 'eslint .'", scripts["lint"])
	}
	if scripts["test"] != "vitest" {
		t.Errorf("scripts[test] = %q, want 'vitest'", scripts["test"])
	}
	if scripts["build"] != "tsc" {
		t.Errorf("scripts[build] = %q, want 'tsc'", scripts["build"])
	}
}

func TestCoderabbitCtx(t *testing.T) {
	ctx, cancel := coderabbitCtx()
	defer cancel()
	if ctx == nil {
		t.Error("coderabbitCtx() returned nil context")
	}
	select {
	case <-ctx.Done():
		t.Error("coderabbitCtx() context should not be cancelled immediately")
	default:
	}
}
