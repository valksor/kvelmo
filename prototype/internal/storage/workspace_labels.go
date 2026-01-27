package storage

import (
	"slices"
)

// AddLabel adds a label to a task. Does nothing if label already exists.
func (w *Workspace) AddLabel(taskID, label string) error {
	work, err := w.LoadWork(taskID)
	if err != nil {
		return err
	}

	// Check for duplicate - don't add if already exists
	if slices.Contains(work.Metadata.Labels, label) {
		return nil
	}

	work.Metadata.Labels = append(work.Metadata.Labels, label)

	return w.SaveWork(work)
}

// RemoveLabel removes a label from a task. No error if label doesn't exist.
func (w *Workspace) RemoveLabel(taskID, label string) error {
	work, err := w.LoadWork(taskID)
	if err != nil {
		return err
	}

	// Filter out the label
	var newLabels []string
	for _, l := range work.Metadata.Labels {
		if l != label {
			newLabels = append(newLabels, l)
		}
	}

	work.Metadata.Labels = newLabels

	return w.SaveWork(work)
}

// SetLabels replaces all labels on a task.
func (w *Workspace) SetLabels(taskID string, labels []string) error {
	work, err := w.LoadWork(taskID)
	if err != nil {
		return err
	}

	work.Metadata.Labels = labels

	return w.SaveWork(work)
}

// GetLabels returns all labels for a task.
func (w *Workspace) GetLabels(taskID string) ([]string, error) {
	work, err := w.LoadWork(taskID)
	if err != nil {
		return nil, err
	}

	return work.Metadata.Labels, nil
}
