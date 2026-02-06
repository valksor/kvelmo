import { useState } from 'react'
import { GitBranch, Clock, FolderGit2, ExternalLink, HelpCircle, Eye } from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'
import type { WorkflowState, ProgressPhase } from '@/types/api'
import { TaskContentModal } from './TaskContentModal'
import { getStateConfigWithProgress } from '@/constants/stateConfig'

interface ActiveWorkCardProps {
  task?: {
    id: string
    state: WorkflowState
    ref?: string
    branch?: string
    worktree_path?: string
    started?: string
  }
  work?: {
    title?: string
    external_key?: string
    description?: string
  }
  progressPhase?: ProgressPhase
}

export function ActiveWorkCard({ task, work, progressPhase }: ActiveWorkCardProps) {
  const [showModal, setShowModal] = useState(false)

  if (!task) {
    return null
  }

  const state = task.state
  // Use progress-aware state config (matches CLI behavior)
  const { displayState, ...config } = getStateConfigWithProgress(state, progressPhase)

  const startedAgo = task.started
    ? formatDistanceToNow(new Date(task.started), { addSuffix: true })
    : 'just now'

  const title = work?.title || task.ref || task.id

  return (
    <>
      <div className="card bg-base-100 shadow-sm overflow-hidden">
        {/* State banner */}
        <div className={`px-6 py-3 ${config.bgClass}`}>
          <div className="flex items-center gap-3">
            <span className="text-2xl">{config.icon}</span>
            <div>
              <span className="text-sm font-semibold uppercase tracking-wide">
                {displayState.replace('_', ' ')}
              </span>
            </div>
          </div>
        </div>

        <div className="card-body">
          {/* Title and external key */}
          <div className="flex items-start justify-between gap-4">
            <div className="flex-1 min-w-0">
              <h2 className="text-2xl font-bold text-base-content truncate">{title}</h2>
              {work?.external_key && (
                <p className="text-sm text-base-content/60 flex items-center gap-1 mt-1">
                  <ExternalLink size={14} />
                  {work.external_key}
                </p>
              )}
            </div>
            <div className="flex items-center gap-2">
              <button
                onClick={() => setShowModal(true)}
                className="btn btn-sm btn-ghost gap-1"
                title="View task details"
              >
                <Eye size={16} />
                View
              </button>
              <span className={`badge ${config.badge} capitalize`}>{displayState}</span>
            </div>
          </div>

          {/* Description preview (if available) */}
          {work?.description && (
            <div className="mt-3 p-3 bg-base-200/50 rounded-lg">
              <p className="text-sm text-base-content/80 line-clamp-3">{work.description}</p>
            </div>
          )}

          {/* Metadata grid */}
          <dl className="grid grid-cols-2 gap-x-6 gap-y-3 mt-4 text-sm">
            {task.branch && (
              <div>
                <dt className="text-base-content/60 text-xs font-medium uppercase tracking-wide mb-1">
                  Branch
                </dt>
                <dd className="font-mono text-base-content flex items-center gap-1">
                  <GitBranch size={14} className="text-base-content/40" />
                  {task.branch}
                </dd>
              </div>
            )}
            {task.ref && (
              <div>
                <dt className="text-base-content/60 text-xs font-medium uppercase tracking-wide mb-1">
                  Source
                </dt>
                <dd className="text-base-content truncate">{task.ref}</dd>
              </div>
            )}
            <div>
              <dt className="text-base-content/60 text-xs font-medium uppercase tracking-wide mb-1">
                Started
              </dt>
              <dd className="text-base-content flex items-center gap-1">
                <Clock size={14} className="text-base-content/40" />
                {startedAgo}
              </dd>
            </div>
            {task.worktree_path && (
              <div>
                <dt className="text-base-content/60 text-xs font-medium uppercase tracking-wide mb-1">
                  Worktree
                </dt>
                <dd
                  className="font-mono text-base-content truncate flex items-center gap-1"
                  title={task.worktree_path}
                >
                  <FolderGit2 size={14} className="text-base-content/40" />
                  {task.worktree_path}
                </dd>
              </div>
            )}
          </dl>

          {/* Indicators */}
          {state === 'waiting' && (
            <div className="flex items-center gap-4 mt-4 pt-4 border-t border-base-200 text-sm">
              <span className="flex items-center gap-1 text-warning">
                <HelpCircle size={16} />
                Waiting for Answer
              </span>
            </div>
          )}
        </div>
      </div>

      {/* Task content modal */}
      <TaskContentModal
        isOpen={showModal}
        onClose={() => setShowModal(false)}
        title={title}
        content={work?.description}
        externalKey={work?.external_key}
        sourceRef={task.ref}
      />
    </>
  )
}
