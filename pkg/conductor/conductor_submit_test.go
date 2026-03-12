package conductor

import (
	"context"
	"strings"
	"testing"
)

// ─── Submit ──────────────────────────────────────────────────────────────────

func TestConductorSubmit_NoTask(t *testing.T) {
	c, _ := New()
	err := c.Submit(context.Background(), false)
	if err == nil {
		t.Error("Submit() with no task should return error")
	}
	if !strings.Contains(err.Error(), "no task loaded") {
		t.Errorf("Submit() error = %q, want 'no task loaded'", err.Error())
	}
}

func TestConductorSubmit_WrongState(t *testing.T) {
	c, _ := New()
	c.ForceWorkUnit(&WorkUnit{
		ID:    "s1",
		Title: "T",
		Source: &Source{
			Provider:  "github",
			Reference: "owner/repo#1",
		},
	})
	// Set cached quality gate pass to avoid synchronous quality gate run
	passed := true
	c.workUnit.QualityGatePassed = &passed
	// StateNone does not allow EventSubmit
	err := c.Submit(context.Background(), false)
	if err == nil {
		t.Error("Submit() from wrong state should return error")
	}
	if !strings.Contains(err.Error(), "cannot submit") {
		t.Errorf("Submit() error = %q, want 'cannot submit'", err.Error())
	}
}

func TestConductorSubmit_QualityGateCachedFail(t *testing.T) {
	c, _ := New()
	c.ForceWorkUnit(&WorkUnit{ID: "s2", Title: "T"})
	c.machine.ForceState(StateReviewing) // Submit is allowed from reviewing
	passed := false
	c.workUnit.QualityGatePassed = &passed
	c.workUnit.QualityGateError = "lint failed"
	err := c.Submit(context.Background(), false)
	if err == nil {
		t.Error("Submit() with failed quality gate should return error")
	}
	if !strings.Contains(err.Error(), "quality gate failed") {
		t.Errorf("Submit() error = %q, want 'quality gate failed'", err.Error())
	}
}
