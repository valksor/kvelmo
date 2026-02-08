package workflow

// Transition defines a state transition.
type Transition struct {
	From    State
	Event   Event
	To      State
	Guards  []GuardFunc
	Effects []EffectFunc
}

// TransitionKey uniquely identifies a transition.
type TransitionKey struct {
	From  State
	Event Event
}

// TransitionTable maps (state, event) pairs to transitions
// The new workflow is phase-based:
//
//	idle -> planning -> idle (specs created)
//	idle -> implementing -> idle (code changed)
//	idle -> reviewing -> idle (review done)
//	idle -> done (finish)
var TransitionTable = map[TransitionKey][]Transition{
	// === Start: Register task ===
	{StateIdle, EventStart}: {
		{From: StateIdle, Event: EventStart, To: StateIdle, Guards: []GuardFunc{GuardHasSource}},
	},

	// === Planning Phase ===
	{StateIdle, EventPlan}: {
		{From: StateIdle, Event: EventPlan, To: StatePlanning, Guards: []GuardFunc{GuardHasDescription}},
	},
	{StatePlanning, EventPlanDone}: {
		{From: StatePlanning, Event: EventPlanDone, To: StateIdle},
	},
	{StatePlanning, EventError}: {
		{From: StatePlanning, Event: EventError, To: StateIdle}, // Return to idle on error
	},
	{StatePlanning, EventCheckpoint}: {
		{From: StatePlanning, Event: EventCheckpoint, To: StateCheckpointing},
	},
	{StatePlanning, EventWait}: {
		{From: StatePlanning, Event: EventWait, To: StateWaiting},
	},
	{StatePlanning, EventPause}: {
		{From: StatePlanning, Event: EventPause, To: StatePaused},
	},

	// === Waiting (user input needed) ===
	{StateIdle, EventWait}: {
		{From: StateIdle, Event: EventWait, To: StateWaiting}, // Workflow needs user decision (e.g., finish action)
	},
	{StateWaiting, EventAnswer}: {
		{From: StateWaiting, Event: EventAnswer, To: StateIdle}, // Ready to continue
	},
	{StateWaiting, EventPlan}: {
		{From: StateWaiting, Event: EventPlan, To: StatePlanning}, // Re-enter planning after answer
	},
	{StateWaiting, EventPause}: {
		{From: StateWaiting, Event: EventPause, To: StatePaused},
	},
	{StateWaiting, EventError}: {
		{From: StateWaiting, Event: EventError, To: StateIdle},
	},

	// === Paused (budget limits) ===
	{StatePaused, EventResume}: {
		{From: StatePaused, Event: EventResume, To: StateIdle},
	},
	{StatePaused, EventError}: {
		{From: StatePaused, Event: EventError, To: StateIdle},
	},

	// === Implementation Phase ===
	{StateIdle, EventImplement}: {
		{From: StateIdle, Event: EventImplement, To: StateImplementing, Guards: []GuardFunc{GuardHasSpecifications}},
	},
	{StateImplementing, EventImplementDone}: {
		{From: StateImplementing, Event: EventImplementDone, To: StateIdle},
	},
	{StateImplementing, EventError}: {
		{From: StateImplementing, Event: EventError, To: StateIdle},
	},
	{StateImplementing, EventCheckpoint}: {
		{From: StateImplementing, Event: EventCheckpoint, To: StateCheckpointing},
	},
	{StateImplementing, EventUndo}: {
		{From: StateImplementing, Event: EventUndo, To: StateReverting, Guards: []GuardFunc{GuardCanUndo}},
	},
	{StateImplementing, EventPause}: {
		{From: StateImplementing, Event: EventPause, To: StatePaused},
	},

	// === Review Phase ===
	{StateIdle, EventReview}: {
		{From: StateIdle, Event: EventReview, To: StateReviewing, Guards: []GuardFunc{GuardCanReview}},
	},
	{StateReviewing, EventReviewDone}: {
		{From: StateReviewing, Event: EventReviewDone, To: StateIdle},
	},
	{StateReviewing, EventError}: {
		{From: StateReviewing, Event: EventError, To: StateIdle},
	},
	{StateReviewing, EventPause}: {
		{From: StateReviewing, Event: EventPause, To: StatePaused},
	},

	// === Finish ===
	{StateIdle, EventFinish}: {
		{From: StateIdle, Event: EventFinish, To: StateDone, Guards: []GuardFunc{GuardCanFinish}},
	},

	// === Failed State Recovery ===
	{StateFailed, EventReset}: {
		{From: StateFailed, Event: EventReset, To: StateIdle},
	},

	// === Checkpointing (return to previous phase) ===
	{StateCheckpointing, EventCheckpointDone}: {
		// After checkpoint, return to idle for user to decide next step
		{From: StateCheckpointing, Event: EventCheckpointDone, To: StateIdle},
	},

	// === Undo/Redo ===
	{StateIdle, EventUndo}: {
		{From: StateIdle, Event: EventUndo, To: StateReverting, Guards: []GuardFunc{GuardCanUndo}},
	},
	{StateIdle, EventRedo}: {
		{From: StateIdle, Event: EventRedo, To: StateRestoring, Guards: []GuardFunc{GuardCanRedo}},
	},
	{StateReverting, EventUndoDone}: {
		{From: StateReverting, Event: EventUndoDone, To: StateIdle},
	},
	{StateReverting, EventError}: {
		{From: StateReverting, Event: EventError, To: StateIdle},
	},
	{StateRestoring, EventRedoDone}: {
		{From: StateRestoring, Event: EventRedoDone, To: StateIdle},
	},
	{StateRestoring, EventError}: {
		{From: StateRestoring, Event: EventError, To: StateIdle},
	},

	// === Abort (explicit per-state, excludes terminal StateDone and StateFailed) ===
	{StateIdle, EventAbort}: {
		{From: StateIdle, Event: EventAbort, To: StateFailed},
	},
	{StatePlanning, EventAbort}: {
		{From: StatePlanning, Event: EventAbort, To: StateFailed},
	},
	{StateImplementing, EventAbort}: {
		{From: StateImplementing, Event: EventAbort, To: StateFailed},
	},
	{StateReviewing, EventAbort}: {
		{From: StateReviewing, Event: EventAbort, To: StateFailed},
	},
	{StateWaiting, EventAbort}: {
		{From: StateWaiting, Event: EventAbort, To: StateFailed},
	},
	{StatePaused, EventAbort}: {
		{From: StatePaused, Event: EventAbort, To: StateFailed},
	},
	{StateCheckpointing, EventAbort}: {
		{From: StateCheckpointing, Event: EventAbort, To: StateFailed},
	},
	{StateReverting, EventAbort}: {
		{From: StateReverting, Event: EventAbort, To: StateFailed},
	},
	{StateRestoring, EventAbort}: {
		{From: StateRestoring, Event: EventAbort, To: StateFailed},
	},
}

// GlobalTransitions apply from any state.
// EventAbort was moved to explicit per-state entries to prevent aborting from
// terminal states (StateDone, StateFailed).
var GlobalTransitions = map[Event]State{}

// GetTransitions returns possible transitions for a state/event pair.
func GetTransitions(from State, event Event) []Transition {
	key := TransitionKey{From: from, Event: event}

	return TransitionTable[key]
}

// GetGlobalTransition returns the target state for global events.
func GetGlobalTransition(event Event) (State, bool) {
	to, ok := GlobalTransitions[event]

	return to, ok
}

// CanTransition checks if a transition is possible without guards.
func CanTransition(from State, event Event) bool {
	_, ok := GlobalTransitions[event]
	if ok {
		return true
	}
	key := TransitionKey{From: from, Event: event}
	transitions, ok := TransitionTable[key]

	return ok && len(transitions) > 0
}
