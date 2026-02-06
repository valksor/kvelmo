package conductor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/valksor/go-mehrhof/internal/provider"
)

func TestPersistSyncedSourceArtifacts_Wrike(t *testing.T) {
	tmp := t.TempDir()
	sourceDir := filepath.Join(tmp, "source")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("mkdir source dir: %v", err)
	}

	sourcePath := filepath.Join(sourceDir, "wrike.txt")
	oldContent := "old wrike content"
	if err := os.WriteFile(sourcePath, []byte(oldContent), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	newContent := "new wrike content"
	changes := provider.ChangeSet{HasChanges: true, DescriptionChanged: true}
	sourceUpdated, previousPath, diffPath, err := persistSyncedSourceArtifacts(
		sourcePath,
		oldContent,
		newContent,
		"wrike",
		changes,
	)
	if err != nil {
		t.Fatalf("persistSyncedSourceArtifacts: %v", err)
	}
	if !sourceUpdated {
		t.Fatal("sourceUpdated = false, want true")
	}

	gotSource, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	if string(gotSource) != newContent {
		t.Fatalf("source content = %q, want %q", string(gotSource), newContent)
	}

	gotPrev, err := os.ReadFile(previousPath)
	if err != nil {
		t.Fatalf("read previous snapshot: %v", err)
	}
	if string(gotPrev) != oldContent {
		t.Fatalf("previous snapshot = %q, want %q", string(gotPrev), oldContent)
	}

	if _, err := os.Stat(diffPath); err != nil {
		t.Fatalf("diff path missing: %v", err)
	}
}

func TestPersistSyncedSourceArtifacts_NonWrike(t *testing.T) {
	tmp := t.TempDir()
	sourceDir := filepath.Join(tmp, "source")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("mkdir source dir: %v", err)
	}

	sourcePath := filepath.Join(sourceDir, "github.txt")
	if err := os.WriteFile(sourcePath, []byte("old"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	sourceUpdated, previousPath, diffPath, err := persistSyncedSourceArtifacts(
		sourcePath,
		"old",
		"new",
		"github",
		provider.ChangeSet{},
	)
	if err != nil {
		t.Fatalf("persistSyncedSourceArtifacts: %v", err)
	}
	if !sourceUpdated {
		t.Fatal("sourceUpdated = false, want true")
	}
	if previousPath != "" {
		t.Fatalf("previousPath = %q, want empty", previousPath)
	}
	if diffPath != "" {
		t.Fatalf("diffPath = %q, want empty", diffPath)
	}

	gotSource, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	if string(gotSource) != "new" {
		t.Fatalf("source content = %q, want %q", string(gotSource), "new")
	}
}
