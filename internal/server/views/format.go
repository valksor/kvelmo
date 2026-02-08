package views

import (
	"fmt"
	"strconv"
	"time"

	"github.com/valksor/go-mehrhof/internal/display"
)

// FormatCost formats a USD cost value with proper display.
func FormatCost(cost float64) string {
	if cost == 0 {
		return "$0.00"
	}
	if cost < 0.01 {
		return "<$0.01"
	}

	return fmt.Sprintf("$%.2f", cost)
}

// FormatNumber formats a number with K/M/B suffixes for readability.
func FormatNumber(n int) string {
	if n < 0 {
		return "-" + FormatNumber(-n)
	}
	if n < 1000 {
		return strconv.Itoa(n)
	}
	if n < 10000 {
		// Show one decimal for numbers like 1.2K
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	if n < 1_000_000 {
		return fmt.Sprintf("%dK", n/1000)
	}
	if n < 10_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n < 1_000_000_000 {
		return fmt.Sprintf("%dM", n/1_000_000)
	}

	return fmt.Sprintf("%.1fB", float64(n)/1_000_000_000)
}

// FormatBytes formats a byte count with KB/MB/GB suffixes for readability.
func FormatBytes(bytes int64) string {
	if bytes < 0 {
		return "-" + FormatBytes(-bytes)
	}
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	}
	if bytes < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	}

	return fmt.Sprintf("%.1f GB", float64(bytes)/(1024*1024*1024))
}

// FormatPercent formats a percentage value.
func FormatPercent(pct float64) string {
	if pct <= 0 {
		return "0%"
	}
	if pct < 1 {
		return "<1%"
	}
	if pct >= 100 {
		return "100%"
	}
	// Show integer if it's a whole number
	if pct == float64(int(pct)) {
		return fmt.Sprintf("%d%%", int(pct))
	}

	return fmt.Sprintf("%.1f%%", pct)
}

// FormatTimeAgo formats a time as a human-readable "time ago" string.
// This is a fixed implementation that avoids the char arithmetic bugs.
func FormatTimeAgo(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	d := time.Since(t)
	if d < 0 {
		return "in the future"
	}

	switch {
	case d < time.Minute:
		return "just now"
	case d < 2*time.Minute:
		return "1 minute ago"
	case d < time.Hour:
		mins := int(d.Minutes())

		return fmt.Sprintf("%d minutes ago", mins)
	case d < 2*time.Hour:
		return "1 hour ago"
	case d < 24*time.Hour:
		hours := int(d.Hours())

		return fmt.Sprintf("%d hours ago", hours)
	case d < 48*time.Hour:
		return "1 day ago"
	case d < 7*24*time.Hour:
		days := int(d.Hours() / 24)

		return fmt.Sprintf("%d days ago", days)
	case d < 14*24*time.Hour:
		return "1 week ago"
	case d < 30*24*time.Hour:
		weeks := int(d.Hours() / (24 * 7))

		return fmt.Sprintf("%d weeks ago", weeks)
	case d < 60*24*time.Hour:
		return "1 month ago"
	case d < 365*24*time.Hour:
		months := int(d.Hours() / (24 * 30))

		return fmt.Sprintf("%d months ago", months)
	default:
		years := int(d.Hours() / (24 * 365))
		if years == 1 {
			return "1 year ago"
		}

		return fmt.Sprintf("%d years ago", years)
	}
}

// FormatDuration formats a duration as a human-readable string.
func FormatDuration(d time.Duration) string {
	if d < 0 {
		return "0s"
	}
	if d < time.Second {
		return "<1s"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		secs := int(d.Seconds()) % 60
		if secs == 0 {
			return fmt.Sprintf("%dm", mins)
		}

		return fmt.Sprintf("%dm %ds", mins, secs)
	}

	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60

	if hours >= 24 {
		days := hours / 24
		hours = hours % 24
		if hours == 0 && mins == 0 {
			return fmt.Sprintf("%dd", days)
		}
		if mins == 0 {
			return fmt.Sprintf("%dd %dh", days, hours)
		}

		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	}

	if mins == 0 {
		return fmt.Sprintf("%dh", hours)
	}

	return fmt.Sprintf("%dh %dm", hours, mins)
}

// FormatDate formats a time as a date string (dd.mm.yyyy).
func FormatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	return t.Format(display.DateFormat)
}

// FormatDateTime formats a time as a datetime string (dd.mm.yyyy hh:mm).
func FormatDateTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	return t.Format(display.DateTimeFormat)
}

// FormatMonth formats a time as a month string (January 2006).
func FormatMonth(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	return t.Format("January 2006")
}

// BudgetColor returns the appropriate progress bar color based on usage percentage.
func BudgetColor(pct float64) string {
	switch {
	case pct >= 90:
		return ProgressRed
	case pct >= 70:
		return ProgressYellow
	default:
		return ProgressGreen
	}
}

// TruncateString truncates a string to maxLen characters, adding ellipsis if needed.
func TruncateString(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return string(runes[:maxLen])
	}

	return string(runes[:maxLen-3]) + "..."
}

// ShortID returns the first 8 characters of an ID.
func ShortID(id string) string {
	if len(id) <= 8 {
		return id
	}

	return id[:8]
}

// Pluralize returns singular or plural form based on count.
func Pluralize(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}

	return plural
}

// FormatCount formats a count with its unit, properly pluralized.
func FormatCount(count int, singular, plural string) string {
	return fmt.Sprintf("%d %s", count, Pluralize(count, singular, plural))
}
