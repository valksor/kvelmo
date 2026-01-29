package stack

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// StacksDir is the subdirectory for stack storage within workspace.
	StacksDir = "stacks"
	// StacksFile is the name of the stacks index file.
	StacksFile = "stacks.yaml"
)

// Storage handles persistence of stacks to disk.
type Storage struct {
	path   string       // Full path to stacks.yaml
	stacks []*Stack     // All stacks
	mu     sync.RWMutex // Protects concurrent access
}

// stacksData is the on-disk format for stacks.
type stacksData struct {
	Version string   `yaml:"version"`
	Stacks  []*Stack `yaml:"stacks"`
}

// NewStorage creates a new stack storage for the given workspace root.
func NewStorage(workspaceRoot string) *Storage {
	return &Storage{
		path:   filepath.Join(workspaceRoot, StacksDir, StacksFile),
		stacks: make([]*Stack, 0),
	}
}

// Load reads stacks from disk. Creates empty storage if file doesn't exist.
func (s *Storage) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		s.stacks = make([]*Stack, 0)

		return nil
	}
	if err != nil {
		return fmt.Errorf("read stacks file: %w", err)
	}

	var sd stacksData
	if err := yaml.Unmarshal(data, &sd); err != nil {
		return fmt.Errorf("parse stacks: %w", err)
	}

	s.stacks = sd.Stacks
	if s.stacks == nil {
		s.stacks = make([]*Stack, 0)
	}

	return nil
}

// Save writes stacks to disk using atomic write pattern.
func (s *Storage) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Ensure directory exists
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create stacks directory: %w", err)
	}

	sd := stacksData{
		Version: "1",
		Stacks:  s.stacks,
	}

	data, err := yaml.Marshal(&sd)
	if err != nil {
		return fmt.Errorf("marshal stacks: %w", err)
	}

	// Atomic write: temp file then rename
	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("write stacks: %w", err)
	}

	if err := os.Rename(tmpPath, s.path); err != nil {
		_ = os.Remove(tmpPath)

		return fmt.Errorf("save stacks: %w", err)
	}

	return nil
}

// GetStack returns the stack with the given ID, or nil if not found.
func (s *Storage) GetStack(stackID string) *Stack {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, stack := range s.stacks {
		if stack.ID == stackID {
			return stack
		}
	}

	return nil
}

// GetStackByTask returns the stack containing the given task ID.
func (s *Storage) GetStackByTask(taskID string) *Stack {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, stack := range s.stacks {
		for _, task := range stack.Tasks {
			if task.ID == taskID {
				return stack
			}
		}
	}

	return nil
}

// AddStack adds a new stack to storage.
func (s *Storage) AddStack(stack *Stack) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for duplicate ID
	for _, existing := range s.stacks {
		if existing.ID == stack.ID {
			return fmt.Errorf("stack already exists: %s", stack.ID)
		}
	}

	s.stacks = append(s.stacks, stack)

	return nil
}

// RemoveStack removes a stack by ID.
func (s *Storage) RemoveStack(stackID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, stack := range s.stacks {
		if stack.ID == stackID {
			s.stacks = append(s.stacks[:i], s.stacks[i+1:]...)

			return true
		}
	}

	return false
}

// ListStacks returns all stacks.
func (s *Storage) ListStacks() []*Stack {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make([]*Stack, len(s.stacks))
	copy(result, s.stacks)

	return result
}

// UpdateStack updates a stack in storage.
func (s *Storage) UpdateStack(stackID string, updater func(*Stack)) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, stack := range s.stacks {
		if stack.ID == stackID {
			updater(stack)
			stack.UpdatedAt = time.Now()

			return nil
		}
	}

	return errors.New("stack not found")
}

// StackCount returns the number of stacks.
func (s *Storage) StackCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.stacks)
}
