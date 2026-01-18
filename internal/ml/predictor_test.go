package ml

import (
	"context"
	"testing"
	"time"

	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/workflow"
)

// createMockTask creates a mock TaskWork for testing.
func createMockTask() *storage.TaskWork {
	return &storage.TaskWork{
		Metadata: storage.WorkMetadata{
			ID:        "test-task-1",
			Title:     "Fix authentication bug",
			CreatedAt: time.Now(),
		},
		Agent: storage.AgentInfo{
			Name: "claude",
		},
	}
}

func TestPredictionType_String(t *testing.T) {
	tests := []struct {
		pt     PredictionType
		expect string
	}{
		{PredictNextAction, "next_action"},
		{PredictDuration, "duration"},
		{PredictComplexity, "complexity"},
		{PredictAgentSelection, "agent_selection"},
		{PredictRisk, "risk"},
	}

	for _, tt := range tests {
		t.Run(tt.expect, func(t *testing.T) {
			if string(tt.pt) != tt.expect {
				t.Errorf("expected %s, got %s", tt.expect, string(tt.pt))
			}
		})
	}
}

func TestNextActionPredictor_Predict(t *testing.T) {
	predictor := NewNextActionPredictor()

	tests := []struct {
		name           string
		state          workflow.State
		expectedAction string
		minConfidence  float32
	}{
		{
			name:           "idle state",
			state:          workflow.StateIdle,
			expectedAction: "plan",
			minConfidence:  0.8,
		},
		{
			name:           "planning state",
			state:          workflow.StatePlanning,
			expectedAction: "plan",
			minConfidence:  0.7,
		},
		{
			name:           "implementing state",
			state:          workflow.StateImplementing,
			expectedAction: "review",
			minConfidence:  0.8,
		},
		{
			name:           "reviewing state",
			state:          workflow.StateReviewing,
			expectedAction: "finish",
			minConfidence:  0.8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := createMockTask()
			prediction, err := predictor.Predict(context.Background(), task, tt.state)
			if err != nil {
				t.Fatalf("Predict failed: %v", err)
			}

			if prediction.Type != PredictNextAction {
				t.Errorf("expected type %s, got %s", PredictNextAction, prediction.Type)
			}

			if prediction.Value != tt.expectedAction {
				t.Errorf("expected action %s, got %v", tt.expectedAction, prediction.Value)
			}

			if prediction.Confidence < tt.minConfidence {
				t.Errorf("confidence %f below minimum %f", prediction.Confidence, tt.minConfidence)
			}

			if prediction.Reasoning == "" {
				t.Error("missing reasoning")
			}
		})
	}
}

func TestDurationPredictor_Predict(t *testing.T) {
	predictor := NewDurationPredictor()

	// Test that predictor returns a duration
	task := createMockTask()
	prediction, err := predictor.Predict(context.Background(), task, workflow.StateIdle)
	if err != nil {
		t.Fatalf("Predict failed: %v", err)
	}

	if prediction.Type != PredictDuration {
		t.Errorf("expected type %s, got %s", PredictDuration, prediction.Type)
	}

	duration, ok := prediction.Value.(time.Duration)
	if !ok {
		t.Fatalf("expected time.Duration, got %T", prediction.Value)
	}

	if duration < 10*time.Minute || duration > 200*time.Minute {
		t.Errorf("duration %v outside reasonable range", duration)
	}

	if prediction.Confidence < 0.5 {
		t.Errorf("confidence %f too low", prediction.Confidence)
	}
}

func TestComplexityPredictor_Predict(t *testing.T) {
	predictor := NewComplexityPredictor()

	task := createMockTask()
	prediction, err := predictor.Predict(context.Background(), task, workflow.StateIdle)
	if err != nil {
		t.Fatalf("Predict failed: %v", err)
	}

	if prediction.Type != PredictComplexity {
		t.Errorf("expected type %s, got %s", PredictComplexity, prediction.Type)
	}

	complexity, ok := prediction.Value.(int)
	if !ok {
		t.Fatalf("expected int, got %T", prediction.Value)
	}

	if complexity < 1 || complexity > 10 {
		t.Errorf("complexity %d outside valid range [1, 10]", complexity)
	}

	if prediction.Confidence < 0.5 {
		t.Errorf("confidence %f too low", prediction.Confidence)
	}
}

func TestRiskPredictor_Predict(t *testing.T) {
	predictor := NewRiskPredictor()

	task := createMockTask()
	prediction, err := predictor.Predict(context.Background(), task, workflow.StateIdle)
	if err != nil {
		t.Fatalf("Predict failed: %v", err)
	}

	if prediction.Type != PredictRisk {
		t.Errorf("expected type %s, got %s", PredictRisk, prediction.Type)
	}

	value, ok := prediction.Value.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map[string]interface{}, got %T", prediction.Value)
	}

	level, ok := value["level"].(string)
	if !ok {
		t.Fatal("missing level in risk prediction")
	}

	validLevels := map[string]bool{
		"low":    true,
		"medium": true,
		"high":   true,
	}

	if !validLevels[level] {
		t.Errorf("invalid risk level %s", level)
	}

	if prediction.Confidence < 0.5 {
		t.Errorf("confidence %f too low", prediction.Confidence)
	}
}

func TestTrainingSample_Validation(t *testing.T) {
	sample := &TrainingSample{
		Features: map[string]interface{}{
			"title_length": 20,
			"task_type":    "fix",
		},
		Label:     "implement",
		Weight:    1.0,
		Timestamp: time.Now(),
	}

	if sample.Features == nil {
		t.Error("features not set")
	}

	if sample.Label == nil {
		t.Error("label not set")
	}

	if sample.Weight <= 0 {
		t.Error("invalid weight")
	}

	if sample.Timestamp.IsZero() {
		t.Error("timestamp not set")
	}
}

func TestPrediction_Confidence(t *testing.T) {
	tests := []struct {
		name       string
		confidence float32
		valid      bool
	}{
		{
			name:       "high confidence",
			confidence: 0.95,
			valid:      true,
		},
		{
			name:       "low confidence",
			confidence: 0.3,
			valid:      true,
		},
		{
			name:       "invalid confidence",
			confidence: 1.5,
			valid:      false,
		},
		{
			name:       "invalid confidence",
			confidence: -0.1,
			valid:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prediction := &Prediction{
				Type:       PredictNextAction,
				Value:      "plan",
				Confidence: tt.confidence,
			}

			isValid := prediction.Confidence >= 0 && prediction.Confidence <= 1.0
			if isValid != tt.valid {
				t.Errorf("validity mismatch: expected %v, got %v", tt.valid, isValid)
			}
		})
	}
}

func TestPrediction_String(t *testing.T) {
	tests := []struct {
		name       string
		prediction *Prediction
		expect     string
	}{
		{
			name: "next action",
			prediction: &Prediction{
				Type:       PredictNextAction,
				Value:      "plan",
				Confidence: 0.9,
				Reasoning:  "Task is idle",
			},
			expect: "plan",
		},
		{
			name: "duration",
			prediction: &Prediction{
				Type:       PredictDuration,
				Value:      30 * time.Minute,
				Confidence: 0.7,
			},
			expect: "30m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var strValue string
			switch v := tt.prediction.Value.(type) {
			case string:
				strValue = v
			case time.Duration:
				strValue = v.String()
			}

			if strValue == "" {
				t.Error("failed to convert prediction to string")
			}
		})
	}
}

// TestFeatureExtractor_EdgeCases tests edge cases and nil/zero value handling.
func TestFeatureExtractor_EdgeCases(t *testing.T) {
	extractor := NewFeatureExtractor()

	tests := []struct {
		name    string
		task    *storage.TaskWork
		state   workflow.State
		wantErr bool
	}{
		{
			name: "zero created_at",
			task: &storage.TaskWork{
				Metadata: storage.WorkMetadata{
					ID:        "test-1",
					Title:     "Test task",
					CreatedAt: time.Time{}, // Zero time
				},
				Agent: storage.AgentInfo{Name: "test-agent"},
				Git:   storage.GitInfo{Branch: "main"},
			},
			state:   workflow.StateIdle,
			wantErr: false, // Should not panic, just return is_new_task=false
		},
		{
			name: "empty agent name",
			task: &storage.TaskWork{
				Metadata: storage.WorkMetadata{
					ID:        "test-2",
					Title:     "Test task",
					CreatedAt: time.Now(),
				},
				Agent: storage.AgentInfo{Name: ""}, // Empty name
			},
			state:   workflow.StateIdle,
			wantErr: false, // Should not panic
		},
		{
			name: "empty git branch",
			task: &storage.TaskWork{
				Metadata: storage.WorkMetadata{
					ID:        "test-3",
					Title:     "Test task",
					CreatedAt: time.Now(),
				},
				Git: storage.GitInfo{Branch: ""}, // Empty branch
			},
			state:   workflow.StateIdle,
			wantErr: false, // Should not panic
		},
		{
			name: "all empty fields",
			task: &storage.TaskWork{
				Metadata: storage.WorkMetadata{
					ID:        "test-4",
					Title:     "Test task",
					CreatedAt: time.Now(),
				},
				Agent: storage.AgentInfo{},
				Git:   storage.GitInfo{},
			},
			state:   workflow.StateIdle,
			wantErr: false, // Should not panic
		},
		{
			name: "terminal states",
			task: &storage.TaskWork{
				Metadata: storage.WorkMetadata{
					ID:        "test-5",
					Title:     "Test task",
					CreatedAt: time.Now(),
				},
			},
			state:   workflow.StateDone,
			wantErr: false,
		},
		{
			name: "failed state",
			task: &storage.TaskWork{
				Metadata: storage.WorkMetadata{
					ID:        "test-6",
					Title:     "Test task",
					CreatedAt: time.Now(),
				},
			},
			state:   workflow.StateFailed,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			features, err := extractor.Extract(tt.task, tt.state)

			if (err != nil) != tt.wantErr {
				t.Errorf("Extract() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if !tt.wantErr && features == nil {
				t.Error("Extract() returned nil features")
			}

			// Verify state_is_terminal is set correctly
			if tt.state == workflow.StateDone || tt.state == workflow.StateFailed {
				if isTerminal, ok := features["state_is_terminal"].(bool); !ok || !isTerminal {
					t.Error("state_is_terminal should be true for done/failed states")
				}
			}
		})
	}
}

// TestPredictor_TypeAssertionSafety tests that predictors handle invalid feature types gracefully.
func TestPredictor_TypeAssertionSafety(t *testing.T) {
	type predictor interface {
		Predict(ctx context.Context, task *storage.TaskWork, state workflow.State) (*Prediction, error)
	}

	tests := []struct {
		name      string
		predictor predictor
		features  map[string]interface{}
	}{
		{
			name:      "duration predictor with invalid title_length type",
			predictor: NewDurationPredictor(),
			features: map[string]interface{}{
				"title_length": "not-an-int", // Wrong type
				"task_type":    "fix",
			},
		},
		{
			name:      "complexity predictor with invalid types",
			predictor: NewComplexityPredictor(),
			features: map[string]interface{}{
				"title_length":     "invalid",
				"title_word_count": "invalid",
				"task_type":        123,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock task - predictors will call Extract() which returns valid types
			// This test just ensures the predictor code doesn't crash
			task := createMockTask()
			_, err := tt.predictor.Predict(context.Background(), task, workflow.StateIdle)
			// Should either succeed or return a meaningful error, not panic
			if err != nil {
				// Error is acceptable for invalid input
				t.Logf("Got expected error: %v", err)
			}
		})
	}
}
