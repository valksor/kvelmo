const HINTS: Record<string, string> = {
  none: 'No active task. Load one with the task panel or `kvelmo start`.',
  loaded: 'Task loaded. Ready to plan \u2014 click Plan or run `kvelmo plan`.',
  planning: 'Planning in progress... The agent is generating specifications.',
  planned: 'Plan complete. Review the spec, then click Implement.',
  implementing: 'Implementation in progress... The agent is writing code.',
  implemented: 'Code complete. Run Review to check quality before submitting.',
  optimizing: 'Optimization in progress...',
  reviewing: 'Review in progress...',
  submitted: 'PR submitted. Waiting for merge, then run Finish to clean up.',
  failed: 'Last action failed. Check the error and try again, or Undo.',
  waiting: 'Waiting for input. Check for quality gate prompts.',
  paused: 'Task paused.',
}

function HintIcon() {
  return (
    <svg
      aria-hidden="true"
      className="w-4 h-4 shrink-0"
      fill="none"
      viewBox="0 0 24 24"
      stroke="currentColor"
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
      />
    </svg>
  )
}

interface WorkflowHintProps {
  state: string
}

export function WorkflowHint({ state }: WorkflowHintProps) {
  const hint = HINTS[state]
  if (!hint) return null

  return (
    <div role="status" className="alert alert-info alert-sm rounded-none py-1.5 px-4 text-sm">
      <HintIcon />
      <span>{hint}</span>
    </div>
  )
}
