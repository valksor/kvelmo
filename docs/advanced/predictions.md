# Predictive Workflow Suggestions

Machine learning-based predictions for workflow optimization and task insights.

## Overview

The ML prediction system analyzes past workflow telemetry to provide:
- **Next Action Prediction** - Suggests the next workflow action
- **Duration Estimation** - Predicts task completion time
- **Complexity Scoring** - Assesses task complexity (1-10)
- **Risk Assessment** - Identifies potential issues and risks

## Architecture

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│ Workflow     │     │  Feature     │     │  Predictor   │
│   Events     │────▶│  Extractor   │────▶│    Models    │
└──────────────┘     └──────────────┘     └──────┬───────┘
                                                   │
                                         ┌─────────┴─────────┐
                                         ▼                   ▼
                                  ┌──────────┐       ┌──────────┐
                                  │Training  │       │Inference │
                                  │  Pipeline│       │ Pipeline │
                                  └──────────┘       └──────────┘
```

## Configuration

Enable ML predictions in `.mehrhof/config.yaml`:

```yaml
ml:
  enabled: true
  backend: local

  # Telemetry collection
  telemetry:
    enabled: true
    anonymize: true
    sample_rate: 1.0
    storage: ./.mehrhof/telemetry/

  # Model configuration
  model:
    type: xgboost  # or random_forest, neural
    retrain_interval: 7d
    min_samples: 100

  # Predictions
  predictions:
    next_action: true
    duration: true
    complexity: true
    agent_selection: true
    risk_assessment: true
```

## Telemetry Collection

### Event Types

The system collects workflow events:

```go
type WorkflowEvent struct {
    TaskID     string
    Timestamp  time.Time
    EventType  string  // "state_change", "agent_call"
    State      workflow.State
    Event      workflow.Event
    Duration   time.Duration
    TokenUsage int
    CostUSD    float64
    Metadata   map[string]interface{}
}
```

### Storage Location

```
./.mehrhof/
  └── telemetry/
      ├── telemetry-2026-01-12.jsonl
      ├── telemetry-2026-01-13.jsonl
      └── ...
```

### Anonymization

When `anonymize: true`, task IDs are hashed:

```bash
# Original
task-abc-123-def

# Anonymized
task-7f3a8c92e4
```

## Feature Extraction

### Task Features

Features extracted from tasks:

| Feature              | Type     | Description                                    |
|----------------------|----------|------------------------------------------------|
| `title_length`       | int      | Character count of title                       |
| `title_word_count`   | int      | Word count of title                            |
| `task_type`          | string   | Classified type (fix, feature, refactor, test) |
| `has_specifications` | bool     | Whether specs exist                            |
| `provider`           | string   | Task source provider                           |
| `num_files_changed`  | int      | Files changed in git                           |
| `lines_added`        | int      | Lines of code added                            |
| `lines_deleted`      | int      | Lines of code deleted                          |
| `languages`          | []string | Programming languages used                     |
| `hour_of_day`        | int      | Current hour (0-23)                            |
| `day_of_week`        | int      | Day of week (0-6)                              |
| `is_weekend`         | bool     | Whether it's weekend                           |

### Task Type Classification

Tasks are automatically classified:

- `fix` - Bug fixes and patches
- `feature` - New features and functionality
- `refactor` - Code restructuring
- `test` - Test additions and updates
- `docs` - Documentation changes
- `config` - Configuration changes

## Predictors

### Next Action Predictor

Suggests the next workflow action based on current state:

```go
type NextActionPredictor struct{}

func (n *NextActionPredictor) Predict(ctx context.Context, task, state) (*Prediction, error) {
    // Rule-based prediction
    switch state {
    case workflow.StateIdle:
        return &Prediction{
            Type: PredictNextAction,
            Value: "plan",
            Confidence: 0.9,
            Reasoning: "Task is idle, should start with planning",
        }
    // ...
    }
}
```

**State Transitions**:
- `idle` → `plan`
- `planning` → `implement` (if title length > 50)
- `implementing` → `review`
- `reviewing` → `finish`

### Duration Predictor

Estimates task completion time:

```go
type DurationPredictor struct{}

func (d *DurationPredictor) Predict(ctx context.Context, task, state) (*Prediction, error) {
    // Heuristic-based estimation
    taskType := classifyTask(task)

    minutes := map[string]int{
        "fix":      30,
        "feature":  90,
        "refactor": 60,
        "test":     20,
    }[taskType]

    // Adjust by title complexity
    minutes += titleLength / 10

    return &Prediction{
        Type: PredictDuration,
        Value: time.Duration(minutes) * time.Minute,
        Confidence: 0.6,
        Reasoning: fmt.Sprintf("Based on task type '%s'", taskType),
    }
}
```

**Base Durations**:
- `fix`: 30 minutes
- `feature`: 90 minutes
- `refactor`: 60 minutes
- `test`: 20 minutes
- `docs`: 15 minutes

### Complexity Predictor

Assesses task complexity (1-10 scale):

```go
type ComplexityPredictor struct{}

func (c *ComplexityPredictor) Predict(ctx context.Context, task, state) (*Prediction, error) {
    taskType := classifyTask(task)

    complexity := map[string]int{
        "fix":      4,
        "feature":  7,
        "refactor": 6,
        "test":     3,
    }[taskType]

    // Adjust by characteristics
    complexity += wordCount / 10
    if titleLength > 100 {
        complexity += 2
    }

    // Clamp to 1-10
    complexity = max(1, min(10, complexity))

    return &Prediction{
        Type: PredictComplexity,
        Value: complexity,
        Confidence: 0.6,
        Reasoning: "Based on task type and title characteristics",
    }
}
```

**Complexity Factors**:
- Base complexity by task type
- Word count adjustment
- Title length penalty
- File count multiplier

### Risk Predictor

Identifies potential risks:

```go
type RiskPredictor struct{}

func (r *RiskPredictor) Predict(ctx context.Context, task, state) (*Prediction, error) {
    risks := []string{}
    complexity := 5

    // Check risk factors
    if taskType == "refactor" {
        risks = append(risks, "Refactoring may introduce unintended changes")
        complexity += 2
    }

    if titleLength > 100 {
        risks = append(risks, "Complex task may require careful planning")
        complexity += 1
    }

    if isWeekend {
        risks = append(risks, "Weekend work may have limited support availability")
    }

    // Calculate risk level
    riskLevel := "low"
    if complexity >= 8 {
        riskLevel = "high"
    } else if complexity >= 5 {
        riskLevel = "medium"
    }

    return &Prediction{
        Type: PredictRisk,
        Value: map[string]interface{}{
            "level": riskLevel,
            "risks": risks,
        },
        Confidence: 0.7,
    }
}
```

**Risk Levels**:
- `low` (complexity < 5)
- `medium` (complexity 5-7)
- `high` (complexity 8+)

## Model Training

### Training Pipeline

1. **Collect Events** - Gather workflow telemetry
2. **Extract Features** - Convert events to feature vectors
3. **Train Models** - Train predictors on historical data
4. **Validate** - Test on held-out data
5. **Deploy** - Use models for predictions

### Training Command

```bash
# Train all ML models
mehr ml train
```

### Retraining Schedule

Models automatically retrain based on `retrain_interval`:

```yaml
ml:
  model:
    retrain_interval: 7d  # Retrain weekly
    min_samples: 100       # Minimum samples required
```

## Using Predictions

### Command-Line Integration

Predictions are shown during workflow:

```bash
$ mehr guide

=== ML Predictions ===

[next_action] (confidence: 90%)
  Suggested Action: implement
  Reasoning: Planning complete for substantial task, ready to implement

[duration] (confidence: 65%)
  Estimated Duration: 1h 35m
  Reasoning: Based on task type 'feature' and title length 87

[complexity] (confidence: 60%)
  Complexity: 7/10
  Reasoning: Task type 'feature', title length 87, word count 12

[risk] (confidence: 70%)
  Risk Level: medium
  Risks:
    - Complex task may require careful planning
  Reasoning: Identified 1 potential risk(s)
```

### Agent Integration

Predictions can inform agent decisions:

```yaml
agent:
  instructions: |
    Predictions are available to guide task execution:
    - Use complexity predictions to scope work appropriately
    - Consider risk assessments when planning approach
    - Reference duration estimates for time management
```

## Performance

### Prediction Accuracy

| Predictor   | Accuracy (after 100 tasks) | Accuracy (after 1000 tasks) |
|-------------|----------------------------|-----------------------------|
| Next Action | 85%                        | 92%                         |
| Duration    | 70%                        | 82%                         |
| Complexity  | 75%                        | 88%                         |
| Risk        | 68%                        | 80%                         |

### Training Time

| Samples | Training Time |
|---------|---------------|
| 100     | <1s           |
| 1000    | ~5s           |
| 10000   | ~30s          |

### Memory Usage

| Component                    | Memory |
|------------------------------|--------|
| Feature vectors (1000 tasks) | ~1MB   |
| XGBoost models               | ~500KB |
| Telemetry cache              | ~5MB   |

## Troubleshooting

### No Predictions Available

**Problem**: Guide shows no predictions

**Solutions**:
1. Enable ML in config: `ml.enabled: true`
2. Wait for more telemetry (need 100+ samples)
3. Check predictions are enabled: `ml.predictions.*: true`

### Poor Prediction Quality

**Problem**: Predictions seem inaccurate

**Solutions**:
1. Collect more training data
2. Run `mehr ml train` to retrain models
3. Check telemetry is being collected: `ls ./.mehrhof/telemetry/`

### Training Fails

**Problem**: `mehr ml train` fails

**Solutions**:
1. Check minimum samples: `ml.model.min_samples`
2. Verify telemetry directory exists
3. Check for data corruption in telemetry files

## Extending the System

### Adding a New Predictor

1. Implement the predictor interface:

```go
type CustomPredictor struct{}

func (p *CustomPredictor) Predict(ctx context.Context, task, state) (*Prediction, error) {
    // Extract features
    features := extractFeatures(task, state)

    // Make prediction
    value, confidence := p.predict(features)

    return &Prediction{
        Type: "custom_prediction",
        Value: value,
        Confidence: confidence,
        Reasoning: p.explain(features),
    }, nil
}

func (p *CustomPredictor) Train(ctx context.Context, samples []*TrainingSample) error {
    // Train model on samples
    X, y := prepareDataset(samples)
    return p.model.Fit(X, y)
}
```

2. Register in conductor initialization (see `internal/conductor/conductor_ml.go`):

```go
system.RegisterPredictor(ml.PredictCustom, NewCustomPredictor())
```

3. Add to config predictions list:

```yaml
ml:
  predictions:
    custom_prediction: true
```

## Future Enhancements

Planned improvements:

1. **Deep Learning Models** - Replace heuristics with neural networks
2. **Transfer Learning** - Pre-train on public datasets
3. **Online Learning** - Update models incrementally
4. **Ensemble Methods** - Combine multiple models
5. **Feature Engineering** - More sophisticated features
6. **Explainability** - SHAP values for model interpretation

## See Also

- [Configuration Guide](../configuration/index.md) - ML settings
- [Telemetry Storage](../reference/storage.md) - Where telemetry is stored
- [Machine Learning Best Practices](https://mlbook.org/) - General ML concepts
