package vcs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFilterAllowedDirtySubmodules(t *testing.T) {
	repoRoot := t.TempDir()

	gitmodules := `[submodule "vendor/lib"]
	path = vendor/lib
	url = https://example.com/lib.git
`
	if err := os.WriteFile(filepath.Join(repoRoot, ".gitmodules"), []byte(gitmodules), 0o644); err != nil {
		t.Fatalf("write .gitmodules: %v", err)
	}

	allow := `{"allow_dirty_submodules":["vendor/lib"]}`
	if err := os.WriteFile(filepath.Join(repoRoot, ".mehrhof-submodules.json"), []byte(allow), 0o644); err != nil {
		t.Fatalf("write allowlist: %v", err)
	}

	g := &Git{repoRoot: repoRoot}
	input := []FileStatus{
		{Index: ' ', WorkDir: 'M', Path: "vendor/lib"},
		{Index: 'M', WorkDir: ' ', Path: "README.md"},
	}

	got := g.filterAllowedDirtySubmodules(input)
	if len(got) != 1 {
		t.Fatalf("filtered len = %d, want 1", len(got))
	}
	if got[0].Path != "README.md" {
		t.Fatalf("remaining path = %q, want %q", got[0].Path, "README.md")
	}
}

func TestFilterAllowedDirtySubmodules_LegacyFile(t *testing.T) {
	repoRoot := t.TempDir()

	gitmodules := `[submodule "src/shared"]
	path = src/shared
	url = https://example.com/shared.git
`
	if err := os.WriteFile(filepath.Join(repoRoot, ".gitmodules"), []byte(gitmodules), 0o644); err != nil {
		t.Fatalf("write .gitmodules: %v", err)
	}

	legacyAllow := `["src/shared"]`
	if err := os.WriteFile(filepath.Join(repoRoot, ".asc-submodules.json"), []byte(legacyAllow), 0o644); err != nil {
		t.Fatalf("write legacy allowlist: %v", err)
	}

	g := &Git{repoRoot: repoRoot}
	input := []FileStatus{
		{Index: ' ', WorkDir: 'M', Path: "src/shared"},
		{Index: ' ', WorkDir: 'M', Path: "go.mod"},
	}

	got := g.filterAllowedDirtySubmodules(input)
	if len(got) != 1 {
		t.Fatalf("filtered len = %d, want 1", len(got))
	}
	if got[0].Path != "go.mod" {
		t.Fatalf("remaining path = %q, want %q", got[0].Path, "go.mod")
	}
}

func TestFilterAllowedDirtySubmodules_NoAllowlist(t *testing.T) {
	repoRoot := t.TempDir()
	g := &Git{repoRoot: repoRoot}

	input := []FileStatus{
		{Index: ' ', WorkDir: 'M', Path: "vendor/lib"},
	}

	got := g.filterAllowedDirtySubmodules(input)
	if len(got) != 1 {
		t.Fatalf("filtered len = %d, want 1", len(got))
	}
}
