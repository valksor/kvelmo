import { useState } from 'react'
import { CheckCircle, XCircle, AlertTriangle, Eye, Wrench, ChevronDown, ChevronRight, ChevronsUpDown } from 'lucide-react'
import type { Review } from '@/types/api'
import { useWorkflowAction } from '@/api/workflow'
import { useImplementReview } from '@/api/task'

interface ReviewsListProps {
  reviews?: Review[]
}

export function ReviewsList({ reviews }: ReviewsListProps) {
  const [expandedReviews, setExpandedReviews] = useState<Set<number>>(new Set())

  if (!reviews || reviews.length === 0) {
    return (
      <div className="card bg-base-100 shadow-sm">
        <div className="card-body">
          <h3 className="text-lg font-bold text-base-content">Reviews</h3>
          <p className="text-base-content/60 text-center py-8">
            No reviews yet. Run <code className="px-2 py-1 bg-base-200 rounded">review</code> after
            implementation.
          </p>
        </div>
      </div>
    )
  }

  const toggleReview = (reviewNumber: number) => {
    setExpandedReviews((prev) => {
      const next = new Set(prev)
      if (next.has(reviewNumber)) {
        next.delete(reviewNumber)
      } else {
        next.add(reviewNumber)
      }
      return next
    })
  }

  const expandAll = () => {
    setExpandedReviews(new Set(reviews.map((r) => r.number)))
  }

  const collapseAll = () => {
    setExpandedReviews(new Set())
  }

  const allExpanded = expandedReviews.size === reviews.length

  return (
    <div className="card bg-base-100 shadow-sm">
      <div className="card-body">
        {/* Header with count and expand/collapse all */}
        <div className="flex items-center justify-between pb-4 border-b border-base-200">
          <h3 className="text-lg font-bold text-base-content">Reviews</h3>
          <div className="flex items-center gap-2">
            <span className="text-sm text-base-content/60">{reviews.length} review(s)</span>
            <button
              onClick={allExpanded ? collapseAll : expandAll}
              className="btn btn-ghost btn-xs"
              title={allExpanded ? 'Collapse all' : 'Expand all'}
            >
              <ChevronsUpDown size={14} />
            </button>
          </div>
        </div>

        {/* Review list */}
        <div className="space-y-4 mt-4">
          {reviews.map((review) => (
            <ReviewItem
              key={review.number}
              review={review}
              expanded={expandedReviews.has(review.number)}
              onToggle={() => toggleReview(review.number)}
            />
          ))}
        </div>
      </div>
    </div>
  )
}

interface ReviewItemProps {
  review: Review
  expanded: boolean
  onToggle: () => void
}

function ReviewItem({ review, expanded, onToggle }: ReviewItemProps) {
  const { mutate: executeAction, isPending: isActionPending } = useWorkflowAction()
  const { mutate: implementReview, isPending: isImplementPending } = useImplementReview()
  const isPending = isActionPending || isImplementPending

  const statusIcon =
    review.status === 'passed' ? (
      <CheckCircle className="w-5 h-5 text-success" />
    ) : review.status === 'failed' ? (
      <XCircle className="w-5 h-5 text-error" />
    ) : (
      <AlertTriangle className="w-5 h-5 text-warning" />
    )

  const statusClass =
    review.status === 'passed'
      ? 'bg-success/20 text-success'
      : review.status === 'failed'
        ? 'bg-error/20 text-error'
        : 'bg-warning/20 text-warning'

  const hasIssues = review.issue_count > 0

  return (
    <div className="rounded-xl bg-base-100 border border-base-200 overflow-hidden">
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
          <h4 className="font-semibold text-base-content">Review #{review.number}</h4>
          {hasIssues && (
            <span className="px-2 py-0.5 text-xs font-medium rounded-full bg-warning/20 text-warning">
              {review.issue_count} issue(s)
            </span>
          )}
        </div>
        <span className={`px-2 py-1 text-xs font-medium rounded-full capitalize ${statusClass}`}>
          {review.status}
        </span>
      </button>

      {/* Collapsed preview - show truncated summary */}
      {!expanded && review.summary && (
        <div className="px-4 pb-4 -mt-2">
          <p className="text-sm text-base-content/60 line-clamp-2 pl-9">{review.summary}</p>
        </div>
      )}

      {/* Expanded content */}
      {expanded && (
        <div className="px-4 pb-4 border-t border-base-200">
          {/* Full summary */}
          {review.summary && (
            <div className="mt-3">
              <span className="text-xs font-medium text-base-content/60 uppercase">Summary</span>
              <div className="text-sm text-base-content/80 whitespace-pre-wrap bg-base-200/50 p-3 rounded-lg mt-1">
                {review.summary}
              </div>
            </div>
          )}

          {/* Actions */}
          <div className="flex gap-2 mt-4">
            <button
              className="btn btn-sm btn-secondary gap-1"
              onClick={(e) => {
                e.stopPropagation()
                executeAction({ action: 'review', options: { view: review.number } })
              }}
              disabled={isPending}
            >
              <Eye size={14} />
              View Full Review
            </button>
            {hasIssues && (
              <button
                className="btn btn-sm btn-primary gap-1"
                onClick={(e) => {
                  e.stopPropagation()
                  implementReview(review.number)
                }}
                disabled={isPending}
              >
                <Wrench size={14} />
                Implement Fixes
              </button>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
