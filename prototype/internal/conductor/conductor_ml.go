package conductor

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/ml"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/workflow"
)

// MLSystem holds the ML prediction system.
type MLSystem struct {
	system       *ml.MLSystem
	config       *storage.MLSettings
	telemetryDir string
}

// InitializeML initializes the ML system from workspace config.
func (c *Conductor) InitializeML(ctx context.Context) error {
	cfg, err := c.workspace.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if cfg == nil || cfg.ML == nil || !cfg.ML.Enabled {
		return nil // ML disabled
	}

	// Create telemetry storage
	telemetryDir := filepath.Join(c.workspace.Root(), ".mehrhof", "telemetry")
	storage, err := ml.NewFileTelemetryStorage(telemetryDir)
	if err != nil {
		return fmt.Errorf("create telemetry storage: %w", err)
	}

	// Create ML system
	system := ml.NewMLSystem(storage, cfg.ML.Telemetry.Anonymize)

	// Register predictors
	system.RegisterPredictor(ml.PredictNextAction, ml.NewNextActionPredictor())
	system.RegisterPredictor(ml.PredictDuration, ml.NewDurationPredictor())
	system.RegisterPredictor(ml.PredictComplexity, ml.NewComplexityPredictor())
	system.RegisterPredictor(ml.PredictRisk, ml.NewRiskPredictor())

	// Store in conductor
	c.ml = &MLSystem{
		system:       system,
		config:       cfg.ML,
		telemetryDir: telemetryDir,
	}

	c.publishProgress("ML prediction system initialized", 100)

	return nil
}

// GetMLPredictions gets all available predictions for the current task.
func (c *Conductor) GetMLPredictions(ctx context.Context) ([]*ml.Prediction, error) {
	if c.ml == nil {
		return nil, nil // ML disabled
	}

	work := c.GetTaskWork()
	if work == nil {
		return nil, nil
	}

	state := c.machine.State()
	if state == "" {
		state = workflow.StateIdle
	}

	predictions, err := c.ml.system.GetAllPredictions(ctx, work, state)
	if err != nil {
		return nil, fmt.Errorf("get predictions: %w", err)
	}

	return predictions, nil
}

// RecordWorkflowEvent records a workflow event for telemetry.
func (c *Conductor) RecordWorkflowEvent(ctx context.Context, event *ml.WorkflowEvent) error {
	if c.ml == nil || !c.ml.config.Telemetry.Enabled {
		return nil // ML disabled or telemetry disabled
	}

	return c.ml.system.RecordEvent(ctx, event)
}

// GetNextActionPrediction predicts the next workflow action.
func (c *Conductor) GetNextActionPrediction(ctx context.Context) (*ml.Prediction, error) {
	if c.ml == nil {
		return nil, errors.New("ML system not initialized")
	}

	work := c.GetTaskWork()
	if work == nil {
		return nil, errors.New("no active task")
	}

	state := c.machine.State()
	if state == "" {
		state = workflow.StateIdle
	}

	return c.ml.system.GetPrediction(ctx, ml.PredictNextAction, work, state)
}

// GetDurationPrediction predicts task completion time.
func (c *Conductor) GetDurationPrediction(ctx context.Context) (*ml.Prediction, error) {
	if c.ml == nil {
		return nil, errors.New("ML system not initialized")
	}

	work := c.GetTaskWork()
	if work == nil {
		return nil, errors.New("no active task")
	}

	state := c.machine.State()
	if state == "" {
		state = workflow.StateIdle
	}

	return c.ml.system.GetPrediction(ctx, ml.PredictDuration, work, state)
}

// TrainMLModels trains all ML predictors with collected data.
func (c *Conductor) TrainMLModels(ctx context.Context) error {
	if c.ml == nil {
		return errors.New("ML system not initialized")
	}

	c.publishProgress("Training ML models...", 0)

	if err := c.ml.system.TrainAll(ctx); err != nil {
		return fmt.Errorf("train models: %w", err)
	}

	c.publishProgress("ML models trained successfully", 100)

	return nil
}

// FormatPredictions formats predictions for display.
func (c *Conductor) FormatPredictions(predictions []*ml.Prediction) string {
	if len(predictions) == 0 {
		return "No predictions available."
	}

	var sb strings.Builder
	sb.WriteString("\n=== ML Predictions ===\n")

	for _, pred := range predictions {
		sb.WriteString(fmt.Sprintf("\n[%s] (confidence: %.0f%%)\n", pred.Type, pred.Confidence*100))

		switch pred.Type {
		case ml.PredictNextAction:
			sb.WriteString(fmt.Sprintf("  Suggested Action: %s\n", pred.Value))
		case ml.PredictDuration:
			if duration, ok := pred.Value.(time.Duration); ok {
				sb.WriteString(fmt.Sprintf("  Estimated Duration: %s\n", duration.Round(time.Minute)))
			} else {
				sb.WriteString(fmt.Sprintf("  Estimated Duration: %v\n", pred.Value))
			}
		case ml.PredictComplexity:
			sb.WriteString(fmt.Sprintf("  Complexity: %d/10\n", pred.Value))
		case ml.PredictAgentSelection:
			sb.WriteString(fmt.Sprintf("  Recommended Agent: %s\n", pred.Value))
		case ml.PredictRisk:
			if value, ok := pred.Value.(map[string]interface{}); ok {
				sb.WriteString(fmt.Sprintf("  Risk Level: %s\n", value["level"]))
				if risks, risksOk := value["risks"].([]string); risksOk {
					for _, risk := range risks {
						sb.WriteString(fmt.Sprintf("    - %s\n", risk))
					}
				}
			} else {
				sb.WriteString(fmt.Sprintf("  Value: %v\n", pred.Value))
			}
		default:
			sb.WriteString(fmt.Sprintf("  Value: %v\n", pred.Value))
		}

		if pred.Reasoning != "" {
			sb.WriteString(fmt.Sprintf("  Reasoning: %s\n", pred.Reasoning))
		}
	}

	return sb.String()
}
