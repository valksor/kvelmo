import { useState, useMemo } from 'react'
import {
  Play,
  Code,
  CheckCircle,
  Flag,
  RefreshCw,
  Undo2,
  Redo2,
  X,
  RotateCcw,
  ChevronDown,
  ChevronRight,
} from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { useWorkflowAction } from '@/api/workflow'
import type {
  WorkflowState,
  WorkflowAction,
  Specification,
  ImplementOptions,
  ProgressPhase,
  WorkflowSyncResponse,
} from '@/types/api'

interface WorkflowActionsProps {
  state?: WorkflowState
  hasTask: boolean
  taskId?: string
  progressPhase?: ProgressPhase
  specs?: Specification[]
}

interface ActionConfig {
  action: WorkflowAction
  label: string
  icon: React.ReactNode
  className: string
  disabled: (state: WorkflowState, hasTask: boolean, phase: ProgressPhase) => boolean
  dangerous?: boolean
  confirm?: string
}

const isActive = (s: WorkflowState) => s.endsWith('ing')

const primaryActions: ActionConfig[] = [
  {
    action: 'plan',
    label: 'Plan',
    icon: <Play size={16} aria-hidden="true" />,
    className: 'btn-info',
    disabled: (state, hasTask) => !hasTask || isActive(state),
  },
  {
    action: 'implement',
    label: 'Implement',
    icon: <Code size={16} aria-hidden="true" />,
    className: 'btn-primary',
    disabled: (state, hasTask, phase) => !hasTask || isActive(state) || phase === 'started',
  },
  {
    action: 'review',
    label: 'Review',
    icon: <CheckCircle size={16} aria-hidden="true" />,
    className: 'btn-secondary',
    disabled: (state, hasTask, phase) => !hasTask || isActive(state) || phase === 'started' || phase === 'planned',
  },
  {
    action: 'finish',
    label: 'Finish',
    icon: <Flag size={16} aria-hidden="true" />,
    className: 'btn-success',
    disabled: (state, hasTask, phase) => !hasTask || isActive(state) || phase === 'started' || phase === 'planned',
  },
]

const advancedActions: ActionConfig[] = [
  {
    action: 'sync',
    label: 'Sync',
    icon: <RefreshCw size={16} aria-hidden="true" />,
    className: 'btn-outline',
    disabled: (state, hasTask, phase) => !hasTask || isActive(state) || phase === 'started',
  },
]

const secondaryActions: ActionConfig[] = [
  {
    action: 'undo',
    label: 'Undo',
    icon: <Undo2 size={16} aria-hidden="true" />,
    className: 'btn-ghost btn-sm',
    disabled: (state, hasTask, phase) => !hasTask || isActive(state) || phase === 'started',
  },
  {
    action: 'redo',
    label: 'Redo',
    icon: <Redo2 size={16} aria-hidden="true" />,
    className: 'btn-ghost btn-sm',
    disabled: (state, hasTask, phase) => !hasTask || isActive(state) || phase === 'started',
  },
  {
    action: 'abandon',
    label: 'Abandon',
    icon: <X size={16} aria-hidden="true" />,
    className: 'btn-ghost btn-sm text-error',
    disabled: (state, hasTask) => !hasTask || isActive(state),
    dangerous: true,
    confirm: 'Are you sure you want to abandon this task?',
  },
  {
    action: 'reset',
    label: 'Reset',
    icon: <RotateCcw size={16} aria-hidden="true" />,
    className: 'btn-ghost btn-sm',
    disabled: (state, hasTask) => !hasTask || isActive(state) || state === 'idle',
  },
]

function isSyncResponse(data: unknown): data is WorkflowSyncResponse {
  if (!data || typeof data !== 'object') {
    return false
  }
  const value = data as Record<string, unknown>
  return typeof value.message === 'string' && typeof value.has_changes === 'boolean'
}

export function WorkflowActions({
  state = 'idle',
  hasTask,
  taskId,
  progressPhase = 'started',
  specs = [],
}: WorkflowActionsProps) {
  const { mutate: executeAction, isPending } = useWorkflowAction()
  const navigate = useNavigate()
  const [syncResult, setSyncResult] = useState<WorkflowSyncResponse | null>(null)

  // Implementation options state
  const [showImplementOptions, setShowImplementOptions] = useState(false)
  const [showAdvancedActions, setShowAdvancedActions] = useState(false)
  const [selectedComponent, setSelectedComponent] = useState('')
  const [parallelCount, setParallelCount] = useState(0)

  // Extract unique components from specs
  const components = useMemo(() => {
    const safeSpecs = specs ?? []
    const uniqueComponents = new Set(
      safeSpecs.map((s) => s.component).filter(Boolean)
    )
    return Array.from(uniqueComponents).sort()
  }, [specs])

  const handleAction = (config: ActionConfig, implementOptions?: ImplementOptions) => {
    if (config.action === 'sync' && !taskId) {
      return
    }
    if (config.dangerous && config.confirm) {
      if (!window.confirm(config.confirm)) return
    }
    const options: Record<string, unknown> | undefined =
      config.action === 'sync' ? { task_id: taskId } : undefined
    executeAction(
      { action: config.action, options, implementOptions },
      {
        onSuccess: (data: unknown) => {
          if (config.action === 'sync') {
            setSyncResult(isSyncResponse(data) ? data : null)
          } else {
            setSyncResult(null)
          }
          if (config.action === 'abandon' || config.action === 'finish') {
            navigate('/')
          }
        },
      }
    )
  }

  const handleImplementWithOptions = () => {
    const options: ImplementOptions = {}
    if (selectedComponent) {
      options.component = selectedComponent
    }
    if (parallelCount > 0) {
      options.parallel = parallelCount
    }
    executeAction({ action: 'implement', implementOptions: options })
  }

  const implementConfig = primaryActions.find((a) => a.action === 'implement')!
  const isImplementDisabled = isPending || implementConfig.disabled(state, hasTask, progressPhase)

  return (
    <div className="card bg-base-100 shadow-sm">
      <div className="card-body">
        <h3 className="font-medium text-sm text-base-content/60 uppercase tracking-wide mb-3">
          Actions
        </h3>
        {syncResult && (
          <div className={`alert mb-3 ${syncResult.has_changes ? 'alert-info' : 'alert-success'}`}>
            <div className="space-y-1 text-sm">
              <div>{syncResult.message}</div>
              {syncResult.spec_generated && (
                <div>
                  Delta specification: <code>{syncResult.spec_generated}</code>
                </div>
              )}
              {syncResult.changes_summary && <div>Summary: {syncResult.changes_summary}</div>}
              {typeof syncResult.source_updated === 'boolean' && (
                <div>Source updated: {syncResult.source_updated ? 'yes' : 'no'}</div>
              )}
              {syncResult.previous_snapshot_path && (
                <div>
                  Previous snapshot: <code>{syncResult.previous_snapshot_path}</code>
                </div>
              )}
              {syncResult.diff_path && (
                <div>
                  Diff file: <code>{syncResult.diff_path}</code>
                </div>
              )}
              {syncResult.warnings && syncResult.warnings.length > 0 && (
                <div>
                  Warnings: {syncResult.warnings.join(' | ')}
                </div>
              )}
            </div>
          </div>
        )}

        {/* Primary actions */}
        <div className="flex flex-col gap-2">
          {primaryActions.map((config) => {
            // Special handling for implement button
            if (config.action === 'implement') {
              return (
                <div key={config.action} className="flex flex-col gap-1">
                  <div className="flex gap-1 items-stretch">
                    <button
                      className={`btn ${config.className} justify-start gap-2 flex-1`}
                      disabled={isImplementDisabled}
                      onClick={() => handleAction(config)}
                    >
                      {config.icon}
                      {config.label}
                    </button>

                    {showAdvancedActions && (
                      <button
                        className={`btn ${config.className} btn-square`}
                        disabled={isImplementDisabled}
                        onClick={() => setShowImplementOptions(!showImplementOptions)}
                        aria-label="Implementation options"
                        aria-expanded={showImplementOptions}
                      >
                        {showImplementOptions ? <ChevronDown size={16} aria-hidden="true" /> : <ChevronRight size={16} aria-hidden="true" />}
                      </button>
                    )}
                  </div>

                  {/* Implementation options panel */}
                  {showAdvancedActions && showImplementOptions && (
                    <div className="p-3 bg-base-200/50 rounded-lg space-y-3 mt-1">
                      <div>
                        <label htmlFor="impl-component" className="block text-xs font-medium text-base-content/60 mb-1">
                          Component
                        </label>
                        <select
                          id="impl-component"
                          value={selectedComponent}
                          onChange={(e) => setSelectedComponent(e.target.value)}
                          className="select select-bordered w-full"
                          disabled={isPending}
                        >
                          <option value="">All components</option>
                          {components.map((c) => (
                            <option key={c} value={c}>
                              {c}
                            </option>
                          ))}
                        </select>
                      </div>
                      <div>
                        <label htmlFor="impl-parallel" className="block text-xs font-medium text-base-content/60 mb-1">
                          Parallel workers
                        </label>
                        <input
                          id="impl-parallel"
                          type="number"
                          min={0}
                          max={10}
                          value={parallelCount}
                          onChange={(e) => setParallelCount(Number(e.target.value))}
                          placeholder="0 = default"
                          className="input input-bordered w-full"
                          disabled={isPending}
                        />
                        <p className="text-xs text-base-content/40 mt-1">0 = sequential execution</p>
                      </div>
                      <button
                        className="btn btn-primary btn-sm w-full"
                        onClick={handleImplementWithOptions}
                        disabled={isImplementDisabled}
                      >
                        Implement with options
                      </button>
                    </div>
                  )}
                </div>
              )
            }

            // Regular button for other actions
            return (
              <button
                key={config.action}
                className={`btn ${config.className} justify-start gap-2`}
                disabled={isPending || config.disabled(state, hasTask, progressPhase)}
                onClick={() => handleAction(config)}
              >
                {config.icon}
                {config.label}
              </button>
            )
          })}
        </div>

        {/* Secondary actions */}
        <div className="divider my-2" />
        <button
          className="btn btn-ghost btn-sm justify-start gap-2"
          type="button"
          onClick={() => {
            const next = !showAdvancedActions
            setShowAdvancedActions(next)
            if (!next) {
              setShowImplementOptions(false)
              setSelectedComponent('')
              setParallelCount(0)
            }
          }}
        >
          {showAdvancedActions ? <ChevronDown size={16} aria-hidden="true" /> : <ChevronRight size={16} aria-hidden="true" />}
          Advanced actions
        </button>

        {showAdvancedActions && (
          <>
            <div className="flex flex-col gap-2">
              {advancedActions.map((config) => (
                <button
                  key={config.action}
                  className={`btn ${config.className} justify-start gap-2`}
                  disabled={isPending || config.disabled(state, hasTask, progressPhase)}
                  onClick={() => handleAction(config)}
                >
                  {config.icon}
                  {config.label}
                </button>
              ))}
            </div>
            <div className="flex flex-wrap gap-1 mt-2">
              {secondaryActions.map((config) => (
                <button
                  key={config.action}
                  className={`btn ${config.className}`}
                  disabled={isPending || config.disabled(state, hasTask, progressPhase)}
                  onClick={() => handleAction(config)}
                  aria-label={config.label}
                >
                  {config.icon}
                </button>
              ))}
            </div>
          </>
        )}
      </div>
    </div>
  )
}
