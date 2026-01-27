package workflow

import (
	"fmt"
	"strings"
)

// DiagramOptions controls diagram generation.
type DiagramOptions struct {
	CurrentState State // Highlight this state
	ShowEvents   bool  // Show event labels on transitions
	Compact      bool  // Use compact layout
}

// StateNode represents a state in the diagram.
type StateNode struct {
	State       State
	Label       string
	Description string
	IsCurrent   bool
	IsTerminal  bool
	IsPhase     bool
}

// TransitionEdge represents a transition between states.
type TransitionEdge struct {
	From     State
	To       State
	Event    Event
	Label    string
	IsGlobal bool // Global transition (from any state)
}

// WorkflowDiagram contains the complete workflow diagram structure.
type WorkflowDiagram struct {
	Nodes   []StateNode
	Edges   []TransitionEdge
	Current State
}

// GenerateDiagram creates a workflow diagram structure.
func GenerateDiagram(machine *Machine, opts DiagramOptions) *WorkflowDiagram {
	diag := &WorkflowDiagram{
		Current: machine.State(),
	}

	// Add all states as nodes
	for state, info := range StateRegistry {
		node := StateNode{
			State:       state,
			Label:       string(state),
			Description: info.Description,
			IsCurrent:   state == opts.CurrentState,
			IsTerminal:  info.Terminal,
			IsPhase:     info.Phase,
		}
		diag.Nodes = append(diag.Nodes, node)
	}

	// Add all transitions as edges
	for _, transitions := range TransitionTable {
		for _, trans := range transitions {
			edge := TransitionEdge{
				From:  trans.From,
				To:    trans.To,
				Event: trans.Event,
				Label: string(trans.Event),
			}
			diag.Edges = append(diag.Edges, edge)
		}
	}

	// Add global transitions (like abort)
	for event, toState := range GlobalTransitions {
		edge := TransitionEdge{
			From:     "", // Global (from any state)
			To:       toState,
			Event:    event,
			Label:    string(event),
			IsGlobal: true,
		}
		diag.Edges = append(diag.Edges, edge)
	}

	return diag
}

// ASCIIDiagram generates an ASCII art diagram of the workflow.
func ASCIIDiagram(machine *Machine, opts DiagramOptions) string {
	// Build layout: phase states horizontally, auxiliary states below
	var sb strings.Builder

	sb.WriteString("Workflow State Machine\n")
	sb.WriteString("======================\n\n")

	// Main flow (top row)
	phaseStates := []State{StateIdle, StatePlanning, StateImplementing, StateReviewing, StateDone}

	sb.WriteString("Main Flow:\n")
	for i, state := range phaseStates {
		isCurrent := state == machine.State()
		var marker string
		if isCurrent {
			marker = ">>> "
		} else {
			marker = "    "
		}

		nodeStr := fmt.Sprintf("%s[%s]", marker, state)
		sb.WriteString(nodeStr)

		if i < len(phaseStates)-1 {
			sb.WriteString(" --> ")
		}
	}
	sb.WriteString("\n\n")

	// Auxiliary states (bottom row)
	auxStates := []State{StateWaiting, StatePaused, StateCheckpointing, StateReverting, StateRestoring, StateFailed}

	sb.WriteString("Auxiliary States:\n")
	for i, state := range auxStates {
		isCurrent := state == machine.State()
		var marker string
		if isCurrent {
			marker = ">>> "
		} else {
			marker = "    "
		}

		nodeStr := fmt.Sprintf("%s[%s]", marker, state)
		sb.WriteString(nodeStr)

		if i < len(auxStates)-1 {
			sb.WriteString("    ")
		}
	}
	sb.WriteString("\n\n")

	// Show valid transitions from current state
	sb.WriteString("Valid Transitions:\n")
	current := machine.State()

	for key, transitions := range TransitionTable {
		if key.From == current {
			for _, trans := range transitions {
				sb.WriteString(fmt.Sprintf("  %s -> %s (%s)\n", trans.From, trans.To, trans.Event))
			}
		}
	}

	// Check global transitions
	for event, toState := range GlobalTransitions {
		sb.WriteString(fmt.Sprintf("  * -> %s (%s - global)\n", toState, event))
	}

	return sb.String()
}

// SVGDiagram generates an SVG diagram of the workflow.
func SVGDiagram(machine *Machine, opts DiagramOptions) string {
	diag := GenerateDiagram(machine, opts)

	var sb strings.Builder

	// SVG header
	width := 800
	height := 400
	sb.WriteString(fmt.Sprintf(`<svg width="%d" height="%d" xmlns="http://www.w3.org/2000/svg">`, width, height))
	sb.WriteString("\n")

	// Styles
	sb.WriteString(`<style>
		.state-box { fill: #f0f0f0; stroke: #333; stroke-width: 2; }
		.state-box.current { fill: #90EE90; stroke: #228B22; }
		.state-box.terminal { fill: #FFE4B5; stroke: #DAA520; }
		.state-box.aux { fill: #E6E6FA; stroke: #9370DB; }
		.state-text { font-family: Arial, sans-serif; font-size: 12px; text-anchor: middle; }
		.transition-line { stroke: #333; stroke-width: 2; fill: none; marker-end: url(#arrowhead); }
		.transition-label { font-family: Arial, sans-serif; font-size: 10px; fill: #666; text-anchor: middle; }
	</style>
`)

	// Arrowhead marker
	sb.WriteString(`<defs>
		<marker id="arrowhead" markerWidth="10" markerHeight="7" refX="9" refY="3.5" orient="auto">
			<polygon points="0 0, 10 3.5, 0 7" fill="#333" />
		</marker>
	</defs>
`)

	// Calculate positions for main flow (horizontal)
	phaseStates := []State{StateIdle, StatePlanning, StateImplementing, StateReviewing, StateDone}
	boxWidth := 100
	boxHeight := 50
	spacing := 50
	startX := 50
	mainY := 80

	statePositions := make(map[State][2]int)

	for i, state := range phaseStates {
		x := startX + i*(boxWidth+spacing)
		statePositions[state] = [2]int{x, mainY}

		// Determine CSS class
		var cssClass string
		if state == machine.State() {
			cssClass = "state-box current"
		} else if state == StateDone {
			cssClass = "state-box terminal"
		} else {
			cssClass = "state-box"
		}

		// Draw state box
		sb.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" class="%s" rx="5" />`,
			x, mainY, boxWidth, boxHeight, cssClass))
		sb.WriteString("\n")

		// Draw state label
		labelX := x + boxWidth/2
		labelY := mainY + boxHeight/2 + 4
		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="state-text">%s</text>`,
			labelX, labelY, state))
		sb.WriteString("\n")
	}

	// Draw transitions between main states
	for _, trans := range diag.Edges {
		if !trans.IsGlobal && trans.From != "" && trans.To != "" {
			fromPos, fromOk := statePositions[trans.From]
			toPos, toOk := statePositions[trans.To]

			if fromOk && toOk {
				// Draw arrow from right side of source to left side of target
				fromX := fromPos[0] + boxWidth
				fromY := fromPos[1] + boxHeight/2
				toX := toPos[0]
				toY := toPos[1] + boxHeight/2

				sb.WriteString(fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" class="transition-line" />`,
					fromX, fromY, toX, toY))
				sb.WriteString("\n")
			}
		}
	}

	// Draw auxiliary states below
	auxStates := []State{StateWaiting, StatePaused, StateCheckpointing, StateReverting, StateRestoring, StateFailed}
	auxY := 250
	auxStartX := 50

	for i, state := range auxStates {
		x := auxStartX + i*(boxWidth+spacing-30) // Slightly tighter spacing
		statePositions[state] = [2]int{x, auxY}

		isCurrent := state == machine.State()
		var cssClass string
		if isCurrent {
			cssClass = "state-box aux current"
		} else if state == StateFailed {
			cssClass = "state-box terminal"
		} else {
			cssClass = "state-box aux"
		}

		// Draw state box
		sb.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" class="%s" rx="5" />`,
			x, auxY, boxWidth, boxHeight, cssClass))
		sb.WriteString("\n")

		// Draw state label (wrap text if needed)
		labelX := x + boxWidth/2
		labelY := auxY + boxHeight/2 + 4
		displayName := string(state)
		if len(displayName) > 10 {
			// Wrap long names
			parts := []string{}
			for i := 0; i < len(displayName); i += 8 {
				end := i + 8
				if end > len(displayName) {
					end = len(displayName)
				}
				parts = append(parts, displayName[i:end])
			}
			for j, part := range parts {
				sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="state-text">%s</text>`,
					labelX, auxY+15+j*14, part))
				sb.WriteString("\n")
			}
		} else {
			sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="state-text">%s</text>`,
				labelX, labelY, displayName))
			sb.WriteString("\n")
		}
	}

	// Close SVG
	sb.WriteString("</svg>\n")

	return sb.String()
}

// GetValidTransitions returns all valid transitions from the current state.
func GetValidTransitions(machine *Machine) []Transition {
	current := machine.State()
	var transitions []Transition

	// Check transition table
	for key, trans := range TransitionTable {
		if key.From == current {
			transitions = append(transitions, trans...)
		}
	}

	// Add global transitions
	for event, toState := range GlobalTransitions {
		transitions = append(transitions, Transition{
			From:  current,
			Event: event,
			To:    toState,
		})
	}

	return transitions
}
