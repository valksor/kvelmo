// Package provider provides utilities for detecting changes in work units.
package provider

import (
	"fmt"
	"strings"
	"time"
)

// ChangeSet describes the differences between two versions of a work unit.
type ChangeSet struct {
	HasChanges         bool         // True if any changes were detected
	DescriptionChanged bool         // True if the description changed
	NewComments        []Comment    // Comments that exist in new but not old
	UpdatedComments    []Comment    // Comments that exist in both but have different text
	NewAttachments     []Attachment // Attachments that exist in new but not old
	RemovedAttachments []Attachment // Attachments that exist in old but not new
	OldDescription     string       // The old description for comparison
	NewDescription     string       // The new description for comparison
	OldTitle           string       // The old title
	NewTitle           string       // The new title
	StatusChanged      bool         // True if status changed
	PriorityChanged    bool         // True if priority changed
	OldStatus          Status
	NewStatus          Status
	OldPriority        Priority
	NewPriority        Priority
}

// DetectChanges compares two work units and returns a ChangeSet describing the differences.
func DetectChanges(old, updated *WorkUnit) ChangeSet {
	changes := ChangeSet{
		OldDescription: old.Description,
		NewDescription: updated.Description,
		OldTitle:       old.Title,
		NewTitle:       updated.Title,
		OldStatus:      old.Status,
		NewStatus:      updated.Status,
		OldPriority:    old.Priority,
		NewPriority:    updated.Priority,
	}

	// Check for title changes
	if old.Title != updated.Title {
		changes.NewTitle = updated.Title
		changes.OldTitle = old.Title
		changes.HasChanges = true
	}

	// Check for description changes
	if old.Description != updated.Description {
		changes.DescriptionChanged = true
		changes.HasChanges = true
	}

	// Check for status changes
	if old.Status != updated.Status {
		changes.StatusChanged = true
		changes.HasChanges = true
	}

	// Check for priority changes
	if old.Priority != updated.Priority {
		changes.PriorityChanged = true
		changes.HasChanges = true
	}

	// Check for new comments
	changes.NewComments = findNewComments(old.Comments, updated.Comments)
	if len(changes.NewComments) > 0 {
		changes.HasChanges = true
	}

	// Check for updated comments
	changes.UpdatedComments = findUpdatedComments(old.Comments, updated.Comments)
	if len(changes.UpdatedComments) > 0 {
		changes.HasChanges = true
	}

	// Check for new attachments
	changes.NewAttachments = findNewAttachments(old.Attachments, updated.Attachments)
	if len(changes.NewAttachments) > 0 {
		changes.HasChanges = true
	}

	// Check for removed attachments
	changes.RemovedAttachments = findRemovedAttachments(old.Attachments, updated.Attachments)
	if len(changes.RemovedAttachments) > 0 {
		changes.HasChanges = true
	}

	return changes
}

// findNewComments returns comments that exist in updated but not in old.
func findNewComments(old, updated []Comment) []Comment {
	if len(updated) == 0 {
		return nil
	}

	oldIDs := make(map[string]bool)
	for _, c := range old {
		oldIDs[c.ID] = true
	}

	var newComments []Comment
	for _, c := range updated {
		if !oldIDs[c.ID] {
			newComments = append(newComments, c)
		}
	}

	return newComments
}

// findUpdatedComments returns comments from updated that have different text than in old.
func findUpdatedComments(old, updated []Comment) []Comment {
	if len(old) == 0 || len(updated) == 0 {
		return nil
	}

	oldComments := make(map[string]Comment)
	for _, c := range old {
		oldComments[c.ID] = c
	}

	var updatedComments []Comment
	for _, c := range updated {
		if oldComment, exists := oldComments[c.ID]; exists {
			// Compare text (trimmed)
			oldText := strings.TrimSpace(oldComment.Body)
			newText := strings.TrimSpace(c.Body)
			if oldText != newText && newText != "" {
				updatedComments = append(updatedComments, c)
			}
		}
	}

	return updatedComments
}

// findNewAttachments returns attachments that exist in updated but not in old.
func findNewAttachments(old, updated []Attachment) []Attachment {
	if len(updated) == 0 {
		return nil
	}

	oldIDs := make(map[string]bool)
	for _, a := range old {
		oldIDs[a.ID] = true
	}

	var newAttachments []Attachment
	for _, a := range updated {
		if !oldIDs[a.ID] {
			newAttachments = append(newAttachments, a)
		}
	}

	return newAttachments
}

// findRemovedAttachments returns attachments that exist in old but not in updated.
func findRemovedAttachments(old, updated []Attachment) []Attachment {
	if len(old) == 0 {
		return nil
	}

	updatedIDs := make(map[string]bool)
	for _, a := range updated {
		updatedIDs[a.ID] = true
	}

	var removed []Attachment
	for _, a := range old {
		if !updatedIDs[a.ID] {
			removed = append(removed, a)
		}
	}

	return removed
}

// Summary returns a human-readable summary of changes.
func (c ChangeSet) Summary() string {
	if !c.HasChanges {
		return "No changes detected"
	}

	var parts []string

	if c.OldTitle != c.NewTitle {
		parts = append(parts, fmt.Sprintf("title changed from %q to %q", c.OldTitle, c.NewTitle))
	}
	if c.StatusChanged {
		parts = append(parts, string(c.NewStatus))
	}
	if c.DescriptionChanged {
		parts = append(parts, "description updated")
	}
	if len(c.NewComments) > 0 {
		parts = append(parts, countedStr(len(c.NewComments), "new comment"))
	}
	if len(c.UpdatedComments) > 0 {
		parts = append(parts, countedStr(len(c.UpdatedComments), "updated comment"))
	}
	if len(c.NewAttachments) > 0 {
		parts = append(parts, countedStr(len(c.NewAttachments), "new attachment"))
	}
	if len(c.RemovedAttachments) > 0 {
		parts = append(parts, countedStr(len(c.RemovedAttachments), "removed attachment"))
	}

	return strings.Join(parts, ", ")
}

// countedStr returns a string like "N item(s)".
func countedStr(count int, item string) string {
	if count == 1 {
		return "1 " + item
	}

	return fmt.Sprintf("%d %ss", count, item)
}

// FormatDiff returns a formatted diff of the changes.
func (c ChangeSet) FormatDiff() string {
	var builder strings.Builder

	builder.WriteString("Changes detected:\n")

	if c.StatusChanged {
		builder.WriteString(fmt.Sprintf("  Status: %s → %s\n", c.OldStatus, c.NewStatus))
	}

	if c.DescriptionChanged {
		builder.WriteString("  Description: updated\n")
	}

	if len(c.NewComments) > 0 {
		builder.WriteString(fmt.Sprintf("  New comments: %d\n", len(c.NewComments)))
		for _, comment := range c.NewComments {
			author := comment.Author.Name
			if author == "" {
				author = comment.Author.ID
			}
			builder.WriteString(fmt.Sprintf("    - %s: %s\n", author, truncate(comment.Body, 60)))
		}
	}

	if len(c.UpdatedComments) > 0 {
		builder.WriteString(fmt.Sprintf("  Updated comments: %d\n", len(c.UpdatedComments)))
	}

	if len(c.NewAttachments) > 0 {
		builder.WriteString(fmt.Sprintf("  New attachments: %d\n", len(c.NewAttachments)))
		for _, att := range c.NewAttachments {
			builder.WriteString(fmt.Sprintf("    - %s\n", att.Name))
		}
	}

	if len(c.RemovedAttachments) > 0 {
		builder.WriteString(fmt.Sprintf("  Removed attachments: %d\n", len(c.RemovedAttachments)))
		for _, att := range c.RemovedAttachments {
			builder.WriteString(fmt.Sprintf("    - %s\n", att.Name))
		}
	}

	return builder.String()
}

// truncate returns a truncated version of a string.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	return s[:maxLen-3] + "..."
}

// GetMostRecentUpdate returns the most recent update timestamp from comments and attachments.
func (c ChangeSet) GetMostRecentUpdate() time.Time {
	var mostRecent time.Time

	for _, comment := range c.NewComments {
		if comment.CreatedAt.After(mostRecent) {
			mostRecent = comment.CreatedAt
		}
	}

	for _, att := range c.NewAttachments {
		if att.CreatedAt.After(mostRecent) {
			mostRecent = att.CreatedAt
		}
	}

	return mostRecent
}

// ResolveAuthor extracts the author name from a comment.
// This is a provider-agnostic helper that works with the standard Comment type.
func ResolveAuthor(comment Comment) string {
	if comment.Author.Name != "" {
		return comment.Author.Name
	}
	if comment.Author.ID != "" {
		return comment.Author.ID
	}

	return ""
}
