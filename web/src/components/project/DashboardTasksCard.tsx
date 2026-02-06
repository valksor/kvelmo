import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { ChevronRight, FolderGit2, ListTree, Loader2, History } from 'lucide-react'
import { useQueues, type PlanTask } from '@/api/project-planning'
import type { TaskHistoryItem } from '@/types/api'
import { getStateConfigWithProgress } from '@/constants/stateConfig'
import { TasksPanel } from '@/components/project/TasksPanel'
import { EditTaskModal } from '@/components/project/EditTaskModal'

type TasksView = 'recent' | 'queue'

interface DashboardTasksCardProps {
  tasks?: TaskHistoryItem[]
  isHistoryLoading?: boolean
}

export function DashboardTasksCard({ tasks, isHistoryLoading }: DashboardTasksCardProps) {
  const [view, setView] = useState<TasksView>('recent')
  const [selectedQueueId, setSelectedQueueId] = useState<string>('')
  const [editingTask, setEditingTask] = useState<PlanTask | null>(null)

  const {
    data: queuesData,
    isLoading: queuesLoading,
    error: queuesError,
  } = useQueues({ enabled: view === 'queue' })

  const queues = useMemo(() => queuesData?.queues ?? [], [queuesData])
  const effectiveSelectedQueueId = useMemo(() => {
    if (view !== 'queue' || queues.length === 0) {
      return ''
    }

    if (selectedQueueId && queues.some((queue) => queue.id === selectedQueueId)) {
      return selectedQueueId
    }

    return queues[0].id
  }, [view, queues, selectedQueueId])

  return (
    <>
      <div className="card bg-base-100 shadow-sm">
        <div className="card-body">
          <div className="flex flex-col gap-3 pb-4 border-b border-base-200 sm:flex-row sm:items-center sm:justify-between">
            <h3 className="text-lg font-bold text-base-content">Tasks</h3>
            <div className="inline-flex items-center bg-base-200 rounded-lg p-1 self-start">
              <button
                type="button"
                className={`btn btn-xs gap-1 ${view === 'recent' ? 'btn-primary' : 'btn-ghost'}`}
                onClick={() => setView('recent')}
              >
                <History size={14} />
                Recent
              </button>
              <button
                type="button"
                className={`btn btn-xs gap-1 ${view === 'queue' ? 'btn-primary' : 'btn-ghost'}`}
                onClick={() => setView('queue')}
              >
                <ListTree size={14} />
                Queue
              </button>
            </div>
          </div>

          {view === 'recent' ? (
            <RecentView tasks={tasks} isLoading={isHistoryLoading} />
          ) : (
            <QueueView
              effectiveSelectedQueueId={effectiveSelectedQueueId}
              onSelectedQueueIdChange={setSelectedQueueId}
              queues={queues}
              isLoading={queuesLoading}
              error={queuesError?.message}
              onEditTask={setEditingTask}
            />
          )}
        </div>
      </div>

      <EditTaskModal task={editingTask} onClose={() => setEditingTask(null)} />
    </>
  )
}

function RecentView({ tasks, isLoading }: { tasks?: TaskHistoryItem[]; isLoading?: boolean }) {
  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Loader2 className="w-6 h-6 animate-spin text-primary" />
      </div>
    )
  }

  return (
    <div className="pt-4">
      {tasks && tasks.length > 0 ? (
        <>
          <div className="divide-y divide-base-200 -mx-4">
            {tasks.slice(0, 10).map((task) => (
              <RecentTaskRow key={task.id} task={task} />
            ))}
          </div>
          <div className="pt-3 text-right">
            <Link to="/history" className="text-sm text-primary hover:underline">
              View All
            </Link>
          </div>
        </>
      ) : (
        <p className="text-base-content/60 text-center py-8">No tasks yet</p>
      )}
    </div>
  )
}

function RecentTaskRow({ task }: { task: TaskHistoryItem }) {
  const { displayState, ...config } = getStateConfigWithProgress(task.state, task.progress_phase)

  return (
    <Link
      to={`/task/${task.id}`}
      className="flex items-center justify-between px-4 py-3 hover:bg-base-200/50 transition-colors group"
    >
      <div className="flex items-center gap-3 min-w-0">
        <span className="text-lg">{config.icon}</span>
        <div className="min-w-0">
          <div className="font-medium text-base-content truncate group-hover:text-primary transition-colors">
            {task.title || task.id}
          </div>
          <div className="flex items-center gap-2 text-xs text-base-content/60 mt-0.5">
            <span>{new Date(task.created_at).toLocaleDateString()}</span>
            {task.worktree_path && (
              <span className="flex items-center gap-1">
                <FolderGit2 size={12} />
                Worktree
              </span>
            )}
          </div>
        </div>
      </div>
      <div className="flex items-center gap-2">
        <span className={`badge ${config.badge} badge-sm capitalize`}>{displayState}</span>
        <ChevronRight size={16} className="text-base-content/40 group-hover:text-primary transition-colors" />
      </div>
    </Link>
  )
}

interface QueueViewProps {
  effectiveSelectedQueueId: string
  onSelectedQueueIdChange: (queueId: string) => void
  queues: Array<{ id: string; title: string; task_count: number; status: string }>
  isLoading: boolean
  error?: string
  onEditTask: (task: PlanTask) => void
}

function QueueView({
  effectiveSelectedQueueId,
  onSelectedQueueIdChange,
  queues,
  isLoading,
  error,
  onEditTask,
}: QueueViewProps) {
  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Loader2 className="w-6 h-6 animate-spin text-primary" />
      </div>
    )
  }

  if (error) {
    return <div className="alert alert-error mt-4">Failed to load queues: {error}</div>
  }

  return (
    <div className="pt-4 space-y-4">
      {queues.length === 0 ? (
        <div className="text-center py-8 text-base-content/60">
          No queues yet. Create one in Project Planning.
        </div>
      ) : (
        <>
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
            <label className="text-sm text-base-content/70" htmlFor="dashboard-queue-select">
              Queue
            </label>
            <select
              id="dashboard-queue-select"
              className="select select-bordered select-sm w-full sm:w-80"
              value={effectiveSelectedQueueId}
              onChange={(e) => onSelectedQueueIdChange(e.target.value)}
            >
              {queues.map((queue) => (
                <option key={queue.id} value={queue.id}>
                  {queue.title || queue.id} ({queue.task_count}, {queue.status})
                </option>
              ))}
            </select>
          </div>

          <TasksPanel queueId={effectiveSelectedQueueId} onEditTask={onEditTask} />
        </>
      )}
    </div>
  )
}
