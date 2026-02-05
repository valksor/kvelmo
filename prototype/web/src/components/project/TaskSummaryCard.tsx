import { Link } from 'react-router-dom'
import { ArrowRight, GitBranch, Clock } from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'
import type { WorkflowState } from '@/types/api'
import { getStateConfig } from '@/constants/stateConfig'

interface TaskSummaryCardProps {
  task?: {
    id: string
    state: WorkflowState
    ref?: string
    branch?: string
    started?: string
  }
  work?: {
    title?: string
    external_key?: string
  }
}

export function TaskSummaryCard({ task, work }: TaskSummaryCardProps) {
  if (!task) {
    return null
  }

  const state = task.state
  const config = getStateConfig(state)
  const title = work?.title || task.ref || task.id

  const startedAgo = task.started
    ? formatDistanceToNow(new Date(task.started), { addSuffix: true })
    : null

  return (
    <div className="card bg-base-100 shadow-sm">
      {/* State indicator bar */}
      <div className={`h-1 ${config.bgClass.replace('/10', '')} rounded-t-2xl`} />

      <div className="card-body">
        <div className="flex items-start justify-between gap-4">
          <div className="flex items-center gap-3">
            <span className="text-2xl">{config.icon}</span>
            <div className="flex-1 min-w-0">
              <h3 className="font-bold text-base-content truncate">{title}</h3>
              <div className="flex items-center gap-3 text-sm text-base-content/60 mt-1">
                {task.branch && (
                  <span className="flex items-center gap-1">
                    <GitBranch size={14} />
                    <code className="text-xs">{task.branch}</code>
                  </span>
                )}
                {startedAgo && (
                  <span className="flex items-center gap-1">
                    <Clock size={14} />
                    {startedAgo}
                  </span>
                )}
              </div>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <span className={`badge ${config.badge} capitalize`}>{state}</span>
            <Link
              to={`/task/${task.id}`}
              className="btn btn-sm btn-primary gap-1"
              title="View task details"
            >
              View
              <ArrowRight size={14} />
            </Link>
          </div>
        </div>
      </div>
    </div>
  )
}
