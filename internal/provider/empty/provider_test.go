package empty

import (
	"context"
	"testing"

	"github.com/valksor/go-mehrhof/internal/provider"
)

func TestProvider_Parse(t *testing.T) {
	p := &Provider{}

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "valid identifier",
			input:   "empty:A-1",
			want:    "A-1",
			wantErr: false,
		},
		{
			name:    "identifier with description",
			input:   "empty:FEATURE-123",
			want:    "FEATURE-123",
			wantErr: false,
		},
		{
			name:    "empty identifier",
			input:   "empty:",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := p.Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)

				return
			}
			if got != tt.want {
				t.Errorf("Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProvider_Fetch(t *testing.T) {
	p := &Provider{}

	wu, err := p.Fetch(context.Background(), "TEST-123")
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if wu.ID != "TEST-123" {
		t.Errorf("Fetch() ID = %v, want %v", wu.ID, "TEST-123")
	}
	if wu.Title != "TEST-123" {
		t.Errorf("Fetch() Title = %v, want %v", wu.Title, "TEST-123")
	}
	if wu.Description != "" {
		t.Errorf("Fetch() Description = %v, want empty", wu.Description)
	}
	if wu.Provider != "empty" {
		t.Errorf("Fetch() Provider = %v, want %v", wu.Provider, "empty")
	}
	if wu.Status != provider.StatusOpen {
		t.Errorf("Fetch() Status = %v, want %v", wu.Status, provider.StatusOpen)
	}
	if wu.Priority != provider.PriorityNormal {
		t.Errorf("Fetch() Priority = %v, want %v", wu.Priority, provider.PriorityNormal)
	}
}

func TestParseMetadataFromNotes_CaseInsensitive(t *testing.T) {
	tests := []struct {
		name         string
		notes        []string
		wantStatus   string
		wantPriority string
		wantLabels   []string
	}{
		{
			name:         "uppercase @STATUS",
			notes:        []string{"@STATUS: done"},
			wantStatus:   "done",
			wantPriority: "",
			wantLabels:   nil,
		},
		{
			name:         "mixed case @Status",
			notes:        []string{"@Status: in_progress"},
			wantStatus:   "in_progress",
			wantPriority: "",
			wantLabels:   nil,
		},
		{
			name:         "uppercase @PRIORITY",
			notes:        []string{"@PRIORITY: critical"},
			wantStatus:   "",
			wantPriority: "critical",
			wantLabels:   nil,
		},
		{
			name:         "mixed case @Priority",
			notes:        []string{"@Priority: low"},
			wantStatus:   "",
			wantPriority: "low",
			wantLabels:   nil,
		},
		{
			name:         "uppercase @LABELS",
			notes:        []string{"@LABELS: bug,urgent"},
			wantStatus:   "",
			wantPriority: "",
			wantLabels:   []string{"bug", "urgent"},
		},
		{
			name:         "mixed case @Labels",
			notes:        []string{"@Labels: feature,enhancement"},
			wantStatus:   "",
			wantPriority: "",
			wantLabels:   []string{"feature", "enhancement"},
		},
		{
			name:         "all metadata with mixed case",
			notes:        []string{"@Status: review", "@Priority: HIGH", "@LaBeLs: bug"},
			wantStatus:   "review",
			wantPriority: "HIGH",
			wantLabels:   []string{"bug"},
		},
		{
			name:         "value preserves case",
			notes:        []string{"@status: In_Progress", "@priority: CrItIcAl"},
			wantStatus:   "In_Progress",
			wantPriority: "CrItIcAl",
			wantLabels:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStatus, gotPriority, gotLabels := ParseMetadataFromNotes(tt.notes)

			if gotStatus != tt.wantStatus {
				t.Errorf("ParseMetadataFromNotes() status = %q, want %q", gotStatus, tt.wantStatus)
			}
			if gotPriority != tt.wantPriority {
				t.Errorf("ParseMetadataFromNotes() priority = %q, want %q", gotPriority, tt.wantPriority)
			}
			if !provider.EqualStringSlices(gotLabels, tt.wantLabels) {
				t.Errorf("ParseMetadataFromNotes() labels = %v, want %v", gotLabels, tt.wantLabels)
			}
		})
	}
}

func TestParseMetadataFromNotes(t *testing.T) {
	tests := []struct {
		name         string
		notes        []string
		wantStatus   string
		wantPriority string
		wantLabels   []string
	}{
		{
			name:         "all metadata present",
			notes:        []string{"@status: in_progress", "@priority: high", "@labels: bug,urgent"},
			wantStatus:   "in_progress",
			wantPriority: "high",
			wantLabels:   []string{"bug", "urgent"},
		},
		{
			name:         "labels with trailing comma",
			notes:        []string{"@labels: bug,urgent,"},
			wantStatus:   "",
			wantPriority: "",
			wantLabels:   []string{"bug", "urgent"},
		},
		{
			name:         "labels with quotes",
			notes:        []string{`@labels: "bug","urgent"`},
			wantStatus:   "",
			wantPriority: "",
			wantLabels:   []string{"bug", "urgent"},
		},
		{
			name:         "no metadata",
			notes:        []string{"just a regular note"},
			wantStatus:   "",
			wantPriority: "",
			wantLabels:   nil,
		},
		{
			name:         "empty notes",
			notes:        []string{},
			wantStatus:   "",
			wantPriority: "",
			wantLabels:   nil,
		},
		{
			name:         "mixed valid and invalid notes",
			notes:        []string{"some text", "@status: done", "more text", "@labels: feature"},
			wantStatus:   "done",
			wantPriority: "",
			wantLabels:   []string{"feature"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStatus, gotPriority, gotLabels := ParseMetadataFromNotes(tt.notes)

			if gotStatus != tt.wantStatus {
				t.Errorf("ParseMetadataFromNotes() status = %q, want %q", gotStatus, tt.wantStatus)
			}
			if gotPriority != tt.wantPriority {
				t.Errorf("ParseMetadataFromNotes() priority = %q, want %q", gotPriority, tt.wantPriority)
			}
			if !provider.EqualStringSlices(gotLabels, tt.wantLabels) {
				t.Errorf("ParseMetadataFromNotes() labels = %v, want %v", gotLabels, tt.wantLabels)
			}
		})
	}
}
