package storage

import (
	"time"

	"github.com/valksor/go-toolkit/history"
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
	h *history.History[CommitAttempt]
}

// NewCommitHistory creates a new history manager for a workspace.
func NewCommitHistory(workspacePath string) *CommitHistory {
	return &CommitHistory{
		h: history.New[CommitAttempt](workspacePath, "commit_history.json"),
	}
}

// LoadAttempts loads all previous attempts from disk.
func (h *CommitHistory) LoadAttempts() ([]CommitAttempt, error) {
	return h.h.Load()
}

// SaveAttempt saves a new attempt to disk.
func (h *CommitHistory) SaveAttempt(attempt CommitAttempt) error {
	return h.h.Save(attempt)
}

// Clear removes all history (useful after successful commits).
func (h *CommitHistory) Clear() error {
	return h.h.Clear()
}
