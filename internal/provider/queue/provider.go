package queue

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/provider"
	"github.com/valksor/go-mehrhof/internal/storage"
)

const ProviderName = "queue"

type Provider struct {
	workspace *storage.Workspace
}

// Info returns provider metadata.
func Info() provider.ProviderInfo {
	return provider.ProviderInfo{
		Name:        ProviderName,
		Description: "Local project queue tasks",
		Schemes:     []string{"queue"},
		Priority:    5,
		Capabilities: provider.CapabilitySet{
			provider.CapRead:     true,
			provider.CapSnapshot: true,
		},
	}
}

// New creates a queue provider backed by the current workspace.
func New(ctx context.Context, _ provider.Config) (any, error) {
	root, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get cwd: %w", err)
	}

	ws, err := storage.OpenWorkspace(ctx, root, nil)
	if err != nil {
		return nil, fmt.Errorf("open workspace: %w", err)
	}

	return &Provider{workspace: ws}, nil
}

// Match checks if input has the queue: scheme prefix.
func (p *Provider) Match(input string) bool {
	return strings.HasPrefix(input, "queue:")
}

// Parse extracts the queue/task identifier from input.
// Input: "queue:<queue-id>/<task-id>" -> ID: "<queue-id>/<task-id>".
func (p *Provider) Parse(input string) (string, error) {
	identifier := strings.TrimPrefix(input, "queue:")
	if identifier == "" {
		return "", errors.New("queue task identifier after 'queue:' prefix")
	}
	if !strings.Contains(identifier, "/") {
		return "", errors.New("queue task identifier must be in '<queue-id>/<task-id>' format")
	}

	return identifier, nil
}

// Fetch loads a queued task and converts it into a WorkUnit.
func (p *Provider) Fetch(ctx context.Context, id string) (*provider.WorkUnit, error) {
	queueID, taskID, err := splitQueueTaskID(id)
	if err != nil {
		return nil, err
	}

	queue, err := storage.LoadTaskQueue(p.workspace, queueID)
	if err != nil {
		return nil, fmt.Errorf("load queue: %w", err)
	}

	task := queue.GetTask(taskID)
	if task == nil {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	now := time.Now()
	wu := &provider.WorkUnit{
		ID:          task.ID,
		ExternalID:  id,
		Provider:    ProviderName,
		Title:       task.Title,
		Description: task.Description,
		Status:      provider.StatusOpen,
		Priority:    mapPriority(task.Priority),
		Labels:      append([]string(nil), task.Labels...),
		Metadata:    make(map[string]any),
		CreatedAt:   now,
		UpdatedAt:   now,
		Source: provider.SourceInfo{
			Type:      ProviderName,
			Reference: "queue:" + id,
			SyncedAt:  now,
		},
		ExternalKey: task.ID,
		TaskType:    "task",
	}

	return wu, nil
}

// Snapshot returns a simple source snapshot for a queued task.
func (p *Provider) Snapshot(ctx context.Context, id string) (*provider.Snapshot, error) {
	queueID, taskID, err := splitQueueTaskID(id)
	if err != nil {
		return nil, err
	}

	queue, err := storage.LoadTaskQueue(p.workspace, queueID)
	if err != nil {
		return nil, fmt.Errorf("load queue: %w", err)
	}

	task := queue.GetTask(taskID)
	if task == nil {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	content := fmt.Sprintf("# %s\n\n%s\n", task.Title, strings.TrimSpace(task.Description))
	if len(task.Labels) > 0 {
		content += fmt.Sprintf("\nLabels: %s\n", strings.Join(task.Labels, ", "))
	}

	return &provider.Snapshot{
		Type:    ProviderName,
		Ref:     "queue:" + id,
		Content: content,
	}, nil
}

func splitQueueTaskID(id string) (string, string, error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid queue task identifier: %s", id)
	}

	return parts[0], parts[1], nil
}

func mapPriority(priority int) provider.Priority {
	switch {
	case priority < 0:
		return provider.PriorityNormal // Invalid priority, use default
	case priority <= 1:
		return provider.PriorityHigh
	case priority == 2:
		return provider.PriorityNormal
	default: // priority >= 3
		return provider.PriorityLow
	}
}

// Register adds queue provider to registry.
func Register(r *provider.Registry) {
	_ = r.Register(Info(), New)
}

var (
	_ provider.Reader      = (*Provider)(nil)
	_ provider.Identifier  = (*Provider)(nil)
	_ provider.Snapshotter = (*Provider)(nil)
)
