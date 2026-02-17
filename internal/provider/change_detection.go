// Package provider provides utilities for detecting changes in work units.
package provider

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/valksor/go-toolkit/workunit"
)

// ChangeSet describes the differences between two versions of a work unit.
type ChangeSet struct {
	HasChanges         bool                  // True if any changes were detected
	DescriptionChanged bool                  // True if the description changed
	NewComments        []workunit.Comment    // Comments that exist in new but not old
	UpdatedComments    []workunit.Comment    // Comments that exist in both but have different text
	NewAttachments     []workunit.Attachment // Attachments that exist in new but not old
	RemovedAttachments []workunit.Attachment // Attachments that exist in old but not new
	OldDescription     string                // The old description for comparison
	NewDescription     string                // The new description for comparison
	OldTitle           string                // The old title
	NewTitle           string                // The new title
	StatusChanged      bool                  // True if status changed
	PriorityChanged    bool                  // True if priority changed
	OldStatus          workunit.Status
	NewStatus          workunit.Status
	OldPriority        workunit.Priority
	NewPriority        workunit.Priority
	LabelsChanged      bool              // True if labels changed
	OldLabels          []string          // Old labels
	NewLabels          []string          // New labels
	AssigneesChanged   bool              // True if assignees changed
	OldAssignees       []workunit.Person // Old assignees
	NewAssignees       []workunit.Person // New assignees
}

// DetectChanges compares two work units and returns a ChangeSet describing the differences.
func DetectChanges(old, updated *workunit.WorkUnit) ChangeSet {
	changes := ChangeSet{
		OldDescription: old.Description,
		NewDescription: updated.Description,
		OldTitle:       old.Title,
		NewTitle:       updated.Title,
		OldStatus:      old.Status,
		NewStatus:      updated.Status,
		OldPriority:    old.Priority,
		NewPriority:    updated.Priority,
		OldLabels:      old.Labels,
		NewLabels:      updated.Labels,
		OldAssignees:   old.Assignees,
		NewAssignees:   updated.Assignees,
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

	// Check for label changes
	if !EqualStringSlices(old.Labels, updated.Labels) {
		changes.LabelsChanged = true
		changes.HasChanges = true
	}

	// Check for assignee changes
	if !equalPersonSlices(old.Assignees, updated.Assignees) {
		changes.AssigneesChanged = true
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
func findNewComments(old, updated []workunit.Comment) []workunit.Comment {
	if len(updated) == 0 {
		return nil
	}

	oldIDs := make(map[string]bool)
	for _, c := range old {
		oldIDs[c.ID] = true
	}

	var newComments []workunit.Comment
	for _, c := range updated {
		if !oldIDs[c.ID] {
			newComments = append(newComments, c)
		}
	}

	return newComments
}

// findUpdatedComments returns comments from updated that have different text than in old.
func findUpdatedComments(old, updated []workunit.Comment) []workunit.Comment {
	if len(old) == 0 || len(updated) == 0 {
		return nil
	}

	oldComments := make(map[string]workunit.Comment)
	for _, c := range old {
		oldComments[c.ID] = c
	}

	var updatedComments []workunit.Comment
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
func findNewAttachments(old, updated []workunit.Attachment) []workunit.Attachment {
	if len(updated) == 0 {
		return nil
	}

	oldIDs := make(map[string]bool)
	for _, a := range old {
		oldIDs[a.ID] = true
	}

	var newAttachments []workunit.Attachment
	for _, a := range updated {
		if !oldIDs[a.ID] {
			newAttachments = append(newAttachments, a)
		}
	}

	return newAttachments
}

// findRemovedAttachments returns attachments that exist in old but not in updated.
func findRemovedAttachments(old, updated []workunit.Attachment) []workunit.Attachment {
	if len(old) == 0 {
		return nil
	}

	updatedIDs := make(map[string]bool)
	for _, a := range updated {
		updatedIDs[a.ID] = true
	}

	var removed []workunit.Attachment
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
		parts = append(parts, fmt.Sprintf("status changed from %s to %s", c.OldStatus, c.NewStatus))
	}
	if c.PriorityChanged {
		parts = append(parts, fmt.Sprintf("priority changed from %s to %s", c.OldPriority, c.NewPriority))
	}
	if c.DescriptionChanged {
		parts = append(parts, "description updated")
	}
	if c.LabelsChanged {
		parts = append(parts, "labels updated")
	}
	if c.AssigneesChanged {
		parts = append(parts, "assignees updated")
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

	if c.PriorityChanged {
		builder.WriteString(fmt.Sprintf("  Priority: %s → %s\n", c.OldPriority, c.NewPriority))
	}

	if c.DescriptionChanged {
		builder.WriteString("  Description: updated\n")
	}

	if c.LabelsChanged {
		oldLabels := formatStringSlice(c.OldLabels)
		newLabels := formatStringSlice(c.NewLabels)
		builder.WriteString(fmt.Sprintf("  Labels: %s → %s\n", oldLabels, newLabels))
	}

	if c.AssigneesChanged {
		oldAssignees := workunit.PersonNames(c.OldAssignees)
		newAssignees := workunit.PersonNames(c.NewAssignees)
		oldStr := formatStringSlice(oldAssignees)
		newStr := formatStringSlice(newAssignees)
		builder.WriteString(fmt.Sprintf("  Assignees: %s → %s\n", oldStr, newStr))
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

// formatStringSlice formats a string slice for display.
// Returns "(none)" for nil/empty slices, otherwise comma-separated values.
func formatStringSlice(slice []string) string {
	if len(slice) == 0 {
		return "(none)"
	}

	return strings.Join(slice, ", ")
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
func ResolveAuthor(comment workunit.Comment) string {
	if comment.Author.Name != "" {
		return comment.Author.Name
	}
	if comment.Author.ID != "" {
		return comment.Author.ID
	}

	return ""
}

// EqualStringSlices compares two string slices for equality.
// The comparison is order-insensitive and treats nil and empty slices as equal.
func EqualStringSlices(a, b []string) bool {
	// Treat nil and empty slices as equal
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) != len(b) {
		return false
	}

	// Fast path: check in-order first (common case)
	// Only allocate and sort if the order differs
	equalInOrder := true
	for i := range a {
		if a[i] != b[i] {
			equalInOrder = false

			break
		}
	}
	if equalInOrder {
		return true
	}

	// For small slices, use a map-based comparison (O(n)) instead of sort (O(n log n))
	// This is more efficient for the typical case of 1-5 labels
	if len(a) <= 10 {
		return equalStringSlicesMap(a, b)
	}

	// For larger slices, use sorting
	aCopy := make([]string, len(a))
	bCopy := make([]string, len(b))
	copy(aCopy, a)
	copy(bCopy, b)

	sort.Strings(aCopy)
	sort.Strings(bCopy)

	for i := range aCopy {
		if aCopy[i] != bCopy[i] {
			return false
		}
	}

	return true
}

// equalStringSlicesMap compares two string slices using a map.
// More efficient than sorting for small slices.
func equalStringSlicesMap(a, b []string) bool {
	// Build a map from the first slice
	seen := make(map[string]int, len(a))
	for _, s := range a {
		seen[s]++
	}

	// Check all elements in second slice exist in first with same counts
	for _, s := range b {
		count, exists := seen[s]
		if !exists || count == 0 {
			return false
		}
		seen[s]--
	}

	// Verify all counts are zero
	for _, count := range seen {
		if count != 0 {
			return false
		}
	}

	return true
}

// equalPersonSlices compares two Person slices for equality.
// The comparison is order-insensitive, compares by ID, and treats duplicates within
// each slice as the same person (e.g., [{ID:"1"}, {ID:"1"}] has one unique person).
// Returns true if both slices contain the same set of unique persons.
func equalPersonSlices(a, b []workunit.Person) bool {
	// Treat nil and empty slices as equal
	if len(a) == 0 && len(b) == 0 {
		return true
	}

	// Deduplicate within each slice (count unique IDs)
	aIDs := make(map[string]bool, len(a))
	for _, p := range a {
		aIDs[p.ID] = true
	}

	bIDs := make(map[string]bool, len(b))
	for _, p := range b {
		bIDs[p.ID] = true
	}

	// Compare unique ID counts
	if len(aIDs) != len(bIDs) {
		return false
	}

	// Verify all IDs from a exist in b
	for id := range aIDs {
		if !bIDs[id] {
			return false
		}
	}

	return true
}
