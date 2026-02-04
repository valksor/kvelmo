package conductor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/valksor/go-mehrhof/internal/agent"
	"github.com/valksor/go-mehrhof/internal/events"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/workflow"
	"github.com/valksor/go-toolkit/eventbus"
)

// TestAskQuestion_NoActiveTask tests that AskQuestion returns an error when no active task.
func TestAskQuestion_NoActiveTask(t *testing.T) {
	c, err := New(WithWorkDir(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	err = c.AskQuestion(ctx, "What is the meaning of life?")

	if err == nil {
		t.Error("AskQuestion should return error when no active task")
	}
	if !strings.Contains(err.Error(), "no active task") {
		t.Errorf("error should mention 'no active task', got: %v", err)
	}
}

// TestAskQuestion_InvalidStates tests that AskQuestion fails in invalid states.
func TestAskQuestion_InvalidStates(t *testing.T) {
	invalidStates := []workflow.State{
		workflow.StateIdle,
		workflow.StateDone,
		workflow.StateFailed,
		workflow.StateWaiting,
		workflow.StatePaused,
		workflow.StateCheckpointing,
		workflow.StateReverting,
		workflow.StateRestoring,
	}

	for _, state := range invalidStates {
		t.Run(string(state), func(t *testing.T) {
			c, err := New(WithWorkDir(t.TempDir()))
			if err != nil {
				t.Fatal(err)
			}

			// Set up active task in invalid state
			c.activeTask = &storage.ActiveTask{
				ID:    "test-task",
				State: string(state),
			}

			ctx := context.Background()
			err = c.AskQuestion(ctx, "Test question?")

			if err == nil {
				t.Errorf("AskQuestion should return error in state %s", state)
			}
			if !strings.Contains(err.Error(), "cannot ask questions in state") {
				t.Errorf("error should mention invalid state, got: %v", err)
			}
		})
	}
}

// TestAskQuestion_ValidStates tests that AskQuestion works in valid states.
func TestAskQuestion_ValidStates(t *testing.T) {
	validStates := []workflow.State{
		workflow.StatePlanning,
		workflow.StateImplementing,
		workflow.StateReviewing,
	}

	for _, state := range validStates {
		t.Run(string(state), func(t *testing.T) {
			tmpDir := t.TempDir()
			ws := openTestWorkspace(t, tmpDir)
			if err := ws.EnsureInitialized(); err != nil {
				t.Fatalf("EnsureInitialized: %v", err)
			}

			// Create task work
			work, err := ws.CreateWork("test-task", storage.SourceInfo{
				Type: "file",
				Ref:  "task.md",
			})
			if err != nil {
				t.Fatalf("CreateWork: %v", err)
			}
			work.Metadata.Title = "Test Task"

			// Create session
			session := &storage.Session{
				Version: "1",
				Kind:    "session",
				Metadata: storage.SessionMetadata{
					StartedAt: time.Now(),
					Type:      string(state),
					Agent:     "test-agent",
				},
				Exchanges: []storage.Exchange{},
			}
			sessionFilename := fmt.Sprintf("%s_session.yaml", state)
			if err := ws.SaveSession("test-task", sessionFilename, session); err != nil {
				t.Fatalf("SaveSession: %v", err)
			}

			// Load session
			loadedSession, err := ws.LoadSession("test-task", sessionFilename)
			if err != nil {
				t.Fatalf("LoadSession: %v", err)
			}

			c, err := New(WithWorkDir(tmpDir))
			if err != nil {
				t.Fatal(err)
			}

			c.workspace = ws
			c.activeTask = &storage.ActiveTask{
				ID:    "test-task",
				State: string(state),
			}
			c.taskWork = work
			c.currentSession = loadedSession

			// Register mock agent
			mockAgent := &mockQuestionAgent{
				name:     "test-agent",
				response: "The answer is 42",
			}
			if err := c.agents.Register(mockAgent); err != nil {
				t.Fatalf("Register agent: %v", err)
			}

			// Set up step agent info in work
			work.Agent.Steps = map[string]storage.StepAgentInfo{
				string(stateToStep(state)): {
					Name: "test-agent",
				},
			}
			if err := ws.SaveWork(work); err != nil {
				t.Fatalf("SaveWork: %v", err)
			}
			c.taskWork = work

			// Track the session before
			exchangesBefore := len(c.currentSession.Exchanges)

			ctx := context.Background()
			err = c.AskQuestion(ctx, "What is the meaning of life?")
			if err != nil {
				t.Errorf("AskQuestion should succeed in state %s, got: %v", state, err)
			}

			// Verify session was updated with Q&A in memory
			// Note: The session modification happens in memory; persisting requires saveCurrentSession()
			exchangesAfter := len(c.currentSession.Exchanges)
			if exchangesAfter < exchangesBefore+2 {
				t.Errorf("session should have at least 2 more exchanges (question + answer), before: %d, after: %d", exchangesBefore, exchangesAfter)
			}

			// Check first new exchange is the user question
			lastExchangeIdx := len(c.currentSession.Exchanges) - 1
			if lastExchangeIdx >= 0 && c.currentSession.Exchanges[lastExchangeIdx].Role == "user" {
				if !strings.Contains(c.currentSession.Exchanges[lastExchangeIdx].Content, "QUESTION:") {
					t.Errorf("last user exchange should contain QUESTION:, got: %s", c.currentSession.Exchanges[lastExchangeIdx].Content)
				}
			}
		})
	}
}

// TestAskQuestion_AgentFailure tests that AskQuestion handles agent failures.
func TestAskQuestion_AgentFailure(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	// Create task work
	work, err := ws.CreateWork("test-task", storage.SourceInfo{
		Type: "file",
		Ref:  "task.md",
	})
	if err != nil {
		t.Fatalf("CreateWork: %v", err)
	}
	work.Metadata.Title = "Test Task"

	c, err := New(WithWorkDir(tmpDir))
	if err != nil {
		t.Fatal(err)
	}

	c.workspace = ws
	c.activeTask = &storage.ActiveTask{
		ID:    "test-task",
		State: "planning",
	}
	c.taskWork = work

	// Don't register any agent - GetAgentForStep will fail

	ctx := context.Background()
	err = c.AskQuestion(ctx, "Test?")

	if err == nil {
		t.Error("AskQuestion should return error when agent not found")
	}
	if !strings.Contains(err.Error(), "get agent for question") {
		t.Errorf("error should mention agent failure, got: %v", err)
	}
}

// TestAskQuestion_BackQuestion tests that AskQuestion handles agent back-questions.
func TestAskQuestion_BackQuestion(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	// Create task work
	work, err := ws.CreateWork("test-task", storage.SourceInfo{
		Type: "file",
		Ref:  "task.md",
	})
	if err != nil {
		t.Fatalf("CreateWork: %v", err)
	}
	work.Metadata.Title = "Test Task"

	// Create session
	session := &storage.Session{
		Version: "1",
		Kind:    "session",
		Metadata: storage.SessionMetadata{
			StartedAt: time.Now(),
			Type:      "implementing",
			Agent:     "test-agent",
		},
		Exchanges: []storage.Exchange{},
	}
	sessionFilename := "implementing_session.yaml"
	if err := ws.SaveSession("test-task", sessionFilename, session); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}

	// Load session
	loadedSession, err := ws.LoadSession("test-task", sessionFilename)
	if err != nil {
		t.Fatalf("LoadSession: %v", err)
	}

	c, err := New(WithWorkDir(tmpDir))
	if err != nil {
		t.Fatal(err)
	}

	c.workspace = ws
	c.activeTask = &storage.ActiveTask{
		ID:    "test-task",
		State: "planning", // Must be planning for EventWait transition to work
	}
	c.taskWork = work
	c.currentSession = loadedSession

	// Set up work unit to satisfy guards, then transition to planning
	c.machine.SetWorkUnit(&workflow.WorkUnit{
		ID:          "test-task",
		Description: "Test task description",
		Source:      &workflow.Source{Reference: "file:task.md"},
	})
	if err := c.machine.Dispatch(context.Background(), workflow.EventPlan); err != nil {
		t.Fatalf("failed to set machine to planning state: %v", err)
	}

	// Register mock agent that asks a back-question
	mockAgent := &mockQuestionAgent{
		name: "test-agent",
		question: &agent.Question{
			Text: "Which approach do you prefer?",
			Options: []agent.QuestionOption{
				{Label: "A", Description: "Option A"},
				{Label: "B", Description: "Option B"},
			},
		},
	}
	if err := c.agents.Register(mockAgent); err != nil {
		t.Fatalf("Register agent: %v", err)
	}

	// Set up step agent info
	work.Agent.Steps = map[string]storage.StepAgentInfo{
		"implementing": {
			Name: "test-agent",
		},
	}
	if err := ws.SaveWork(work); err != nil {
		t.Fatalf("SaveWork: %v", err)
	}
	c.taskWork = work

	ctx := context.Background()
	err = c.AskQuestion(ctx, "Should I use A or B?")

	if !errors.Is(err, ErrPendingQuestion) {
		t.Errorf("AskQuestion should return ErrPendingQuestion when agent asks back-question, got: %v", err)
	}

	// Verify pending question was saved
	pendingQuestion, err := ws.LoadPendingQuestion("test-task")
	if err != nil {
		t.Fatalf("LoadPendingQuestion: %v", err)
	}
	if pendingQuestion.Question != "Which approach do you prefer?" {
		t.Errorf("pending question text mismatch, got: %s", pendingQuestion.Question)
	}
	if len(pendingQuestion.Options) != 2 {
		t.Errorf("pending question should have 2 options, got: %d", len(pendingQuestion.Options))
	}
}

// TestAskQuestion_EventPublishing tests that AskQuestion publishes events.
func TestAskQuestion_EventPublishing(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	// Create task work
	work, err := ws.CreateWork("test-task", storage.SourceInfo{
		Type: "file",
		Ref:  "task.md",
	})
	if err != nil {
		t.Fatalf("CreateWork: %v", err)
	}
	work.Metadata.Title = "Test Task"

	// Create session
	session := &storage.Session{
		Version: "1",
		Kind:    "session",
		Metadata: storage.SessionMetadata{
			StartedAt: time.Now(),
			Type:      "planning",
			Agent:     "test-agent",
		},
		Exchanges: []storage.Exchange{},
	}
	sessionFilename := "planning_session.yaml"
	if err := ws.SaveSession("test-task", sessionFilename, session); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}

	// Load session
	loadedSession, err := ws.LoadSession("test-task", sessionFilename)
	if err != nil {
		t.Fatalf("LoadSession: %v", err)
	}

	c, err := New(WithWorkDir(tmpDir))
	if err != nil {
		t.Fatal(err)
	}

	c.workspace = ws
	c.activeTask = &storage.ActiveTask{
		ID:    "test-task",
		State: "planning",
	}
	c.taskWork = work
	c.currentSession = loadedSession

	// Track published events
	var publishedEvents []eventbus.Event
	eventBus := c.GetEventBus()
	unsubID := eventBus.SubscribeAll(func(e eventbus.Event) {
		if e.Type == events.TypeAgentMessage {
			publishedEvents = append(publishedEvents, e)
		}
	})
	defer eventBus.Unsubscribe(unsubID)

	// Register mock agent
	mockAgent := &mockQuestionAgent{
		name:     "test-agent",
		response: "Test response",
		events: []agent.Event{
			{Type: "content", Data: map[string]any{"text": "Hello"}},
			{Type: "content", Data: map[string]any{"text": " World"}},
		},
	}
	if err := c.agents.Register(mockAgent); err != nil {
		t.Fatalf("Register agent: %v", err)
	}

	// Set up step agent info
	work.Agent.Steps = map[string]storage.StepAgentInfo{
		"planning": {
			Name: "test-agent",
		},
	}
	if err := ws.SaveWork(work); err != nil {
		t.Fatalf("SaveWork: %v", err)
	}
	c.taskWork = work

	ctx := context.Background()
	err = c.AskQuestion(ctx, "Test?")
	if err != nil {
		t.Errorf("AskQuestion should succeed, got: %v", err)
	}

	// Verify events were published
	if len(publishedEvents) != 2 {
		t.Errorf("expected 2 published events, got: %d", len(publishedEvents))
	}
}

// TestAskQuestion_SessionHistory tests that AskQuestion includes session history.
func TestAskQuestion_SessionHistory(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	// Create task work
	work, err := ws.CreateWork("test-task", storage.SourceInfo{
		Type: "file",
		Ref:  "task.md",
	})
	if err != nil {
		t.Fatalf("CreateWork: %v", err)
	}
	work.Metadata.Title = "Test Task"

	// Create session with existing exchanges
	session := &storage.Session{
		Version: "1",
		Kind:    "session",
		Metadata: storage.SessionMetadata{
			StartedAt: time.Now(),
			Type:      "implementing",
			Agent:     "test-agent",
		},
		Exchanges: []storage.Exchange{
			{Role: "user", Content: "First question", Timestamp: time.Now().Add(-2 * time.Minute)},
			{Role: "assistant", Content: "First answer", Timestamp: time.Now().Add(-1 * time.Minute)},
			{Role: "user", Content: "Second question", Timestamp: time.Now().Add(-30 * time.Second)},
			{Role: "assistant", Content: "Second answer", Timestamp: time.Now().Add(-20 * time.Second)},
		},
	}
	sessionFilename := "implementing_session.yaml"
	if err := ws.SaveSession("test-task", sessionFilename, session); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}

	// Load session
	loadedSession, err := ws.LoadSession("test-task", sessionFilename)
	if err != nil {
		t.Fatalf("LoadSession: %v", err)
	}

	c, err := New(WithWorkDir(tmpDir))
	if err != nil {
		t.Fatal(err)
	}

	c.workspace = ws
	c.activeTask = &storage.ActiveTask{
		ID:    "test-task",
		State: "implementing",
	}
	c.taskWork = work
	c.currentSession = loadedSession

	// Track the prompt sent to the agent
	var capturedPrompt string
	mockAgent := &mockQuestionAgent{
		name: "test-agent",
		capturePrompt: func(p string) {
			capturedPrompt = p
		},
		response: "Test response",
	}
	if err := c.agents.Register(mockAgent); err != nil {
		t.Fatalf("Register agent: %v", err)
	}

	// Set up step agent info
	work.Agent.Steps = map[string]storage.StepAgentInfo{
		"implementing": {
			Name: "test-agent",
		},
	}
	if err := ws.SaveWork(work); err != nil {
		t.Fatalf("SaveWork: %v", err)
	}
	c.taskWork = work

	ctx := context.Background()
	err = c.AskQuestion(ctx, "New question?")
	if err != nil {
		t.Errorf("AskQuestion should succeed, got: %v", err)
	}

	// Verify prompt contains session history
	if capturedPrompt == "" {
		t.Fatal("prompt was not captured")
	}
	if !strings.Contains(capturedPrompt, "Recent Conversation") {
		t.Error("prompt should contain 'Recent Conversation' section")
	}
	if !strings.Contains(capturedPrompt, "First question") {
		t.Error("prompt should contain first question from history")
	}
}

// TestAskQuestion_WithSpecification tests that AskQuestion includes specification content.
func TestAskQuestion_WithSpecification(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	// Create task work
	work, err := ws.CreateWork("test-task", storage.SourceInfo{
		Type: "file",
		Ref:  "task.md",
	})
	if err != nil {
		t.Fatalf("CreateWork: %v", err)
	}
	work.Metadata.Title = "Test Task"

	// Save a specification
	specContent := "# Specification 1\n\nImplement feature X with REST API."
	if err := ws.SaveSpecification("test-task", 1, specContent); err != nil {
		t.Fatalf("SaveSpecification: %v", err)
	}

	// Create session
	session := &storage.Session{
		Version: "1",
		Kind:    "session",
		Metadata: storage.SessionMetadata{
			StartedAt: time.Now(),
			Type:      "planning",
			Agent:     "test-agent",
		},
		Exchanges: []storage.Exchange{},
	}
	sessionFilename := "planning_session.yaml"
	if err := ws.SaveSession("test-task", sessionFilename, session); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}

	// Load session
	loadedSession, err := ws.LoadSession("test-task", sessionFilename)
	if err != nil {
		t.Fatalf("LoadSession: %v", err)
	}

	c, err := New(WithWorkDir(tmpDir))
	if err != nil {
		t.Fatal(err)
	}

	c.workspace = ws
	c.activeTask = &storage.ActiveTask{
		ID:    "test-task",
		State: "planning",
	}
	c.taskWork = work
	c.currentSession = loadedSession

	// Track the prompt sent to the agent
	var capturedPrompt string
	mockAgent := &mockQuestionAgent{
		name: "test-agent",
		capturePrompt: func(p string) {
			capturedPrompt = p
		},
		response: "Test response",
	}
	if err := c.agents.Register(mockAgent); err != nil {
		t.Fatalf("Register agent: %v", err)
	}

	// Set up step agent info
	work.Agent.Steps = map[string]storage.StepAgentInfo{
		"planning": {
			Name: "test-agent",
		},
	}
	if err := ws.SaveWork(work); err != nil {
		t.Fatalf("SaveWork: %v", err)
	}
	c.taskWork = work

	ctx := context.Background()
	err = c.AskQuestion(ctx, "How should I implement this?")
	if err != nil {
		t.Errorf("AskQuestion should succeed, got: %v", err)
	}

	// Verify prompt contains specification
	if capturedPrompt == "" {
		t.Fatal("prompt was not captured")
	}
	if !strings.Contains(capturedPrompt, "Current Specification") {
		t.Error("prompt should contain 'Current Specification' section")
	}
	if !strings.Contains(capturedPrompt, "Implement feature X") {
		t.Error("prompt should contain specification content")
	}
}

// mockQuestionAgent is a mock agent for testing AskQuestion.
type mockQuestionAgent struct {
	name          string
	response      string
	question      *agent.Question
	events        []agent.Event
	capturePrompt func(string)
}

func (m *mockQuestionAgent) Name() string {
	return m.name
}

func (m *mockQuestionAgent) Run(ctx context.Context, prompt string) (*agent.Response, error) {
	if m.capturePrompt != nil {
		m.capturePrompt(prompt)
	}

	return &agent.Response{
		Summary:  m.response,
		Messages: []string{m.response},
		Question: m.question,
	}, nil
}

func (m *mockQuestionAgent) RunStream(ctx context.Context, prompt string) (<-chan agent.Event, <-chan error) {
	eventCh := make(chan agent.Event, len(m.events))
	errCh := make(chan error, 1)

	for _, e := range m.events {
		eventCh <- e
	}
	close(eventCh)
	close(errCh)

	return eventCh, errCh
}

func (m *mockQuestionAgent) RunWithCallback(ctx context.Context, prompt string, cb agent.StreamCallback) (*agent.Response, error) {
	if m.capturePrompt != nil {
		m.capturePrompt(prompt)
	}

	// Send events through callback
	for _, e := range m.events {
		if cb != nil {
			_ = cb(e)
		}
	}

	return &agent.Response{
		Summary:  m.response,
		Messages: []string{m.response},
		Question: m.question,
	}, nil
}

func (m *mockQuestionAgent) Available() error {
	return nil
}

func (m *mockQuestionAgent) WithEnv(key, value string) agent.Agent {
	return m
}

func (m *mockQuestionAgent) WithArgs(args ...string) agent.Agent {
	return m
}

// stateToStep converts a workflow State to its corresponding Step.
func stateToStep(state workflow.State) workflow.Step {
	switch state {
	case workflow.StatePlanning:
		return workflow.StepPlanning
	case workflow.StateImplementing:
		return workflow.StepImplementing
	case workflow.StateReviewing:
		return workflow.StepReviewing
	case workflow.StateIdle, workflow.StateDone, workflow.StateFailed, workflow.StateWaiting, workflow.StatePaused, workflow.StateCheckpointing, workflow.StateReverting, workflow.StateRestoring:
		return ""
	}

	return ""
}

// TestBuildQuestionPrompt tests the buildQuestionPrompt function.
func TestBuildQuestionPrompt(t *testing.T) {
	tests := []struct {
		name                 string
		title                string
		question             string
		specificationContent string
		sessionHistory       string
		wantContain          []string
	}{
		{
			name:     "minimal prompt",
			title:    "Test Task",
			question: "What should I do?",
			wantContain: []string{
				"Test Task",
				"What should I do?",
				"User Question",
			},
		},
		{
			name:                 "with specification",
			title:                "Task",
			question:             "Question?",
			specificationContent: "# Spec\nImplement X.",
			wantContain: []string{
				"Current Specification",
				"Implement X",
			},
		},
		{
			name:           "with session history",
			title:          "Task",
			question:       "Question?",
			sessionHistory: "**user:** First\n**assistant:** Answer\n",
			wantContain: []string{
				"Recent Conversation",
				"First",
				"Answer",
			},
		},
		{
			name:                 "full prompt",
			title:                "Complete Task",
			question:             "Final question?",
			specificationContent: "# Specification\nDetails here.",
			sessionHistory:       "**user:** Q1\n**assistant:** A1\n",
			wantContain: []string{
				"Complete Task",
				"Current Specification",
				"Details here",
				"Recent Conversation",
				"Q1",
				"A1",
				"Final question?",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildQuestionPrompt(tt.title, tt.question, tt.specificationContent, tt.sessionHistory)

			for _, want := range tt.wantContain {
				if !strings.Contains(got, want) {
					t.Errorf("buildQuestionPrompt() missing %q\nGot:\n%s", want, got)
				}
			}

			// Verify structure
			if !strings.Contains(got, "## Task") {
				t.Error("prompt should have '## Task' section")
			}
			if !strings.Contains(got, "## User Question") {
				t.Error("prompt should have '## User Question' section")
			}
		})
	}
}

// =============================================================================
// AnswerQuestion Tests
// =============================================================================

// TestAnswerQuestion_NoActiveTask tests that AnswerQuestion returns error when no task.
func TestAnswerQuestion_NoActiveTask(t *testing.T) {
	c, err := New(WithWorkDir(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}

	err = c.AnswerQuestion(context.Background(), "my answer")
	if err == nil {
		t.Error("AnswerQuestion should return error when no active task")
	}
	if !strings.Contains(err.Error(), "no active task") {
		t.Errorf("error should mention 'no active task', got: %v", err)
	}
}

// TestAnswerQuestion_NoPendingQuestion tests that AnswerQuestion fails without pending question.
func TestAnswerQuestion_NoPendingQuestion(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	// Create work directory
	_, err := ws.CreateWork("test-task", storage.SourceInfo{Type: "file", Ref: "task.md"})
	if err != nil {
		t.Fatalf("CreateWork: %v", err)
	}

	// Create conductor with active task but NO pending question
	eventBus := eventbus.NewBus()
	c := &Conductor{
		workspace: ws,
		machine:   workflow.NewMachine(eventBus),
		eventBus:  eventBus,
		activeTask: &storage.ActiveTask{
			ID:    "test-task",
			State: string(workflow.StateWaiting),
		},
	}

	err = c.AnswerQuestion(context.Background(), "my answer")
	if err == nil {
		t.Error("AnswerQuestion should return error when no pending question")
	}
	if !strings.Contains(err.Error(), "no pending question") {
		t.Errorf("error should mention 'no pending question', got: %v", err)
	}
}

// TestAnswerQuestion_WithPendingQuestion tests the normal flow with a pending question.
func TestAnswerQuestion_WithPendingQuestion(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	// Create work directory
	_, err := ws.CreateWork("test-task", storage.SourceInfo{Type: "file", Ref: "task.md"})
	if err != nil {
		t.Fatalf("CreateWork: %v", err)
	}

	// Save a pending question
	pendingQ := &storage.PendingQuestion{
		Question: "What database should I use?",
		Options: []storage.QuestionOption{
			{Label: "PostgreSQL", Description: "Relational database"},
			{Label: "MongoDB", Description: "Document database"},
		},
	}
	if err := ws.SavePendingQuestion("test-task", pendingQ); err != nil {
		t.Fatalf("SavePendingQuestion: %v", err)
	}

	// Verify pending question exists
	if !ws.HasPendingQuestion("test-task") {
		t.Fatal("pending question should exist")
	}

	// Create conductor in waiting state
	eventBus := eventbus.NewBus()
	machine := workflow.NewMachine(eventBus)

	// Set up work unit to satisfy guards
	machine.SetWorkUnit(&workflow.WorkUnit{
		ID:          "test-task",
		Description: "Test task description",
		Source:      &workflow.Source{Reference: "file:task.md"},
	})

	// Transition machine to waiting state: idle -> planning -> waiting
	_ = machine.Dispatch(context.Background(), workflow.EventPlan) // idle -> planning
	_ = machine.Dispatch(context.Background(), workflow.EventWait) // planning -> waiting

	c := &Conductor{
		workspace: ws,
		machine:   machine,
		eventBus:  eventBus,
		activeTask: &storage.ActiveTask{
			ID:    "test-task",
			State: string(workflow.StateWaiting),
		},
	}

	// Answer the question
	err = c.AnswerQuestion(context.Background(), "PostgreSQL")
	if err != nil {
		t.Fatalf("AnswerQuestion: %v", err)
	}

	// Verify pending question was cleared
	if ws.HasPendingQuestion("test-task") {
		t.Error("pending question should be cleared after answering")
	}

	// Verify state transitioned to idle
	if c.activeTask.State != string(workflow.StateIdle) {
		t.Errorf("state should be idle after answering, got: %s", c.activeTask.State)
	}

	// Verify answer was saved as note
	notes, err := ws.ReadNotes("test-task")
	if err != nil {
		t.Fatalf("ReadNotes: %v", err)
	}
	if !strings.Contains(notes, "**Q:**") || !strings.Contains(notes, "**A:**") {
		t.Errorf("notes should contain Q&A format, got: %s", notes)
	}
	if !strings.Contains(notes, "PostgreSQL") {
		t.Errorf("notes should contain the answer 'PostgreSQL', got: %s", notes)
	}
}

// TestAnswerQuestion_TransitionsFromWaiting tests state transition from waiting to idle.
func TestAnswerQuestion_TransitionsFromWaiting(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openTestWorkspace(t, tmpDir)
	if err := ws.EnsureInitialized(); err != nil {
		t.Fatalf("EnsureInitialized: %v", err)
	}

	_, err := ws.CreateWork("test-task", storage.SourceInfo{Type: "file", Ref: "task.md"})
	if err != nil {
		t.Fatalf("CreateWork: %v", err)
	}

	// Save pending question
	if err := ws.SavePendingQuestion("test-task", &storage.PendingQuestion{
		Question: "Continue?",
	}); err != nil {
		t.Fatalf("SavePendingQuestion: %v", err)
	}

	eventBus := eventbus.NewBus()
	machine := workflow.NewMachine(eventBus)

	// Set up work unit to satisfy guards
	machine.SetWorkUnit(&workflow.WorkUnit{
		ID:          "test-task",
		Description: "Test task description",
		Source:      &workflow.Source{Reference: "file:task.md"},
	})

	// Get machine into waiting state: idle -> planning -> waiting
	_ = machine.Dispatch(context.Background(), workflow.EventPlan)
	_ = machine.Dispatch(context.Background(), workflow.EventWait)

	if machine.State() != workflow.StateWaiting {
		t.Fatalf("machine should be in waiting state, got: %s", machine.State())
	}

	c := &Conductor{
		workspace: ws,
		machine:   machine,
		eventBus:  eventBus,
		activeTask: &storage.ActiveTask{
			ID:    "test-task",
			State: string(workflow.StateWaiting),
		},
	}

	// Answer should transition state
	if err := c.AnswerQuestion(context.Background(), "yes"); err != nil {
		t.Fatalf("AnswerQuestion: %v", err)
	}

	// Machine should now be in idle state
	if machine.State() != workflow.StateIdle {
		t.Errorf("machine state should be idle, got: %s", machine.State())
	}
}

// TestResetState_FromWaitingWithoutPendingQuestion tests the edge case where
// state is "waiting" but there's no pending question (e.g., old bug cleared
// the question but didn't transition state).
func TestResetState_FromWaitingWithoutPendingQuestion(t *testing.T) {
	tmpDir := t.TempDir()
	ws := openResetTestWorkspace(t, tmpDir)

	eventBus := eventbus.NewBus()
	machine := workflow.NewMachine(eventBus)

	// Set up work unit to satisfy guards
	machine.SetWorkUnit(&workflow.WorkUnit{
		ID:          "test-task",
		Description: "Test task description",
		Source:      &workflow.Source{Reference: "file:task.md"},
	})

	// Get machine into waiting state: idle -> planning -> waiting
	_ = machine.Dispatch(context.Background(), workflow.EventPlan)
	_ = machine.Dispatch(context.Background(), workflow.EventWait)

	c := &Conductor{
		workspace: ws,
		machine:   machine,
		eventBus:  eventBus,
		activeTask: &storage.ActiveTask{
			ID:    "test-task",
			State: string(workflow.StateWaiting),
		},
	}

	// Verify no pending question
	if ws.HasPendingQuestion("test-task") {
		t.Fatal("should have no pending question for this test")
	}

	// ResetState should work to recover from this stuck state
	if err := c.ResetState(context.Background()); err != nil {
		t.Fatalf("ResetState: %v", err)
	}

	// State should now be idle
	if c.activeTask.State != string(workflow.StateIdle) {
		t.Errorf("state should be idle after reset, got: %s", c.activeTask.State)
	}
}
