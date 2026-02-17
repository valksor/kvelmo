package linear

import (
	"testing"

	"github.com/valksor/go-toolkit/capability"
	"github.com/valksor/go-toolkit/workunit"
)

// ──────────────────────────────────────────────────────────────────────────────
// Provider.Match tests
// ──────────────────────────────────────────────────────────────────────────────

func TestProviderMatch(t *testing.T) {
	p := &Provider{}

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "linear scheme with colon",
			input: "linear:ENG-123",
			want:  true,
		},
		{
			name:  "ln scheme with colon",
			input: "ln:ENG-123",
			want:  true,
		},
		{
			name:  "linear scheme uppercase",
			input: "LINEAR:ENG-123",
			want:  false,
		},
		{
			name:  "no scheme",
			input: "ENG-123",
			want:  false,
		},
		{
			name:  "different scheme",
			input: "github:123",
			want:  false,
		},
		{
			name:  "empty string",
			input: "",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.Match(tt.input)
			if got != tt.want {
				t.Errorf("Provider.Match(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Provider.Parse tests
// ──────────────────────────────────────────────────────────────────────────────

func TestProviderParse(t *testing.T) {
	p := &Provider{}

	tests := []struct {
		name        string
		input       string
		want        string
		errContains string
		wantErr     bool
	}{
		{
			name:  "valid linear scheme reference",
			input: "linear:ENG-123",
			want:  "ENG-123",
		},
		{
			name:  "valid ln scheme reference",
			input: "ln:PROD-456",
			want:  "PROD-456",
		},
		{
			name:  "valid URL reference",
			input: "linear:https://linear.app/team/issue/ENG-123-title",
			want:  "ENG-123",
		},
		{
			name:        "invalid format - no dash",
			input:       "ENG123",
			wantErr:     true,
			errContains: "unrecognized",
		},
		{
			name:        "empty string",
			input:       "",
			wantErr:     true,
			errContains: "empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := p.Parse(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Provider.Parse(%q) expected error, got nil", tt.input)

					return
				}
				if tt.errContains != "" && !containsString(err.Error(), tt.errContains) {
					t.Errorf("Provider.Parse(%q) error = %v, want to contain %q", tt.input, err, tt.errContains)
				}

				return
			}

			if err != nil {
				t.Errorf("Provider.Parse(%q) unexpected error: %v", tt.input, err)

				return
			}

			if got != tt.want {
				t.Errorf("Provider.Parse(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Info tests
// ──────────────────────────────────────────────────────────────────────────────

func TestInfo(t *testing.T) {
	info := Info()

	if info.Name != ProviderName {
		t.Errorf("Info().Name = %q, want %q", info.Name, ProviderName)
	}

	if info.Priority != 20 {
		t.Errorf("Info().Priority = %d, want 20", info.Priority)
	}

	expectedSchemes := []string{"linear", "ln"}
	if len(info.Schemes) != 2 {
		t.Errorf("Info().Schemes length = %d, want 2", len(info.Schemes))
	} else {
		for i, scheme := range info.Schemes {
			if scheme != expectedSchemes[i] {
				t.Errorf("Info().Schemes[%d] = %q, want %q", i, scheme, expectedSchemes[i])
			}
		}
	}

	// Check capabilities
	expectedCaps := capability.CapabilitySet{
		capability.CapRead:           true,
		capability.CapList:           true,
		capability.CapFetchComments:  true,
		capability.CapComment:        true,
		capability.CapUpdateStatus:   true,
		capability.CapManageLabels:   true,
		capability.CapCreateWorkUnit: true,
		capability.CapSnapshot:       true,
	}

	for cap, shouldHave := range expectedCaps {
		if info.Capabilities.Has(cap) != shouldHave {
			t.Errorf("Info().Capabilities.Has(%v) = %v, want %v", cap, info.Capabilities.Has(cap), shouldHave)
		}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// mapLinearStatus tests
// ──────────────────────────────────────────────────────────────────────────────

func TestMapLinearStatus(t *testing.T) {
	tests := []struct {
		name  string
		state *State
		want  workunit.Status
	}{
		{
			name:  "nil state",
			state: nil,
			want:  workunit.StatusOpen,
		},
		{
			name:  "Backlog state",
			state: &State{Name: "Backlog"},
			want:  workunit.StatusOpen,
		},
		{
			name:  "Todo state",
			state: &State{Name: "Todo"},
			want:  workunit.StatusOpen,
		},
		{
			name:  "Unstarted state",
			state: &State{Name: "Unstarted"},
			want:  workunit.StatusOpen,
		},
		{
			name:  "In Progress state",
			state: &State{Name: "In Progress"},
			want:  workunit.StatusInProgress,
		},
		{
			name:  "Started state",
			state: &State{Name: "Started"},
			want:  workunit.StatusInProgress,
		},
		{
			name:  "In Review state",
			state: &State{Name: "In Review"},
			want:  workunit.StatusInProgress,
		},
		{
			name:  "Done state",
			state: &State{Name: "Done"},
			want:  workunit.StatusDone,
		},
		{
			name:  "Completed state",
			state: &State{Name: "Completed"},
			want:  workunit.StatusDone,
		},
		{
			name:  "Canceled state",
			state: &State{Name: "Canceled"},
			want:  workunit.StatusClosed,
		},
		{
			name:  "Cancelled state",
			state: &State{Name: "Cancelled"},
			want:  workunit.StatusClosed,
		},
		{
			name:  "unknown state - defaults to open",
			state: &State{Name: "UnknownState"},
			want:  workunit.StatusOpen,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapLinearStatus(tt.state)
			if got != tt.want {
				t.Errorf("mapLinearStatus(%v) = %v, want %v", tt.state, got, tt.want)
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// mapLinearPriority tests
// ──────────────────────────────────────────────────────────────────────────────

func TestMapLinearPriority(t *testing.T) {
	tests := []struct {
		name     string
		priority int
		want     workunit.Priority
	}{
		{
			name:     "Urgent priority",
			priority: 1,
			want:     workunit.PriorityCritical,
		},
		{
			name:     "High priority",
			priority: 2,
			want:     workunit.PriorityHigh,
		},
		{
			name:     "Medium priority",
			priority: 3,
			want:     workunit.PriorityNormal,
		},
		{
			name:     "Low priority",
			priority: 4,
			want:     workunit.PriorityLow,
		},
		{
			name:     "No priority",
			priority: 0,
			want:     workunit.PriorityNormal,
		},
		{
			name:     "Unknown priority - defaults to normal",
			priority: 99,
			want:     workunit.PriorityNormal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapLinearPriority(tt.priority)
			if got != tt.want {
				t.Errorf("mapLinearPriority(%d) = %v, want %v", tt.priority, got, tt.want)
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// mapProviderStatusToLinearStateName tests
// ──────────────────────────────────────────────────────────────────────────────

func TestMapProviderStatusToLinearStateName(t *testing.T) {
	tests := []struct {
		name   string
		status workunit.Status
		want   string
	}{
		{
			name:   "Open status",
			status: workunit.StatusOpen,
			want:   "Todo",
		},
		{
			name:   "In Progress status",
			status: workunit.StatusInProgress,
			want:   "In Progress",
		},
		{
			name:   "Review status",
			status: workunit.StatusReview,
			want:   "In Review",
		},
		{
			name:   "Done status",
			status: workunit.StatusDone,
			want:   "Done",
		},
		{
			name:   "Closed status",
			status: workunit.StatusClosed,
			want:   "Canceled",
		},
		{
			name:   "Unknown status - defaults to Todo",
			status: workunit.Status("unknown"),
			want:   "Todo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapProviderStatusToLinearStateName(tt.status)
			if got != tt.want {
				t.Errorf("mapProviderStatusToLinearStateName(%v) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// mapProviderPriorityToLinear tests
// ──────────────────────────────────────────────────────────────────────────────

func TestMapProviderPriorityToLinear(t *testing.T) {
	tests := []struct {
		name     string
		priority workunit.Priority
		want     int
	}{
		{
			name:     "Critical priority",
			priority: workunit.PriorityCritical,
			want:     1, // Urgent
		},
		{
			name:     "High priority",
			priority: workunit.PriorityHigh,
			want:     2, // High
		},
		{
			name:     "Normal priority",
			priority: workunit.PriorityNormal,
			want:     3, // Medium
		},
		{
			name:     "Low priority",
			priority: workunit.PriorityLow,
			want:     4, // Low
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapProviderPriorityToLinear(tt.priority)
			if got == nil {
				t.Errorf("mapProviderPriorityToLinear(%v) = nil, want %d", tt.priority, tt.want)

				return
			}
			if *got != tt.want {
				t.Errorf("mapProviderPriorityToLinear(%v) = %d, want %d", tt.priority, *got, tt.want)
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Helper functions
// ──────────────────────────────────────────────────────────────────────────────

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && contains(s, substr)))
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}
