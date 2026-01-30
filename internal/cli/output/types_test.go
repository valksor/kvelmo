package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		wantErr bool
	}{
		{
			name:    "simple struct",
			input:   TokenUsage{InputTokens: 100, OutputTokens: 50, TotalTokens: 150},
			wantErr: false,
		},
		{
			name:    "nil value",
			input:   nil,
			wantErr: false,
		},
		{
			name:    "nested struct",
			input:   CostMetrics{InputTokens: 100, OutputTokens: 50, TotalTokens: 150, CostUSD: 0.01},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := WriteJSONTo(&buf, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("WriteJSONTo() error = %v, wantErr %v", err, tt.wantErr)

				return
			}
			if !tt.wantErr && buf.Len() == 0 {
				t.Error("WriteJSONTo() produced empty output")
			}
		})
	}
}

func TestWriteJSONTo_Formatting(t *testing.T) {
	input := TokenUsage{InputTokens: 100, OutputTokens: 50, TotalTokens: 150}

	var buf bytes.Buffer
	if err := WriteJSONTo(&buf, input); err != nil {
		t.Fatalf("WriteJSONTo() error = %v", err)
	}

	// Check that output is indented (contains newlines and spaces)
	out := buf.String()
	if !strings.Contains(out, "\n") {
		t.Error("WriteJSONTo() output is not indented (no newlines)")
	}
	if !strings.Contains(out, "  ") {
		t.Error("WriteJSONTo() output is not indented (no double spaces)")
	}
}

func TestTokenUsage_JSONMarshaling(t *testing.T) {
	usage := TokenUsage{
		InputTokens:  1000,
		OutputTokens: 500,
		CachedTokens: 200,
		TotalTokens:  1500,
	}

	data, err := json.Marshal(usage)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded TokenUsage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded != usage {
		t.Errorf("Round-trip failed: got %+v, want %+v", decoded, usage)
	}
}

func TestCostMetrics_JSONMarshaling(t *testing.T) {
	metrics := CostMetrics{
		InputTokens:  1000,
		OutputTokens: 500,
		TotalTokens:  1500,
		CachedTokens: 200,
		CostUSD:      0.015,
	}

	data, err := json.Marshal(metrics)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded CostMetrics
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded != metrics {
		t.Errorf("Round-trip failed: got %+v, want %+v", decoded, metrics)
	}
}

func TestStepCost_JSONMarshaling(t *testing.T) {
	cost := StepCost{
		InputTokens:  1000,
		OutputTokens: 500,
		CachedTokens: 200,
		TotalTokens:  1500,
		CostUSD:      0.015,
		Calls:        3,
	}

	data, err := json.Marshal(cost)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded StepCost
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded != cost {
		t.Errorf("Round-trip failed: got %+v, want %+v", decoded, cost)
	}
}

func TestSpecificationSummary_JSONMarshaling(t *testing.T) {
	summary := SpecificationSummary{
		Draft:        2,
		Ready:        3,
		Implementing: 1,
		Done:         5,
	}

	data, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded SpecificationSummary
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded != summary {
		t.Errorf("Round-trip failed: got %+v, want %+v", decoded, summary)
	}
}

func TestCheckpoint_JSONMarshaling(t *testing.T) {
	checkpoint := Checkpoint{
		Number:    1,
		Message:   "Initial checkpoint",
		ID:        "abc123",
		Timestamp: "2024-01-15T10:30:00Z",
	}

	data, err := json.Marshal(checkpoint)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded Checkpoint
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded != checkpoint {
		t.Errorf("Round-trip failed: got %+v, want %+v", decoded, checkpoint)
	}
}

func TestSession_JSONMarshaling(t *testing.T) {
	session := Session{
		Kind:         "planning",
		StartTime:    "2024-01-15T10:30:00Z",
		InputTokens:  1000,
		OutputTokens: 500,
	}

	data, err := json.Marshal(session)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded Session
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded != session {
		t.Errorf("Round-trip failed: got %+v, want %+v", decoded, session)
	}
}

func TestSpecification_JSONMarshaling(t *testing.T) {
	spec := Specification{
		Number:      1,
		Title:       "Add user authentication",
		Status:      "implementing",
		CreatedAt:   "2024-01-15T10:30:00Z",
		CompletedAt: "",
	}

	data, err := json.Marshal(spec)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded Specification
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded != spec {
		t.Errorf("Round-trip failed: got %+v, want %+v", decoded, spec)
	}
}
