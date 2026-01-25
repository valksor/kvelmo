package cost

import (
	"testing"
)

func TestASCIIBarChart(t *testing.T) {
	tests := []struct {
		name         string
		bars         []Bar
		opts         ChartOptions
		wantContains []string
	}{
		{
			name: "simple bar chart",
			bars: []Bar{
				{Label: "Task A", Value: 50, MaxValue: 100},
				{Label: "Task B", Value: 75, MaxValue: 100},
			},
			opts: ChartOptions{
				Title: "Test Chart",
				Width: 40,
			},
			wantContains: []string{"Test Chart", "Task A", "Task B"},
		},
		{
			name: "with values",
			bars: []Bar{
				{Label: "Item", Value: 1000},
			},
			opts: ChartOptions{
				Width:      20,
				ShowValues: true,
			},
			wantContains: []string{"Item", "1,000"},
		},
		{
			name: "long label truncation",
			bars: []Bar{
				{Label: "This is a very long label that should be truncated", Value: 50},
			},
			opts: ChartOptions{
				Width: 30,
			},
			wantContains: []string{"..."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ASCIIBarChart(tt.bars, tt.opts)
			if got == "" {
				t.Error("ASCIIBarChart returned empty string")
			}
			for _, want := range tt.wantContains {
				if !contains(got, want) {
					t.Errorf("ASCIIBarChart() should contain %q", want)
				}
			}
		})
	}
}

func TestASCIILineChart(t *testing.T) {
	tests := []struct {
		name       string
		dataPoints []struct {
			Label string
			Value int
		}
		opts           ChartOptions
		shouldHaveData bool
	}{
		{
			name: "simple line chart",
			dataPoints: []struct {
				Label string
				Value int
			}{
				{"Mon", 10},
				{"Tue", 20},
				{"Wed", 15},
			},
			opts: ChartOptions{
				Title: "Weekly Trend",
			},
			shouldHaveData: true,
		},
		{
			name: "empty data",
			dataPoints: []struct {
				Label string
				Value int
			}{},
			opts:           ChartOptions{},
			shouldHaveData: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ASCIILineChart(tt.dataPoints, tt.opts)
			if got == "" {
				t.Error("ASCIILineChart returned empty string")
			}
			if tt.shouldHaveData {
				// Just check that the chart has some content
				if len(got) < 10 {
					t.Error("ASCIILineChart output seems too short")
				}
			} else {
				if !contains(got, "(no data)") {
					t.Error("ASCIILineChart() should contain '(no data)' for empty input")
				}
			}
		})
	}
}

func TestASCIIPieChart(t *testing.T) {
	tests := []struct {
		name   string
		slices []struct {
			Label   string
			Value   int
			Percent float64
		}
		opts         ChartOptions
		wantContains []string
	}{
		{
			name: "simple pie chart",
			slices: []struct {
				Label   string
				Value   int
				Percent float64
			}{
				{"Planning", 30, 30.0},
				{"Implementing", 50, 50.0},
				{"Review", 20, 20.0},
			},
			opts: ChartOptions{
				Title: "Time Distribution",
			},
			wantContains: []string{"Time Distribution", "Planning", "Implementing", "Review"},
		},
		{
			name: "with percentages",
			slices: []struct {
				Label   string
				Value   int
				Percent float64
			}{
				{"Step A", 25, 25.0},
				{"Step B", 75, 75.0},
			},
			opts:         ChartOptions{},
			wantContains: []string{"25.0%"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ASCIIPieChart(tt.slices, tt.opts)
			if got == "" {
				t.Error("ASCIIPieChart returned empty string")
			}
			for _, want := range tt.wantContains {
				if !contains(got, want) {
					t.Errorf("ASCIIPieChart() should contain %q", want)
				}
			}
		})
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "0"},
		{1, "1"},
		{10, "10"},
		{100, "100"},
		{1000, "1,000"},
		{10000, "10,000"},
		{100000, "100,000"},
		{1000000, "1,000,000"},
		{1234567, "1,234,567"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := formatNumber(tt.input); got != tt.want {
				t.Errorf("formatNumber(%d) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestAbs(t *testing.T) {
	tests := []struct {
		input int
		want  int
	}{
		{0, 0},
		{1, 1},
		{-1, 1},
		{100, 100},
		{-100, 100},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			if got := abs(tt.input); got != tt.want {
				t.Errorf("abs(%d) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestVerticalBarChart(t *testing.T) {
	bars := []Bar{
		{Label: "A", Value: 10, MaxValue: 20},
		{Label: "B", Value: 15, MaxValue: 20},
	}
	opts := ChartOptions{
		Title:    "Vertical",
		Vertical: true,
	}

	got := ASCIIBarChart(bars, opts)
	if !contains(got, "Vertical") {
		t.Error("Vertical chart should contain title")
	}
	if !contains(got, "A") {
		t.Error("Vertical chart should contain label A")
	}
	if !contains(got, "B") {
		t.Error("Vertical chart should contain label B")
	}
}

// Helper function.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsMiddle(s, substr))))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}
