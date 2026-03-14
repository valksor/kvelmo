/**
 * Visual step indicator for the task lifecycle pipeline.
 * Shows the current phase with completed/active/future states.
 */

interface WorkflowStep {
  id: string
  label: string
  states: string[] // Task states that map to this step
}

const WORKFLOW_STEPS: WorkflowStep[] = [
  { id: 'load', label: 'Load', states: ['loaded'] },
  { id: 'plan', label: 'Plan', states: ['planning', 'planned'] },
  { id: 'implement', label: 'Implement', states: ['implementing', 'implemented'] },
  { id: 'simplify', label: 'Simplify', states: ['simplifying'] },
  { id: 'optimize', label: 'Optimize', states: ['optimizing'] },
  { id: 'review', label: 'Review', states: ['reviewing'] },
  { id: 'submit', label: 'Submit', states: ['submitted'] },
]

// States that indicate work in progress (show animation)
const ACTIVE_STATES = new Set(['planning', 'implementing', 'simplifying', 'optimizing', 'reviewing'])

interface WorkflowStepperProps {
  state: string
}

export function WorkflowStepper({ state }: WorkflowStepperProps) {
  if (state === 'none') return null

  const isFailed = state === 'failed'
  const isPaused = state === 'paused' || state === 'waiting'
  const isActive = ACTIVE_STATES.has(state)
  const stepIndex = WORKFLOW_STEPS.findIndex(step => step.states.includes(state))

  return (
    <div className="flex items-center gap-0.5 px-3 py-1.5 bg-base-200/50 border-b border-base-300 overflow-x-auto">
      {/* Global status indicator for failed/paused (not tied to a specific step) */}
      {(isFailed || isPaused) && (
        <span className={`text-[10px] sm:text-xs font-medium mr-1 ${isFailed ? 'text-error' : 'text-warning'}`}>
          {isFailed ? 'Failed' : 'Paused'}
        </span>
      )}
      {WORKFLOW_STEPS.map((step, index) => {
        const isCompleted = stepIndex > index
        const isCurrent = stepIndex === index

        let dotClass = 'w-2 h-2 rounded-full flex-shrink-0 transition-colors'
        let labelClass = 'text-[10px] sm:text-xs transition-colors whitespace-nowrap'
        let connectorClass = 'flex-1 h-px min-w-2 max-w-6 transition-colors'

        if (isCompleted) {
          dotClass += ' bg-success'
          labelClass += ' text-success'
          connectorClass += ' bg-success/50'
        } else if (isCurrent && isActive) {
          dotClass += ' bg-primary animate-pulse'
          labelClass += ' text-primary font-medium'
          connectorClass += ' bg-base-300'
        } else if (isCurrent) {
          dotClass += ' bg-primary'
          labelClass += ' text-primary font-medium'
          connectorClass += ' bg-base-300'
        } else {
          dotClass += ' bg-base-300'
          labelClass += ' text-base-content/30'
          connectorClass += ' bg-base-300'
        }

        return (
          <div key={step.id} className="contents">
            <div className="flex items-center gap-1">
              <div className={dotClass} />
              <span className={labelClass}>{step.label}</span>
            </div>
            {index < WORKFLOW_STEPS.length - 1 && (
              <div className={connectorClass} />
            )}
          </div>
        )
      })}
    </div>
  )
}
