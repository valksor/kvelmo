// Package hierarchy provides common utilities for parent-child task relationships.
//
// Many task providers support hierarchical relationships (subtasks, parent issues,
// child items). This package consolidates common patterns and errors.
package hierarchy

import (
	"context"
	"errors"
	"strings"

	"github.com/valksor/go-mehrhof/internal/provider"
)

// Common errors for hierarchy operations.
var (
	// ErrNotASubtask is returned when a work unit is not a subtask
	// (i.e., it has no parent or is not a hierarchical child).
	ErrNotASubtask = errors.New("not a subtask")

	// ErrNoParent is returned when a subtask has no parent
	// (orphaned subtask or relationship not found).
	ErrNoParent = errors.New("no parent found")
)

// SubtaskPattern defines common patterns for detecting subtasks from their ID.
type SubtaskPattern struct {
	// Contains is a substring that indicates a subtask ID.
	// e.g., "-task-" for GitHub/GitLab, ":task-" for Bitbucket
	Contains string

	// Prefix is a prefix that indicates a subtask ID.
	// e.g., "checkitem/" for Trello
	Prefix string
}

// IsSubtaskID checks if the given ID matches the subtask pattern.
func (p SubtaskPattern) IsSubtaskID(id string) bool {
	if p.Contains != "" && strings.Contains(id, p.Contains) {
		return true
	}
	if p.Prefix != "" && strings.HasPrefix(id, p.Prefix) {
		return true
	}

	return false
}

// Common subtask patterns used by various providers.
var (
	// GitHubSubtaskPattern matches GitHub task list items (owner/repo#123-task-1).
	GitHubSubtaskPattern = SubtaskPattern{Contains: "-task-"}

	// GitLabSubtaskPattern matches GitLab task list items (project#123-task-1).
	GitLabSubtaskPattern = SubtaskPattern{Contains: "-task-"}

	// BitbucketSubtaskPattern matches Bitbucket task list items (project:123:task-1).
	BitbucketSubtaskPattern = SubtaskPattern{Contains: ":task-"}

	// TrelloSubtaskPattern matches Trello checklist items.
	TrelloSubtaskPattern = SubtaskPattern{Contains: "/checkitem/"}
)

// ExtractParentID extracts the parent ID from a subtask ID by removing the subtask suffix.
// Works with patterns like "owner/repo#123-task-1" -> "owner/repo#123"
// or "project:123:task-1" -> "project:123".
//
// Returns the parent ID and true if a subtask pattern was found.
// Returns the original ID and false if no subtask pattern was found.
func ExtractParentID(subtaskID string, pattern SubtaskPattern) (string, bool) {
	if pattern.Contains != "" {
		idx := strings.Index(subtaskID, pattern.Contains)
		if idx != -1 {
			return subtaskID[:idx], true
		}
	}
	if pattern.Prefix != "" {
		if strings.HasPrefix(subtaskID, pattern.Prefix) {
			// For prefix patterns like "checkitem/", we can't extract parent from ID
			// The provider needs to fetch metadata to find the parent
			return "", false
		}
	}

	return subtaskID, false
}

// FetcherFunc is a function that fetches a work unit by ID.
type FetcherFunc func(ctx context.Context, id string) (*provider.WorkUnit, error)

// FetchParentByID is a helper for providers where the parent ID can be extracted from the subtask ID.
// It extracts the parent ID using the pattern and fetches it using the provided function.
func FetchParentByID(ctx context.Context, subtaskID string, pattern SubtaskPattern, fetcher FetcherFunc) (*provider.WorkUnit, error) {
	parentID, isSubtask := ExtractParentID(subtaskID, pattern)
	if !isSubtask {
		return nil, ErrNotASubtask
	}
	if parentID == "" {
		return nil, ErrNoParent
	}

	return fetcher(ctx, parentID)
}
