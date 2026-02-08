import { useState } from 'react'
import { formatDate } from '@/utils/format'
import {
  CheckCircle,
  Circle,
  Loader2,
  ChevronDown,
  ChevronRight,
  Copy,
  Check,
  ChevronsUpDown,
  FileDiff,
} from 'lucide-react'
import { useSpecificationFileDiff } from '@/api/task'
import { AccessibleModal } from '@/components/ui/AccessibleModal'
import { VisualUnifiedDiff } from '@/components/task/VisualUnifiedDiff'
import { VisualCombinedDiff } from '@/components/task/VisualCombinedDiff'
import type { Specification } from '@/types/api'

// User-friendly status labels for non-technical users
const statusLabels: Record<string, string> = {
  completed: 'Done',
  in_progress: 'Working',
  pending: 'Not Started',
  failed: 'Problem',
}

interface SpecificationsListProps {
  specs?: Specification[]
  isLoading?: boolean
  taskId?: string
}

export function SpecificationsList({ specs, isLoading, taskId }: SpecificationsListProps) {
  const [expandedSpecs, setExpandedSpecs] = useState<Set<number>>(new Set())
  const [diffTarget, setDiffTarget] = useState<{ specNumber: number; filePath: string } | null>(null)
  const [diffContent, setDiffContent] = useState('')
  const [diffError, setDiffError] = useState('')
  const [diffMode, setDiffMode] = useState<'split' | 'combined' | 'raw'>('split')
  const { mutateAsync: fetchDiff, isPending: isLoadingDiff } = useSpecificationFileDiff(taskId)

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

  if (!specs || specs.length === 0) {
    return (
      <div className="card bg-base-100 shadow-sm">
        <div className="card-body">
          <h3 className="text-lg font-bold text-base-content">Specifications</h3>
          <p className="text-base-content/60 text-center py-8">
            No specifications yet. Start planning to generate them.
            <span className="block mt-1 text-xs">
              Technical: <code className="px-2 py-1 bg-base-200 rounded">plan</code>
            </span>
          </p>
        </div>
      </div>
    )
  }

  const completed = specs.filter((s) => s.status === 'completed').length
  const total = specs.length
  const progress = total > 0 ? (completed / total) * 100 : 0

  const toggleSpec = (specNumber: number) => {
    setExpandedSpecs((prev) => {
      const next = new Set(prev)
      if (next.has(specNumber)) {
        next.delete(specNumber)
      } else {
        next.add(specNumber)
      }
      return next
    })
  }

  const expandAll = () => {
    setExpandedSpecs(new Set(specs.map((s) => s.number)))
  }

  const collapseAll = () => {
    setExpandedSpecs(new Set())
  }

  const allExpanded = expandedSpecs.size === specs.length

  const closeDiff = () => {
    setDiffTarget(null)
    setDiffContent('')
    setDiffError('')
    setDiffMode('split')
  }

  const handleOpenDiff = async (specNumber: number, filePath: string) => {
    if (!taskId) {
      setDiffTarget({ specNumber, filePath })
      setDiffContent('')
      setDiffError('Task ID is missing, cannot load diff.')
      return
    }

    setDiffTarget({ specNumber, filePath })
    setDiffContent('')
    setDiffError('')
    setDiffMode('split')

    try {
      const response = await fetchDiff({ specNumber, filePath })
      setDiffContent(response.diff)
    } catch (error) {
      setDiffError(error instanceof Error ? error.message : 'Failed to load diff')
    }
  }

  return (
    <>
      <div className="card bg-base-100 shadow-sm">
        <div className="card-body">
          {/* Header with count and expand/collapse all */}
          <div className="flex items-center justify-between pb-4 border-b border-base-200">
            <h3 className="text-lg font-bold text-base-content">Specifications</h3>
            <div className="flex items-center gap-2">
              <span className="text-sm text-base-content/60">
                {completed}/{total} complete
              </span>
              <button
                onClick={allExpanded ? collapseAll : expandAll}
                className="btn btn-ghost btn-xs"
                title={allExpanded ? 'Collapse all' : 'Expand all'}
                aria-label={allExpanded ? 'Collapse all specifications' : 'Expand all specifications'}
              >
                <ChevronsUpDown size={14} />
              </button>
            </div>
          </div>

          {/* Progress bar */}
          <div className="mt-4 mb-6">
            <div className="h-2 bg-base-200 rounded-full overflow-hidden">
              <div
                className="h-full bg-gradient-to-r from-success to-success/80 transition-all duration-500"
                style={{ width: `${progress}%` }}
              />
            </div>
            <p className="text-xs text-base-content/60 mt-1">{Math.round(progress)}% complete</p>
          </div>

          {/* Specification list */}
          <div className="space-y-4">
            {specs.map((spec) => (
              <SpecificationItem
                key={spec.number}
                spec={spec}
                expanded={expandedSpecs.has(spec.number)}
                onToggle={() => toggleSpec(spec.number)}
                onOpenDiff={handleOpenDiff}
              />
            ))}
          </div>
        </div>
      </div>
      <AccessibleModal
        isOpen={!!diffTarget}
        onClose={closeDiff}
        title="File Diff"
        size="5xl"
        actions={(
          <button onClick={closeDiff} className="btn">
            Close
          </button>
        )}
      >
        {diffTarget && (
          <>
            <p className="text-xs text-base-content/60 mb-3">
              specification-{diffTarget.specNumber} ·{' '}
              <span className="font-mono">{diffTarget.filePath}</span>
            </p>

            {!isLoadingDiff && !diffError && diffContent && (
              <div className="flex justify-end mb-3">
                <div className="join" role="group" aria-label="Diff view mode">
                  <button
                    className={`btn btn-sm join-item ${diffMode === 'split' ? 'btn-primary' : 'btn-ghost'}`}
                    onClick={() => setDiffMode('split')}
                  >
                    Split
                  </button>
                  <button
                    className={`btn btn-sm join-item ${diffMode === 'combined' ? 'btn-primary' : 'btn-ghost'}`}
                    onClick={() => setDiffMode('combined')}
                  >
                    Combined
                  </button>
                  <button
                    className={`btn btn-sm join-item ${diffMode === 'raw' ? 'btn-primary' : 'btn-ghost'}`}
                    onClick={() => setDiffMode('raw')}
                  >
                    Raw
                  </button>
                </div>
              </div>
            )}

            <div className="bg-base-200/60 rounded-lg p-3 max-h-[70vh] overflow-auto">
              {isLoadingDiff ? (
                <div className="flex items-center justify-center py-10">
                  <Loader2 className="w-5 h-5 animate-spin text-primary" />
                </div>
              ) : diffError ? (
                <p className="text-sm text-error whitespace-pre-wrap">{diffError}</p>
              ) : diffContent ? (
                diffMode === 'split' ? (
                  <VisualUnifiedDiff diff={diffContent} />
                ) : diffMode === 'combined' ? (
                  <VisualCombinedDiff diff={diffContent} />
                ) : (
                  <pre className="text-xs font-mono whitespace-pre-wrap">{diffContent}</pre>
                )
              ) : (
                <p className="text-sm text-base-content/60">No diff found for this file.</p>
              )}
            </div>
          </>
        )}
      </AccessibleModal>
    </>
  )
}

interface SpecificationItemProps {
  spec: Specification
  expanded: boolean
  onToggle: () => void
  onOpenDiff: (specNumber: number, filePath: string) => void
}

function SpecificationItem({
  spec,
  expanded,
  onToggle,
  onOpenDiff,
}: SpecificationItemProps) {
  const [copied, setCopied] = useState(false)

  const statusIcon =
    spec.status === 'completed' ? (
      <CheckCircle className="w-5 h-5 text-success" />
    ) : spec.status === 'in_progress' ? (
      <Loader2 className="w-5 h-5 text-primary animate-spin" />
    ) : (
      <Circle className="w-5 h-5 text-base-content/40" />
    )

  const isActive = spec.status === 'in_progress'

  const handleCopy = async (e: React.MouseEvent) => {
    e.stopPropagation()
    if (spec.description) {
      await navigator.clipboard.writeText(spec.description)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    }
  }

  return (
    <div
      className={`rounded-xl bg-base-100 border border-base-200 overflow-hidden ${
        isActive ? 'ring-2 ring-primary ring-offset-2' : ''
      }`}
    >
      {/* Clickable header row */}
      <button
        onClick={onToggle}
        className="w-full p-4 flex items-center justify-between hover:bg-base-200/50 transition-colors text-left"
      >
        <div className="flex items-center gap-2">
          {expanded ? (
            <ChevronDown size={16} className="text-base-content/50" />
          ) : (
            <ChevronRight size={16} className="text-base-content/50" />
          )}
          {statusIcon}
          <h4 className="font-semibold text-base-content">
            {spec.title || `Spec #${spec.number}`}
          </h4>
        </div>
        <div className="flex items-center gap-2">
          {spec.component && (
            <span className="px-2 py-1 text-xs font-medium rounded-full bg-base-200 text-base-content/60">
              {spec.component}
            </span>
          )}
          <span
            className={`px-2 py-1 text-xs font-medium rounded-full ${
              spec.status === 'completed'
                ? 'bg-success/20 text-success'
                : spec.status === 'in_progress'
                  ? 'bg-primary/20 text-primary'
                  : 'bg-base-200 text-base-content/60'
            }`}
          >
            {statusLabels[spec.status] || spec.status}
          </span>
        </div>
      </button>

      {/* Collapsed preview */}
      {!expanded && spec.description && (
        <div className="px-4 pb-4 -mt-2">
          <p className="text-sm text-base-content/60 line-clamp-2 pl-9">{spec.description}</p>
          {spec.implemented_files && spec.implemented_files.length > 0 && (
            <p className="text-xs text-success mt-2 pl-9">
              {spec.implemented_files.length} implemented file
              {spec.implemented_files.length === 1 ? '' : 's'}
            </p>
          )}
        </div>
      )}

      {/* Expanded content */}
      {expanded && (
        <div className="px-4 pb-4 border-t border-base-200">
          {/* Full description */}
          {spec.description && (
            <div className="mt-3">
              <div className="flex items-center justify-between mb-2">
                <span className="text-xs font-medium text-base-content/60 uppercase">Description</span>
                <button
                  onClick={handleCopy}
                  className="btn btn-ghost btn-xs gap-1"
                  title="Copy description"
                >
                  {copied ? (
                    <>
                      <Check size={12} className="text-success" />
                      Copied
                    </>
                  ) : (
                    <>
                      <Copy size={12} />
                      Copy
                    </>
                  )}
                </button>
              </div>
              <div className="text-sm text-base-content/80 whitespace-pre-wrap bg-base-200/50 p-3 rounded-lg">
                {spec.description}
              </div>
            </div>
          )}

          {/* Timestamps */}
          {(spec.created_at || spec.completed_at) && (
            <div className="mt-3 pt-3 border-t border-base-200 flex items-center gap-4 text-xs text-base-content/60">
              {spec.created_at && <span>Created: {formatDate(spec.created_at)}</span>}
              {spec.completed_at && (
                <span className="text-success">
                  Completed: {formatDate(spec.completed_at)}
                </span>
              )}
            </div>
          )}

          {spec.implemented_files && spec.implemented_files.length > 0 && (
            <div className="mt-3 pt-3 border-t border-base-200">
              <span className="text-xs font-medium text-base-content/60 uppercase">
                Implemented Files ({spec.implemented_files.length})
              </span>
              <div className="mt-2 flex flex-wrap gap-2">
                {spec.implemented_files.map((filePath, index) => (
                  <button
                    key={`${filePath}-${index}`}
                    className="px-2 py-1 text-xs rounded bg-success/10 text-success font-mono hover:bg-success/20 transition-colors inline-flex items-center gap-1"
                    title={filePath}
                    onClick={() => onOpenDiff(spec.number, filePath)}
                  >
                    <FileDiff size={12} />
                    {filePath}
                  </button>
                ))}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
