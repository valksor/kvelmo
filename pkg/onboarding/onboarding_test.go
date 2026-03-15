package onboarding

import (
	"path/filepath"
	"testing"
)

func TestTracker_CompleteAndStatus(t *testing.T) {
	path := filepath.Join(t.TempDir(), "onboarding.json")
	tracker := New(path)

	// Initially no steps completed.
	status, err := tracker.Status()
	if err != nil {
		t.Fatalf("status: %v", err)
	}

	if len(status) != 0 {
		t.Fatalf("expected empty status, got %d entries", len(status))
	}

	// Complete a step.
	if err := tracker.Complete(StepInstallCheck); err != nil {
		t.Fatalf("complete: %v", err)
	}

	status, err = tracker.Status()
	if err != nil {
		t.Fatalf("status after complete: %v", err)
	}

	if !status[StepInstallCheck] {
		t.Error("expected StepInstallCheck to be complete")
	}

	if status[StepFirstProject] {
		t.Error("expected StepFirstProject to be incomplete")
	}
}

func TestTracker_IsComplete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "onboarding.json")
	tracker := New(path)

	// Not complete with no steps done.
	complete, err := tracker.IsComplete()
	if err != nil {
		t.Fatalf("is complete: %v", err)
	}

	if complete {
		t.Error("expected incomplete with no steps done")
	}

	// Complete all steps.
	for _, step := range AllSteps {
		if err := tracker.Complete(step); err != nil {
			t.Fatalf("complete %s: %v", step, err)
		}
	}

	complete, err = tracker.IsComplete()
	if err != nil {
		t.Fatalf("is complete after all: %v", err)
	}

	if !complete {
		t.Error("expected complete after all steps done")
	}
}

func TestTracker_Reset(t *testing.T) {
	path := filepath.Join(t.TempDir(), "onboarding.json")
	tracker := New(path)

	// Complete a step, then reset.
	if err := tracker.Complete(StepFirstTask); err != nil {
		t.Fatalf("complete: %v", err)
	}

	if err := tracker.Reset(); err != nil {
		t.Fatalf("reset: %v", err)
	}

	status, err := tracker.Status()
	if err != nil {
		t.Fatalf("status after reset: %v", err)
	}

	if len(status) != 0 {
		t.Fatalf("expected empty status after reset, got %d entries", len(status))
	}

	// Reset on non-existent file should not error.
	if err := tracker.Reset(); err != nil {
		t.Fatalf("reset non-existent: %v", err)
	}
}
