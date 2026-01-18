package ml

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/workflow"
)

// PredictionType defines the type of prediction.
type PredictionType string

const (
	PredictNextAction     PredictionType = "next_action"
	PredictDuration       PredictionType = "duration"
	PredictComplexity     PredictionType = "complexity"
	PredictAgentSelection PredictionType = "agent_selection"
	PredictRisk           PredictionType = "risk"
)

// Prediction represents a machine learning prediction.
type Prediction struct {
	Type       PredictionType         `json:"type"`
	Value      interface{}            `json:"value"`
	Confidence float32                `json:"confidence"`
	Reasoning  string                 `json:"reasoning,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
}

// Predictor is the interface for ML predictions.
type Predictor interface {
	Predict(ctx context.Context, task *storage.TaskWork, state workflow.State) (*Prediction, error)
	Train(ctx context.Context, samples []*TrainingSample) error
}

// TrainingSample represents a training data point.
type TrainingSample struct {
	Features  map[string]interface{} `json:"features"`
	Label     interface{}            `json:"label"`
	Weight    float32                `json:"weight"`
	Timestamp time.Time              `json:"timestamp"`
	TaskID    string                 `json:"task_id"`
}

// TelemetryCollector collects workflow event data for ML training.
type TelemetryCollector struct {
	mu        sync.Mutex
	storage   TelemetryStorage
	enabled   bool
	anonymize bool
}

// TelemetryStorage stores and retrieves telemetry data.
type TelemetryStorage interface {
	StoreEvent(ctx context.Context, event *WorkflowEvent) error
	LoadEvents(ctx context.Context, opts EventQueryOptions) ([]*WorkflowEvent, error)
	LoadSamples(ctx context.Context) ([]*TrainingSample, error)
}

// WorkflowEvent represents a single workflow event for telemetry.
type WorkflowEvent struct {
	TaskID     string                 `json:"task_id"`
	Timestamp  time.Time              `json:"timestamp"`
	EventType  string                 `json:"event_type"` // "state_change", "agent_call", etc.
	State      workflow.State         `json:"state"`
	Event      workflow.Event         `json:"event"`
	Duration   time.Duration          `json:"duration,omitempty"`
	TokenUsage int                    `json:"token_usage,omitempty"`
	CostUSD    float64                `json:"cost_usd,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// EventQueryOptions specifies filters for querying events.
type EventQueryOptions struct {
	TaskID    string    `json:"task_id,omitempty"`
	StartTime time.Time `json:"start_time,omitempty"`
	EndTime   time.Time `json:"end_time,omitempty"`
	EventType string    `json:"event_type,omitempty"`
	Limit     int       `json:"limit,omitempty"`
}

// NewTelemetryCollector creates a new telemetry collector.
func NewTelemetryCollector(storage TelemetryStorage, anonymize bool) *TelemetryCollector {
	return &TelemetryCollector{
		storage:   storage,
		enabled:   true,
		anonymize: anonymize,
	}
}

// RecordEvent records a workflow event.
func (t *TelemetryCollector) RecordEvent(ctx context.Context, event *WorkflowEvent) error {
	if !t.enabled {
		return nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.anonymize {
		event = t.anonymizeEvent(event)
	}

	return t.storage.StoreEvent(ctx, event)
}

// anonymizeEvent removes sensitive information from an event.
func (t *TelemetryCollector) anonymizeEvent(event *WorkflowEvent) *WorkflowEvent {
	anonymized := *event
	anonymized.TaskID = hashTaskID(event.TaskID)

	return &anonymized
}

// hashTaskID creates a consistent hash for a task ID using SHA-256.
func hashTaskID(taskID string) string {
	h := sha256.Sum256([]byte(taskID))
	// Use first 8 bytes of the hash for a shorter but still secure identifier
	return "task-" + hex.EncodeToString(h[:8])
}

// FileTelemetryStorage stores telemetry data on disk.
type FileTelemetryStorage struct {
	baseDir string
	mu      sync.Mutex
}

// NewFileTelemetryStorage creates a new file-based telemetry storage.
func NewFileTelemetryStorage(baseDir string) (*FileTelemetryStorage, error) {
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("create telemetry directory: %w", err)
	}

	return &FileTelemetryStorage{
		baseDir: baseDir,
	}, nil
}

// StoreEvent stores an event to disk.
func (f *FileTelemetryStorage) StoreEvent(ctx context.Context, event *WorkflowEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Store events in daily files
	date := event.Timestamp.Format("2006-01-02")
	filename := fmt.Sprintf("telemetry-%s.jsonl", date)

	filePath := filepath.Join(f.baseDir, filename)

	// Append event to file
	line := eventToString(event) + "\n"

	// Open file with O_CREATE|O_APPEND to handle both create and append atomically
	// This avoids the TOCTOU race condition between stat and file operations
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open telemetry file: %w", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			// Log but don't fail the operation
			fmt.Fprintf(os.Stderr, "warning: failed to close telemetry file: %v\n", cerr)
		}
	}()

	if _, err := file.WriteString(line); err != nil {
		return fmt.Errorf("write telemetry: %w", err)
	}

	return nil
}

// LoadEvents loads events from disk, applying filters from opts.
func (f *FileTelemetryStorage) LoadEvents(ctx context.Context, opts EventQueryOptions) ([]*WorkflowEvent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Determine date range for files to load
	startDate := opts.StartTime.Format("2006-01-02")
	endDate := opts.EndTime.Format("2006-01-02")

	// If no date range specified, load last 30 days
	if opts.StartTime.IsZero() {
		endDate = time.Now().Format("2006-01-02")
		startDate = time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	}

	var events []*WorkflowEvent

	// Read all telemetry files in date range
	current := startDate
	for current <= endDate {
		filename := fmt.Sprintf("telemetry-%s.jsonl", current)
		filePath := filepath.Join(f.baseDir, filename)

		fileEvents, err := f.loadEventsFromFile(filePath, opts)
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("load events from %s: %w", filename, err)
		}
		events = append(events, fileEvents...)

		// Move to next day
		t, _ := time.Parse("2006-01-02", current)
		current = t.AddDate(0, 0, 1).Format("2006-01-02")
	}

	// Apply limit
	if opts.Limit > 0 && len(events) > opts.Limit {
		events = events[:opts.Limit]
	}

	return events, nil
}

// loadEventsFromFile loads events from a single JSONL file.
func (f *FileTelemetryStorage) loadEventsFromFile(filePath string, opts EventQueryOptions) ([]*WorkflowEvent, error) {
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []*WorkflowEvent{}, nil
		}

		return nil, err
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			// Log but don't fail - we already have the data we need
			fmt.Fprintf(os.Stderr, "warning: failed to close telemetry file: %v\n", cerr)
		}
	}()

	var events []*WorkflowEvent
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Bytes()
		var event WorkflowEvent
		if err := json.Unmarshal(line, &event); err != nil {
			// Log preview of malformed line (up to 100 chars)
			preview := string(line)
			if len(preview) > 100 {
				preview = preview[:100] + "..."
			}
			slog.Warn("Skipping malformed telemetry line", "error", err, "line_preview", preview)
			// Skip malformed lines
			continue
		}

		// Apply filters
		if opts.TaskID != "" && event.TaskID != opts.TaskID {
			continue
		}
		if !opts.StartTime.IsZero() && event.Timestamp.Before(opts.StartTime) {
			continue
		}
		if !opts.EndTime.IsZero() && event.Timestamp.After(opts.EndTime) {
			continue
		}
		if opts.EventType != "" && event.EventType != opts.EventType {
			continue
		}

		events = append(events, &event)
	}

	return events, scanner.Err()
}

// LoadSamples loads training samples from stored events.
func (f *FileTelemetryStorage) LoadSamples(ctx context.Context) ([]*TrainingSample, error) {
	events, err := f.LoadEvents(ctx, EventQueryOptions{})
	if err != nil {
		return nil, fmt.Errorf("load events: %w", err)
	}

	// Group events by task to create samples
	taskEvents := make(map[string][]*WorkflowEvent)
	for _, event := range events {
		taskEvents[event.TaskID] = append(taskEvents[event.TaskID], event)
	}

	var samples []*TrainingSample
	for taskID, taskEventList := range taskEvents {
		// Create sample from task events
		sample := eventsToSample(taskID, taskEventList)
		if sample != nil {
			samples = append(samples, sample)
		}
	}

	return samples, nil
}

// eventsToSample converts a group of events for a task into a training sample.
func eventsToSample(taskID string, events []*WorkflowEvent) *TrainingSample {
	if len(events) == 0 {
		return nil
	}

	// Calculate duration from first to last event
	duration := events[len(events)-1].Timestamp.Sub(events[0].Timestamp)

	// Count state transitions and agent calls
	var stateChanges, agentCalls int
	var totalTokens int
	var totalCost float64

	for _, event := range events {
		switch event.EventType {
		case "state_change":
			stateChanges++
		case "agent_call":
			agentCalls++
		}
		totalTokens += event.TokenUsage
		totalCost += event.CostUSD
	}

	// Create features map
	features := map[string]interface{}{
		"state_changes":  stateChanges,
		"agent_calls":    agentCalls,
		"total_tokens":   totalTokens,
		"total_cost_usd": totalCost,
		"duration_sec":   duration.Seconds(),
	}

	// Label is the final state
	label := events[len(events)-1].State

	return &TrainingSample{
		Features:  features,
		Label:     string(label),
		Weight:    1.0,
		Timestamp: events[0].Timestamp,
		TaskID:    taskID,
	}
}

// eventToString converts an event to JSON for storage.
func eventToString(event *WorkflowEvent) string {
	data, err := json.Marshal(event)
	if err != nil {
		// Fallback to manual encoding if JSON fails
		return fmt.Sprintf(`{"task_id":"%s","timestamp":"%s","type":"%s","state":"%s","event":"%s"}`,
			event.TaskID,
			event.Timestamp.Format(time.RFC3339),
			event.EventType,
			string(event.State),
			string(event.Event),
		)
	}

	return string(data)
}

// MLSystem manages all ML predictors.
type MLSystem struct {
	predictors map[PredictionType]Predictor
	collector  *TelemetryCollector
	storage    TelemetryStorage
	enabled    bool
}

// NewMLSystem creates a new ML system.
func NewMLSystem(storage TelemetryStorage, anonymize bool) *MLSystem {
	collector := NewTelemetryCollector(storage, anonymize)

	return &MLSystem{
		predictors: make(map[PredictionType]Predictor),
		collector:  collector,
		storage:    storage,
		enabled:    true,
	}
}

// RegisterPredictor registers a predictor for a prediction type.
func (m *MLSystem) RegisterPredictor(pType PredictionType, predictor Predictor) {
	m.predictors[pType] = predictor
}

// GetPrediction gets a prediction for a specific type.
func (m *MLSystem) GetPrediction(ctx context.Context, pType PredictionType, task *storage.TaskWork, state workflow.State) (*Prediction, error) {
	if !m.enabled {
		return nil, errors.New("ML system not enabled")
	}

	predictor, ok := m.predictors[pType]
	if !ok {
		return nil, fmt.Errorf("no predictor registered for type: %s", pType)
	}

	prediction, err := predictor.Predict(ctx, task, state)
	if err != nil {
		return nil, fmt.Errorf("predict: %w", err)
	}

	return prediction, nil
}

// GetAllPredictions gets all available predictions.
func (m *MLSystem) GetAllPredictions(ctx context.Context, task *storage.TaskWork, state workflow.State) ([]*Prediction, error) {
	if !m.enabled {
		return nil, nil
	}

	var predictions []*Prediction

	for pType, predictor := range m.predictors {
		prediction, err := predictor.Predict(ctx, task, state)
		if err != nil {
			continue // Skip failed predictions
		}
		prediction.Type = pType
		prediction.Timestamp = time.Now()
		predictions = append(predictions, prediction)
	}

	return predictions, nil
}

// RecordEvent records a workflow event.
func (m *MLSystem) RecordEvent(ctx context.Context, event *WorkflowEvent) error {
	return m.collector.RecordEvent(ctx, event)
}

// TrainAll trains all predictors with collected data.
func (m *MLSystem) TrainAll(ctx context.Context) error {
	samples, err := m.storage.LoadSamples(ctx)
	if err != nil {
		return fmt.Errorf("load samples: %w", err)
	}

	if len(samples) == 0 {
		return errors.New("no training samples available")
	}

	for pType, predictor := range m.predictors {
		if err := predictor.Train(ctx, samples); err != nil {
			return fmt.Errorf("train %s predictor: %w", pType, err)
		}
	}

	return nil
}
