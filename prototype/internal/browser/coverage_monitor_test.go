package browser

import (
	"context"
	"math"
	"testing"

	"github.com/go-rod/rod/lib/proto"
)

// TestCoverageMonitor tests coverage tracking and data processing.
func TestCoverageMonitor(t *testing.T) {
	t.Run("NewCoverageMonitor", func(t *testing.T) {
		tests := []struct {
			name     string
			trackJS  bool
			trackCSS bool
		}{
			{"both enabled", true, true},
			{"JS only", true, false},
			{"CSS only", false, true},
			{"neither", false, false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				mon := NewCoverageMonitor(tt.trackJS, tt.trackCSS)
				if mon.trackJS != tt.trackJS {
					t.Errorf("trackJS = %v, want %v", mon.trackJS, tt.trackJS)
				}
				if mon.trackCSS != tt.trackCSS {
					t.Errorf("trackCSS = %v, want %v", mon.trackCSS, tt.trackCSS)
				}
				if mon.page != nil {
					t.Error("page should be nil before Start()")
				}
			})
		}
	})

	t.Run("CollectWithoutStart", func(t *testing.T) {
		mon := NewCoverageMonitor(true, true)

		_, _, _, err := mon.Collect(context.Background())
		if err == nil {
			t.Error("Collect() should fail when monitor not started")
		}
		if err.Error() != "coverage monitor not started" {
			t.Errorf("error = %q, want 'coverage monitor not started'", err.Error())
		}
	})

	t.Run("ProcessJSCoverage", func(t *testing.T) {
		mon := NewCoverageMonitor(true, false)

		tests := []struct {
			name       string
			scripts    []*proto.ProfilerScriptCoverage
			wantCount  int
			wantUsed   int
			wantRanges int
		}{
			{
				name:      "empty scripts",
				scripts:   []*proto.ProfilerScriptCoverage{},
				wantCount: 0,
			},
			{
				name: "skip scripts without URL",
				scripts: []*proto.ProfilerScriptCoverage{
					{ScriptID: "1", URL: ""},
				},
				wantCount: 0,
			},
			{
				name: "single script with used ranges",
				scripts: []*proto.ProfilerScriptCoverage{
					{
						ScriptID: "1",
						URL:      "https://example.com/main.js",
						Functions: []*proto.ProfilerFunctionCoverage{
							{
								Ranges: []*proto.ProfilerCoverageRange{
									{StartOffset: 0, EndOffset: 100, Count: 5},
									{StartOffset: 100, EndOffset: 200, Count: 0},
								},
							},
						},
					},
				},
				wantCount:  1,
				wantUsed:   100, // Only the first range (0-100) has Count > 0
				wantRanges: 2,
			},
			{
				name: "multiple scripts",
				scripts: []*proto.ProfilerScriptCoverage{
					{
						ScriptID: "1",
						URL:      "https://example.com/main.js",
						Functions: []*proto.ProfilerFunctionCoverage{
							{
								Ranges: []*proto.ProfilerCoverageRange{
									{StartOffset: 0, EndOffset: 500, Count: 1},
								},
							},
						},
					},
					{
						ScriptID: "2",
						URL:      "https://example.com/vendor.js",
						Functions: []*proto.ProfilerFunctionCoverage{
							{
								Ranges: []*proto.ProfilerCoverageRange{
									{StartOffset: 0, EndOffset: 1000, Count: 0},
								},
							},
						},
					},
				},
				wantCount: 2,
				wantUsed:  500, // Only first script's range is used
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				entries := mon.processJSCoverage(tt.scripts)
				if len(entries) != tt.wantCount {
					t.Errorf("got %d entries, want %d", len(entries), tt.wantCount)
				}

				if tt.wantCount > 0 {
					totalUsed := 0
					for _, entry := range entries {
						totalUsed += entry.UsedBytes
					}
					if totalUsed != tt.wantUsed {
						t.Errorf("total used bytes = %d, want %d", totalUsed, tt.wantUsed)
					}
				}

				if tt.wantRanges > 0 && len(entries) > 0 {
					if len(entries[0].Ranges) != tt.wantRanges {
						t.Errorf("ranges = %d, want %d", len(entries[0].Ranges), tt.wantRanges)
					}
				}
			})
		}
	})

	t.Run("ProcessJSCoverageFields", func(t *testing.T) {
		mon := NewCoverageMonitor(true, false)

		// TotalBytes uses max-range logic: the outer function range [0, 800]
		// is the largest and becomes the TotalBytes for this script.
		scripts := []*proto.ProfilerScriptCoverage{
			{
				ScriptID: "42",
				URL:      "https://example.com/app.js",
				Functions: []*proto.ProfilerFunctionCoverage{
					{
						Ranges: []*proto.ProfilerCoverageRange{
							{StartOffset: 0, EndOffset: 800, Count: 1},
							{StartOffset: 0, EndOffset: 500, Count: 3},
							{StartOffset: 500, EndOffset: 800, Count: 0},
						},
					},
				},
			},
		}

		entries := mon.processJSCoverage(scripts)
		if len(entries) != 1 {
			t.Fatalf("got %d entries, want 1", len(entries))
		}

		entry := entries[0]
		if entry.ScriptID != "42" {
			t.Errorf("ScriptID = %q, want '42'", entry.ScriptID)
		}
		if entry.URL != "https://example.com/app.js" {
			t.Errorf("URL = %q, want 'https://example.com/app.js'", entry.URL)
		}
		// UsedBytes = sum of ranges with Count > 0: 800 + 500 = 1300
		if entry.UsedBytes != 1300 {
			t.Errorf("UsedBytes = %d, want 1300", entry.UsedBytes)
		}
		// TotalBytes = max range size across all ranges (largest = 800)
		if entry.TotalBytes != 800 {
			t.Errorf("TotalBytes = %d, want 800", entry.TotalBytes)
		}

		// Verify individual range details
		if len(entry.Ranges) != 3 {
			t.Fatalf("Ranges count = %d, want 3", len(entry.Ranges))
		}
		if entry.Ranges[1].Count != 3 {
			t.Errorf("Range[1].Count = %d, want 3", entry.Ranges[1].Count)
		}
		if entry.Ranges[2].Count != 0 {
			t.Errorf("Range[2].Count = %d, want 0", entry.Ranges[2].Count)
		}
	})

	t.Run("ProcessCSSCoverage", func(t *testing.T) {
		mon := NewCoverageMonitor(false, true)

		tests := []struct {
			name      string
			rules     []*proto.CSSRuleUsage
			wantCount int
			wantUsed  int
			wantTotal int
		}{
			{
				name:      "empty rules",
				rules:     []*proto.CSSRuleUsage{},
				wantCount: 0,
			},
			{
				name: "single stylesheet with mixed usage",
				rules: []*proto.CSSRuleUsage{
					{StyleSheetID: "sheet-1", StartOffset: 0, EndOffset: 100, Used: true},
					{StyleSheetID: "sheet-1", StartOffset: 100, EndOffset: 250, Used: false},
				},
				wantCount: 1,
				wantUsed:  100,
				wantTotal: 250,
			},
			{
				name: "multiple stylesheets",
				rules: []*proto.CSSRuleUsage{
					{StyleSheetID: "sheet-1", StartOffset: 0, EndOffset: 100, Used: true},
					{StyleSheetID: "sheet-2", StartOffset: 0, EndOffset: 200, Used: true},
					{StyleSheetID: "sheet-1", StartOffset: 100, EndOffset: 300, Used: false},
				},
				wantCount: 2,
				wantUsed:  300, // 100 from sheet-1 + 200 from sheet-2
				wantTotal: 500, // 300 from sheet-1 + 200 from sheet-2
			},
			{
				name: "all rules unused",
				rules: []*proto.CSSRuleUsage{
					{StyleSheetID: "sheet-1", StartOffset: 0, EndOffset: 500, Used: false},
					{StyleSheetID: "sheet-1", StartOffset: 500, EndOffset: 1000, Used: false},
				},
				wantCount: 1,
				wantUsed:  0,
				wantTotal: 1000,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				entries := mon.processCSSCoverage(tt.rules)
				if len(entries) != tt.wantCount {
					t.Errorf("got %d entries, want %d", len(entries), tt.wantCount)
				}

				totalUsed := 0
				totalBytes := 0
				for _, entry := range entries {
					totalUsed += entry.UsedBytes
					totalBytes += entry.TotalBytes
				}

				if totalUsed != tt.wantUsed {
					t.Errorf("total used bytes = %d, want %d", totalUsed, tt.wantUsed)
				}
				if totalBytes != tt.wantTotal {
					t.Errorf("total bytes = %d, want %d", totalBytes, tt.wantTotal)
				}
			})
		}
	})

	t.Run("ProcessCSSCoverageGrouping", func(t *testing.T) {
		mon := NewCoverageMonitor(false, true)

		// All rules belong to same stylesheet — should produce 1 entry
		rules := []*proto.CSSRuleUsage{
			{StyleSheetID: "sheet-1", StartOffset: 0, EndOffset: 50, Used: true},
			{StyleSheetID: "sheet-1", StartOffset: 50, EndOffset: 120, Used: false},
			{StyleSheetID: "sheet-1", StartOffset: 120, EndOffset: 200, Used: true},
		}

		entries := mon.processCSSCoverage(rules)
		if len(entries) != 1 {
			t.Fatalf("got %d entries, want 1 (grouped by stylesheet)", len(entries))
		}

		entry := entries[0]
		if entry.StyleSheetID != "sheet-1" {
			t.Errorf("StyleSheetID = %q, want 'sheet-1'", entry.StyleSheetID)
		}
		if len(entry.Ranges) != 3 {
			t.Errorf("Ranges count = %d, want 3", len(entry.Ranges))
		}
		if entry.UsedBytes != 130 { // 50 + 80
			t.Errorf("UsedBytes = %d, want 130", entry.UsedBytes)
		}
		if entry.TotalBytes != 200 { // 50 + 70 + 80
			t.Errorf("TotalBytes = %d, want 200", entry.TotalBytes)
		}
	})
}

// TestCoverageSummaryCalculations tests the coverage percentage math.
func TestCoverageSummaryCalculations(t *testing.T) {
	tests := []struct {
		name         string
		jsUsed       int
		jsTotal      int
		cssUsed      int
		cssTotal     int
		wantJSPct    float64
		wantCSSPct   float64
		wantTotalPct float64
	}{
		{
			name:         "50% JS, 25% CSS",
			jsUsed:       500,
			jsTotal:      1000,
			cssUsed:      250,
			cssTotal:     1000,
			wantJSPct:    50.0,
			wantCSSPct:   25.0,
			wantTotalPct: 37.5, // 750 / 2000
		},
		{
			name:         "100% coverage",
			jsUsed:       1000,
			jsTotal:      1000,
			cssUsed:      500,
			cssTotal:     500,
			wantJSPct:    100.0,
			wantCSSPct:   100.0,
			wantTotalPct: 100.0,
		},
		{
			name:         "0% coverage",
			jsUsed:       0,
			jsTotal:      1000,
			cssUsed:      0,
			cssTotal:     500,
			wantJSPct:    0.0,
			wantCSSPct:   0.0,
			wantTotalPct: 0.0,
		},
		{
			name:         "zero total bytes",
			jsUsed:       0,
			jsTotal:      0,
			cssUsed:      0,
			cssTotal:     0,
			wantJSPct:    0.0,
			wantCSSPct:   0.0,
			wantTotalPct: 0.0,
		},
		{
			name:         "JS only",
			jsUsed:       300,
			jsTotal:      1000,
			cssUsed:      0,
			cssTotal:     0,
			wantJSPct:    30.0,
			wantCSSPct:   0.0,
			wantTotalPct: 30.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := &CoverageSummary{
				JSUsedBytes:   tt.jsUsed,
				JSTotalBytes:  tt.jsTotal,
				CSSUsedBytes:  tt.cssUsed,
				CSSTotalBytes: tt.cssTotal,
			}

			// Calculate percentages (mirrors the logic in Collect())
			if summary.JSTotalBytes > 0 {
				summary.JSPercentage = float64(summary.JSUsedBytes) / float64(summary.JSTotalBytes) * 100
			}
			if summary.CSSTotalBytes > 0 {
				summary.CSSPercentage = float64(summary.CSSUsedBytes) / float64(summary.CSSTotalBytes) * 100
			}
			summary.TotalUsed = summary.JSUsedBytes + summary.CSSUsedBytes
			summary.TotalSize = summary.JSTotalBytes + summary.CSSTotalBytes
			if summary.TotalSize > 0 {
				summary.TotalPercentage = float64(summary.TotalUsed) / float64(summary.TotalSize) * 100
			}

			if !floatEqual(summary.JSPercentage, tt.wantJSPct) {
				t.Errorf("JSPercentage = %.2f, want %.2f", summary.JSPercentage, tt.wantJSPct)
			}
			if !floatEqual(summary.CSSPercentage, tt.wantCSSPct) {
				t.Errorf("CSSPercentage = %.2f, want %.2f", summary.CSSPercentage, tt.wantCSSPct)
			}
			if !floatEqual(summary.TotalPercentage, tt.wantTotalPct) {
				t.Errorf("TotalPercentage = %.2f, want %.2f", summary.TotalPercentage, tt.wantTotalPct)
			}
		})
	}
}

// floatEqual compares two floats with a small tolerance.
func floatEqual(a, b float64) bool {
	return math.Abs(a-b) < 0.001
}
