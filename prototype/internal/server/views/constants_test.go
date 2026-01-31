package views

import "testing"

func TestGetStateDisplay(t *testing.T) {
	tests := []struct {
		name  string
		state string
		want  StateDisplayInfo
	}{
		{
			name:  "idle state",
			state: StateIdle,
			want: StateDisplayInfo{
				Icon:     "○",
				Badge:    "Ready",
				Color:    "text-base-content/60",
				BarColor: "bg-base-300",
			},
		},
		{
			name:  "planning state",
			state: StatePlanning,
			want: StateDisplayInfo{
				Icon:     "◐",
				Badge:    "Planning...",
				Color:    "text-info",
				BarColor: "bg-info",
			},
		},
		{
			name:  "implementing state",
			state: StateImplementing,
			want: StateDisplayInfo{
				Icon:     "◑",
				Badge:    "Implementing...",
				Color:    "text-warning",
				BarColor: "bg-warning",
			},
		},
		{
			name:  "reviewing state",
			state: StateReviewing,
			want: StateDisplayInfo{
				Icon:     "◉",
				Badge:    "Reviewing...",
				Color:    "text-primary",
				BarColor: "bg-primary",
			},
		},
		{
			name:  "done state",
			state: StateDone,
			want: StateDisplayInfo{
				Icon:     "●",
				Badge:    "Done",
				Color:    "text-success",
				BarColor: "bg-success",
			},
		},
		{
			name:  "failed state",
			state: StateFailed,
			want: StateDisplayInfo{
				Icon:     "✗",
				Badge:    "Failed",
				Color:    "text-error",
				BarColor: "bg-error",
			},
		},
		{
			name:  "waiting state",
			state: StateWaiting,
			want: StateDisplayInfo{
				Icon:     "?",
				Badge:    "Waiting...",
				Color:    "text-warning",
				BarColor: "bg-warning",
			},
		},
		{
			name:  "paused state",
			state: StatePaused,
			want: StateDisplayInfo{
				Icon:     "⏸",
				Badge:    "Paused",
				Color:    "text-neutral",
				BarColor: "bg-neutral",
			},
		},
		{
			name:  "unknown state returns fallback",
			state: "unknown-state",
			want: StateDisplayInfo{
				Icon:     "?",
				Badge:    "unknown-state",
				Color:    "text-base-content/60",
				BarColor: "bg-base-300",
			},
		},
		{
			name:  "empty state returns fallback",
			state: "",
			want: StateDisplayInfo{
				Icon:     "?",
				Badge:    "",
				Color:    "text-base-content/60",
				BarColor: "bg-base-300",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetStateDisplay(tt.state)
			if got.Icon != tt.want.Icon {
				t.Errorf("GetStateDisplay() Icon = %q, want %q", got.Icon, tt.want.Icon)
			}
			if got.Badge != tt.want.Badge {
				t.Errorf("GetStateDisplay() Badge = %q, want %q", got.Badge, tt.want.Badge)
			}
			if got.Color != tt.want.Color {
				t.Errorf("GetStateDisplay() Color = %q, want %q", got.Color, tt.want.Color)
			}
			if got.BarColor != tt.want.BarColor {
				t.Errorf("GetStateDisplay() BarColor = %q, want %q", got.BarColor, tt.want.BarColor)
			}
		})
	}
}

func TestGetSpecStatusDisplay(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   SpecStatusDisplayInfo
	}{
		{
			name:   "pending status",
			status: SpecStatusPending,
			want: SpecStatusDisplayInfo{
				Icon:  "○",
				Color: "text-base-content/40",
			},
		},
		{
			name:   "active status",
			status: SpecStatusActive,
			want: SpecStatusDisplayInfo{
				Icon:  "◐",
				Color: "text-info",
			},
		},
		{
			name:   "completed status",
			status: SpecStatusCompleted,
			want: SpecStatusDisplayInfo{
				Icon:  "●",
				Color: "text-success",
			},
		},
		{
			name:   "skipped status",
			status: SpecStatusSkipped,
			want: SpecStatusDisplayInfo{
				Icon:  "⊘",
				Color: "text-base-content/40",
			},
		},
		{
			name:   "unknown status returns fallback",
			status: "unknown-status",
			want: SpecStatusDisplayInfo{
				Icon:  "?",
				Color: "text-base-content/40",
			},
		},
		{
			name:   "empty status returns fallback",
			status: "",
			want: SpecStatusDisplayInfo{
				Icon:  "?",
				Color: "text-base-content/40",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetSpecStatusDisplay(tt.status)
			if got.Icon != tt.want.Icon {
				t.Errorf("GetSpecStatusDisplay() Icon = %q, want %q", got.Icon, tt.want.Icon)
			}
			if got.Color != tt.want.Color {
				t.Errorf("GetSpecStatusDisplay() Color = %q, want %q", got.Color, tt.want.Color)
			}
		})
	}
}

func TestLabelColor(t *testing.T) {
	// Test that different labels produce deterministic colors
	labels := []string{"bug", "feature", "enhancement", "docs", "test"}
	colors := make(map[string]bool)

	for _, label := range labels {
		color := LabelColor(label)
		if color == "" {
			t.Errorf("LabelColor(%q) returned empty string", label)
		}
		colors[color] = true
	}

	// We expect at least some variety in colors
	if len(colors) < 2 {
		t.Error("LabelColor() should produce different colors for different labels")
	}

	// Test that the same label always produces the same color
	label1 := "bug"
	color1 := LabelColor(label1)
	color2 := LabelColor(label1)
	if color1 != color2 {
		t.Errorf("LabelColor() should be deterministic for same label: got %q, want %q", color2, color1)
	}

	// Test empty label
	emptyColor := LabelColor("")
	if emptyColor == "" {
		t.Error("LabelColor(\"\") should return a valid color")
	}
}
