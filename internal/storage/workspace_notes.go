package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// NotesPath returns the path to notes.md.
func (w *Workspace) NotesPath(taskID string) string {
	return filepath.Join(w.WorkPath(taskID), notesFileName)
}

// AppendNote adds a note to notes.md.
func (w *Workspace) AppendNote(taskID, content, state string) error {
	notesPath := w.NotesPath(taskID)

	// Read existing content
	existing, _ := os.ReadFile(notesPath)

	// Format new note
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	stateTag := ""
	if state != "" {
		stateTag = fmt.Sprintf(" [%s]", state)
	}
	newNote := fmt.Sprintf("\n## %s%s\n\n%s\n", timestamp, stateTag, content)

	// Use strings.Builder for efficient concatenation
	var b strings.Builder
	b.Grow(len(existing) + len(newNote))
	b.Write(existing)
	b.WriteString(newNote)

	return os.WriteFile(notesPath, []byte(b.String()), 0o644)
}

// ReadNotes reads the notes file content.
func (w *Workspace) ReadNotes(taskID string) (string, error) {
	data, err := os.ReadFile(w.NotesPath(taskID))
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// ParseNotes parses raw notes markdown into note entries.
// Notes are stored with format: ## timestamp [state]\n\ncontent.
func ParseNotes(content string) []Note {
	var notes []Note
	lines := strings.Split(content, "\n")

	var currentNote *Note
	noteNumber := 0

	for _, line := range lines {
		// Check for note heading (## timestamp [state])
		if strings.HasPrefix(line, "## ") {
			// Save previous note if exists
			if currentNote != nil && strings.TrimSpace(currentNote.Content) != "" {
				currentNote.Content = strings.TrimSpace(currentNote.Content)
				notes = append(notes, *currentNote)
			}

			noteNumber++
			headerText := strings.TrimPrefix(line, "## ")
			headerText = strings.TrimSpace(headerText)

			// Parse state if present (format: "2026-01-27 10:30:05 [planning]")
			var timestamp, state string
			if idx := strings.Index(headerText, " ["); idx != -1 {
				timestamp = headerText[:idx]
				state = strings.TrimSuffix(headerText[idx+2:], "]")
			} else {
				timestamp = headerText
			}

			// Parse timestamp
			parsed, err := time.Parse("2006-01-02 15:04:05", timestamp)
			if err != nil {
				// Try without seconds
				parsed, err = time.Parse("2006-01-02 15:04", timestamp)
				if err != nil {
					parsed = time.Now()
				}
			}

			currentNote = &Note{
				Number:    noteNumber,
				Timestamp: parsed,
				State:     state,
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
	if currentNote != nil && strings.TrimSpace(currentNote.Content) != "" {
		currentNote.Content = strings.TrimSpace(currentNote.Content)
		notes = append(notes, *currentNote)
	}

	return notes
}

// LoadNotes loads and parses notes for a task.
// Returns an empty slice (not an error) if the notes file doesn't exist yet.
func (w *Workspace) LoadNotes(taskID string) ([]Note, error) {
	content, err := w.ReadNotes(taskID)
	if err != nil {
		if os.IsNotExist(err) {
			return []Note{}, nil // No notes file yet = empty notes
		}

		return nil, err
	}

	return ParseNotes(content), nil
}
