import { useProjectStore } from '../stores/projectStore'
import { EmptyState } from './EmptyState'

interface CheckpointsWidgetProps {
  embedded?: boolean
}

export function CheckpointsWidget({ embedded = false }: CheckpointsWidgetProps) {
  const { checkpoints, redoStack, goToCheckpoint, undo, redo, loading } = useProjectStore()

  const hasCheckpoints = checkpoints.length > 0 || redoStack.length > 0

  const content = (
    <div>
      {!hasCheckpoints ? (
            <EmptyState title="No checkpoints yet" description="Checkpoints are created during planning and implementation" icon="🕐" />
          ) : (
            <>
              {/* Checkpoint Timeline */}
              <div className="space-y-2 mb-4 max-h-[200px] overflow-auto">
                {checkpoints.map((cp, i) => (
                  <button
                    key={cp.sha}
                    onClick={() => goToCheckpoint(cp.sha)}
                    disabled={loading}
                    aria-label={`Go to checkpoint ${checkpoints.length - i}: ${cp.message || cp.sha.slice(0, 8)}`}
                    className="w-full text-left p-3 rounded-lg bg-base-300 hover:bg-base-100 border border-transparent hover:border-primary/30 transition-all duration-150 disabled:opacity-50 group"
                  >
                    <div className="flex items-center gap-3">
                      <div className="w-6 h-6 rounded-full bg-primary/20 flex items-center justify-center text-primary text-xs font-semibold">
                        {checkpoints.length - i}
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="font-mono text-sm text-base-content/80 group-hover:text-base-content transition-colors">
                          {cp.sha.slice(0, 8)}
                        </div>
                        {cp.message && (
                          <div className="text-xs text-base-content/60 truncate">{cp.message}</div>
                        )}
                      </div>
                      <svg aria-hidden="true" className="w-4 h-4 text-base-content/40 group-hover:text-primary transition-colors" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                      </svg>
                    </div>
                  </button>
                ))}
              </div>

              {/* Quick Actions */}
              <div className="flex gap-2 pt-4 border-t border-base-300">
                <button
                  onClick={() => undo()}
                  disabled={checkpoints.length === 0 || loading}
                  className="btn btn-ghost flex-1 btn-sm"
                >
                  <svg aria-hidden="true" className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 10h10a8 8 0 018 8v2M3 10l6 6m-6-6l6-6" />
                  </svg>
                  Undo ({checkpoints.length})
                </button>
                <button
                  onClick={() => redo()}
                  disabled={redoStack.length === 0 || loading}
                  className="btn btn-ghost flex-1 btn-sm"
                >
                  Redo ({redoStack.length})
                  <svg aria-hidden="true" className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 10h-10a8 8 0 00-8 8v2m18-10l-6 6m6-6l-6-6" />
                  </svg>
                </button>
              </div>
            </>
          )}
        </div>
      )

  if (embedded) {
    return content
  }

  return (
    <section className="card bg-base-200">
      <div className="card-body">
        <h2 className="card-title text-base-content flex items-center gap-2">
          <svg aria-hidden="true" className="w-5 h-5 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          Checkpoints
        </h2>
        <div className="mt-4">
          {content}
        </div>
      </div>
    </section>
  )
}
