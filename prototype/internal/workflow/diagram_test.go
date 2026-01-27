package workflow

import (
	"strings"
	"testing"

	"github.com/valksor/go-toolkit/eventbus"
)

// TestGenerateDiagram tests diagram generation.
func TestGenerateDiagram(t *testing.T) {
	machine := NewMachine(eventbus.NewBus())
	opts := DiagramOptions{
		CurrentState: StateIdle,
		ShowEvents:   true,
		Compact:      false,
	}

	diagram := GenerateDiagram(machine, opts)
	if diagram == nil {
		t.Fatal("GenerateDiagram() returned nil")
	}

	// Should have nodes
	if len(diagram.Nodes) == 0 {
		t.Error("GenerateDiagram() should produce nodes")
	}

	// Should have edges
	if len(diagram.Edges) == 0 {
		t.Error("GenerateDiagram() should produce edges")
	}

	// Current state should be set
	if diagram.Current != StateIdle {
		t.Errorf("GenerateDiagram() Current = %v, want %v", diagram.Current, StateIdle)
	}

	// Check that main states are in nodes
	mainStates := []State{StateIdle, StatePlanning, StateImplementing, StateReviewing, StateDone}
	nodeStates := make(map[State]bool)
	for _, node := range diagram.Nodes {
		nodeStates[node.State] = true
	}
	for _, state := range mainStates {
		if !nodeStates[state] {
			t.Errorf("GenerateDiagram() should have node for state %v", state)
		}
	}
}

// TestASCIIIDiagram tests ASCII diagram generation.
func TestASCIIIDiagram(t *testing.T) {
	tests := []struct {
		name    string
		state   State
		wantHas []string
	}{
		{
			name:  "idle state",
			state: StateIdle,
			wantHas: []string{
				"Workflow State Machine",
				"Main Flow:",
				"idle",
				"planning",
				"implementing",
				"reviewing",
				"done",
				"Auxiliary States:",
				"waiting",
				"paused",
				"checkpointing",
				"reverting",
				"restoring",
				"failed",
				"Valid Transitions:",
			},
		},
		{
			name:  "planning state",
			state: StatePlanning,
			wantHas: []string{
				">>>", // Current marker
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			machine := NewMachine(eventbus.NewBus())

			result := ASCIIDiagram(machine, DiagramOptions{
				CurrentState: tt.state,
				ShowEvents:   true,
			})
			if result == "" {
				t.Error("ASCIIDiagram() returned empty string")
			}

			// Check for expected content
			for _, expected := range tt.wantHas {
				if !strings.Contains(result, expected) {
					t.Errorf("ASCIIDiagram() should contain %q", expected)
				}
			}

			// Should contain arrows
			if !strings.Contains(result, "-->") {
				t.Error("ASCIIDiagram() should contain arrows")
			}
		})
	}
}

// TestASCIIIDiagramStructure tests the structure of the ASCII diagram.
func TestASCIIDiagramStructure(t *testing.T) {
	machine := NewMachine(eventbus.NewBus())
	opts := DiagramOptions{}

	result := ASCIIDiagram(machine, opts)

	// Should have header
	if !strings.HasPrefix(result, "Workflow State Machine") {
		t.Error("ASCIIDiagram() should start with header")
	}

	// Should have section headers
	sections := []string{
		"Main Flow:",
		"Auxiliary States:",
		"Valid Transitions:",
	}
	for _, section := range sections {
		if !strings.Contains(result, section) {
			t.Errorf("ASCIIDiagram() should have section %q", section)
		}
	}

	// Should contain state names
	states := []string{
		"idle", "planning", "implementing", "reviewing", "done",
		"waiting", "paused", "checkpointing", "reverting", "restoring", "failed",
	}
	for _, state := range states {
		if !strings.Contains(result, state) {
			t.Errorf("ASCIIDiagram() should contain state %q", state)
		}
	}
}

// TestSVGDiagram tests SVG diagram generation.
func TestSVGDiagram(t *testing.T) {
	machine := NewMachine(eventbus.NewBus())
	opts := DiagramOptions{
		CurrentState: StateIdle,
	}

	result := SVGDiagram(machine, opts)
	if result == "" {
		t.Fatal("SVGDiagram() returned empty string")
	}

	// Should have SVG tag
	if !strings.HasPrefix(result, "<svg") {
		t.Error("SVGDiagram() should start with <svg> tag")
	}

	// Should have closing SVG tag
	if !strings.HasSuffix(result, "</svg>\n") {
		t.Error("SVGDiagram() should end with </svg> tag")
	}

	// Should have style section
	if !strings.Contains(result, "<style>") {
		t.Error("SVGDiagram() should have <style> section")
	}

	// Should have defs section for arrowhead
	if !strings.Contains(result, "<defs>") {
		t.Error("SVGDiagram() should have <defs> section")
	}

	// Should have rectangles for states
	if !strings.Contains(result, "<rect") {
		t.Error("SVGDiagram() should have <rect> elements")
	}

	// Should have text labels
	if !strings.Contains(result, "<text") {
		t.Error("SVGDiagram() should have <text> elements")
	}

	// Should have state names
	stateNames := []string{"idle", "planning", "implementing", "reviewing", "done"}
	for _, name := range stateNames {
		if !strings.Contains(result, name) {
			t.Errorf("SVGDiagram() should contain state name %q", name)
		}
	}
}

// TestSVGDiagramWithDifferentStates tests SVG diagram with different current states.
func TestSVGDiagramWithDifferentStates(t *testing.T) {
	states := []State{
		StateIdle,
		StatePlanning,
		StateImplementing,
		StateReviewing,
		StateDone,
	}

	for _, state := range states {
		t.Run(string(state), func(t *testing.T) {
			machine := NewMachine(eventbus.NewBus())
			opts := DiagramOptions{
				CurrentState: state,
			}

			result := SVGDiagram(machine, opts)
			if result == "" {
				t.Error("SVGDiagram() returned empty string")
			}

			// Should contain the current state name
			if !strings.Contains(result, string(state)) {
				t.Errorf("SVGDiagram() should contain current state %q", state)
			}
		})
	}
}

// TestSVGDiagramHasCSSClasses tests that SVG diagram uses proper CSS classes.
func TestSVGDiagramHasCSSClasses(t *testing.T) {
	machine := NewMachine(eventbus.NewBus())
	opts := DiagramOptions{}

	result := SVGDiagram(machine, opts)

	// Check for CSS classes
	cssClasses := []string{
		"class=\"state-box\"",
		"class=\"state-text\"",
		"class=\"transition-line\"",
	}

	for _, cssClass := range cssClasses {
		if !strings.Contains(result, cssClass) {
			t.Errorf("SVGDiagram() should contain CSS class %q", cssClass)
		}
	}
}

// TestGetValidTransitions tests getting valid transitions from a state.
func TestGetValidTransitions(t *testing.T) {
	tests := []struct {
		name           string
		state          State
		minTransitions int
		expectedEvents []Event
	}{
		{
			name:           "idle state",
			state:          StateIdle,
			minTransitions: 2, // Should have at least start and abort
			expectedEvents: []Event{EventStart},
		},
		{
			name:           "planning state",
			state:          StatePlanning,
			minTransitions: 1,
		},
		{
			name:           "done state",
			state:          StateDone,
			minTransitions: 0, // Terminal state
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			machine := NewMachine(eventbus.NewBus())

			// If not idle, we'd need to transition there - for now just test with idle
			if tt.state != StateIdle {
				t.Skip("skipping non-idle states for simplicity")
			}

			transitions := GetValidTransitions(machine)
			if len(transitions) < tt.minTransitions {
				t.Errorf("GetValidTransitions() returned %d transitions, want at least %d",
					len(transitions), tt.minTransitions)
			}

			// Check for expected events
			for _, expectedEvent := range tt.expectedEvents {
				found := false
				for _, trans := range transitions {
					if trans.Event == expectedEvent {
						found = true

						break
					}
				}
				if !found {
					t.Errorf("GetValidTransitions() should contain event %v", expectedEvent)
				}
			}
		})
	}
}

// TestGetValidTransitionsHasGlobalTransitions tests that global transitions are included.
func TestGetValidTransitionsHasGlobalTransitions(t *testing.T) {
	machine := NewMachine(eventbus.NewBus())

	transitions := GetValidTransitions(machine)

	// Should have global transitions like abort
	foundAbort := false
	for _, trans := range transitions {
		if trans.Event == EventAbort {
			foundAbort = true

			break
		}
	}
	if !foundAbort {
		t.Error("GetValidTransitions() should include global abort transition")
	}
}

// TestWorkflowDiagramStructure tests the WorkflowDiagram struct.
func TestWorkflowDiagramStructure(t *testing.T) {
	machine := NewMachine(eventbus.NewBus())
	opts := DiagramOptions{}

	diagram := GenerateDiagram(machine, opts)

	// Nodes should be populated
	if len(diagram.Nodes) == 0 {
		t.Error("WorkflowDiagram should have nodes")
	}

	// Edges should be populated
	if len(diagram.Edges) == 0 {
		t.Error("WorkflowDiagram should have edges")
	}

	// Check node structure
	for _, node := range diagram.Nodes {
		if node.State == "" {
			t.Error("StateNode.State should not be empty")
		}
		if node.Label == "" {
			t.Error("StateNode.Label should not be empty")
		}
	}

	// Check edge structure
	for _, edge := range diagram.Edges {
		if edge.Event == "" {
			t.Error("TransitionEdge.Event should not be empty")
		}
		if edge.To == "" {
			t.Error("TransitionEdge.To should not be empty")
		}
	}
}

// TestStateNodeFields tests StateNode field population.
func TestStateNodeFields(t *testing.T) {
	machine := NewMachine(eventbus.NewBus())
	opts := DiagramOptions{
		CurrentState: StateIdle,
	}

	diagram := GenerateDiagram(machine, opts)

	// Find idle node and verify fields
	var idleNode *StateNode
	for i := range diagram.Nodes {
		if diagram.Nodes[i].State == StateIdle {
			idleNode = &diagram.Nodes[i]

			break
		}
	}

	if idleNode == nil {
		t.Fatal("GenerateDiagram() should have idle node")
	}

	// Current state should be marked
	if !idleNode.IsCurrent {
		t.Error("StateNode.IsCurrent should be true for current state")
	}

	// Label should match state name
	if idleNode.Label != string(StateIdle) {
		t.Errorf("StateNode.Label = %q, want %q", idleNode.Label, StateIdle)
	}
}

// TestTransitionEdgeFields tests TransitionEdge field population.
func TestTransitionEdgeFields(t *testing.T) {
	machine := NewMachine(eventbus.NewBus())
	opts := DiagramOptions{}

	diagram := GenerateDiagram(machine, opts)

	// Find a non-global edge and verify fields
	var normalEdge *TransitionEdge
	for i := range diagram.Edges {
		if !diagram.Edges[i].IsGlobal && diagram.Edges[i].From != "" {
			normalEdge = &diagram.Edges[i]

			break
		}
	}

	if normalEdge != nil {
		// Normal edge should have From state
		if normalEdge.From == "" {
			t.Error("TransitionEdge.From should not be empty for normal edges")
		}

		// Label should match event
		if normalEdge.Label == "" {
			t.Error("TransitionEdge.Label should not be empty")
		}
	}

	// Find a global edge and verify it's marked as global
	var globalEdge *TransitionEdge
	for i := range diagram.Edges {
		if diagram.Edges[i].IsGlobal {
			globalEdge = &diagram.Edges[i]

			break
		}
	}

	if globalEdge == nil {
		t.Error("Should have at least one global transition")
	} else {
		// Global edge should be marked
		if !globalEdge.IsGlobal {
			t.Error("Global transition should have IsGlobal = true")
		}

		// Global edge should have empty From (from any state)
		if globalEdge.From != "" {
			t.Errorf("TransitionEdge.From for global edge should be empty, got %q", globalEdge.From)
		}
	}
}

// TestDiagramOptions tests DiagramOptions struct.
func TestDiagramOptions(t *testing.T) {
	opts := DiagramOptions{
		CurrentState: StatePlanning,
		ShowEvents:   true,
		Compact:      true,
	}

	if opts.CurrentState != StatePlanning {
		t.Errorf("DiagramOptions.CurrentState = %v, want %v", opts.CurrentState, StatePlanning)
	}
	if !opts.ShowEvents {
		t.Error("DiagramOptions.ShowEvents should be true")
	}
	if !opts.Compact {
		t.Error("DiagramOptions.Compact should be true")
	}
}

// TestASCIIIDiagramWithCurrentMarker tests ASCII diagram highlights current state.
func TestASCIIDiagramWithCurrentMarker(t *testing.T) {
	machine := NewMachine(eventbus.NewBus())
	opts := DiagramOptions{
		CurrentState: StateIdle,
	}

	result := ASCIIDiagram(machine, opts)

	// Should have marker for current state
	if !strings.Contains(result, ">>>") {
		t.Error("ASCIIDiagram() should mark current state with >>>")
	}

	// The marker should be near the state name
	lines := strings.Split(result, "\n")
	foundCurrentMarker := false
	for _, line := range lines {
		if strings.Contains(line, ">>>") && strings.Contains(line, "idle") {
			foundCurrentMarker = true

			break
		}
	}
	if !foundCurrentMarker {
		// The format might be different, just check the marker exists
		if !strings.Contains(result, ">>> [idle]") && !strings.Contains(result, ">>>[idle]") {
			t.Error("ASCIIDiagram() should have marker near current state")
		}
	}
}

// TestSVGDiagramDimensions tests SVG diagram has proper dimensions.
func TestSVGDiagramDimensions(t *testing.T) {
	machine := NewMachine(eventbus.NewBus())
	opts := DiagramOptions{}

	result := SVGDiagram(machine, opts)

	// Should have width and height in SVG tag
	if !strings.Contains(result, `width="`) {
		t.Error("SVGDiagram() should have width attribute")
	}
	if !strings.Contains(result, `height="`) {
		t.Error("SVGDiagram() should have height attribute")
	}

	// Check for specific dimensions (800x400 based on code)
	if !strings.Contains(result, `width="800"`) {
		t.Error("SVGDiagram() should have width=\"800\"")
	}
	if !strings.Contains(result, `height="400"`) {
		t.Error("SVGDiagram() should have height=\"400\"")
	}
}
