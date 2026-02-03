package storage

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// QueueNote represents a single note entry with timestamp.
type QueueNote struct {
	Timestamp string    // ISO timestamp
	Content   string    // Note content
	CreatedAt time.Time // Parsed time
}

// QueueNotesPath returns the notes directory for a queue.
func (ws *Workspace) QueueNotesPath(queueID string) string {
	return filepath.Join(ws.workspaceRoot, QueuesDir, queueID, "notes")
}

// QueueNotePath returns the notes file path for a specific task.
func (ws *Workspace) QueueNotePath(queueID, taskID string) string {
	return filepath.Join(ws.QueueNotesPath(queueID), taskID+".md")
}

// PlanOutputPath returns the path to the plan output file for a queue.
func (ws *Workspace) PlanOutputPath(queueID string) string {
	return filepath.Join(ws.workspaceRoot, QueuesDir, queueID, "plan.md")
}

// SavePlanOutput saves the raw AI planning output for debugging.
// This is written alongside queue.yaml so users can inspect what the AI returned.
func (ws *Workspace) SavePlanOutput(queueID, content string) error {
	planPath := ws.PlanOutputPath(queueID)
	dir := filepath.Dir(planPath)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create queue directory: %w", err)
	}

	if err := os.WriteFile(planPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write plan output: %w", err)
	}

	return nil
}

// LoadQueueNotes loads notes for a specific task.
// Returns notes in chronological order (oldest first).
func (ws *Workspace) LoadQueueNotes(queueID, taskID string) ([]QueueNote, error) {
	notesFile := ws.QueueNotePath(queueID, taskID)

	data, err := os.ReadFile(notesFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []QueueNote{}, nil
		}

		return nil, fmt.Errorf("read notes file: %w", err)
	}

	notes, err := parseNotesFile(string(data))
	if err != nil {
		return nil, fmt.Errorf("parse notes: %w", err)
	}

	return notes, nil
}

// AppendQueueNote adds a note to a queue task.
// Creates the notes directory and file if they don't exist.
func (ws *Workspace) AppendQueueNote(queueID, taskID, content string) error {
	notesDir := ws.QueueNotesPath(queueID)
	if err := os.MkdirAll(notesDir, 0o755); err != nil {
		return fmt.Errorf("create notes directory: %w", err)
	}

	notesFile := ws.QueueNotePath(queueID, taskID)

	f, err := os.OpenFile(notesFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open notes file: %w", err)
	}
	defer func() { _ = f.Close() }()

	timestamp := time.Now().Format("2006-01-02 15:04")

	// Write note with markdown heading
	if _, err := fmt.Fprintf(f, "\n## %s\n\n%s\n", timestamp, content); err != nil {
		return fmt.Errorf("write note: %w", err)
	}

	return nil
}

// SetQueueNotes replaces all notes for a task.
// Useful for optimize operations that rewrite notes.
func (ws *Workspace) SetQueueNotes(queueID, taskID string, notes []QueueNote) error {
	notesDir := ws.QueueNotesPath(queueID)
	if err := os.MkdirAll(notesDir, 0o755); err != nil {
		return fmt.Errorf("create notes directory: %w", err)
	}

	notesFile := ws.QueueNotePath(queueID, taskID)

	var content strings.Builder
	for _, note := range notes {
		content.WriteString(fmt.Sprintf("\n## %s\n\n%s\n", note.Timestamp, note.Content))
	}

	if err := os.WriteFile(notesFile, []byte(content.String()), 0o644); err != nil {
		return fmt.Errorf("write notes file: %w", err)
	}

	return nil
}

// parseNotesFile parses a notes markdown file into note entries.
// Expected format:
//
//	## 2026-01-27 10:30
//
//	Note content here
//
//	## 2026-01-27 11:00
//
//	Another note
func parseNotesFile(content string) ([]QueueNote, error) {
	var notes []QueueNote

	scanner := bufio.NewScanner(strings.NewReader(content))
	var currentNote *QueueNote

	for scanner.Scan() {
		line := scanner.Text()

		// Check for note heading (## timestamp)
		if strings.HasPrefix(line, "## ") {
			// Save previous note if exists
			if currentNote != nil && currentNote.Content != "" {
				notes = append(notes, *currentNote)
			}

			timestamp := strings.TrimPrefix(line, "## ")
			timestamp = strings.TrimSpace(timestamp)

			// Parse timestamp
			parsed, err := time.Parse("2006-01-02 15:04", timestamp)
			if err != nil {
				parsed = time.Now() // Fallback to now if parsing fails
			}

			currentNote = &QueueNote{
				Timestamp: timestamp,
				CreatedAt: parsed,
			}
		} else if currentNote != nil {
			// Append line to current note content
			if currentNote.Content != "" {
				currentNote.Content += "\n"
			}
			currentNote.Content += line
		}
	}

	// Add last note
	if currentNote != nil && currentNote.Content != "" {
		// Trim leading/trailing whitespace from content
		currentNote.Content = strings.TrimSpace(currentNote.Content)
		if currentNote.Content != "" {
			notes = append(notes, *currentNote)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan notes: %w", err)
	}

	return notes, nil
}

// FormatQueueNotes formats notes as a markdown string for display.
func FormatQueueNotes(notes []QueueNote) string {
	if len(notes) == 0 {
		return "*No notes yet.*"
	}

	var sb strings.Builder
	for _, note := range notes {
		sb.WriteString(fmt.Sprintf("**%s:** %s\n", note.Timestamp, note.Content))
	}

	return sb.String()
}

// QueueNotesPlainText returns notes as plain text for AI prompts.
func QueueNotesPlainText(notes []QueueNote) string {
	if len(notes) == 0 {
		return ""
	}

	var sb strings.Builder
	for i, note := range notes {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(fmt.Sprintf("- [%s] %s", note.Timestamp, note.Content))
	}

	return sb.String()
}
