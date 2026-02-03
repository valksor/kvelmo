package library

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/valksor/go-mehrhof/internal/storage"
	"gopkg.in/yaml.v3"
)

// Store handles disk I/O for library collections.
type Store struct {
	rootDir     string
	projectID   string
	lockTimeout time.Duration
}

// NewStore creates a store for the given location.
// If shared is true, uses the shared location; otherwise uses project-namespaced storage.
func NewStore(ctx context.Context, repoRoot string, shared bool, lockTimeout time.Duration) (*Store, error) {
	homeDir, err := storage.GetMehrhofHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get mehrhof home: %w", err)
	}

	var rootDir, projectID string
	if shared {
		rootDir = filepath.Join(homeDir, "library", "shared")
	} else {
		projectID, err = storage.GenerateProjectID(ctx, repoRoot)
		if err != nil {
			return nil, fmt.Errorf("generate project ID: %w", err)
		}
		rootDir = filepath.Join(homeDir, "library", projectID)
	}

	if lockTimeout == 0 {
		lockTimeout = 10 * time.Second
	}

	return &Store{
		rootDir:     rootDir,
		projectID:   projectID,
		lockTimeout: lockTimeout,
	}, nil
}

// NewStoreWithRoot creates a store with an explicit root directory (for testing).
func NewStoreWithRoot(rootDir string, lockTimeout time.Duration) *Store {
	if lockTimeout == 0 {
		lockTimeout = 10 * time.Second
	}

	return &Store{
		rootDir:     rootDir,
		lockTimeout: lockTimeout,
	}
}

// RootDir returns the store's root directory.
func (s *Store) RootDir() string {
	return s.rootDir
}

// ProjectID returns the store's project ID (empty for shared store).
func (s *Store) ProjectID() string {
	return s.projectID
}

// manifestPath returns the path to the manifest file.
func (s *Store) manifestPath() string {
	return filepath.Join(s.rootDir, "manifest.yaml")
}

// lockPath returns the path to the lock file.
func (s *Store) lockPath() string {
	return filepath.Join(s.rootDir, ".manifest.lock")
}

// collectionDir returns the directory for a collection.
func (s *Store) collectionDir(id string) string {
	return filepath.Join(s.rootDir, "collections", id)
}

// pagesDir returns the pages directory for a collection.
func (s *Store) pagesDir(id string) string {
	return filepath.Join(s.collectionDir(id), "pages")
}

// metaPath returns the path to a collection's metadata file.
func (s *Store) metaPath(id string) string {
	return filepath.Join(s.collectionDir(id), "meta.yaml")
}

// ensureDir creates a directory if it doesn't exist.
func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

// LoadManifest loads the manifest from disk.
func (s *Store) LoadManifest() (*Manifest, error) {
	path := s.manifestPath()

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return NewManifest(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}

	if m.Collections == nil {
		m.Collections = make([]*Collection, 0)
	}

	return &m, nil
}

// SaveManifest saves the manifest to disk with file locking.
func (s *Store) SaveManifest(m *Manifest) error {
	if err := ensureDir(s.rootDir); err != nil {
		return fmt.Errorf("create root dir: %w", err)
	}

	return storage.WithLockTimeout(s.lockPath(), s.lockTimeout, func() error {
		return s.atomicWriteManifest(m)
	})
}

// atomicWriteManifest writes the manifest using a temp file + rename for atomicity.
func (s *Store) atomicWriteManifest(m *Manifest) error {
	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}

	path := s.manifestPath()
	tmpPath := path + ".tmp"

	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("write temp manifest: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)

		return fmt.Errorf("rename manifest: %w", err)
	}

	return nil
}

// LoadCollectionMeta loads a collection's metadata.
func (s *Store) LoadCollectionMeta(id string) (*CollectionMeta, error) {
	path := s.metaPath(id)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read collection meta: %w", err)
	}

	var meta CollectionMeta
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("parse collection meta: %w", err)
	}

	if meta.Pages == nil {
		meta.Pages = make([]*Page, 0)
	}

	return &meta, nil
}

// SaveCollectionMeta saves a collection's metadata.
func (s *Store) SaveCollectionMeta(id string, meta *CollectionMeta) error {
	dir := s.collectionDir(id)
	if err := ensureDir(dir); err != nil {
		return fmt.Errorf("create collection dir: %w", err)
	}

	data, err := yaml.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshal collection meta: %w", err)
	}

	path := s.metaPath(id)
	tmpPath := path + ".tmp"

	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("write temp meta: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)

		return fmt.Errorf("rename meta: %w", err)
	}

	return nil
}

// WritePage writes a page's content to disk.
func (s *Store) WritePage(collectionID, pagePath, content string) error {
	fullPath := filepath.Join(s.pagesDir(collectionID), pagePath)

	// Create parent directories
	if err := ensureDir(filepath.Dir(fullPath)); err != nil {
		return fmt.Errorf("create page dir: %w", err)
	}

	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write page: %w", err)
	}

	return nil
}

// ReadPage reads a page's content from disk.
func (s *Store) ReadPage(collectionID, pagePath string) (string, error) {
	fullPath := filepath.Join(s.pagesDir(collectionID), pagePath)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("read page: %w", err)
	}

	return string(data), nil
}

// DeleteCollection removes a collection and all its pages.
func (s *Store) DeleteCollection(id string) error {
	dir := s.collectionDir(id)

	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove collection dir: %w", err)
	}

	return nil
}

// CollectionExists checks if a collection directory exists.
func (s *Store) CollectionExists(id string) bool {
	_, err := os.Stat(s.collectionDir(id))

	return err == nil
}

// ListPageFiles returns all page files in a collection.
func (s *Store) ListPageFiles(id string) ([]string, error) {
	pagesDir := s.pagesDir(id)

	var files []string
	err := filepath.Walk(pagesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			rel, err := filepath.Rel(pagesDir, path)
			if err != nil {
				return err
			}
			files = append(files, rel)
		}

		return nil
	})

	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list page files: %w", err)
	}

	return files, nil
}

// SaveCollection saves a collection to the manifest (with locking).
func (s *Store) SaveCollection(c *Collection) error {
	return storage.WithLockTimeout(s.lockPath(), s.lockTimeout, func() error {
		manifest, err := s.LoadManifest()
		if err != nil {
			return fmt.Errorf("load manifest: %w", err)
		}

		manifest.AddCollection(c)

		return s.atomicWriteManifest(manifest)
	})
}

// RemoveCollectionFromManifest removes a collection from the manifest (with locking).
func (s *Store) RemoveCollectionFromManifest(id string) error {
	return storage.WithLockTimeout(s.lockPath(), s.lockTimeout, func() error {
		manifest, err := s.LoadManifest()
		if err != nil {
			return fmt.Errorf("load manifest: %w", err)
		}

		if !manifest.RemoveCollection(id) {
			return fmt.Errorf("collection %q not found in manifest", id)
		}

		return s.atomicWriteManifest(manifest)
	})
}

// GetCollection loads a collection from the manifest.
func (s *Store) GetCollection(id string) (*Collection, error) {
	manifest, err := s.LoadManifest()
	if err != nil {
		return nil, err
	}

	c := manifest.GetCollection(id)
	if c == nil {
		return nil, fmt.Errorf("collection %q not found", id)
	}

	return c, nil
}

// ListCollections returns all collections in the manifest.
func (s *Store) ListCollections() ([]*Collection, error) {
	manifest, err := s.LoadManifest()
	if err != nil {
		return nil, err
	}

	return manifest.Collections, nil
}

// WritePages writes multiple pages incrementally.
// Returns pages successfully written and any errors encountered.
func (s *Store) WritePages(collectionID string, pages []*CrawledPage) ([]*Page, []error) {
	var written []*Page
	var errors []error

	for _, cp := range pages {
		if cp.Error != nil {
			errors = append(errors, cp.Error)

			continue
		}

		if err := s.WritePage(collectionID, cp.Path, cp.Content); err != nil {
			errors = append(errors, fmt.Errorf("write %s: %w", cp.Path, err))

			continue
		}

		written = append(written, cp.ToPage())
	}

	return written, errors
}

// ListIncompleteCollections returns collection IDs with interrupted crawls.
func (s *Store) ListIncompleteCollections() ([]string, error) {
	collectionsDir := filepath.Join(s.rootDir, "collections")
	entries, err := os.ReadDir(collectionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("list collections: %w", err)
	}

	var incomplete []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		statePath := filepath.Join(collectionsDir, entry.Name(), ".crawl-state.yaml")
		if _, err := os.Stat(statePath); err == nil {
			incomplete = append(incomplete, entry.Name())
		}
	}

	return incomplete, nil
}
