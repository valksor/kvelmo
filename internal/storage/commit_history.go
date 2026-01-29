package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// CommitAttempt records a grouping attempt for context in subsequent runs.
type CommitAttempt struct {
	Timestamp time.Time     `json:"timestamp"`
	Groups    []ChangeGroup `json:"groups"`
	Note      string        `json:"note,omitempty"`
	FileHash  string        `json:"file_hash,omitempty"` // Hash of files to detect if they changed
	IsDryRun  bool          `json:"dry_run"`             // Whether this was a dry run
}

// ChangeGroup represents a group of files for a commit.
type ChangeGroup struct {
	Files   []string `json:"files"`
	Message string   `json:"message,omitempty"` // Generated commit message
}

// CommitHistory manages persistence of commit grouping attempts.
type CommitHistory struct {
	workspacePath string
	historyFile   string
}

// NewCommitHistory creates a new history manager for a workspace.
func NewCommitHistory(workspacePath string) *CommitHistory {
	historyFile := filepath.Join(workspacePath, "commit_history.json")

	return &CommitHistory{
		workspacePath: workspacePath,
		historyFile:   historyFile,
	}
}

// LoadAttempts loads all previous attempts from disk.
func (h *CommitHistory) LoadAttempts() ([]CommitAttempt, error) {
	data, err := os.ReadFile(h.historyFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []CommitAttempt{}, nil
		}

		return nil, err
	}

	var attempts []CommitAttempt
	if err := json.Unmarshal(data, &attempts); err != nil {
		return nil, err
	}

	return attempts, nil
}

// SaveAttempt saves a new attempt to disk.
func (h *CommitHistory) SaveAttempt(attempt CommitAttempt) error {
	attempts, err := h.LoadAttempts()
	if err != nil {
		return err
	}

	// Keep only last 10 attempts to avoid bloat
	attempts = append(attempts, attempt)
	if len(attempts) > 10 {
		attempts = attempts[len(attempts)-10:]
	}

	data, err := json.MarshalIndent(attempts, "", "  ")
	if err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(h.historyFile), 0o755); err != nil {
		return err
	}

	return os.WriteFile(h.historyFile, data, 0o644)
}

// Clear removes all history (useful after successful commits).
func (h *CommitHistory) Clear() error {
	return os.Remove(h.historyFile)
}
