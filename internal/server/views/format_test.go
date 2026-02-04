package views

import (
	"testing"
	"time"
)

func TestFormatCost(t *testing.T) {
	tests := []struct {
		name string
		cost float64
		want string
	}{
		{
			name: "zero cost",
			cost: 0,
			want: "$0.00",
		},
		{
			name: "very small cost",
			cost: 0.001,
			want: "<$0.01",
		},
		{
			name: "normal cost",
			cost: 1.23,
			want: "$1.23",
		},
		{
			name: "large cost",
			cost: 123.456,
			want: "$123.46",
		},
		{
			name: "very large cost",
			cost: 1000.999,
			want: "$1001.00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatCost(tt.cost)
			if got != tt.want {
				t.Errorf("FormatCost() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		name string
		n    int
		want string
	}{
		{
			name: "zero",
			n:    0,
			want: "0",
		},
		{
			name: "single digit",
			n:    5,
			want: "5",
		},
		{
			name: "hundreds",
			n:    999,
			want: "999",
		},
		{
			name: "one thousand with decimals",
			n:    1200,
			want: "1.2K",
		},
		{
			name: "ten thousands",
			n:    15000,
			want: "15K",
		},
		{
			name: "hundred thousands",
			n:    999000,
			want: "999K",
		},
		{
			name: "million with decimals",
			n:    1500000,
			want: "1.5M",
		},
		{
			name: "ten millions",
			n:    15000000,
			want: "15M",
		},
		{
			name: "billion",
			n:    1500000000,
			want: "1.5B",
		},
		{
			name: "negative number",
			n:    -5000,
			want: "-5.0K",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatNumber(tt.n)
			if got != tt.want {
				t.Errorf("FormatNumber() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatPercent(t *testing.T) {
	tests := []struct {
		name string
		pct  float64
		want string
	}{
		{
			name: "zero percent",
			pct:  0,
			want: "0%",
		},
		{
			name: "negative percent",
			pct:  -5,
			want: "0%",
		},
		{
			name: "less than 1 percent",
			pct:  0.5,
			want: "<1%",
		},
		{
			name: "whole number percent",
			pct:  42,
			want: "42%",
		},
		{
			name: "decimal percent",
			pct:  42.5,
			want: "42.5%",
		},
		{
			name: "at 100 percent",
			pct:  100,
			want: "100%",
		},
		{
			name: "over 100 percent",
			pct:  150,
			want: "100%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatPercent(tt.pct)
			if got != tt.want {
				t.Errorf("FormatPercent() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatTimeAgo(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name string
		t    time.Time
		want string
	}{
		{
			name: "zero time",
			t:    time.Time{},
			want: "",
		},
		{
			name: "just now",
			t:    now.Add(-30 * time.Second),
			want: "just now",
		},
		{
			name: "1 minute ago",
			t:    now.Add(-90 * time.Second),
			want: "1 minute ago",
		},
		{
			name: "5 minutes ago",
			t:    now.Add(-5 * time.Minute),
			want: "5 minutes ago",
		},
		{
			name: "1 hour ago",
			t:    now.Add(-90 * time.Minute),
			want: "1 hour ago",
		},
		{
			name: "3 hours ago",
			t:    now.Add(-3 * time.Hour),
			want: "3 hours ago",
		},
		{
			name: "1 day ago",
			t:    now.Add(-36 * time.Hour),
			want: "1 day ago",
		},
		{
			name: "5 days ago",
			t:    now.Add(-5 * 24 * time.Hour),
			want: "5 days ago",
		},
		{
			name: "1 week ago",
			t:    now.Add(-10 * 24 * time.Hour),
			want: "1 week ago",
		},
		{
			name: "3 weeks ago",
			t:    now.Add(-21 * 24 * time.Hour),
			want: "3 weeks ago",
		},
		{
			name: "1 month ago",
			t:    now.Add(-45 * 24 * time.Hour),
			want: "1 month ago",
		},
		{
			name: "6 months ago",
			t:    now.Add(-180 * 24 * time.Hour),
			want: "6 months ago",
		},
		{
			name: "1 year ago",
			t:    now.Add(-400 * 24 * time.Hour),
			want: "1 year ago",
		},
		{
			name: "2 years ago",
			t:    now.Add(-800 * 24 * time.Hour),
			want: "2 years ago",
		},
		{
			name: "future time",
			t:    now.Add(1 * time.Hour),
			want: "in the future",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatTimeAgo(tt.t)
			if got != tt.want {
				t.Errorf("FormatTimeAgo() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{
			name: "zero duration",
			d:    0,
			want: "<1s",
		},
		{
			name: "negative duration",
			d:    -1 * time.Second,
			want: "0s",
		},
		{
			name: "less than 1 second",
			d:    500 * time.Millisecond,
			want: "<1s",
		},
		{
			name: "seconds only",
			d:    45 * time.Second,
			want: "45s",
		},
		{
			name: "1 minute",
			d:    1 * time.Minute,
			want: "1m",
		},
		{
			name: "minutes and seconds",
			d:    2*time.Minute + 30*time.Second,
			want: "2m 30s",
		},
		{
			name: "1 hour",
			d:    1 * time.Hour,
			want: "1h",
		},
		{
			name: "hours and minutes",
			d:    2*time.Hour + 30*time.Minute,
			want: "2h 30m",
		},
		{
			name: "1 day",
			d:    24 * time.Hour,
			want: "1d",
		},
		{
			name: "days and hours",
			d:    2*24*time.Hour + 6*time.Hour,
			want: "2d 6h",
		},
		{
			name: "days hours minutes",
			d:    2*24*time.Hour + 6*time.Hour + 30*time.Minute,
			want: "2d 6h 30m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatDuration(tt.d)
			if got != tt.want {
				t.Errorf("FormatDuration() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatDate(t *testing.T) {
	tests := []struct {
		name string
		t    time.Time
		want string
	}{
		{
			name: "zero time",
			t:    time.Time{},
			want: "",
		},
		{
			name: "valid date",
			t:    time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			want: "2024-01-15",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatDate(tt.t)
			if got != tt.want {
				t.Errorf("FormatDate() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatDateTime(t *testing.T) {
	tests := []struct {
		name string
		t    time.Time
		want string
	}{
		{
			name: "zero time",
			t:    time.Time{},
			want: "",
		},
		{
			name: "valid datetime",
			t:    time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			want: "2024-01-15 10:30",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatDateTime(tt.t)
			if got != tt.want {
				t.Errorf("FormatDateTime() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatMonth(t *testing.T) {
	tests := []struct {
		name string
		t    time.Time
		want string
	}{
		{
			name: "zero time",
			t:    time.Time{},
			want: "",
		},
		{
			name: "January 2024",
			t:    time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			want: "January 2024",
		},
		{
			name: "December 2023",
			t:    time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC),
			want: "December 2023",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatMonth(tt.t)
			if got != tt.want {
				t.Errorf("FormatMonth() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBudgetColor(t *testing.T) {
	tests := []struct {
		name string
		pct  float64
		want string
	}{
		{
			name: "low usage",
			pct:  50,
			want: ProgressGreen,
		},
		{
			name: "medium usage",
			pct:  75,
			want: ProgressYellow,
		},
		{
			name: "high usage",
			pct:  95,
			want: ProgressRed,
		},
		{
			name: "at 90 threshold",
			pct:  90,
			want: ProgressRed,
		},
		{
			name: "at 70 threshold",
			pct:  70,
			want: ProgressYellow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BudgetColor(tt.pct)
			if got != tt.want {
				t.Errorf("BudgetColor() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		maxLen int
		want   string
	}{
		{
			name:   "empty string",
			s:      "",
			maxLen: 10,
			want:   "",
		},
		{
			name:   "zero max length",
			s:      "hello",
			maxLen: 0,
			want:   "",
		},
		{
			name:   "negative max length",
			s:      "hello",
			maxLen: -1,
			want:   "",
		},
		{
			name:   "shorter than max",
			s:      "hello",
			maxLen: 10,
			want:   "hello",
		},
		{
			name:   "exactly max length",
			s:      "hello",
			maxLen: 5,
			want:   "hello",
		},
		{
			name:   "needs truncation",
			s:      "hello world",
			maxLen: 8,
			want:   "hello...",
		},
		{
			name:   "truncate to 3 or less",
			s:      "hello",
			maxLen: 3,
			want:   "hel",
		},
		{
			name:   "truncate unicode",
			s:      "hello世界",
			maxLen: 7,
			want:   "hello世界", // 7 runes = 5 + 2, exactly at limit
		},
		{
			name:   "truncate unicode with ellipsis",
			s:      "hello世界",
			maxLen: 6,
			want:   "hel...", // Takes first 3 runes + "..."
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateString(tt.s, tt.maxLen)
			if got != tt.want {
				t.Errorf("TruncateString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestShortID(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want string
	}{
		{
			name: "short ID",
			id:   "abc123",
			want: "abc123",
		},
		{
			name: "exactly 8 chars",
			id:   "12345678",
			want: "12345678",
		},
		{
			name: "long ID",
			id:   "1234567890abcdef",
			want: "12345678",
		},
		{
			name: "empty string",
			id:   "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShortID(tt.id)
			if got != tt.want {
				t.Errorf("ShortID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPluralize(t *testing.T) {
	tests := []struct {
		name     string
		count    int
		singular string
		plural   string
		want     string
	}{
		{
			name:     "singular form",
			count:    1,
			singular: "task",
			plural:   "tasks",
			want:     "task",
		},
		{
			name:     "plural form",
			count:    5,
			singular: "task",
			plural:   "tasks",
			want:     "tasks",
		},
		{
			name:     "zero uses plural",
			count:    0,
			singular: "item",
			plural:   "items",
			want:     "items",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Pluralize(tt.count, tt.singular, tt.plural)
			if got != tt.want {
				t.Errorf("Pluralize() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatCount(t *testing.T) {
	tests := []struct {
		name     string
		count    int
		singular string
		plural   string
		want     string
	}{
		{
			name:     "one item",
			count:    1,
			singular: "task",
			plural:   "tasks",
			want:     "1 task",
		},
		{
			name:     "multiple items",
			count:    5,
			singular: "task",
			plural:   "tasks",
			want:     "5 tasks",
		},
		{
			name:     "zero items",
			count:    0,
			singular: "item",
			plural:   "items",
			want:     "0 items",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatCount(tt.count, tt.singular, tt.plural)
			if got != tt.want {
				t.Errorf("FormatCount() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToFloat(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  float64
	}{
		{
			name:  "int positive",
			input: 42,
			want:  42.0,
		},
		{
			name:  "int zero",
			input: 0,
			want:  0.0,
		},
		{
			name:  "int negative",
			input: -10,
			want:  -10.0,
		},
		{
			name:  "int64 positive",
			input: int64(1000000),
			want:  1000000.0,
		},
		{
			name:  "int64 negative",
			input: int64(-500),
			want:  -500.0,
		},
		{
			name:  "int32 positive",
			input: int32(100),
			want:  100.0,
		},
		{
			name:  "int32 negative",
			input: int32(-50),
			want:  -50.0,
		},
		{
			name:  "uint positive",
			input: uint(999),
			want:  999.0,
		},
		{
			name:  "uint64 large",
			input: uint64(18446744073709551615),
			want:  18446744073709551615.0,
		},
		{
			name:  "float32 positive",
			input: float32(2.5),
			want:  2.5,
		},
		{
			name:  "float64 positive",
			input: 3.14,
			want:  3.14,
		},
		{
			name:  "float64 zero",
			input: 0.0,
			want:  0.0,
		},
		{
			name:  "float64 negative",
			input: -2.5,
			want:  -2.5,
		},
		{
			name:  "string returns zero",
			input: "not a number",
			want:  0.0,
		},
		{
			name:  "nil returns zero",
			input: nil,
			want:  0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toFloat(tt.input)
			if got != tt.want {
				t.Errorf("toFloat(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
