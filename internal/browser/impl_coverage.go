package browser

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// CoverageMonitor tracks JavaScript and CSS coverage for a browser tab.
// Unlike streaming monitors (network/console), coverage uses a start/collect/stop pattern.
type CoverageMonitor struct {
	page     *rod.Page
	trackJS  bool
	trackCSS bool
}

// NewCoverageMonitor creates a new coverage monitor.
func NewCoverageMonitor(trackJS, trackCSS bool) *CoverageMonitor {
	return &CoverageMonitor{
		trackJS:  trackJS,
		trackCSS: trackCSS,
	}
}

// Start enables coverage tracking on the page.
func (m *CoverageMonitor) Start(ctx context.Context, page *rod.Page) error {
	m.page = page
	ctxPage := page.Context(ctx)

	if m.trackJS {
		_ = proto.ProfilerEnable{}.Call(ctxPage)
		_, err := proto.ProfilerStartPreciseCoverage{CallCount: true, Detailed: true}.Call(ctxPage)
		if err != nil {
			return fmt.Errorf("start JS coverage: %w", err)
		}
	}

	if m.trackCSS {
		_ = proto.CSSEnable{}.Call(ctxPage)
		err := proto.CSSStartRuleUsageTracking{}.Call(ctxPage)
		if err != nil {
			return fmt.Errorf("start CSS coverage: %w", err)
		}
	}

	return nil
}

// Collect gathers coverage data and stops tracking.
// Returns JS entries, CSS entries, and a summary.
func (m *CoverageMonitor) Collect(ctx context.Context) (*CoverageSummary, []JSCoverageEntry, []CSSCoverageEntry, error) {
	if m.page == nil {
		return nil, nil, nil, errors.New("coverage monitor not started")
	}

	ctxPage := m.page.Context(ctx)
	var jsEntries []JSCoverageEntry
	var cssEntries []CSSCoverageEntry
	summary := &CoverageSummary{}

	// Collect JS coverage
	if m.trackJS {
		result, err := proto.ProfilerTakePreciseCoverage{}.Call(ctxPage)
		if err != nil {
			slog.Debug("failed to take JS coverage", "error", err)
		} else {
			jsEntries = m.processJSCoverage(result.Result)
			for _, entry := range jsEntries {
				summary.JSUsedBytes += entry.UsedBytes
				summary.JSTotalBytes += entry.TotalBytes
			}
		}

		// Stop JS profiling
		_ = proto.ProfilerStopPreciseCoverage{}.Call(ctxPage)
		_ = proto.ProfilerDisable{}.Call(ctxPage)
	}

	// Collect CSS coverage
	if m.trackCSS {
		result, err := proto.CSSStopRuleUsageTracking{}.Call(ctxPage)
		if err != nil {
			slog.Debug("failed to stop CSS coverage tracking", "error", err)
		} else {
			cssEntries = m.processCSSCoverage(result.RuleUsage)
			for _, entry := range cssEntries {
				summary.CSSUsedBytes += entry.UsedBytes
				summary.CSSTotalBytes += entry.TotalBytes
			}
		}
	}

	// Calculate percentages
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

	return summary, jsEntries, cssEntries, nil
}

// processJSCoverage converts CDP profiler coverage into our JSCoverageEntry type.
func (m *CoverageMonitor) processJSCoverage(scripts []*proto.ProfilerScriptCoverage) []JSCoverageEntry {
	entries := make([]JSCoverageEntry, 0, len(scripts))

	for _, script := range scripts {
		// Skip scripts without URL (internal/eval scripts)
		if script.URL == "" {
			continue
		}

		entry := JSCoverageEntry{
			ScriptID: string(script.ScriptID),
			URL:      script.URL,
		}

		// Calculate used bytes from function coverage ranges
		for _, fn := range script.Functions {
			for _, r := range fn.Ranges {
				rangeSize := r.EndOffset - r.StartOffset
				if rangeSize > entry.TotalBytes {
					entry.TotalBytes = rangeSize
				}
				entry.Ranges = append(entry.Ranges, CoverageRange{
					StartOffset: r.StartOffset,
					EndOffset:   r.EndOffset,
					Count:       r.Count,
				})
				if r.Count > 0 {
					entry.UsedBytes += r.EndOffset - r.StartOffset
				}
			}
		}

		entries = append(entries, entry)
	}

	return entries
}

// processCSSCoverage converts CDP CSS rule usage into our CSSCoverageEntry type.
func (m *CoverageMonitor) processCSSCoverage(rules []*proto.CSSRuleUsage) []CSSCoverageEntry {
	// Group rules by stylesheet
	sheetMap := make(map[string]*CSSCoverageEntry)

	for _, rule := range rules {
		sheetID := string(rule.StyleSheetID)
		entry, exists := sheetMap[sheetID]
		if !exists {
			entry = &CSSCoverageEntry{
				StyleSheetID: sheetID,
			}
			sheetMap[sheetID] = entry
		}

		rangeSize := int(rule.EndOffset - rule.StartOffset)
		entry.TotalBytes += rangeSize
		entry.Ranges = append(entry.Ranges, CoverageRange{
			StartOffset: int(rule.StartOffset),
			EndOffset:   int(rule.EndOffset),
		})

		if rule.Used {
			entry.UsedBytes += rangeSize
		}
	}

	entries := make([]CSSCoverageEntry, 0, len(sheetMap))
	for _, entry := range sheetMap {
		entries = append(entries, *entry)
	}

	return entries
}
