package memory

import (
	"testing"
	"time"
)

func TestMatchTypeFilter(t *testing.T) {
	tests := []struct {
		name       string
		docType    DocumentType
		typeFilter interface{}
		want       bool
	}{
		{
			name:       "matching string type",
			docType:    TypeCodeChange,
			typeFilter: "code_change",
			want:       true,
		},
		{
			name:       "non-matching string type",
			docType:    TypeCodeChange,
			typeFilter: "solution",
			want:       false,
		},
		{
			name:       "matching type in string array",
			docType:    TypeCodeChange,
			typeFilter: []string{"solution", "code_change"},
			want:       true,
		},
		{
			name:       "non-matching type in string array",
			docType:    TypeCodeChange,
			typeFilter: []string{"solution", "error"},
			want:       false,
		},
		{
			name:       "empty string array",
			docType:    TypeCodeChange,
			typeFilter: []string{},
			want:       false,
		},
		{
			name:       "type matches first in array",
			docType:    TypeSolution,
			typeFilter: []string{"solution", "code_change"},
			want:       true,
		},
		{
			name:       "type matches last in array",
			docType:    TypeError,
			typeFilter: []string{"solution", "code_change", "error"},
			want:       true,
		},
		{
			name:       "unknown filter type - string conversion match",
			docType:    TypeCodeChange,
			typeFilter: TypeCodeChange, // Same type, different instance
			want:       true,
		},
		{
			name:       "unknown filter type - different type",
			docType:    TypeCodeChange,
			typeFilter: 123,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchTypeFilter(tt.docType, tt.typeFilter)
			if got != tt.want {
				t.Errorf("matchTypeFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchesFilter(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-1 * time.Hour)
	later := now.Add(1 * time.Hour)

	doc := &Document{
		ID:        "doc-1",
		Type:      TypeCodeChange,
		Content:   "Fixed a bug",
		CreatedAt: now,
		Metadata: map[string]interface{}{
			"file":     "main.go",
			"language": "go",
		},
	}

	tests := []struct {
		name   string
		doc    *Document
		filter map[string]interface{}
		want   bool
	}{
		{
			name:   "no filter - passes",
			doc:    doc,
			filter: nil,
			want:   true,
		},
		{
			name:   "empty filter - passes",
			doc:    doc,
			filter: map[string]interface{}{},
			want:   true,
		},
		{
			name:   "matching type filter",
			doc:    doc,
			filter: map[string]interface{}{"type": "code_change"},
			want:   true,
		},
		{
			name:   "non-matching type filter",
			doc:    doc,
			filter: map[string]interface{}{"type": "solution"},
			want:   false,
		},
		{
			name:   "matching type in array",
			doc:    doc,
			filter: map[string]interface{}{"type": []string{"solution", "code_change"}},
			want:   true,
		},
		{
			name: "time_from before doc time - passes",
			doc:  doc,
			filter: map[string]interface{}{
				"time_from": earlier,
			},
			want: true,
		},
		{
			name: "time_from after doc time - fails",
			doc:  doc,
			filter: map[string]interface{}{
				"time_from": later,
			},
			want: false,
		},
		{
			name: "time_to after doc time - passes",
			doc:  doc,
			filter: map[string]interface{}{
				"time_to": later,
			},
			want: true,
		},
		{
			name: "time_to before doc time - fails",
			doc:  doc,
			filter: map[string]interface{}{
				"time_to": earlier,
			},
			want: false,
		},
		{
			name: "time range includes doc - passes",
			doc:  doc,
			filter: map[string]interface{}{
				"time_from": earlier,
				"time_to":   later,
			},
			want: true,
		},
		{
			name: "matching metadata key-value",
			doc:  doc,
			filter: map[string]interface{}{
				"file": "main.go",
			},
			want: true,
		},
		{
			name: "non-matching metadata value",
			doc:  doc,
			filter: map[string]interface{}{
				"file": "other.go",
			},
			want: false,
		},
		{
			name: "missing metadata key",
			doc:  doc,
			filter: map[string]interface{}{
				"missing_key": "value",
			},
			want: false,
		},
		{
			name: "combined filters - all pass",
			doc:  doc,
			filter: map[string]interface{}{
				"type":     "code_change",
				"file":     "main.go",
				"language": "go",
			},
			want: true,
		},
		{
			name: "combined filters - one fails",
			doc:  doc,
			filter: map[string]interface{}{
				"type": "code_change",
				"file": "wrong.go",
			},
			want: false,
		},
		{
			name: "min_score filter ignored in matchesFilter",
			doc:  doc,
			filter: map[string]interface{}{
				"min_score": float32(0.9),
			},
			want: true, // min_score is handled elsewhere
		},
		{
			name: "doc with no metadata and metadata filter",
			doc: &Document{
				ID:        "doc-2",
				Type:      TypeSolution,
				CreatedAt: now,
				Metadata:  nil,
			},
			filter: map[string]interface{}{
				"file": "main.go",
			},
			want: false,
		},
		{
			name: "empty metadata map with filter",
			doc: &Document{
				ID:        "doc-3",
				Type:      TypeSolution,
				CreatedAt: now,
				Metadata:  map[string]interface{}{},
			},
			filter: map[string]interface{}{
				"file": "main.go",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesFilter(tt.doc, tt.filter)
			if got != tt.want {
				t.Errorf("matchesFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}
