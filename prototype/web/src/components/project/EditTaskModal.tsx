import { useState, useEffect } from 'react'
import { X, Loader2, Save } from 'lucide-react'
import { useUpdateTask, type PlanTask, type UpdateTaskRequest } from '@/api/project-planning'

interface EditTaskModalProps {
  task: PlanTask | null
  onClose: () => void
}

export function EditTaskModal({ task, onClose }: EditTaskModalProps) {
  const updateTask = useUpdateTask()

  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [priority, setPriority] = useState(3)
  const [status, setStatus] = useState<string>('pending')
  const [parentId, setParentId] = useState('')
  const [dependsOn, setDependsOn] = useState('')
  const [labels, setLabels] = useState('')

  // Reset form when task changes - this is a valid pattern for form initialization
  useEffect(() => {
    if (task) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- form initialization from props
      setTitle(task.title)
      setDescription(task.description)
      setPriority(task.priority)
      setStatus(task.status)
      setParentId(task.parent_id || '')
      setDependsOn(task.depends_on.join(', '))
      setLabels(task.labels.join(', '))
    }
  }, [task])

  if (!task) return null

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    const data: UpdateTaskRequest = {
      title,
      description,
      priority,
      status,
      parent_id: parentId || undefined,
      depends_on: dependsOn
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean),
      labels: labels
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean),
    }

    await updateTask.mutateAsync({ taskId: task.id, data })
    onClose()
  }

  return (
    <div className="modal modal-open">
      <div className="modal-box max-w-2xl">
        {/* Header */}
        <div className="flex items-center justify-between mb-4">
          <h3 className="font-bold text-lg">Edit Task</h3>
          <button className="btn btn-ghost btn-sm btn-circle" onClick={onClose}>
            <X size={18} />
          </button>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit} className="space-y-4">
          {/* Title */}
          <div className="form-control">
            <label className="label">
              <span className="label-text">Title</span>
            </label>
            <input
              type="text"
              className="input input-bordered w-full"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              required
            />
          </div>

          {/* Description */}
          <div className="form-control">
            <label className="label">
              <span className="label-text">Description</span>
            </label>
            <textarea
              className="textarea textarea-bordered w-full h-32"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Task description (markdown supported)"
            />
          </div>

          {/* Priority + Status */}
          <div className="grid grid-cols-2 gap-4">
            <div className="form-control">
              <label className="label">
                <span className="label-text">Priority</span>
              </label>
              <select
                className="select select-bordered w-full"
                value={priority}
                onChange={(e) => setPriority(Number(e.target.value))}
              >
                <option value={1}>Highest (1)</option>
                <option value={2}>High (2)</option>
                <option value={3}>Medium (3)</option>
                <option value={4}>Low (4)</option>
                <option value={5}>Lowest (5)</option>
              </select>
            </div>
            <div className="form-control">
              <label className="label">
                <span className="label-text">Status</span>
              </label>
              <select
                className="select select-bordered w-full"
                value={status}
                onChange={(e) => setStatus(e.target.value)}
              >
                <option value="pending">Pending</option>
                <option value="ready">Ready</option>
                <option value="blocked">Blocked</option>
                <option value="submitted">Submitted</option>
              </select>
            </div>
          </div>

          {/* Parent ID */}
          <div className="form-control">
            <label className="label">
              <span className="label-text">Parent Task ID</span>
            </label>
            <input
              type="text"
              className="input input-bordered w-full font-mono text-sm"
              value={parentId}
              onChange={(e) => setParentId(e.target.value)}
              placeholder="Optional parent task ID"
            />
          </div>

          {/* Dependencies */}
          <div className="form-control">
            <label className="label">
              <span className="label-text">Dependencies</span>
              <span className="label-text-alt">Comma-separated task IDs</span>
            </label>
            <input
              type="text"
              className="input input-bordered w-full font-mono text-sm"
              value={dependsOn}
              onChange={(e) => setDependsOn(e.target.value)}
              placeholder="task-1, task-2"
            />
          </div>

          {/* Labels */}
          <div className="form-control">
            <label className="label">
              <span className="label-text">Labels</span>
              <span className="label-text-alt">Comma-separated</span>
            </label>
            <input
              type="text"
              className="input input-bordered w-full"
              value={labels}
              onChange={(e) => setLabels(e.target.value)}
              placeholder="bug, urgent, backend"
            />
          </div>

          {/* Error display */}
          {updateTask.error && (
            <div className="alert alert-error text-sm">
              {updateTask.error instanceof Error
                ? updateTask.error.message
                : 'Failed to update task'}
            </div>
          )}

          {/* Actions */}
          <div className="modal-action">
            <button type="button" className="btn btn-ghost" onClick={onClose}>
              Cancel
            </button>
            <button
              type="submit"
              className="btn btn-primary gap-2"
              disabled={updateTask.isPending}
            >
              {updateTask.isPending ? (
                <Loader2 size={16} className="animate-spin" />
              ) : (
                <Save size={16} />
              )}
              Save Changes
            </button>
          </div>
        </form>
      </div>
      <div className="modal-backdrop" onClick={onClose} />
    </div>
  )
}
