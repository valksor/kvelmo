package export

import (
	"context"
	"errors"
	"testing"

	"github.com/valksor/go-mehrhof/internal/agent"
)

// mockAgent is a mock agent for testing.
type mockAgent struct {
	response string
	err      error
}

func (m *mockAgent) Name() string {
	return "mock"
}

func (m *mockAgent) Run(ctx context.Context, prompt string) (*agent.Response, error) {
	if m.err != nil {
		return nil, m.err
	}

	return &agent.Response{
		Summary: m.response,
	}, nil
}

func (m *mockAgent) RunStream(ctx context.Context, prompt string) (<-chan agent.Event, <-chan error) {
	eventCh := make(chan agent.Event)
	errCh := make(chan error, 1)
	close(eventCh)
	close(errCh)

	return eventCh, errCh
}

func (m *mockAgent) RunWithCallback(ctx context.Context, prompt string, cb agent.StreamCallback) (*agent.Response, error) {
	if m.err != nil {
		return nil, m.err
	}

	return &agent.Response{Summary: m.response}, nil
}

func (m *mockAgent) Available() error {
	return nil
}

func (m *mockAgent) WithEnv(key, value string) agent.Agent {
	return m
}

func (m *mockAgent) WithArgs(args ...string) agent.Agent {
	return m
}

// TestParseProjectPlanWithSchema_SchemaExtraction tests schema-driven extraction.
func TestParseProjectPlanWithSchema_SchemaExtraction(t *testing.T) {
	ctx := context.Background()
	agent := &mockAgent{
		response: `{"tasks":[{"id":"task-1","title":"Test Task","priority":1,"status":"ready"}],"questions":[],"blockers":[]}`,
	}

	plan := ParseProjectPlanWithSchema(ctx, "some content", agent)

	if plan == nil {
		t.Fatal("ParseProjectPlanWithSchema returned nil")
	}

	if len(plan.Tasks) != 1 {
		t.Errorf("got %d tasks, want 1", len(plan.Tasks))
	}

	if plan.Tasks[0].ID != "task-1" {
		t.Errorf("got task ID %q, want 'task-1'", plan.Tasks[0].ID)
	}

	if plan.Tasks[0].Title != "Test Task" {
		t.Errorf("got task title %q, want 'Test Task'", plan.Tasks[0].Title)
	}
}

// TestParseProjectPlanWithSchema_SchemaExtractionWithMultipleTasks tests extraction of multiple tasks.
func TestParseProjectPlanWithSchema_SchemaExtractionWithMultipleTasks(t *testing.T) {
	ctx := context.Background()
	agent := &mockAgent{
		response: `{
			"tasks": [
				{"id":"task-1","title":"First Task","priority":1,"status":"ready"},
				{"id":"task-2","title":"Second Task","priority":2,"status":"pending","depends_on":["task-1"]}
			],
			"questions": ["Question 1?"],
			"blockers": []
		}`,
	}

	plan := ParseProjectPlanWithSchema(ctx, "some content", agent)

	if len(plan.Tasks) != 2 {
		t.Fatalf("got %d tasks, want 2", len(plan.Tasks))
	}

	if plan.Tasks[0].ID != "task-1" {
		t.Errorf("first task ID = %q, want 'task-1'", plan.Tasks[0].ID)
	}

	if plan.Tasks[1].ID != "task-2" {
		t.Errorf("second task ID = %q, want 'task-2'", plan.Tasks[1].ID)
	}

	if len(plan.Tasks[1].DependsOn) != 1 || plan.Tasks[1].DependsOn[0] != "task-1" {
		t.Errorf("second task depends_on = %v, want [task-1]", plan.Tasks[1].DependsOn)
	}

	if len(plan.Questions) != 1 || plan.Questions[0] != "Question 1?" {
		t.Errorf("questions = %v, want ['Question 1?']", plan.Questions)
	}
}

// TestParseProjectPlanWithSchema_SchemaExtractionFails_fallsBackToRegex tests that regex parsing is used when schema extraction fails.
func TestParseProjectPlanWithSchema_SchemaExtractionFails_fallsBackToRegex(t *testing.T) {
	ctx := context.Background()
	agent := &mockAgent{
		err: errors.New("agent unavailable"),
	}

	// Content that regex can parse
	content := `## Tasks
### task-1: Test Task
- **Priority**: 1
- **Status**: ready
- **Description**: Test description
`

	plan := ParseProjectPlanWithSchema(ctx, content, agent)

	if plan == nil {
		t.Fatal("ParseProjectPlanWithSchema returned nil")
	}

	if len(plan.Tasks) != 1 {
		t.Errorf("got %d tasks, want 1 (regex fallback)", len(plan.Tasks))
	}

	if plan.Tasks[0].ID != "task-1" {
		t.Errorf("got task ID %q, want 'task-1' (regex fallback)", plan.Tasks[0].ID)
	}
}

// TestParseProjectPlanWithSchema_NoAgent_fallsBackToRegex tests that nil agent uses regex parsing.
func TestParseProjectPlanWithSchema_NoAgent_fallsBackToRegex(t *testing.T) {
	ctx := context.Background()

	// Content that regex can parse
	content := `## Tasks
### task-1: Test Task
- **Priority**: 1
- **Status**: ready
`

	plan := ParseProjectPlanWithSchema(ctx, content, nil)

	if plan == nil {
		t.Fatal("ParseProjectPlanWithSchema returned nil")
	}

	if len(plan.Tasks) != 1 {
		t.Errorf("got %d tasks, want 1 (regex fallback)", len(plan.Tasks))
	}
}

// TestParseProjectPlanWithSchema_SchemaReturnsInvalidJson_fallsBackToRegex tests that invalid JSON falls back to regex.
func TestParseProjectPlanWithSchema_SchemaReturnsInvalidJson_fallsBackToRegex(t *testing.T) {
	ctx := context.Background()
	agent := &mockAgent{
		response: `this is not valid json`,
	}

	// Content that regex can parse
	content := `## Tasks
### task-1: Test Task
- **Priority**: 1
- **Status**: ready
`

	plan := ParseProjectPlanWithSchema(ctx, content, agent)

	if plan == nil {
		t.Fatal("ParseProjectPlanWithSchema returned nil")
	}

	if len(plan.Tasks) != 1 {
		t.Errorf("got %d tasks, want 1 (regex fallback after invalid JSON)", len(plan.Tasks))
	}
}

// TestParseProjectPlanWithSchema_SchemaReturnsEmpty_fallsBackToRegex tests that empty result falls back to regex.
func TestParseProjectPlanWithSchema_SchemaReturnsEmpty_fallsBackToRegex(t *testing.T) {
	ctx := context.Background()
	agent := &mockAgent{
		response: `{"tasks":[],"questions":[],"blockers":[]}`,
	}

	// Content that regex can parse
	content := `## Tasks
### task-1: Test Task
- **Priority**: 1
- **Status**: ready
`

	plan := ParseProjectPlanWithSchema(ctx, content, agent)

	if plan == nil {
		t.Fatal("ParseProjectPlanWithSchema returned nil")
	}

	// Empty schema result should fall back to regex
	if len(plan.Tasks) != 1 {
		t.Errorf("got %d tasks, want 1 (regex fallback after empty result)", len(plan.Tasks))
	}
}

// TestParseProjectPlan_BackwardCompatibility tests that ParseProjectPlan still works.
func TestParseProjectPlan_BackwardCompatibility(t *testing.T) {
	content := `## Tasks
### task-1: Test Task
- **Priority**: 1
- **Status**: ready
- **Labels**: backend,api
- **Description**: Test description

### task-2: Another Task
- **Priority**: 2
- **Status**: pending
- **Depends on**: task-1

## Questions
1. Question one?
2. Question two?

## Blockers
- Blocker one
`

	plan := ParseProjectPlan(content)

	if plan == nil {
		t.Fatal("ParseProjectPlan returned nil")
	}

	if len(plan.Tasks) != 2 {
		t.Errorf("got %d tasks, want 2", len(plan.Tasks))
	}

	if len(plan.Questions) != 2 {
		t.Errorf("got %d questions, want 2", len(plan.Questions))
	}

	if len(plan.Blockers) != 1 {
		t.Errorf("got %d blockers, want 1", len(plan.Blockers))
	}
}
