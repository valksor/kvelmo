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
