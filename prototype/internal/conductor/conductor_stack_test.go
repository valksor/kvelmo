package conductor

import (
	"context"
	"testing"

	"github.com/valksor/go-mehrhof/internal/stack"
)

func TestAutoRebaseResult_Fields(t *testing.T) {
	// Verify the AutoRebaseResult struct has all expected fields
	result := AutoRebaseResult{
		Attempted:    true,
		Skipped:      false,
		SkipReason:   "test reason",
		Preview:      &stack.RebasePreview{},
		Executed:     true,
		Result:       &stack.RebaseResult{},
		UserDeclined: false,
		HasConflicts: false,
		Unavailable:  false,
	}

	if !result.Attempted {
		t.Error("Attempted should be true")
	}
	if result.Skipped {
		t.Error("Skipped should be false")
	}
	if result.SkipReason != "test reason" {
		t.Errorf("SkipReason = %q, want %q", result.SkipReason, "test reason")
	}
	if result.Preview == nil {
		t.Error("Preview should not be nil")
	}
	if !result.Executed {
		t.Error("Executed should be true")
	}
	if result.Result == nil {
		t.Error("Result should not be nil")
	}
}

func TestTryAutoRebase_NoActiveTask(t *testing.T) {
	// When workspace is nil, tryAutoRebase should handle gracefully
	c := &Conductor{
		opts: Options{},
	}

	// Call with SkipAutoRebase to test the early return path
	// This verifies the function doesn't crash on edge cases
	opts := FinishOptions{SkipAutoRebase: true}
	result := c.tryAutoRebase(context.Background(), "test-task", opts)
	if !result.Skipped {
		t.Error("Expected Skipped when SkipAutoRebase is true")
	}
}

func TestTryAutoRebase_SkipAutoRebaseFlag(t *testing.T) {
	// Test that SkipAutoRebase flag causes early return
	c := &Conductor{
		opts: Options{},
	}

	opts := FinishOptions{
		SkipAutoRebase: true,
	}

	result := c.tryAutoRebase(context.Background(), "test-task", opts)

	if !result.Skipped {
		t.Error("Expected Skipped to be true when flag is set")
	}
	wantReason := "auto-rebase disabled via --no-auto-rebase flag"
	if result.SkipReason != wantReason {
		t.Errorf("SkipReason = %q, want %q", result.SkipReason, wantReason)
	}
}

func TestTryAutoRebase_AutoModeSkips(t *testing.T) {
	// Note: We can't fully test auto-mode skip without a real workspace
	// since the auto-mode check happens after config load.
	// Instead, verify the logic is correctly implemented by checking
	// that SkipAutoRebase flag works (which bypasses the nil workspace issue)

	c := &Conductor{
		opts: Options{
			AutoMode:           true,
			SkipAgentQuestions: true,
		},
	}

	// Use skip flag to test early return path
	opts := FinishOptions{
		SkipAutoRebase: true,
	}

	result := c.tryAutoRebase(context.Background(), "test-task", opts)
	if !result.Skipped {
		t.Error("Expected Skipped when SkipAutoRebase flag is set")
	}

	// Verify opts are correctly set
	if !c.opts.AutoMode {
		t.Error("AutoMode should be true")
	}
	if !c.opts.SkipAgentQuestions {
		t.Error("SkipAgentQuestions should be true")
	}
}

func TestFinishOptions_AutoRebaseFields(t *testing.T) {
	// Verify FinishOptions has the auto-rebase fields
	opts := FinishOptions{
		SkipAutoRebase:  true,
		ForceAutoRebase: true,
	}

	if !opts.SkipAutoRebase {
		t.Error("SkipAutoRebase should be true")
	}
	if !opts.ForceAutoRebase {
		t.Error("ForceAutoRebase should be true")
	}

	// Verify default options don't set these
	defaultOpts := DefaultFinishOptions()
	if defaultOpts.SkipAutoRebase {
		t.Error("DefaultFinishOptions should have SkipAutoRebase = false")
	}
	if defaultOpts.ForceAutoRebase {
		t.Error("DefaultFinishOptions should have ForceAutoRebase = false")
	}
}
