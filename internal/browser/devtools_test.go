package browser

import (
	"testing"

	"github.com/go-rod/rod/lib/proto"
)

// TestNetworkMonitorOptions tests network monitor configuration.
func TestNetworkMonitorOptions(t *testing.T) {
	t.Run("DefaultNetworkMonitorOptions", func(t *testing.T) {
		opts := DefaultNetworkMonitorOptions()
		if opts.CaptureBody {
			t.Error("CaptureBody should be false by default")
		}
		if opts.MaxBodySize != 1024*1024 {
			t.Errorf("MaxBodySize = %d, want %d (1MB)", opts.MaxBodySize, 1024*1024)
		}
	})

	t.Run("NewNetworkMonitorWithOptions", func(t *testing.T) {
		opts := NetworkMonitorOptions{
			CaptureBody: true,
			MaxBodySize: 512 * 1024,
		}

		mon := NewNetworkMonitorWithOptions(opts)
		if !mon.opts.CaptureBody {
			t.Error("CaptureBody should be true")
		}
		if mon.opts.MaxBodySize != 512*1024 {
			t.Errorf("MaxBodySize = %d, want %d", mon.opts.MaxBodySize, 512*1024)
		}
	})

	t.Run("NewNetworkMonitorWithOptionsZeroMaxBody", func(t *testing.T) {
		opts := NetworkMonitorOptions{
			CaptureBody: true,
			MaxBodySize: 0, // Should default to 1MB
		}

		mon := NewNetworkMonitorWithOptions(opts)
		if mon.opts.MaxBodySize != 1024*1024 {
			t.Errorf("MaxBodySize = %d, want %d (1MB default)", mon.opts.MaxBodySize, 1024*1024)
		}
	})

	t.Run("NewNetworkMonitorWithOptionsNegativeMaxBody", func(t *testing.T) {
		opts := NetworkMonitorOptions{
			CaptureBody: true,
			MaxBodySize: -100, // Should default to 1MB
		}

		mon := NewNetworkMonitorWithOptions(opts)
		if mon.opts.MaxBodySize != 1024*1024 {
			t.Errorf("MaxBodySize = %d, want %d (1MB default)", mon.opts.MaxBodySize, 1024*1024)
		}
	})
}

// TestTruncateBody tests body truncation logic.
func TestTruncateBody(t *testing.T) {
	mon := NewNetworkMonitorWithOptions(NetworkMonitorOptions{
		CaptureBody: true,
		MaxBodySize: 20,
	})

	tests := []struct {
		name       string
		body       string
		wantPrefix string
		truncated  bool
	}{
		{
			name:       "short body unchanged",
			body:       "hello",
			wantPrefix: "hello",
			truncated:  false,
		},
		{
			name:       "exact limit unchanged",
			body:       "12345678901234567890",
			wantPrefix: "12345678901234567890",
			truncated:  false,
		},
		{
			name:       "over limit truncated",
			body:       "123456789012345678901", // 21 chars
			wantPrefix: "12345678901234567890",
			truncated:  true,
		},
		{
			name:       "empty body",
			body:       "",
			wantPrefix: "",
			truncated:  false,
		},
		{
			name:       "large body truncated",
			body:       "AAAAAAAAAABBBBBBBBBBCCCCCCCCCCDDDDDDDDDDD", // 42 chars
			wantPrefix: "AAAAAAAAAABBBBBBBBBB",
			truncated:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mon.truncateBody(tt.body)

			if tt.truncated {
				if len(result) <= len(tt.body) && result == tt.body {
					t.Error("expected body to be truncated")
				}
				// Check prefix
				if result[:20] != tt.wantPrefix {
					t.Errorf("truncated prefix = %q, want %q", result[:20], tt.wantPrefix)
				}
				// Check suffix contains truncation marker
				if !contains(result, "truncated") {
					t.Error("truncated body should contain 'truncated' marker")
				}
			} else {
				if result != tt.wantPrefix {
					t.Errorf("result = %q, want %q", result, tt.wantPrefix)
				}
			}
		})
	}
}

// TestConvertCSSProperties tests the CSS property conversion helper.
func TestConvertCSSProperties(t *testing.T) {
	t.Run("empty properties", func(t *testing.T) {
		result := convertCSSProperties(nil)
		if len(result) != 0 {
			t.Errorf("got %d properties, want 0", len(result))
		}
	})

	t.Run("multiple properties", func(t *testing.T) {
		props := []*proto.CSSCSSProperty{
			{Name: "color", Value: "red", Important: false},
			{Name: "font-size", Value: "16px", Important: true},
			{Name: "display", Value: "flex", Important: false},
		}

		result := convertCSSProperties(props)
		if len(result) != 3 {
			t.Fatalf("got %d properties, want 3", len(result))
		}

		if result[0].Name != "color" || result[0].Value != "red" {
			t.Errorf("prop[0] = {%s: %s}, want {color: red}", result[0].Name, result[0].Value)
		}
		if !result[1].Important {
			t.Error("font-size should be marked as important")
		}
		if result[2].Name != "display" || result[2].Value != "flex" {
			t.Errorf("prop[2] = {%s: %s}, want {display: flex}", result[2].Name, result[2].Value)
		}
	})
}

// TestConvertRuleMatch tests the CSS rule match conversion helper.
func TestConvertRuleMatch(t *testing.T) {
	t.Run("basic rule with selector", func(t *testing.T) {
		rm := &proto.CSSRuleMatch{
			Rule: &proto.CSSCSSRule{
				Origin: "author",
				SelectorList: &proto.CSSSelectorList{
					Text: ".my-class > p",
				},
				Style: &proto.CSSCSSStyle{
					CSSProperties: []*proto.CSSCSSProperty{
						{Name: "color", Value: "blue"},
					},
				},
				StyleSheetID: "sheet-1",
			},
		}

		result := convertRuleMatch(rm)
		if result.Selector != ".my-class > p" {
			t.Errorf("Selector = %q, want '.my-class > p'", result.Selector)
		}
		if result.Origin != "author" {
			t.Errorf("Origin = %q, want 'author'", result.Origin)
		}
		if result.SourceURL != "sheet-1" {
			t.Errorf("SourceURL = %q, want 'sheet-1'", result.SourceURL)
		}
		if len(result.Properties) != 1 {
			t.Fatalf("Properties count = %d, want 1", len(result.Properties))
		}
		if result.Properties[0].Name != "color" {
			t.Errorf("Properties[0].Name = %q, want 'color'", result.Properties[0].Name)
		}
	})

	t.Run("user-agent rule without stylesheet", func(t *testing.T) {
		rm := &proto.CSSRuleMatch{
			Rule: &proto.CSSCSSRule{
				Origin: "user-agent",
				SelectorList: &proto.CSSSelectorList{
					Text: "*",
				},
				Style: &proto.CSSCSSStyle{
					CSSProperties: []*proto.CSSCSSProperty{
						{Name: "display", Value: "block"},
						{Name: "margin", Value: "0px"},
					},
				},
				// No StyleSheetID for user-agent rules
			},
		}

		result := convertRuleMatch(rm)
		if result.Origin != "user-agent" {
			t.Errorf("Origin = %q, want 'user-agent'", result.Origin)
		}
		if result.SourceURL != "" {
			t.Errorf("SourceURL = %q, want empty for user-agent", result.SourceURL)
		}
		if len(result.Properties) != 2 {
			t.Errorf("Properties count = %d, want 2", len(result.Properties))
		}
	})

	t.Run("rule without selector list", func(t *testing.T) {
		rm := &proto.CSSRuleMatch{
			Rule: &proto.CSSCSSRule{
				Origin: "author",
				Style: &proto.CSSCSSStyle{
					CSSProperties: []*proto.CSSCSSProperty{},
				},
			},
		}

		result := convertRuleMatch(rm)
		if result.Selector != "" {
			t.Errorf("Selector = %q, want empty when no SelectorList", result.Selector)
		}
	})
}

// TestScriptSourceType tests the ScriptSource type.
func TestScriptSourceType(t *testing.T) {
	src := ScriptSource{
		ScriptID: "42",
		URL:      "https://example.com/app.js",
		Source:   "console.log('hello');",
		Length:   21,
	}

	if src.ScriptID != "42" {
		t.Errorf("ScriptID = %q, want '42'", src.ScriptID)
	}
	if src.Length != 21 {
		t.Errorf("Length = %d, want 21", src.Length)
	}
}

// TestContainsHelper tests the case-insensitive contains function.
func TestContainsHelper(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		substr string
		want   bool
	}{
		{"exact match", "hello", "hello", true},
		{"case insensitive", "Hello World", "hello", true},
		{"substring", "hello world", "world", true},
		{"no match", "hello", "xyz", false},
		{"empty substr", "hello", "", true},
		{"empty string", "", "hello", false},
		{"both empty", "", "", true},
		{"mixed case match", "JavaScript", "javascript", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contains(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}
