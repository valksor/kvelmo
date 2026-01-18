package ml

import (
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/workflow"
)

// FeatureExtractor extracts features from tasks for ML training.
type FeatureExtractor struct{}

// NewFeatureExtractor creates a new feature extractor.
func NewFeatureExtractor() *FeatureExtractor {
	return &FeatureExtractor{}
}

// Extract extracts features from a task and state.
func (f *FeatureExtractor) Extract(task *storage.TaskWork, state workflow.State) (map[string]interface{}, error) {
	features := make(map[string]interface{})

	// Task features
	features["title_length"] = len(task.Metadata.Title)
	features["title_word_count"] = len(strings.Fields(task.Metadata.Title))
	features["has_external_key"] = task.Metadata.ExternalKey != ""
	features["task_type"] = f.classifyTaskType(task)
	// Handle zero time to avoid panic
	if !task.Metadata.CreatedAt.IsZero() {
		features["is_new_task"] = time.Since(task.Metadata.CreatedAt) < 24*time.Hour
	} else {
		features["is_new_task"] = false
	}

	// State features
	features["current_state"] = string(state)
	features["state_is_terminal"] = state == workflow.StateDone || state == workflow.StateFailed

	// Time features
	now := time.Now()
	features["hour_of_day"] = now.Hour()
	features["day_of_week"] = int(now.Weekday())
	features["is_weekend"] = now.Weekday() == time.Saturday || now.Weekday() == time.Sunday

	// Agent features
	if task.Agent.Name != "" {
		features["agent_name"] = task.Agent.Name
	}

	// Git features
	if task.Git.Branch != "" {
		features["has_branch"] = true
		features["branch_created"] = !task.Git.CreatedAt.IsZero()
	}

	// Cost features
	features["has_cost_data"] = task.Costs.TotalCostUSD > 0

	return features, nil
}

// classifyTaskType classifies a task by type.
func (f *FeatureExtractor) classifyTaskType(task *storage.TaskWork) string {
	title := strings.ToLower(task.Metadata.Title)

	switch {
	case strings.Contains(title, "fix") || strings.Contains(title, "bug"):
		return "fix"
	case strings.Contains(title, "feature") || strings.Contains(title, "add"):
		return "feature"
	case strings.Contains(title, "refactor") || strings.Contains(title, "clean"):
		return "refactor"
	case strings.Contains(title, "test") || strings.Contains(title, "spec"):
		return "test"
	case strings.Contains(title, "doc"):
		return "documentation"
	default:
		return "other"
	}
}

// ExtractFeaturesForMultiple extracts features for multiple tasks.
func (f *FeatureExtractor) ExtractFeaturesForMultiple(tasks []*storage.TaskWork, states []workflow.State) ([]map[string]interface{}, error) {
	features := make([]map[string]interface{}, len(tasks))

	for i, task := range tasks {
		feature, err := f.Extract(task, states[i])
		if err != nil {
			return nil, err
		}
		features[i] = feature
	}

	return features, nil
}

// FeatureNames returns the list of feature names.
func (f *FeatureExtractor) FeatureNames() []string {
	return []string{
		"title_length",
		"title_word_count",
		"has_external_key",
		"task_type",
		"is_new_task",
		"current_state",
		"state_is_terminal",
		"hour_of_day",
		"day_of_week",
		"is_weekend",
		"agent_name",
		"has_branch",
		"branch_created",
		"has_cost_data",
	}
}
