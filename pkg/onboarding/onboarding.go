// Package onboarding provides a guided step tracker for new users.
// It persists completion state in a JSON file.
package onboarding

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/valksor/kvelmo/pkg/paths"
)

// Step represents a named onboarding milestone.
type Step string

const (
	StepInstallCheck   Step = "install_check"
	StepFirstProject   Step = "first_project"
	StepFirstTask      Step = "first_task"
	StepFirstPlan      Step = "first_plan"
	StepFirstImplement Step = "first_implement"
	StepFirstSubmit    Step = "first_submit"
)

// AllSteps is the ordered list of all onboarding steps.
var AllSteps = []Step{
	StepInstallCheck,
	StepFirstProject,
	StepFirstTask,
	StepFirstPlan,
	StepFirstImplement,
	StepFirstSubmit,
}

// Tracker manages onboarding step completion in a JSON file.
type Tracker struct {
	path string
}

// New creates a Tracker that persists state at path.
// If path is empty, defaults to <BaseDir>/onboarding.json.
func New(path string) *Tracker {
	if path == "" {
		path = filepath.Join(paths.BaseDir(), "onboarding.json")
	}

	return &Tracker{path: path}
}

// Status loads and returns the step completion map.
func (t *Tracker) Status() (map[Step]bool, error) {
	status, err := t.load()
	if err != nil {
		return nil, err
	}

	return status, nil
}

// Complete marks a step as completed and saves the state.
func (t *Tracker) Complete(step Step) error {
	status, err := t.load()
	if err != nil {
		return err
	}

	status[step] = true

	return t.save(status)
}

// IsComplete returns true if all steps have been completed.
func (t *Tracker) IsComplete() (bool, error) {
	status, err := t.load()
	if err != nil {
		return false, err
	}

	for _, step := range AllSteps {
		if !status[step] {
			return false, nil
		}
	}

	return true, nil
}

// Reset deletes the onboarding state file.
func (t *Tracker) Reset() error {
	err := os.Remove(t.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	return err
}

func (t *Tracker) load() (map[Step]bool, error) {
	data, err := os.ReadFile(t.path)
	if errors.Is(err, os.ErrNotExist) {
		return make(map[Step]bool), nil
	}

	if err != nil {
		return nil, fmt.Errorf("read onboarding state: %w", err)
	}

	var status map[Step]bool
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, fmt.Errorf("parse onboarding state: %w", err)
	}

	return status, nil
}

func (t *Tracker) save(status map[Step]bool) error {
	if err := os.MkdirAll(filepath.Dir(t.path), 0o755); err != nil {
		return fmt.Errorf("ensure onboarding dir: %w", err)
	}

	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal onboarding state: %w", err)
	}

	if err := os.WriteFile(t.path, data, 0o644); err != nil {
		return fmt.Errorf("write onboarding state: %w", err)
	}

	return nil
}
