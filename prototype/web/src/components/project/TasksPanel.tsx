import { useId, useState } from 'react'
import {
  Loader2,
  Edit2,
  Play,
  Sparkles,
  Send,
  ChevronRight,
  AlertCircle,
  CheckCircle,
  Clock,
  Lock,
} from 'lucide-react'
import {
  useQueueTasks,
  useSubmitTasks,
  useReorderTasks,
  useStartImplementation,
  type PlanTask,
} from '@/api/project-planning'
import { TASK_SUBMISSION_PROVIDERS } from '@/constants/taskOptions'

interface TasksPanelProps {
  queueId?: string
  onEditTask: (task: PlanTask) => void
}

export function TasksPanel({ queueId, onEditTask }: TasksPanelProps) {
  const id = useId()
  const { data, isLoading, error } = useQueueTasks(queueId)
  const submitTasks = useSubmitTasks()
  const reorderTasks = useReorderTasks()
  const startImpl = useStartImplementation()

  const [provider, setProvider] = useState('github')
  const [mention, setMention] = useState('')
  const [dryRun, setDryRun] = useState(false)

  if (!queueId) {
    return (
      <div className="text-center py-12 text-base-content/60">
        Select a queue to view its tasks
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="w-6 h-6 animate-spin text-primary" aria-hidden="true" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="alert alert-error">
        Failed to load tasks: {error.message}
      </div>
    )
  }

  const tasks = data?.tasks || []
  const queueTitle = data?.queue_title || 'Tasks'

  const handleSubmit = async () => {
    if (!queueId) return
    await submitTasks.mutateAsync({
      queue_id: queueId,
      provider,
      mention: mention || undefined,
      dry_run: dryRun,
    })
  }

  const handleReorder = async () => {
    if (!queueId) return
    await reorderTasks.mutateAsync({ queue_id: queueId })
  }

  const handleStartImpl = async () => {
    if (!queueId) return
    await startImpl.mutateAsync({ queue_id: queueId })
  }

  // Build task tree for hierarchical display
  const rootTasks = tasks.filter((t) => !t.parent_id)
  const getChildren = (parentId: string) => tasks.filter((t) => t.parent_id === parentId)

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h3 className="font-medium">{queueTitle}</h3>
        <span className="text-sm text-base-content/60">
          {tasks.length} task{tasks.length !== 1 ? 's' : ''}
        </span>
      </div>

      {/* Tasks table */}
      {tasks.length === 0 ? (
        <div className="text-center py-8 text-base-content/60">
          No tasks in this queue
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="table table-sm">
            <thead>
              <tr>
                <th>Task</th>
                <th>Status</th>
                <th>Priority</th>
                <th>Dependencies</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {rootTasks.map((task) => (
                <TaskRows
                  key={task.id}
                  task={task}
                  depth={0}
                  getChildren={getChildren}
                  onEdit={onEditTask}
                />
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Actions */}
      {tasks.length > 0 && (
        <div className="card bg-base-200/50 p-4 space-y-4">
          {/* AI Reorder */}
          <div className="flex items-center justify-between">
            <span className="text-sm text-base-content/70">Optimize task order with AI</span>
            <button
              className="btn btn-ghost btn-sm gap-1"
              onClick={handleReorder}
              disabled={reorderTasks.isPending}
            >
              {reorderTasks.isPending ? (
                <Loader2 size={14} className="animate-spin" aria-hidden="true" />
              ) : (
                <Sparkles size={14} aria-hidden="true" />
              )}
              AI Suggest Order
            </button>
          </div>

          {/* Submit to provider */}
          <div className="border-t border-base-300 pt-4">
            <h4 className="text-sm font-medium mb-3">Submit to Provider</h4>
            <div className="flex flex-wrap gap-3 items-end">
              <div>
                <label className="text-xs text-base-content/60" htmlFor={`${id}-provider`}>Provider</label>
                <select
                  id={`${id}-provider`}
                  className="select select-bordered"
                  value={provider}
                  onChange={(e) => setProvider(e.target.value)}
                >
                  {TASK_SUBMISSION_PROVIDERS.map((p) => (
                    <option key={p.value} value={p.value}>
                      {p.label}
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label className="text-xs text-base-content/60" htmlFor={`${id}-mention`}>Mention (optional)</label>
                <input
                  id={`${id}-mention`}
                  type="text"
                  placeholder="@username"
                  className="input input-bordered w-32"
                  value={mention}
                  onChange={(e) => setMention(e.target.value)}
                />
              </div>
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  className="checkbox checkbox-primary"
                  checked={dryRun}
                  onChange={(e) => setDryRun(e.target.checked)}
                />
                <span className="text-sm">Dry run</span>
              </label>
              <button
                className="btn btn-primary btn-sm gap-1"
                onClick={handleSubmit}
                disabled={submitTasks.isPending}
              >
                {submitTasks.isPending ? (
                  <Loader2 size={14} className="animate-spin" aria-hidden="true" />
                ) : (
                  <Send size={14} aria-hidden="true" />
                )}
                Submit
              </button>
            </div>
          </div>

          {/* Start Implementation */}
          <div className="border-t border-base-300 pt-4">
            <button
              className="btn btn-success w-full gap-2"
              onClick={handleStartImpl}
              disabled={startImpl.isPending}
            >
              {startImpl.isPending ? (
                <Loader2 size={16} className="animate-spin" aria-hidden="true" />
              ) : (
                <Play size={16} aria-hidden="true" />
              )}
              Start Implementation
            </button>
          </div>
        </div>
      )}
    </div>
  )
}

interface TaskRowsProps {
  task: PlanTask
  depth: number
  getChildren: (parentId: string) => PlanTask[]
  onEdit: (task: PlanTask) => void
}

function TaskRows({ task, depth, getChildren, onEdit }: TaskRowsProps) {
  const children = getChildren(task.id)

  const statusIcon = {
    ready: <CheckCircle size={14} className="text-success" aria-hidden="true" />,
    pending: <Clock size={14} className="text-base-content/50" aria-hidden="true" />,
    blocked: <Lock size={14} className="text-warning" aria-hidden="true" />,
    submitted: <AlertCircle size={14} className="text-info" aria-hidden="true" />,
  }[task.status] || <Clock size={14} aria-hidden="true" />

  const statusBadge = {
    ready: 'badge-success',
    pending: 'badge-ghost',
    blocked: 'badge-warning',
    submitted: 'badge-info',
  }[task.status] || 'badge-ghost'

  const priorityLabel = ['', 'Highest', 'High', 'Medium', 'Low', 'Lowest'][task.priority] || ''

  return (
    <>
      <tr className="hover">
        <td>
          <div className="flex items-center gap-1" style={{ paddingLeft: depth * 20 }}>
            {children.length > 0 && <ChevronRight size={14} className="text-base-content/40" aria-hidden="true" />}
            <div>
              <div className="font-medium">{task.title}</div>
              <div className="text-xs text-base-content/50 font-mono">{task.id}</div>
            </div>
          </div>
        </td>
        <td>
          <span className={`badge badge-sm gap-1 ${statusBadge}`}>
            {statusIcon}
            {task.status}
          </span>
        </td>
        <td className="text-sm">{priorityLabel}</td>
        <td>
          {task.depends_on.length > 0 ? (
            <span className="text-xs text-base-content/60">
              {task.depends_on.length} dep{task.depends_on.length !== 1 ? 's' : ''}
            </span>
          ) : (
            <span className="text-xs text-base-content/40">-</span>
          )}
        </td>
        <td>
          <button
            className="btn btn-ghost btn-xs"
            onClick={() => onEdit(task)}
            aria-label="Edit task"
          >
            <Edit2 size={14} aria-hidden="true" />
          </button>
        </td>
      </tr>
      {children.map((child) => (
        <TaskRows
          key={child.id}
          task={child}
          depth={depth + 1}
          getChildren={getChildren}
          onEdit={onEdit}
        />
      ))}
    </>
  )
}
