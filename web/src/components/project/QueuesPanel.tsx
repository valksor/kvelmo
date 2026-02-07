import { Loader2, Trash2, Eye, RefreshCw, FolderOpen } from 'lucide-react'
import { useQueues, useDeleteQueue, type QueueSummary } from '@/api/project-planning'
import { formatDistanceToNow } from 'date-fns'

interface QueuesPanelProps {
  onSelectQueue: (queueId: string) => void
  selectedQueueId?: string
}

export function QueuesPanel({ onSelectQueue, selectedQueueId }: QueuesPanelProps) {
  const { data, isLoading, error, refetch } = useQueues()
  const deleteQueue = useDeleteQueue()

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="w-6 h-6 animate-spin text-primary" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="alert alert-error">
        Failed to load queues: {error.message}
      </div>
    )
  }

  const queues = data?.queues || []

  if (queues.length === 0) {
    return (
      <div className="text-center py-12">
        <FolderOpen className="w-12 h-12 mx-auto text-base-content/30 mb-4" />
        <p className="text-base-content/60">No queues yet</p>
        <p className="text-sm text-base-content/40 mt-1">
          Create a plan using the "Create Plan" tab to generate a task queue
        </p>
      </div>
    )
  }

  const handleDelete = async (queueId: string, e: React.MouseEvent) => {
    e.stopPropagation()
    if (confirm('Delete this queue and all its tasks?')) {
      await deleteQueue.mutateAsync(queueId)
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium text-base-content/70">
          {queues.length} queue{queues.length !== 1 ? 's' : ''}
        </h3>
        <button className="btn btn-ghost btn-xs" onClick={() => refetch()}>
          <RefreshCw size={14} />
          Refresh
        </button>
      </div>

      <div className="space-y-2">
        {queues.map((queue) => (
          <QueueCard
            key={queue.id}
            queue={queue}
            isSelected={selectedQueueId === queue.id}
            onSelect={() => onSelectQueue(queue.id)}
            onDelete={(e) => handleDelete(queue.id, e)}
            isDeleting={deleteQueue.isPending}
          />
        ))}
      </div>
    </div>
  )
}

interface QueueCardProps {
  queue: QueueSummary
  isSelected: boolean
  onSelect: () => void
  onDelete: (e: React.MouseEvent) => void
  isDeleting: boolean
}

function QueueCard({ queue, isSelected, onSelect, onDelete, isDeleting }: QueueCardProps) {
  const createdAgo = formatDistanceToNow(new Date(queue.created_at), { addSuffix: true })

  const statusBadge = {
    active: 'badge-primary',
    completed: 'badge-success',
    pending: 'badge-ghost',
  }[queue.status] || 'badge-ghost'

  return (
    <div
      role="button"
      tabIndex={0}
      aria-pressed={isSelected}
      className={`card bg-base-100 shadow-sm cursor-pointer transition-all hover:shadow-md ${
        isSelected ? 'ring-2 ring-primary' : ''
      }`}
      onClick={onSelect}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          onSelect()
        }
      }}
    >
      <div className="card-body py-3 px-4">
        <div className="flex items-start justify-between gap-2">
          <div className="flex-1 min-w-0">
            <h4 className="font-medium truncate">{queue.title || queue.id}</h4>
            <p className="text-xs text-base-content/50 truncate">{queue.source}</p>
          </div>
          <div className="flex items-center gap-2">
            <span className={`badge badge-sm ${statusBadge}`}>{queue.status}</span>
          </div>
        </div>

        <div className="flex items-center justify-between mt-2">
          <div className="flex items-center gap-3 text-xs text-base-content/60">
            <span>{queue.task_count} task{queue.task_count !== 1 ? 's' : ''}</span>
            <span>{createdAgo}</span>
          </div>
          <div className="flex items-center gap-1">
            <button
              className="btn btn-ghost btn-xs"
              onClick={onSelect}
              title="View tasks"
              aria-label="View queue tasks"
            >
              <Eye size={14} />
            </button>
            <button
              className="btn btn-ghost btn-xs text-error"
              onClick={onDelete}
              disabled={isDeleting}
              title="Delete queue"
              aria-label="Delete queue"
            >
              <Trash2 size={14} />
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
