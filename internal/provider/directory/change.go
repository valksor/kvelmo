package directory

import (
	"fmt"
	"strings"
	"time"
)

// DirectoryChanges represents differences between two directory snapshots.
type DirectoryChanges struct {
	Added    []FileInfo
	Removed  []FileInfo
	Modified []FileInfo
}

// CompareDirectorySnapshots compares two directory snapshots and returns the differences.
// Detects new files, removed files, and modified files (size or mtime changed).
//
// Note: This comparison uses file size and modification time as a fast heuristic.
//   - May miss content changes that preserve file size (rare but possible)
//   - May report false positives if file is touched without content changes
//   - Does not detect permission or ownership changes
//   - For accuracy-critical use cases, consider content hashing (SHA-256)
func CompareDirectorySnapshots(old, updated DirectorySnapshot) DirectoryChanges {
	var changes DirectoryChanges

	// Build maps for easier comparison
	oldFiles := make(map[string]FileInfo)
	for _, f := range old.Files {
		oldFiles[f.Path] = f
	}

	newFiles := make(map[string]FileInfo)
	for _, f := range updated.Files {
		newFiles[f.Path] = f
	}

	// Find added files (in new but not in old)
	for path, newFile := range newFiles {
		if _, exists := oldFiles[path]; !exists {
			changes.Added = append(changes.Added, newFile)
		}
	}

	// Find removed and modified files
	for path, oldFile := range oldFiles {
		newFile, exists := newFiles[path]
		if !exists {
			// File was removed
			changes.Removed = append(changes.Removed, oldFile)
		} else {
			// File exists in both, check if modified
			if oldFile.Size != newFile.Size || oldFile.ModTime != newFile.ModTime {
				changes.Modified = append(changes.Modified, newFile)
			}
		}
	}

	return changes
}

// HasChanges returns true if there are any changes in the DirectoryChanges.
func (d DirectoryChanges) HasChanges() bool {
	return len(d.Added) > 0 || len(d.Removed) > 0 || len(d.Modified) > 0
}

// Summary returns a human-readable summary of changes.
func (d DirectoryChanges) Summary() string {
	if len(d.Added) == 0 && len(d.Modified) == 0 && len(d.Removed) == 0 {
		return "No directory changes"
	}

	var builder strings.Builder
	first := true

	if len(d.Added) > 0 {
		builder.WriteString(countedStr(len(d.Added), "new file"))
		first = false
	}
	if len(d.Modified) > 0 {
		if !first {
			builder.WriteString(", ")
		}
		builder.WriteString(countedStr(len(d.Modified), "modified file"))
		first = false
	}
	if len(d.Removed) > 0 {
		if !first {
			builder.WriteString(", ")
		}
		builder.WriteString(countedStr(len(d.Removed), "removed file"))
	}

	return builder.String()
}

// MostRecentChange returns the most recent modification time from all changes.
func (d DirectoryChanges) MostRecentChange() time.Time {
	var mostRecent time.Time

	checkRecent := func(t time.Time) {
		if t.After(mostRecent) {
			mostRecent = t
		}
	}

	for _, f := range d.Added {
		checkRecent(f.ModTime)
	}
	for _, f := range d.Modified {
		checkRecent(f.ModTime)
	}
	for _, f := range d.Removed {
		checkRecent(f.ModTime)
	}

	return mostRecent
}

// countedStr returns a string like "N item(s)".
func countedStr(count int, item string) string {
	if count == 1 {
		return "1 " + item
	}

	return fmt.Sprintf("%d %ss", count, item)
}
