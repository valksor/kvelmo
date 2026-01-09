package wrike

import (
	"context"
	"fmt"
)

// AddLabels adds tags (labels) to a task.
//
// Note: This uses optimistic concurrency - if multiple processes modify labels
// concurrently, one may overwrite the other's changes. Wrike's API does not
// support atomic add/remove operations.
func (p *Provider) AddLabels(ctx context.Context, workUnitID string, labels []string) error {
	// Early exit if no labels to add
	if len(labels) == 0 {
		return nil
	}

	// Fetch current task to get existing tags
	task, err := p.client.GetTask(ctx, workUnitID)
	if err != nil {
		return fmt.Errorf("get task: %w", err)
	}

	// Merge existing tags with new tags (avoid duplicates)
	existingTags := make(map[string]bool)
	for _, tag := range task.Tags {
		existingTags[tag] = true
	}

	// Track if any new labels were actually added
	originalLen := len(task.Tags)
	for _, label := range labels {
		if !existingTags[label] {
			task.Tags = append(task.Tags, label)
			existingTags[label] = true // Prevent duplicates within the input
		}
	}

	// Skip API call if no new labels were added
	if len(task.Tags) == originalLen {
		return nil
	}

	// Update the task with the merged tags
	if err := p.client.UpdateTaskTags(ctx, workUnitID, task.Tags); err != nil {
		return fmt.Errorf("update task tags: %w", err)
	}

	return nil
}

// RemoveLabels removes tags (labels) from a task.
//
// Note: This uses optimistic concurrency - if multiple processes modify labels
// concurrently, one may overwrite the other's changes. Wrike's API does not
// support atomic add/remove operations.
func (p *Provider) RemoveLabels(ctx context.Context, workUnitID string, labels []string) error {
	// Early exit if no labels to remove
	if len(labels) == 0 {
		return nil
	}

	// Fetch current task to get existing tags
	task, err := p.client.GetTask(ctx, workUnitID)
	if err != nil {
		return fmt.Errorf("get task: %w", err)
	}

	// Create a set of labels to remove
	labelsToRemove := make(map[string]bool)
	for _, label := range labels {
		labelsToRemove[label] = true
	}

	// Filter out the labels to remove
	var newTags []string
	for _, tag := range task.Tags {
		if !labelsToRemove[tag] {
			newTags = append(newTags, tag)
		}
	}

	// Skip API call if no labels were actually removed
	if len(newTags) == len(task.Tags) {
		return nil
	}

	// Update the task with the filtered tags
	if err := p.client.UpdateTaskTags(ctx, workUnitID, newTags); err != nil {
		return fmt.Errorf("update task tags: %w", err)
	}

	return nil
}
