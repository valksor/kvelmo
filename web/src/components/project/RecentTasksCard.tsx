import { Link } from 'react-router-dom'
import { ChevronRight, Loader2, FolderGit2 } from 'lucide-react'
import type { TaskHistoryItem } from '@/types/api'
import { getStateConfig } from '@/constants/stateConfig'

interface RecentTasksCardProps {
  tasks?: TaskHistoryItem[]
  isLoading?: boolean
}

export function RecentTasksCard({ tasks, isLoading }: RecentTasksCardProps) {
  if (isLoading) {
    return (
      <div className="card bg-base-100 shadow-sm">
        <div className="card-body">
          <div className="flex items-center justify-center py-8">
            <Loader2 className="w-6 h-6 animate-spin text-primary" />
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="card bg-base-100 shadow-sm">
      <div className="card-body">
        {/* Header */}
        <div className="flex items-center justify-between pb-4 border-b border-base-200">
          <h3 className="text-lg font-bold text-base-content">Recent Tasks</h3>
          <Link to="/history" className="text-sm text-primary hover:underline">
            View All
          </Link>
        </div>

        {/* Task list */}
        {tasks && tasks.length > 0 ? (
          <div className="divide-y divide-base-200 -mx-4">
            {tasks.slice(0, 10).map((task) => (
              <TaskRow key={task.id} task={task} />
            ))}
          </div>
        ) : (
          <p className="text-base-content/60 text-center py-8">No tasks yet</p>
        )}
      </div>
    </div>
  )
}

interface TaskRowProps {
  task: TaskHistoryItem
}

function TaskRow({ task }: TaskRowProps) {
  const config = getStateConfig(task.state)

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
        <span className={`badge ${config.badge} badge-sm capitalize`}>{task.state}</span>
        <ChevronRight size={16} className="text-base-content/40 group-hover:text-primary transition-colors" />
      </div>
    </Link>
  )
}
