import { useState } from 'react'
import {
  GitBranch,
  Clock,
  ExternalLink,
  Eye,
  ChevronDown,
  ChevronRight,
  GitPullRequest,
  CheckCircle2,
} from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'
import type { WorkResponse } from '@/types/api'
import { TaskContentModal } from './TaskContentModal'

interface CompletedWorkCardProps {
  work: WorkResponse['work']
}

export function CompletedWorkCard({ work }: CompletedWorkCardProps) {
  const [showModal, setShowModal] = useState(false)
  const [showTechnicalDetails, setShowTechnicalDetails] = useState(false)

  const title = work.metadata.title || work.metadata.id
  const createdAgo = formatDistanceToNow(new Date(work.metadata.created_at), { addSuffix: true })
  const updatedAgo = formatDistanceToNow(new Date(work.metadata.updated_at), { addSuffix: true })

  const hasPR = work.metadata.pull_request != null

  return (
    <>
      <div className="card bg-base-100 shadow-sm overflow-hidden">
        {/* State banner - completed */}
        <div className="px-6 py-3 bg-success/10">
          <div className="flex items-center gap-3">
            <span className="text-2xl">
              {hasPR ? <GitPullRequest className="text-success" /> : <CheckCircle2 className="text-success" />}
            </span>
            <div>
              <span className="text-sm font-semibold uppercase tracking-wide text-success">
                {hasPR ? 'Completed with PR' : 'Completed'}
              </span>
            </div>
          </div>
        </div>

        <div className="card-body">
          {/* Title and external key */}
          <div className="flex items-start justify-between gap-4">
            <div className="flex-1 min-w-0">
              <h2 className="text-2xl font-bold text-base-content truncate">{title}</h2>
              {work.metadata.external_key && (
                <p className="text-sm text-base-content/60 flex items-center gap-1 mt-1">
                  <ExternalLink size={14} />
                  {work.metadata.external_key}
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
              <span className="badge badge-success capitalize">done</span>
            </div>
          </div>

          {/* PR Badge - prominent display */}
          {work.metadata.pull_request && (
            <div className="mt-3">
              <a
                href={work.metadata.pull_request.url}
                target="_blank"
                rel="noopener noreferrer"
                className="btn btn-primary btn-sm gap-2"
              >
                <GitPullRequest size={16} />
                View PR #{work.metadata.pull_request.number}
                <ExternalLink size={14} />
              </a>
            </div>
          )}

          {/* Description preview (if available) */}
          {work.description && (
            <div className="mt-3 p-3 bg-base-200/50 rounded-lg">
              <p className="text-sm text-base-content/80 line-clamp-3">{work.description}</p>
            </div>
          )}

          <dl className="grid grid-cols-2 gap-4 mt-4 text-sm">
            <div>
              <dt className="text-base-content/60 text-xs font-medium uppercase tracking-wide mb-1">
                Created
              </dt>
              <dd className="text-base-content flex items-center gap-1">
                <Clock size={14} className="text-base-content/40" />
                {createdAgo}
              </dd>
            </div>
            <div>
              <dt className="text-base-content/60 text-xs font-medium uppercase tracking-wide mb-1">
                Completed
              </dt>
              <dd className="text-base-content flex items-center gap-1">
                <Clock size={14} className="text-base-content/40" />
                {updatedAgo}
              </dd>
            </div>
          </dl>

          {/* Labels */}
          {work.metadata.labels && work.metadata.labels.length > 0 && (
            <div className="flex flex-wrap gap-1 mt-3">
              {work.metadata.labels.map((label) => (
                <span key={label} className="badge badge-outline badge-sm">
                  {label}
                </span>
              ))}
            </div>
          )}

          {(work.git.branch || work.source.ref) && (
            <div className="mt-3">
              <button
                type="button"
                className="btn btn-ghost btn-sm px-0"
                onClick={() => setShowTechnicalDetails((prev) => !prev)}
              >
                {showTechnicalDetails ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
                Technical details
              </button>

              {showTechnicalDetails && (
                <dl className="grid grid-cols-2 gap-x-6 gap-y-3 mt-2 text-sm p-3 rounded-lg bg-base-200/50">
                  {work.git.branch && (
                    <div>
                      <dt className="text-base-content/60 text-xs font-medium uppercase tracking-wide mb-1">
                        Branch
                      </dt>
                      <dd className="font-mono text-base-content flex items-center gap-1">
                        <GitBranch size={14} className="text-base-content/40" />
                        {work.git.branch}
                      </dd>
                    </div>
                  )}
                  {work.git.base_branch && (
                    <div>
                      <dt className="text-base-content/60 text-xs font-medium uppercase tracking-wide mb-1">
                        Base Branch
                      </dt>
                      <dd className="font-mono text-base-content flex items-center gap-1">
                        <GitBranch size={14} className="text-base-content/40" />
                        {work.git.base_branch}
                      </dd>
                    </div>
                  )}
                  {work.source.ref && (
                    <div className="col-span-2">
                      <dt className="text-base-content/60 text-xs font-medium uppercase tracking-wide mb-1">
                        Source
                      </dt>
                      <dd className="text-base-content truncate">{work.source.ref}</dd>
                    </div>
                  )}
                </dl>
              )}
            </div>
          )}
        </div>
      </div>

      {/* Task content modal */}
      <TaskContentModal
        isOpen={showModal}
        onClose={() => setShowModal(false)}
        title={title}
        content={work.description}
        externalKey={work.metadata.external_key}
        sourceRef={work.source.ref}
      />
    </>
  )
}
