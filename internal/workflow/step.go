package workflow

// Step represents a workflow step that can have its own agent configuration.
type Step string

const (
	// StepPlanning is the planning phase where specifications are created.
	StepPlanning Step = "planning"
	// StepImplementing is the implementation phase where code is written.
	StepImplementing Step = "implementing"
	// StepReviewImplementing is the phase where review fixes are implemented.
	// This is distinct from StepImplementing to allow separate agent configuration
	// and tracking when implementing fixes from code review feedback.
	StepReviewImplementing Step = "review_implementing"
	// StepReviewing is the review phase where code is reviewed.
	StepReviewing Step = "reviewing"
	// StepCheckpointing is the checkpointing phase for git operations.
	StepCheckpointing Step = "checkpointing"
	// StepSimplifying is the simplification phase where content is refined.
	StepSimplifying Step = "simplifying"
	// StepOptimizing is the prompt optimization phase where prompts are refined.
	StepOptimizing Step = "optimizing"
	// StepQuality is the quality review phase in auto mode.
	StepQuality Step = "quality"
	// StepPRReview is the standalone PR review step.
	StepPRReview Step = "pr_review"
)

// AllSteps returns all valid step names.
func AllSteps() []Step {
	return []Step{
		StepPlanning,
		StepImplementing,
		StepReviewImplementing,
		StepReviewing,
		StepCheckpointing,
		StepSimplifying,
		StepOptimizing,
		StepQuality,
		StepPRReview,
	}
}

// IsValidStep checks if a string is a valid step name.
func IsValidStep(s string) bool {
	for _, step := range AllSteps() {
		if string(step) == s {
			return true
		}
	}

	return false
}

// String returns the string representation of the step.
func (s Step) String() string {
	return string(s)
}
