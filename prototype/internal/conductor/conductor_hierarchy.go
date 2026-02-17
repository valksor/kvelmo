package conductor

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-toolkit/workunit"
)

// ErrNotASubtask is returned when a work unit is not a subtask.
var ErrNotASubtask = errors.New("not a subtask")

// HierarchicalContext holds parent and sibling task information.
// This provides agents with broader context when working on subtasks.
type HierarchicalContext struct {
	Parent   *workunit.WorkUnit   // The parent task (if this is a subtask)
	Siblings []*workunit.WorkUnit // Sibling subtasks (if includeSiblings was requested)
}

// FetchHierarchicalContext retrieves parent and optionally sibling tasks for a work unit.
// This is useful when working on subtasks to provide agents with broader context.
//
// Parameters:
//   - ctx: Context for cancellation
//   - p: The provider instance (as returned by providers.Resolve)
//   - workUnit: The work unit to fetch hierarchy for
//   - includeSiblings: If true, fetch sibling subtasks from the parent
//
// Returns:
//   - *HierarchicalContext: Parent and optionally sibling tasks (nil if not a subtask or provider doesn't support parent fetching)
//   - error: Any error during fetching
func (c *Conductor) FetchHierarchicalContext(ctx context.Context, p any, workUnit *workunit.WorkUnit, includeSiblings bool) (*HierarchicalContext, error) {
	// Check if this work unit is a subtask
	if !isSubtask(workUnit) {
		return nil, ErrNotASubtask
	}

	// Check if provider supports parent fetching
	parentFetcher, ok := p.(workunit.ParentFetcher)
	if !ok {
		// Provider doesn't support parent fetching, return gracefully
		return nil, ErrNotASubtask
	}

	result := &HierarchicalContext{}

	// Fetch parent task
	parent, err := parentFetcher.FetchParent(ctx, workUnit.ID)
	if err != nil {
		// Log but don't fail - parent context is optional
		// The parent may have been deleted or access may be restricted
		return nil, fmt.Errorf("fetch parent: %w", err)
	}
	if parent != nil {
		result.Parent = parent
	}

	// Fetch siblings if requested and parent is available
	if includeSiblings && result.Parent != nil {
		// Check if provider supports subtask fetching
		subtaskFetcher, ok := p.(workunit.SubtaskFetcher)
		if ok {
			siblings, err := subtaskFetcher.FetchSubtasks(ctx, result.Parent.ID)
			if err == nil && len(siblings) > 0 {
				// Filter out the current work unit from siblings
				result.Siblings = filterSelf(siblings, workUnit.ID)
			}
			// If subtask fetching fails, we simply don't include siblings
		}
	}

	return result, nil
}

// FetchHierarchicalContextFromConfig retrieves hierarchical context based on workspace configuration.
// This reads the context section from workspace config to determine what to include.
// CLI options (c.opts.WithParent, c.opts.WithSiblings, c.opts.MaxSiblings) override workspace config.
func (c *Conductor) FetchHierarchicalContextFromConfig(ctx context.Context, p any, workUnit *workunit.WorkUnit) (*HierarchicalContext, error) {
	cfg, err := c.workspace.LoadConfig()
	if err != nil {
		// Use defaults if config can't be loaded
		return c.FetchHierarchicalContext(ctx, p, workUnit, true)
	}

	// Determine includeParent (CLI option overrides config)
	includeParent := true
	if c.opts.WithParent != nil {
		includeParent = *c.opts.WithParent
	} else if cfg.Context != nil {
		includeParent = cfg.Context.IncludeParent
	}

	// Determine includeSiblings (CLI option overrides config)
	includeSiblings := true
	if c.opts.WithSiblings != nil {
		includeSiblings = *c.opts.WithSiblings
	} else if cfg.Context != nil {
		includeSiblings = cfg.Context.IncludeSiblings
	}

	// Determine maxSiblings (CLI option overrides config)
	maxSiblings := 5
	if c.opts.MaxSiblings != nil {
		maxSiblings = *c.opts.MaxSiblings
	} else if cfg.Context != nil && cfg.Context.MaxSiblings > 0 {
		maxSiblings = cfg.Context.MaxSiblings
	}

	ctxInfo, err := c.FetchHierarchicalContext(ctx, p, workUnit, includeSiblings)
	if err != nil {
		// If not a subtask or provider doesn't support it, return gracefully
		return nil, err
	}

	// Apply max siblings limit
	if ctxInfo != nil && ctxInfo.Siblings != nil && maxSiblings > 0 && len(ctxInfo.Siblings) > maxSiblings {
		ctxInfo.Siblings = ctxInfo.Siblings[:maxSiblings]
	}

	// If parent was disabled via option or config, remove it from result
	if !includeParent && ctxInfo != nil {
		ctxInfo.Parent = nil
	}

	return ctxInfo, nil
}

// BuildHierarchyMetadata creates hierarchy metadata for storage.
// This is called when saving task work to persist hierarchical context.
func BuildHierarchyMetadata(workUnit *workunit.WorkUnit, hierarchy *HierarchicalContext) *storage.HierarchyInfo {
	if hierarchy == nil {
		return nil
	}

	info := &storage.HierarchyInfo{}

	if hierarchy.Parent != nil {
		info.ParentID = hierarchy.Parent.ID
		info.ParentTitle = hierarchy.Parent.Title
	}

	if len(hierarchy.Siblings) > 0 {
		info.SiblingIDs = make([]string, len(hierarchy.Siblings))
		for i, s := range hierarchy.Siblings {
			info.SiblingIDs[i] = s.ID
		}
	}

	return info
}

// isSubtask checks if a work unit is a subtask by examining its metadata.
func isSubtask(wu *workunit.WorkUnit) bool {
	if wu == nil || wu.Metadata == nil {
		return false
	}

	// Check for explicit is_subtask flag
	if isSubtask, ok := wu.Metadata["is_subtask"].(bool); ok && isSubtask {
		return true
	}

	// Check for parent_id presence
	if _, ok := wu.Metadata["parent_id"].(string); ok {
		return true
	}

	// Heuristic: task items in markdown-based providers (GitHub, GitLab, Bitbucket)
	// These have synthetic IDs with "-task-" or ":task-" pattern
	if strings.Contains(wu.ID, "-task-") || strings.Contains(wu.ID, ":task-") {
		return true
	}

	return false
}

// filterSelf removes the current work unit from a list of siblings.
func filterSelf(siblings []*workunit.WorkUnit, selfID string) []*workunit.WorkUnit {
	result := make([]*workunit.WorkUnit, 0, len(siblings))
	for _, s := range siblings {
		if s.ID != selfID {
			result = append(result, s)
		}
	}

	return result
}
