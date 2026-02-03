package library

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// CrawlStateManager handles persistence of crawl state for resume capability.
type CrawlStateManager struct {
	store        *Store
	collectionID string
	state        *CrawlState
	dirty        bool
	lastSave     time.Time
}

// NewCrawlStateManager creates a state manager for a collection.
func NewCrawlStateManager(store *Store, collectionID string) *CrawlStateManager {
	return &CrawlStateManager{
		store:        store,
		collectionID: collectionID,
		lastSave:     time.Now(),
	}
}

// statePath returns the path to the crawl state file.
func (m *CrawlStateManager) statePath() string {
	return filepath.Join(m.store.collectionDir(m.collectionID), ".crawl-state.yaml")
}

// HasIncompleteState checks if an incomplete crawl exists.
func (m *CrawlStateManager) HasIncompleteState() bool {
	_, err := os.Stat(m.statePath())

	return err == nil
}

// LoadState loads existing crawl state from disk.
func (m *CrawlStateManager) LoadState() (*CrawlState, error) {
	data, err := os.ReadFile(m.statePath())
	if err != nil {
		return nil, fmt.Errorf("read crawl state: %w", err)
	}

	var state CrawlState
	if err := yaml.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parse crawl state: %w", err)
	}

	m.state = &state
	m.dirty = false

	return &state, nil
}

// InitState creates new crawl state with discovered URLs.
func (m *CrawlStateManager) InitState(source string, config *CrawlConfig, discoveredURLs []string) *CrawlState {
	m.state = NewCrawlState(m.collectionID, source, config)
	m.state.DiscoveredURLs = discoveredURLs
	m.state.Phase = PhaseFetching

	// Mark all URLs as pending
	for _, url := range discoveredURLs {
		m.state.ProcessedURLs[url] = URLStatus{Status: URLStatusPending}
	}

	m.dirty = true

	return m.state
}

// GetState returns the current state (may be nil if not loaded/initialized).
func (m *CrawlStateManager) GetState() *CrawlState {
	return m.state
}

// MarkURL updates status for a URL.
func (m *CrawlStateManager) MarkURL(url string, status URLStatus) {
	if m.state == nil {
		return
	}
	status.ProcessedAt = time.Now()
	m.state.ProcessedURLs[url] = status
	m.state.LastUpdatedAt = time.Now()
	m.dirty = true
}

// MarkSuccess marks a URL as successfully fetched.
func (m *CrawlStateManager) MarkSuccess(url, pagePath, contentHash string) {
	m.MarkURL(url, URLStatus{
		Status:      URLStatusSuccess,
		PagePath:    pagePath,
		ContentHash: contentHash,
	})
}

// MarkFailed marks a URL as failed with error.
func (m *CrawlStateManager) MarkFailed(url string, err error, retryCount int) {
	m.MarkURL(url, URLStatus{
		Status:     URLStatusFailed,
		Error:      err.Error(),
		RetryCount: retryCount,
	})
}

// MarkSkipped marks a URL as skipped (e.g., robots.txt disallowed).
func (m *CrawlStateManager) MarkSkipped(url, reason string) {
	m.MarkURL(url, URLStatus{
		Status: URLStatusSkipped,
		Error:  reason,
	})
}

// SaveState persists current state to disk.
func (m *CrawlStateManager) SaveState() error {
	if m.state == nil {
		return nil
	}

	// Don't save if not dirty (no changes since last save)
	if !m.dirty {
		return nil
	}

	// Ensure collection directory exists
	collDir := m.store.collectionDir(m.collectionID)
	if err := os.MkdirAll(collDir, 0o755); err != nil {
		return fmt.Errorf("create collection dir: %w", err)
	}

	m.state.LastUpdatedAt = time.Now()

	data, err := yaml.Marshal(m.state)
	if err != nil {
		return fmt.Errorf("marshal crawl state: %w", err)
	}

	path := m.statePath()
	tmpPath := path + ".tmp"

	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("write crawl state: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)

		return fmt.Errorf("rename crawl state: %w", err)
	}

	m.dirty = false
	m.lastSave = time.Now()

	return nil
}

// ShouldCheckpoint returns true if it's time to save state based on page count or time.
func (m *CrawlStateManager) ShouldCheckpoint(pageCount, pageInterval int, timeInterval time.Duration) bool {
	// Checkpoint every N pages
	if pageInterval > 0 && pageCount > 0 && pageCount%pageInterval == 0 {
		return true
	}
	// Checkpoint after time interval
	if timeInterval > 0 && time.Since(m.lastSave) >= timeInterval {
		return true
	}

	return false
}

// DeleteState removes the state file (called on successful completion).
func (m *CrawlStateManager) DeleteState() error {
	path := m.statePath()
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete crawl state: %w", err)
	}
	m.state = nil
	m.dirty = false

	return nil
}

// SetPhase updates the crawl phase.
func (m *CrawlStateManager) SetPhase(phase CrawlPhase) {
	if m.state == nil {
		return
	}
	m.state.Phase = phase
	m.state.LastUpdatedAt = time.Now()
	m.dirty = true
}

// ComputeContentHash generates a SHA256 hash of content for change detection.
func ComputeContentHash(content string) string {
	h := sha256.Sum256([]byte(content))

	return hex.EncodeToString(h[:])
}
