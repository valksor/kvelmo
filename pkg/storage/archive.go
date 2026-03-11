package storage

import (
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/valksor/kvelmo/pkg/meta"
	"gopkg.in/yaml.v3"
)

// ArchivedTask is a lightweight record of a completed task.
type ArchivedTask struct {
	ID          string    `yaml:"id" json:"id"`
	Title       string    `yaml:"title" json:"title"`
	Branch      string    `yaml:"branch,omitempty" json:"branch,omitempty"`
	Source      string    `yaml:"source,omitempty" json:"source,omitempty"`
	FinalState  string    `yaml:"final_state" json:"final_state"` // "finished", "abandoned", etc.
	StartedAt   time.Time `yaml:"started_at" json:"started_at"`
	CompletedAt time.Time `yaml:"completed_at" json:"completed_at"`
}

// ArchiveFile returns the path to the archive index file.
func (s *Store) ArchiveFile() string {
	return filepath.Join(s.projectRoot, meta.OrgDir, "archive.yaml")
}

// ArchiveTask appends a completed task to the archive.
func (s *Store) ArchiveTask(task ArchivedTask) error {
	dir := filepath.Dir(s.ArchiveFile())
	if err := EnsureDir(dir); err != nil {
		return err
	}

	tasks, _ := s.ListArchivedTasks() // ignore error, start fresh if corrupt
	tasks = append(tasks, task)

	data, err := yaml.Marshal(tasks)
	if err != nil {
		return err
	}

	return os.WriteFile(s.ArchiveFile(), data, 0o644)
}

// ListArchivedTasks returns all archived tasks, newest first.
func (s *Store) ListArchivedTasks() ([]ArchivedTask, error) {
	data, err := os.ReadFile(s.ArchiveFile())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, err
	}

	var tasks []ArchivedTask
	if err := yaml.Unmarshal(data, &tasks); err != nil {
		return nil, err
	}

	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].CompletedAt.After(tasks[j].CompletedAt)
	})

	return tasks, nil
}
