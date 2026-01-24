package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// TaskQueueVersion is the current version of the queue format.
	TaskQueueVersion = "1"
	// QueuesDir is the subdirectory for queue storage.
	QueuesDir = "queues"
)

// TaskQueue represents a collection of tasks for a project planning workflow.
// Tasks are stored locally for review/editing before submission to external providers.
type TaskQueue struct {
	Version   string        `yaml:"version"`
	ID        string        `yaml:"id"`
	Title     string        `yaml:"title,omitempty"`
	Source    string        `yaml:"source,omitempty"`    // Original source reference
	Status    QueueStatus   `yaml:"status"`              // draft, ready, submitted
	Tasks     []*QueuedTask `yaml:"tasks"`               // Ordered list of tasks
	Questions []string      `yaml:"questions,omitempty"` // Unresolved questions from AI
	Blockers  []string      `yaml:"blockers,omitempty"`  // Identified blockers
	CreatedAt time.Time     `yaml:"created_at"`
	UpdatedAt time.Time     `yaml:"updated_at"`

	mu   sync.RWMutex `yaml:"-"`
	path string       `yaml:"-"` // path to queue file
}

// QueueStatus represents the state of a task queue.
type QueueStatus string

const (
	QueueStatusDraft     QueueStatus = "draft"     // Being reviewed/edited
	QueueStatusReady     QueueStatus = "ready"     // Ready for submission
	QueueStatusSubmitted QueueStatus = "submitted" // Submitted to provider
)

// QueuedTask represents a single task within a queue.
type QueuedTask struct {
	ID          string     `yaml:"id"`                     // Local ID (task-1, task-2, etc.)
	Title       string     `yaml:"title"`                  // Task title
	Description string     `yaml:"description,omitempty"`  // Detailed description
	Status      TaskStatus `yaml:"status"`                 // pending, ready, blocked, submitted
	Priority    int        `yaml:"priority"`               // 1 = highest priority
	DependsOn   []string   `yaml:"depends_on,omitempty"`   // Task IDs this depends on
	Blocks      []string   `yaml:"blocks,omitempty"`       // Task IDs this blocks (computed)
	Labels      []string   `yaml:"labels,omitempty"`       // Labels/tags
	Assignee    string     `yaml:"assignee,omitempty"`     // Assignee identifier
	ExternalID  string     `yaml:"external_id,omitempty"`  // Provider ID after submission
	ExternalURL string     `yaml:"external_url,omitempty"` // Provider URL after submission
}

// TaskStatus represents the state of a single task.
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"   // Not yet ready
	TaskStatusReady     TaskStatus = "ready"     // Ready to start (no blockers)
	TaskStatusBlocked   TaskStatus = "blocked"   // Waiting on dependencies
	TaskStatusSubmitted TaskStatus = "submitted" // Submitted to provider
)

// NewTaskQueue creates a new empty task queue.
func NewTaskQueue(id, title, source string) *TaskQueue {
	now := time.Now()

	return &TaskQueue{
		Version:   TaskQueueVersion,
		ID:        id,
		Title:     title,
		Source:    source,
		Status:    QueueStatusDraft,
		Tasks:     make([]*QueuedTask, 0),
		Questions: make([]string, 0),
		Blockers:  make([]string, 0),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// LoadTaskQueue loads a task queue from disk.
func LoadTaskQueue(ws *Workspace, queueID string) (*TaskQueue, error) {
	path := ws.QueuePath(queueID)
	if path == "" {
		return nil, errors.New("could not determine queue path")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("queue not found: %s", queueID)
		}

		return nil, fmt.Errorf("read queue file: %w", err)
	}

	var queue TaskQueue
	if err := yaml.Unmarshal(data, &queue); err != nil {
		return nil, fmt.Errorf("parse queue: %w", err)
	}

	queue.path = path
	if queue.Tasks == nil {
		queue.Tasks = make([]*QueuedTask, 0)
	}
	if queue.Questions == nil {
		queue.Questions = make([]string, 0)
	}
	if queue.Blockers == nil {
		queue.Blockers = make([]string, 0)
	}

	return &queue, nil
}

// Save writes the queue to disk using atomic write pattern.
func (q *TaskQueue) Save() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.path == "" {
		return errors.New("queue path not set")
	}

	q.UpdatedAt = time.Now()

	// Ensure directory exists
	dir := filepath.Dir(q.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create queue directory: %w", err)
	}

	data, err := yaml.Marshal(q)
	if err != nil {
		return fmt.Errorf("marshal queue: %w", err)
	}

	// Atomic write: temp file then rename
	tmpPath := q.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("write queue: %w", err)
	}

	if err := os.Rename(tmpPath, q.path); err != nil {
		_ = os.Remove(tmpPath)

		return fmt.Errorf("save queue: %w", err)
	}

	return nil
}

// AddTask adds a task to the queue.
func (q *TaskQueue) AddTask(task *QueuedTask) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.Tasks = append(q.Tasks, task)
	q.UpdatedAt = time.Now()
}

// GetTask retrieves a task by ID.
func (q *TaskQueue) GetTask(taskID string) *QueuedTask {
	q.mu.RLock()
	defer q.mu.RUnlock()

	for _, task := range q.Tasks {
		if task.ID == taskID {
			return task
		}
	}

	return nil
}

// UpdateTask updates a task in the queue.
func (q *TaskQueue) UpdateTask(taskID string, updater func(*QueuedTask)) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	for _, task := range q.Tasks {
		if task.ID == taskID {
			updater(task)
			q.UpdatedAt = time.Now()

			return nil
		}
	}

	return fmt.Errorf("task not found: %s", taskID)
}

// RemoveTask removes a task from the queue.
func (q *TaskQueue) RemoveTask(taskID string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	for i, task := range q.Tasks {
		if task.ID == taskID {
			q.Tasks = append(q.Tasks[:i], q.Tasks[i+1:]...)
			q.UpdatedAt = time.Now()

			return true
		}
	}

	return false
}

// ReorderTask moves a task to a new position.
func (q *TaskQueue) ReorderTask(taskID string, newIndex int) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	taskIndex := -1
	for i, task := range q.Tasks {
		if task.ID == taskID {
			taskIndex = i

			break
		}
	}

	if taskIndex == -1 {
		return fmt.Errorf("task not found: %s", taskID)
	}

	if newIndex < 0 || newIndex >= len(q.Tasks) {
		return fmt.Errorf("invalid index: %d", newIndex)
	}

	// Remove task from current position
	task := q.Tasks[taskIndex]
	q.Tasks = append(q.Tasks[:taskIndex], q.Tasks[taskIndex+1:]...)

	// Insert at new position
	q.Tasks = append(q.Tasks[:newIndex], append([]*QueuedTask{task}, q.Tasks[newIndex:]...)...)
	q.UpdatedAt = time.Now()

	return nil
}

// ComputeBlocksRelations updates the Blocks field for all tasks based on DependsOn.
func (q *TaskQueue) ComputeBlocksRelations() {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Clear existing Blocks
	for _, task := range q.Tasks {
		task.Blocks = nil
	}

	// Build reverse mapping
	for _, task := range q.Tasks {
		for _, depID := range task.DependsOn {
			for _, depTask := range q.Tasks {
				if depTask.ID == depID {
					depTask.Blocks = append(depTask.Blocks, task.ID)

					break
				}
			}
		}
	}
}

// ComputeTaskStatuses updates task statuses based on dependencies.
// Tasks with unsubmitted dependencies are marked as blocked.
func (q *TaskQueue) ComputeTaskStatuses() {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Build a map of submitted tasks
	submitted := make(map[string]bool)
	for _, task := range q.Tasks {
		if task.Status == TaskStatusSubmitted {
			submitted[task.ID] = true
		}
	}

	// Update statuses
	for _, task := range q.Tasks {
		if task.Status == TaskStatusSubmitted {
			continue // Don't change submitted tasks
		}

		blocked := false
		for _, depID := range task.DependsOn {
			if !submitted[depID] {
				blocked = true

				break
			}
		}

		if blocked {
			task.Status = TaskStatusBlocked
		} else if task.Status == TaskStatusBlocked {
			task.Status = TaskStatusReady
		}
	}
}

// GetReadyTasks returns tasks that are ready to start (not blocked, not submitted).
func (q *TaskQueue) GetReadyTasks() []*QueuedTask {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var ready []*QueuedTask
	for _, task := range q.Tasks {
		if task.Status == TaskStatusReady {
			ready = append(ready, task)
		}
	}

	return ready
}

// GetBlockedTasks returns tasks that are blocked by dependencies.
func (q *TaskQueue) GetBlockedTasks() []*QueuedTask {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var blocked []*QueuedTask
	for _, task := range q.Tasks {
		if task.Status == TaskStatusBlocked {
			blocked = append(blocked, task)
		}
	}

	return blocked
}

// NextTaskID generates the next task ID (task-1, task-2, etc.).
func (q *TaskQueue) NextTaskID() string {
	q.mu.RLock()
	defer q.mu.RUnlock()

	maxNum := 0
	for _, task := range q.Tasks {
		var num int
		if _, err := fmt.Sscanf(task.ID, "task-%d", &num); err == nil {
			if num > maxNum {
				maxNum = num
			}
		}
	}

	return fmt.Sprintf("task-%d", maxNum+1)
}

// TaskCount returns the number of tasks in the queue.
func (q *TaskQueue) TaskCount() int {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return len(q.Tasks)
}

// QueuePath returns the path to a queue file within the workspace.
func (ws *Workspace) QueuePath(queueID string) string {
	return filepath.Join(ws.workspaceRoot, QueuesDir, queueID, "queue.yaml")
}

// SaveTaskQueue saves a task queue to the workspace.
func (ws *Workspace) SaveTaskQueue(queue *TaskQueue) error {
	if queue.path == "" {
		queue.path = ws.QueuePath(queue.ID)
	}

	return queue.Save()
}

// ListQueues returns all queue IDs in the workspace.
func (ws *Workspace) ListQueues() ([]string, error) {
	queuesDir := filepath.Join(ws.workspaceRoot, QueuesDir)

	entries, err := os.ReadDir(queuesDir)
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read queues directory: %w", err)
	}

	var queueIDs []string
	for _, entry := range entries {
		if entry.IsDir() {
			queuePath := filepath.Join(queuesDir, entry.Name(), "queue.yaml")
			if _, err := os.Stat(queuePath); err == nil {
				queueIDs = append(queueIDs, entry.Name())
			}
		}
	}

	// Sort by name
	sort.Strings(queueIDs)

	return queueIDs, nil
}

// DeleteQueue removes a queue and its directory.
func (ws *Workspace) DeleteQueue(queueID string) error {
	queueDir := filepath.Join(ws.workspaceRoot, QueuesDir, queueID)

	return os.RemoveAll(queueDir)
}

// QueueExists checks if a queue exists.
func (ws *Workspace) QueueExists(queueID string) bool {
	path := ws.QueuePath(queueID)
	_, err := os.Stat(path)

	return err == nil
}
