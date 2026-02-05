import { useState } from 'react'
import { CheckCircle, Circle, Loader2, ChevronDown, ChevronRight, Copy, Check, ChevronsUpDown } from 'lucide-react'
import type { Specification } from '@/types/api'

interface SpecificationsListProps {
  specs?: Specification[]
  isLoading?: boolean
}

export function SpecificationsList({ specs, isLoading }: SpecificationsListProps) {
  const [expandedSpecs, setExpandedSpecs] = useState<Set<number>>(new Set())

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
            No specifications yet. Run <code className="px-2 py-1 bg-base-200 rounded">plan</code> to
            generate them.
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

  return (
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
            />
          ))}
        </div>
      </div>
    </div>
  )
}

interface SpecificationItemProps {
  spec: Specification
  expanded: boolean
  onToggle: () => void
}

function SpecificationItem({ spec, expanded, onToggle }: SpecificationItemProps) {
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
            {spec.status}
          </span>
        </div>
      </button>

      {/* Collapsed preview */}
      {!expanded && spec.description && (
        <div className="px-4 pb-4 -mt-2">
          <p className="text-sm text-base-content/60 line-clamp-2 pl-9">{spec.description}</p>
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
              {spec.created_at && <span>Created: {new Date(spec.created_at).toLocaleDateString()}</span>}
              {spec.completed_at && (
                <span className="text-success">
                  Completed: {new Date(spec.completed_at).toLocaleDateString()}
                </span>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
