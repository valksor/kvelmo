package conductor

import (
	"errors"
	"strings"
	"testing"
)

func TestEnrichError_Nil(t *testing.T) {
	if got := EnrichError(nil, "test"); got != nil {
		t.Errorf("EnrichError(nil) = %v, want nil", got)
	}
}

func TestEnrichError_AlreadyUserError(t *testing.T) {
	ue := &UserError{Message: "already enriched", Code: "test"}
	got := EnrichError(ue, "plan")
	if got != ue {
		t.Error("EnrichError should return existing UserError as-is")
	}
}

func TestEnrichError_KnownPatterns(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		phase    string
		wantMsg  string
		wantCode string
	}{
		{
			name:     "no agents",
			err:      errors.New("no available agents found"),
			phase:    "plan",
			wantMsg:  "No AI agent is connected",
			wantCode: "agent_unavailable",
		},
		{
			name:     "agent connect",
			err:      errors.New("connect failed (both WebSocket and CLI): timeout"),
			phase:    "plan",
			wantMsg:  "Could not connect to AI agent",
			wantCode: "agent_connect",
		},
		{
			name:     "push failure",
			err:      errors.New("push branch main: authentication required"),
			phase:    "submit",
			wantMsg:  "Git push failed",
			wantCode: "git_push",
		},
		{
			name:     "no task",
			err:      errors.New("no task loaded"),
			phase:    "plan",
			wantMsg:  "No task is loaded",
			wantCode: "no_task",
		},
		{
			name:     "quality gate",
			err:      errors.New("quality gate failed: lint errors"),
			phase:    "submit",
			wantMsg:  "Code quality checks did not pass",
			wantCode: "quality_gate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EnrichError(tt.err, tt.phase)
			if got.Message != tt.wantMsg {
				t.Errorf("Message = %q, want %q", got.Message, tt.wantMsg)
			}
			if got.Code != tt.wantCode {
				t.Errorf("Code = %q, want %q", got.Code, tt.wantCode)
			}
			if got.Fix == "" {
				t.Error("Fix should not be empty")
			}
			if !errors.Is(got.Cause, tt.err) {
				t.Error("Cause should be the original error")
			}
		})
	}
}

func TestEnrichError_UnknownError(t *testing.T) {
	err := errors.New("something completely unexpected")
	got := EnrichError(err, "implement")

	if !strings.Contains(got.Message, "implement") {
		t.Errorf("Message should mention phase, got %q", got.Message)
	}
	if got.Code != "unknown" {
		t.Errorf("Code = %q, want unknown", got.Code)
	}
}

func TestUserError_Error(t *testing.T) {
	ue := &UserError{Message: "test message", Cause: errors.New("root cause")}
	if ue.Error() != "test message" {
		t.Errorf("Error() = %q, want %q", ue.Error(), "test message")
	}
}

func TestUserError_Unwrap(t *testing.T) {
	cause := errors.New("root cause")
	ue := &UserError{Message: "wrapped", Cause: cause}
	if !errors.Is(ue, cause) {
		t.Error("errors.Is should find the wrapped cause")
	}
}
