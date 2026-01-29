package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCommitHistory_LoadAttempts_NoHistory(t *testing.T) {
	t.Parallel()

	// Create a temp directory
	tmpDir := t.TempDir()
	history := NewCommitHistory(tmpDir)

	attempts, err := history.LoadAttempts()
	if err != nil {
		t.Fatalf("LoadAttempts() error = %v", err)
	}

	if len(attempts) != 0 {
		t.Errorf("LoadAttempts() returned %d attempts, want 0", len(attempts))
	}
}

func TestCommitHistory_SaveAndLoad(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	history := NewCommitHistory(tmpDir)

	attempt := CommitAttempt{
		Timestamp: time.Now().Truncate(time.Second),
		Groups: []ChangeGroup{
			{
				Files:   []string{"file1.go", "file2.go"},
				Message: "Add feature",
			},
		},
		Note:     "Test grouping",
		FileHash: "abc123",
		IsDryRun: true,
	}

	if err := history.SaveAttempt(attempt); err != nil {
		t.Fatalf("SaveAttempt() error = %v", err)
	}

	// Verify file was created
	historyFile := filepath.Join(tmpDir, "commit_history.json")
	if _, err := os.Stat(historyFile); os.IsNotExist(err) {
		t.Fatal("SaveAttempt() did not create history file")
	}

	// Load and verify
	loaded, err := history.LoadAttempts()
	if err != nil {
		t.Fatalf("LoadAttempts() error = %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("LoadAttempts() returned %d attempts, want 1", len(loaded))
	}

	if loaded[0].Note != attempt.Note {
		t.Errorf("LoadAttempts() note = %q, want %q", loaded[0].Note, attempt.Note)
	}

	if loaded[0].FileHash != attempt.FileHash {
		t.Errorf("LoadAttempts() fileHash = %q, want %q", loaded[0].FileHash, attempt.FileHash)
	}

	if !loaded[0].IsDryRun {
		t.Errorf("LoadAttempts() isDryRun = false, want true")
	}

	if len(loaded[0].Groups) != 1 {
		t.Fatalf("LoadAttempts() groups length = %d, want 1", len(loaded[0].Groups))
	}
}

func TestCommitHistory_MaxTenAttempts(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	history := NewCommitHistory(tmpDir)

	// Save 15 attempts
	for i := range 15 {
		attempt := CommitAttempt{
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
			Groups: []ChangeGroup{
				{Files: []string{"file.go"}},
			},
		}
		if err := history.SaveAttempt(attempt); err != nil {
			t.Fatalf("SaveAttempt() error = %v", err)
		}
	}

	// Should only have last 10
	loaded, err := history.LoadAttempts()
	if err != nil {
		t.Fatalf("LoadAttempts() error = %v", err)
	}

	if len(loaded) != 10 {
		t.Errorf("LoadAttempts() returned %d attempts, want 10 (max)", len(loaded))
	}
}

func TestCommitHistory_Clear(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	history := NewCommitHistory(tmpDir)

	// Save an attempt
	attempt := CommitAttempt{
		Timestamp: time.Now(),
		Groups: []ChangeGroup{
			{Files: []string{"file.go"}},
		},
	}
	if err := history.SaveAttempt(attempt); err != nil {
		t.Fatalf("SaveAttempt() error = %v", err)
	}

	// Clear
	if err := history.Clear(); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	// Verify empty
	loaded, err := history.LoadAttempts()
	if err != nil {
		t.Fatalf("LoadAttempts() error = %v", err)
	}

	if len(loaded) != 0 {
		t.Errorf("LoadAttempts() after Clear() returned %d attempts, want 0", len(loaded))
	}
}
