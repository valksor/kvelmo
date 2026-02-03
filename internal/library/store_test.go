package library

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewStoreWithRoot(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStoreWithRoot(tmpDir, 5*time.Second)

	if store.RootDir() != tmpDir {
		t.Errorf("RootDir() = %q, want %q", store.RootDir(), tmpDir)
	}
}

func TestManifestRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStoreWithRoot(tmpDir, 5*time.Second)

	// Create a manifest with collections
	manifest := NewManifest()
	manifest.AddCollection(&Collection{
		ID:          "test-collection",
		Name:        "Test Collection",
		Source:      "https://example.com",
		SourceType:  SourceURL,
		IncludeMode: IncludeModeAuto,
		Paths:       []string{"test/**"},
		PulledAt:    time.Now().UTC().Truncate(time.Second),
		PageCount:   5,
		TotalSize:   1024,
		Location:    "project",
	})

	// Save manifest
	if err := store.SaveManifest(manifest); err != nil {
		t.Fatalf("SaveManifest failed: %v", err)
	}

	// Load manifest
	loaded, err := store.LoadManifest()
	if err != nil {
		t.Fatalf("LoadManifest failed: %v", err)
	}

	if len(loaded.Collections) != 1 {
		t.Errorf("got %d collections, want 1", len(loaded.Collections))
	}

	c := loaded.Collections[0]
	if c.ID != "test-collection" {
		t.Errorf("ID = %q, want %q", c.ID, "test-collection")
	}
	if c.Name != "Test Collection" {
		t.Errorf("Name = %q, want %q", c.Name, "Test Collection")
	}
	if c.SourceType != SourceURL {
		t.Errorf("SourceType = %q, want %q", c.SourceType, SourceURL)
	}
}

func TestManifestOperations(t *testing.T) {
	manifest := NewManifest()

	// Add collection
	c1 := &Collection{ID: "c1", Name: "Collection 1"}
	manifest.AddCollection(c1)

	if got := manifest.GetCollection("c1"); got == nil {
		t.Error("GetCollection returned nil for existing collection")
	}

	if got := manifest.GetCollection("nonexistent"); got != nil {
		t.Error("GetCollection returned non-nil for nonexistent collection")
	}

	// Update collection (same ID)
	c1Updated := &Collection{ID: "c1", Name: "Updated Name"}
	manifest.AddCollection(c1Updated)

	if len(manifest.Collections) != 1 {
		t.Errorf("expected 1 collection after update, got %d", len(manifest.Collections))
	}
	if manifest.Collections[0].Name != "Updated Name" {
		t.Error("collection was not updated")
	}

	// Add another collection
	c2 := &Collection{ID: "c2", Name: "Collection 2"}
	manifest.AddCollection(c2)

	if len(manifest.Collections) != 2 {
		t.Errorf("expected 2 collections, got %d", len(manifest.Collections))
	}

	// Remove collection
	if !manifest.RemoveCollection("c1") {
		t.Error("RemoveCollection returned false for existing collection")
	}

	if len(manifest.Collections) != 1 {
		t.Errorf("expected 1 collection after remove, got %d", len(manifest.Collections))
	}

	if manifest.RemoveCollection("nonexistent") {
		t.Error("RemoveCollection returned true for nonexistent collection")
	}
}

func TestCollectionMetaRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStoreWithRoot(tmpDir, 5*time.Second)

	meta := NewCollectionMeta("https://example.com", SourceURL)
	meta.CrawlConfig = DefaultCrawlConfig()
	meta.Pages = []*Page{
		{Path: "index.md", Title: "Index", SizeBytes: 100},
		{Path: "guide/intro.md", Title: "Introduction", SizeBytes: 200},
	}

	collectionID := "test-col"

	// Save meta
	if err := store.SaveCollectionMeta(collectionID, meta); err != nil {
		t.Fatalf("SaveCollectionMeta failed: %v", err)
	}

	// Load meta
	loaded, err := store.LoadCollectionMeta(collectionID)
	if err != nil {
		t.Fatalf("LoadCollectionMeta failed: %v", err)
	}

	if loaded.Source != "https://example.com" {
		t.Errorf("Source = %q, want %q", loaded.Source, "https://example.com")
	}
	if loaded.SourceType != SourceURL {
		t.Errorf("SourceType = %q, want %q", loaded.SourceType, SourceURL)
	}
	if len(loaded.Pages) != 2 {
		t.Errorf("got %d pages, want 2", len(loaded.Pages))
	}
}

func TestPageReadWrite(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStoreWithRoot(tmpDir, 5*time.Second)

	collectionID := "test-col"
	pagePath := "guide/intro.md"
	content := "# Introduction\n\nThis is the intro."

	// Write page
	if err := store.WritePage(collectionID, pagePath, content); err != nil {
		t.Fatalf("WritePage failed: %v", err)
	}

	// Read page
	got, err := store.ReadPage(collectionID, pagePath)
	if err != nil {
		t.Fatalf("ReadPage failed: %v", err)
	}

	if got != content {
		t.Errorf("content mismatch:\ngot:  %q\nwant: %q", got, content)
	}

	// Verify file exists on disk
	fullPath := filepath.Join(tmpDir, "collections", collectionID, "pages", pagePath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		t.Error("page file does not exist on disk")
	}
}

func TestListPageFiles(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStoreWithRoot(tmpDir, 5*time.Second)

	collectionID := "test-col"

	// Write some pages
	pages := []string{"index.md", "guide/intro.md", "guide/advanced.md", "api/reference.md"}
	for _, p := range pages {
		if err := store.WritePage(collectionID, p, "content"); err != nil {
			t.Fatalf("WritePage(%s) failed: %v", p, err)
		}
	}

	// List pages
	files, err := store.ListPageFiles(collectionID)
	if err != nil {
		t.Fatalf("ListPageFiles failed: %v", err)
	}

	if len(files) != len(pages) {
		t.Errorf("got %d files, want %d", len(files), len(pages))
	}

	// Empty collection
	emptyFiles, err := store.ListPageFiles("nonexistent")
	if err != nil {
		t.Fatalf("ListPageFiles for nonexistent failed: %v", err)
	}
	if len(emptyFiles) != 0 {
		t.Errorf("expected 0 files for nonexistent collection, got %d", len(emptyFiles))
	}
}

func TestDeleteCollection(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStoreWithRoot(tmpDir, 5*time.Second)

	collectionID := "to-delete"

	// Create collection with pages
	if err := store.WritePage(collectionID, "index.md", "content"); err != nil {
		t.Fatalf("WritePage failed: %v", err)
	}

	if !store.CollectionExists(collectionID) {
		t.Error("collection should exist after writing page")
	}

	// Delete
	if err := store.DeleteCollection(collectionID); err != nil {
		t.Fatalf("DeleteCollection failed: %v", err)
	}

	if store.CollectionExists(collectionID) {
		t.Error("collection should not exist after deletion")
	}
}

func TestSaveAndGetCollection(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStoreWithRoot(tmpDir, 5*time.Second)

	c := &Collection{
		ID:          "test-id",
		Name:        "Test",
		Source:      "https://example.com",
		SourceType:  SourceURL,
		IncludeMode: IncludeModeAuto,
		PulledAt:    time.Now().UTC().Truncate(time.Second),
	}

	// Save
	if err := store.SaveCollection(c); err != nil {
		t.Fatalf("SaveCollection failed: %v", err)
	}

	// Get
	got, err := store.GetCollection("test-id")
	if err != nil {
		t.Fatalf("GetCollection failed: %v", err)
	}

	if got.Name != "Test" {
		t.Errorf("Name = %q, want %q", got.Name, "Test")
	}

	// Get nonexistent
	_, err = store.GetCollection("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent collection")
	}
}

func TestListCollections(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStoreWithRoot(tmpDir, 5*time.Second)

	// Empty list
	cols, err := store.ListCollections()
	if err != nil {
		t.Fatalf("ListCollections failed: %v", err)
	}
	if len(cols) != 0 {
		t.Errorf("expected 0 collections, got %d", len(cols))
	}

	// Add collections
	for i := range 3 {
		c := &Collection{
			ID:   GenerateCollectionID("", "source"+string(rune('0'+i))),
			Name: "Collection " + string(rune('0'+i)),
		}
		if err := store.SaveCollection(c); err != nil {
			t.Fatalf("SaveCollection failed: %v", err)
		}
	}

	cols, err = store.ListCollections()
	if err != nil {
		t.Fatalf("ListCollections failed: %v", err)
	}
	if len(cols) != 3 {
		t.Errorf("expected 3 collections, got %d", len(cols))
	}
}

func TestWritePages(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStoreWithRoot(tmpDir, 5*time.Second)

	collectionID := "test-col"
	pages := []*CrawledPage{
		{Path: "good.md", Content: "good content", SizeBytes: 12},
		{Path: "also-good.md", Content: "also good", SizeBytes: 9},
		{Path: "failed.md", Error: os.ErrPermission}, // Simulated error
	}

	written, errors := store.WritePages(collectionID, pages)

	if len(written) != 2 {
		t.Errorf("expected 2 written pages, got %d", len(written))
	}
	if len(errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(errors))
	}

	// Verify written pages
	for _, w := range written {
		content, err := store.ReadPage(collectionID, w.Path)
		if err != nil {
			t.Errorf("ReadPage(%s) failed: %v", w.Path, err)
		}
		if content == "" {
			t.Errorf("ReadPage(%s) returned empty content", w.Path)
		}
	}
}
