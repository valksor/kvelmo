import { Link } from 'react-router-dom'
import { Clock, GitBranch, ExternalLink } from 'lucide-react'
import type { TaskResponse } from '@/types/api'
import { formatDistanceToNow } from 'date-fns'
import { getStateConfig } from '@/constants/stateConfig'
import { formatTokens, formatCostSimple } from '@/utils/format'

interface TaskCardProps {
  task: TaskResponse
}

export function TaskCard({ task }: TaskCardProps) {
  if (!task.active || !task.task) {
    return null
  }

  const { task: activeTask, work } = task
  const state = activeTask.state
  const config = getStateConfig(state)

  const startedAgo = activeTask.started
    ? formatDistanceToNow(new Date(activeTask.started), { addSuffix: true })
    : 'just now'

  return (
    <div className="card bg-base-100 shadow-sm">
      {/* Progress bar at top */}
      <div className={`h-1 ${config.color} rounded-t-2xl`} />

      <div className="card-body">
        {/* Header: Title + State badge */}
        <div className="flex items-start justify-between gap-4">
          <div className="flex-1 min-w-0">
            <h2 className="card-title text-lg">
              <Link to={`/task/${activeTask.id}`} className="hover:underline truncate">
                {work?.title || activeTask.ref || activeTask.id}
              </Link>
            </h2>
            {work?.external_key && (
              <p className="text-sm text-base-content/60 flex items-center gap-1 mt-1">
                <ExternalLink size={14} />
                {work.external_key}
              </p>
            )}
          </div>
          <span className={`badge ${config.badge} capitalize`}>{state}</span>
        </div>

        {/* Metadata row */}
        <div className="flex flex-wrap gap-4 text-sm text-base-content/60 mt-2">
          {activeTask.branch && (
            <span className="flex items-center gap-1">
              <GitBranch size={14} />
              <code className="text-xs">{activeTask.branch}</code>
            </span>
          )}
          <span className="flex items-center gap-1">
            <Clock size={14} />
            {startedAgo}
          </span>
        </div>

        {/* Costs summary if available */}
        {work?.costs?.total_cost_usd != null && (
          <div className="mt-3 pt-3 border-t border-base-200">
            <div className="flex gap-4 text-sm">
              <span>
                <span className="text-base-content/60">Cost:</span>{' '}
                <span className="font-medium">{formatCostSimple(work.costs.total_cost_usd)}</span>
              </span>
              {work.costs.total_input_tokens != null && (
                <span>
                  <span className="text-base-content/60">Tokens:</span>{' '}
                  <span className="font-medium">{formatTokens(work.costs.total_input_tokens + (work.costs.total_output_tokens || 0))}</span>
                </span>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
