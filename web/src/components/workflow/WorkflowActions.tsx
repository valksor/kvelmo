import { Play, Code, CheckCircle, Flag, Undo2, Redo2, X, RotateCcw } from 'lucide-react'
import { useWorkflowAction } from '@/api/workflow'
import type { WorkflowState, WorkflowAction } from '@/types/api'

interface WorkflowActionsProps {
  state?: WorkflowState
  hasTask: boolean
}

interface ActionConfig {
  action: WorkflowAction
  label: string
  icon: React.ReactNode
  className: string
  disabled: (state: WorkflowState, hasTask: boolean) => boolean
  dangerous?: boolean
  confirm?: string
}

const actions: ActionConfig[] = [
  {
    action: 'plan',
    label: 'Plan',
    icon: <Play size={16} />,
    className: 'btn-info',
    // Plan always enabled when there's a task
    disabled: (_, hasTask) => !hasTask,
  },
  {
    action: 'implement',
    label: 'Implement',
    icon: <Code size={16} />,
    className: 'btn-primary',
    // Implement disabled in idle (need to plan first)
    disabled: (state, hasTask) => !hasTask || state === 'idle',
  },
  {
    action: 'review',
    label: 'Review',
    icon: <CheckCircle size={16} />,
    className: 'btn-secondary',
    // Review disabled until after implementing
    disabled: (state, hasTask) => !hasTask || state === 'idle' || state === 'planning',
  },
  {
    action: 'finish',
    label: 'Finish',
    icon: <Flag size={16} />,
    className: 'btn-success',
    // Finish after implementing (review is optional)
    disabled: (state, hasTask) => !hasTask || state === 'idle' || state === 'planning',
  },
]

const secondaryActions: ActionConfig[] = [
  {
    action: 'undo',
    label: 'Undo',
    icon: <Undo2 size={16} />,
    className: 'btn-ghost btn-sm',
    // Undo disabled in idle (nothing to undo)
    disabled: (state, hasTask) => !hasTask || state === 'idle',
  },
  {
    action: 'redo',
    label: 'Redo',
    icon: <Redo2 size={16} />,
    className: 'btn-ghost btn-sm',
    // Redo available same as undo (disabled in idle)
    disabled: (state, hasTask) => !hasTask || state === 'idle',
  },
  {
    action: 'abandon',
    label: 'Abandon',
    icon: <X size={16} />,
    className: 'btn-ghost btn-sm text-error',
    // Abandon enabled whenever there's a task
    disabled: (_, hasTask) => !hasTask,
    dangerous: true,
    confirm: 'Are you sure you want to abandon this task?',
  },
  {
    action: 'reset',
    label: 'Reset',
    icon: <RotateCcw size={16} />,
    className: 'btn-ghost btn-sm',
    // Reset disabled in idle
    disabled: (state, hasTask) => !hasTask || state === 'idle',
  },
]

export function WorkflowActions({ state = 'idle', hasTask }: WorkflowActionsProps) {
  const { mutate: executeAction, isPending } = useWorkflowAction()

  const handleAction = (config: ActionConfig) => {
    if (config.dangerous && config.confirm) {
      if (!window.confirm(config.confirm)) return
    }
    executeAction({ action: config.action })
  }

  return (
    <div className="card bg-base-100 shadow-sm">
      <div className="card-body">
        <h3 className="font-medium text-sm text-base-content/60 uppercase tracking-wide mb-3">
          Actions
        </h3>

        {/* Primary actions */}
        <div className="flex flex-col gap-2">
          {actions.map((config) => (
            <button
              key={config.action}
              className={`btn ${config.className} justify-start gap-2`}
              disabled={isPending || config.disabled(state, hasTask)}
              onClick={() => handleAction(config)}
            >
              {config.icon}
              {config.label}
            </button>
          ))}
        </div>

        {/* Secondary actions */}
        <div className="divider my-2" />
        <div className="flex flex-wrap gap-1">
          {secondaryActions.map((config) => (
            <button
              key={config.action}
              className={`btn ${config.className}`}
              disabled={isPending || config.disabled(state, hasTask)}
              onClick={() => handleAction(config)}
              title={config.label}
            >
              {config.icon}
            </button>
          ))}
        </div>
      </div>
    </div>
  )
}
