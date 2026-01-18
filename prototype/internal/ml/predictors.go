package ml

import (
	"context"
	"fmt"
	"time"

	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/workflow"
)

// NextActionPredictor predicts the next workflow action.
type NextActionPredictor struct {
	extractor *FeatureExtractor
}

// NewNextActionPredictor creates a new next action predictor.
func NewNextActionPredictor() *NextActionPredictor {
	return &NextActionPredictor{
		extractor: NewFeatureExtractor(),
	}
}

// Predict predicts the next action.
func (n *NextActionPredictor) Predict(ctx context.Context, task *storage.TaskWork, state workflow.State) (*Prediction, error) {
	features, err := n.extractor.Extract(task, state)
	if err != nil {
		return nil, err
	}

	// Simple rule-based prediction for now
	// In production, this would use a trained model
	var nextAction string
	var confidence float32
	var reasoning string

	switch state {
	case workflow.StateIdle:
		nextAction = "plan"
		confidence = 0.9
		reasoning = "Task is idle, should start with planning"
	case workflow.StatePlanning:
		titleLen, ok := features["title_length"].(int)
		if !ok {
			titleLen = 0
		}
		if task.Metadata.Title != "" && titleLen > 50 {
			nextAction = "implement"
			confidence = 0.85
			reasoning = "Planning complete for substantial task, ready to implement"
		} else {
			nextAction = "plan"
			confidence = 0.7
			reasoning = "Task needs more detailed planning"
		}
	case workflow.StateImplementing:
		nextAction = "review"
		confidence = 0.8
		reasoning = "Implementation complete, should review"
	case workflow.StateReviewing:
		nextAction = "finish"
		confidence = 0.9
		reasoning = "Review complete, ready to finish"
	case workflow.StateDone, workflow.StateFailed, workflow.StateWaiting, workflow.StateCheckpointing, workflow.StateReverting, workflow.StateRestoring:
		nextAction = "continue"
		confidence = 0.5
		reasoning = "Terminal or waiting state, continue current work"
	}

	return &Prediction{
		Type:       PredictNextAction,
		Value:      nextAction,
		Confidence: confidence,
		Reasoning:  reasoning,
		Metadata:   features,
		Timestamp:  time.Now(),
	}, nil
}

// Train trains the predictor with samples.
func (n *NextActionPredictor) Train(ctx context.Context, samples []*TrainingSample) error {
	// For now, just log - would train actual model in production
	fmt.Printf("Training NextActionPredictor with %d samples\n", len(samples))

	return nil
}

// DurationPredictor predicts task completion time.
type DurationPredictor struct {
	extractor *FeatureExtractor
}

// NewDurationPredictor creates a new duration predictor.
func NewDurationPredictor() *DurationPredictor {
	return &DurationPredictor{
		extractor: NewFeatureExtractor(),
	}
}

// Predict predicts task duration.
func (d *DurationPredictor) Predict(ctx context.Context, task *storage.TaskWork, state workflow.State) (*Prediction, error) {
	features, err := d.extractor.Extract(task, state)
	if err != nil {
		return nil, err
	}

	// Simple heuristic-based prediction
	titleLen, ok := features["title_length"].(int)
	if !ok {
		titleLen = 0
	}
	taskType, ok := features["task_type"].(string)
	if !ok {
		taskType = "other"
	}

	var minutes int
	var confidence float32

	// Base duration by task type
	switch taskType {
	case "fix":
		minutes = 30
		confidence = 0.7
	case "feature":
		minutes = 90
		confidence = 0.6
	case "refactor":
		minutes = 60
		confidence = 0.65
	case "test":
		minutes = 20
		confidence = 0.75
	default:
		minutes = 45
		confidence = 0.5
	}

	// Adjust by title complexity
	minutes += titleLen / 10

	return &Prediction{
		Type:       PredictDuration,
		Value:      time.Duration(minutes) * time.Minute,
		Confidence: confidence,
		Reasoning:  fmt.Sprintf("Based on task type '%s' and title length %d", taskType, titleLen),
		Metadata:   features,
		Timestamp:  time.Now(),
	}, nil
}

// Train trains the predictor with samples.
func (d *DurationPredictor) Train(ctx context.Context, samples []*TrainingSample) error {
	fmt.Printf("Training DurationPredictor with %d samples\n", len(samples))

	return nil
}

// ComplexityPredictor predicts task complexity (1-10 scale).
type ComplexityPredictor struct {
	extractor *FeatureExtractor
}

// NewComplexityPredictor creates a new complexity predictor.
func NewComplexityPredictor() *ComplexityPredictor {
	return &ComplexityPredictor{
		extractor: NewFeatureExtractor(),
	}
}

// Predict predicts task complexity.
func (c *ComplexityPredictor) Predict(ctx context.Context, task *storage.TaskWork, state workflow.State) (*Prediction, error) {
	features, err := c.extractor.Extract(task, state)
	if err != nil {
		return nil, err
	}

	titleLen, ok := features["title_length"].(int)
	if !ok {
		titleLen = 0
	}
	wordCount, ok := features["title_word_count"].(int)
	if !ok {
		wordCount = 0
	}
	taskType, ok := features["task_type"].(string)
	if !ok {
		taskType = "other"
	}

	var complexity int
	var confidence float32

	// Base complexity by task type
	switch taskType {
	case "fix":
		complexity = 4
	case "feature":
		complexity = 7
	case "refactor":
		complexity = 6
	case "test":
		complexity = 3
	default:
		complexity = 5
	}

	// Adjust by title characteristics
	complexity += wordCount / 10
	if titleLen > 100 {
		complexity += 2
	}

	// Clamp to 1-10
	if complexity < 1 {
		complexity = 1
	}
	if complexity > 10 {
		complexity = 10
	}

	confidence = 0.6

	return &Prediction{
		Type:       PredictComplexity,
		Value:      complexity,
		Confidence: confidence,
		Reasoning:  fmt.Sprintf("Task type '%s', title length %d, word count %d", taskType, titleLen, wordCount),
		Metadata:   features,
		Timestamp:  time.Now(),
	}, nil
}

// Train trains the predictor with samples.
func (c *ComplexityPredictor) Train(ctx context.Context, samples []*TrainingSample) error {
	fmt.Printf("Training ComplexityPredictor with %d samples\n", len(samples))

	return nil
}

// RiskPredictor predicts potential risks.
type RiskPredictor struct {
	extractor *FeatureExtractor
}

// NewRiskPredictor creates a new risk predictor.
func NewRiskPredictor() *RiskPredictor {
	return &RiskPredictor{
		extractor: NewFeatureExtractor(),
	}
}

// Predict predicts task risks.
func (r *RiskPredictor) Predict(ctx context.Context, task *storage.TaskWork, state workflow.State) (*Prediction, error) {
	features, err := r.extractor.Extract(task, state)
	if err != nil {
		return nil, err
	}

	risks := make([]string, 0)
	complexity := 5 // Default

	// Check for risk factors
	taskType, ok := features["task_type"].(string)
	if !ok {
		taskType = "other"
	}
	if taskType == "refactor" {
		risks = append(risks, "Refactoring may introduce unintended changes")
		complexity += 2
	}

	titleLen, ok := features["title_length"].(int)
	if !ok {
		titleLen = 0
	}
	if titleLen > 100 {
		risks = append(risks, "Complex task may require careful planning")
		complexity += 1
	}

	isWeekend, ok := features["is_weekend"].(bool)
	if !ok {
		isWeekend = false
	}
	if isWeekend {
		risks = append(risks, "Weekend work may have limited support availability")
	}

	// Calculate risk level
	var riskLevel string
	var confidence float32

	if complexity >= 8 {
		riskLevel = "high"
		confidence = 0.7
	} else if complexity >= 5 {
		riskLevel = "medium"
		confidence = 0.6
	} else {
		riskLevel = "low"
		confidence = 0.8
	}

	if len(risks) == 0 {
		risks = append(risks, "No specific risks identified")
	}

	return &Prediction{
		Type: PredictRisk,
		Value: map[string]interface{}{
			"level": riskLevel,
			"risks": risks,
		},
		Confidence: confidence,
		Reasoning:  fmt.Sprintf("Identified %d potential risk(s)", len(risks)),
		Metadata:   features,
		Timestamp:  time.Now(),
	}, nil
}

// Train trains the predictor with samples.
func (r *RiskPredictor) Train(ctx context.Context, samples []*TrainingSample) error {
	fmt.Printf("Training RiskPredictor with %d samples\n", len(samples))

	return nil
}
