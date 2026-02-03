package library

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNewManager(t *testing.T) {
	tmpDir := t.TempDir()

	m, err := NewManager(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	if m.projectStore == nil {
		t.Error("expected project store to be initialized")
	}
	if m.sharedStore == nil {
		t.Error("expected shared store to be initialized")
	}
	if m.config == nil {
		t.Error("expected config to be initialized")
	}
}

func TestNewManager_NoRepoRoot(t *testing.T) {
	m, err := NewManager(context.Background(), "")
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	if m.projectStore != nil {
		t.Error("expected project store to be nil when no repo root")
	}
	if m.sharedStore == nil {
		t.Error("expected shared store to be initialized")
	}
}

func TestManager_PullFile(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	// Create test files
	docsDir := filepath.Join(tmpDir, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(docsDir, "guide.md"), []byte("# Guide\n\nThis is a guide."), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := NewManager(ctx, tmpDir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	result, err := m.Pull(ctx, docsDir, &PullOptions{
		Name: "Test Docs",
	})
	if err != nil {
		t.Fatalf("Pull failed: %v", err)
	}

	if result.PagesWritten != 1 {
		t.Errorf("PagesWritten = %d, want 1", result.PagesWritten)
	}
	if result.Collection.Name != "Test Docs" {
		t.Errorf("Name = %q, want %q", result.Collection.Name, "Test Docs")
	}
}

func TestManager_List(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(testFile, []byte("# Test"), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := NewManager(ctx, tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Pull a collection
	_, err = m.Pull(ctx, testFile, &PullOptions{Name: "Test"})
	if err != nil {
		t.Fatal(err)
	}

	// List collections
	collections, err := m.List(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(collections) != 1 {
		t.Fatalf("got %d collections, want 1", len(collections))
	}
	if collections[0].Name != "Test" {
		t.Errorf("Name = %q, want %q", collections[0].Name, "Test")
	}
}

func TestManager_Show(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	testFile := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(testFile, []byte("# Test Content"), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := NewManager(ctx, tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	result, err := m.Pull(ctx, testFile, &PullOptions{Name: "My Docs"})
	if err != nil {
		t.Fatal(err)
	}

	// Show by ID
	coll, err := m.Show(ctx, result.Collection.ID)
	if err != nil {
		t.Fatalf("Show failed: %v", err)
	}

	if coll.Name != "My Docs" {
		t.Errorf("Name = %q, want %q", coll.Name, "My Docs")
	}
}

func TestManager_Remove(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	testFile := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(testFile, []byte("# Test"), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := NewManager(ctx, tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	result, err := m.Pull(ctx, testFile, &PullOptions{Name: "ToRemove"})
	if err != nil {
		t.Fatal(err)
	}

	// Verify it exists
	if _, err := m.Show(ctx, result.Collection.ID); err != nil {
		t.Fatal("collection should exist before removal")
	}

	// Remove it
	if err := m.Remove(ctx, result.Collection.ID, false); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	// Verify it's gone
	if _, err := m.Show(ctx, result.Collection.ID); err == nil {
		t.Error("collection should not exist after removal")
	}
}

func TestManager_Update(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	testFile := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(testFile, []byte("# Original"), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := NewManager(ctx, tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	result, err := m.Pull(ctx, testFile, &PullOptions{Name: "Updatable"})
	if err != nil {
		t.Fatal(err)
	}

	// Modify the source file
	if err := os.WriteFile(testFile, []byte("# Updated Content"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Update the collection
	_, err = m.Update(ctx, result.Collection.ID)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify the content was updated
	pages, err := m.ListPages(ctx, result.Collection.ID)
	if err != nil {
		t.Fatal(err)
	}

	if len(pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(pages))
	}

	_, content, err := m.ShowPage(ctx, result.Collection.ID, pages[0])
	if err != nil {
		t.Fatal(err)
	}

	if content != "# Updated Content" {
		t.Errorf("content not updated: %q", content)
	}
}

func TestManager_GetStore(t *testing.T) {
	tmpDir := t.TempDir()

	m, err := NewManager(context.Background(), tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	projectStore := m.GetStore(false)
	if projectStore == nil {
		t.Error("expected project store")
	}

	sharedStore := m.GetStore(true)
	if sharedStore == nil {
		t.Error("expected shared store")
	}

	if projectStore == sharedStore {
		t.Error("stores should be different")
	}
}

func TestSumPageSizes(t *testing.T) {
	pages := []*CrawledPage{
		{SizeBytes: 100},
		{SizeBytes: 200},
		{SizeBytes: 50},
	}

	total := sumPageSizes(pages)
	if total != 350 {
		t.Errorf("total = %d, want 350", total)
	}
}

func TestDeriveNameFromSource(t *testing.T) {
	tests := []struct {
		source string
		want   string
	}{
		{"https://react.dev/reference", "react-dev-reference"},
		{"/home/user/docs", "docs"},
		{"./local", "local"},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			got := deriveNameFromSource(tt.source)
			if got != tt.want {
				t.Errorf("deriveNameFromSource(%q) = %q, want %q", tt.source, got, tt.want)
			}
		})
	}
}
