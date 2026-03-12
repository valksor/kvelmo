import { useState, useCallback } from 'react'
import { useProjectStore } from '../stores/projectStore'

interface ActionsWidgetProps {
  embedded?: boolean
}

type RetryableAction = {
  label: string
  fn: () => Promise<void>
}

export function ActionsWidget({ embedded = false }: ActionsWidgetProps) {
  const {
    state, plan, implement, simplify, optimize, review, submit, undo, redo, abort, abandon, update,
    finish, refresh, approveRemote, mergeRemote, deleteTask,
    loading, checkpoints, redoStack, error
  } = useProjectStore()

  const [showAbandonModal, setShowAbandonModal] = useState(false)
  const [abandonKeepBranch, setAbandonKeepBranch] = useState(false)
  const [showSubmitModal, setShowSubmitModal] = useState(false)
  const [submitDeleteBranch, setSubmitDeleteBranch] = useState(false)
  const [showFinishModal, setShowFinishModal] = useState(false)
  const [finishDeleteRemote, setFinishDeleteRemote] = useState(false)
  const [showDeleteModal, setShowDeleteModal] = useState(false)
  const [updateNotice, setUpdateNotice] = useState<string | null>(null)
  const [refreshResult, setRefreshResult] = useState<{ action: string; message: string } | null>(null)
  const [lastAction, setLastAction] = useState<RetryableAction | null>(null)

  const tracked = useCallback((label: string, fn: () => Promise<void>) => {
    return async () => {
      setLastAction({ label, fn })
      useProjectStore.setState({ error: null })
      await fn()
    }
  }, [])

  const canSubmit = state === 'reviewing'
  const canUndo = checkpoints.length > 0
  const canRedo = redoStack.length > 0
  const canAbort = state !== 'none' && state !== 'submitted'
  const isActive = state !== 'none'
  const canUpdate = state === 'loaded' || state === 'planned' || state === 'implemented'
  const canForcePlan = state === 'planned'
  const canForceImplement = state === 'implemented'
  const canSimplify = state === 'implemented'
  const canOptimize = state === 'implemented'
  const canReview = state === 'implemented'
  const canFinish = state === 'submitted'

  const handleAbandon = async () => {
    setShowAbandonModal(false)
    await abandon(abandonKeepBranch)
    setAbandonKeepBranch(false)
  }

  const handleSubmit = async () => {
    setShowSubmitModal(false)
    await submit({ delete_branch: submitDeleteBranch })
    setSubmitDeleteBranch(false)
  }

  const handleUpdate = async () => {
    const result = await update()
    if (result.changed) {
      setUpdateNotice(
        result.specification_generated
          ? 'Task updated from source — new specification generated.'
          : 'Task content updated from source.'
      )
    } else {
      setUpdateNotice('Task is already up to date.')
    }
    setTimeout(() => setUpdateNotice(null), 4000)
  }

  const handleFinish = async () => {
    setShowFinishModal(false)
    await finish({ delete_remote: finishDeleteRemote })
    setFinishDeleteRemote(false)
  }

  const handleRefresh = async () => {
    const result = await refresh()
    if (result) {
      setRefreshResult({ action: result.action, message: result.message })
      setTimeout(() => setRefreshResult(null), 6000)
    }
  }

  const handleDelete = async () => {
    setShowDeleteModal(false)
    await deleteTask()
  }

  const content = (
    <>
      <div className="space-y-2">
        {/* Update from Source */}
        {canUpdate && (
          <button
            onClick={tracked('Update', handleUpdate)}
            disabled={loading}
            className="btn btn-outline btn-info w-full btn-sm"
          >
            <svg aria-hidden="true" className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            Update from Source
          </button>
        )}

        {/* Update notice toast */}
        {updateNotice && (
          <div className="alert alert-info py-2 text-sm">
            <svg className="w-4 h-4 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            {updateNotice}
          </div>
        )}

        {/* Plan button with force re-run option */}
        <div className="flex gap-1">
          <button
            onClick={tracked('Plan', () => plan(false))}
            disabled={state !== 'loaded' || loading}
            className="btn btn-primary flex-1"
          >
            <svg aria-hidden="true" className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01M9 16h.01" />
            </svg>
            Plan
          </button>
          {canForcePlan && (
            <button
              onClick={tracked('Plan', () => plan(true))}
              disabled={loading}
              className="btn btn-primary btn-outline btn-square"
              aria-label="Force re-run planning"
            >
              <svg aria-hidden="true" className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
            </button>
          )}
        </div>

        {/* Implement button with force re-run option */}
        <div className="flex gap-1">
          <button
            onClick={tracked('Implement', () => implement(false))}
            disabled={state !== 'planned' || loading}
            className="btn btn-success flex-1"
          >
            <svg aria-hidden="true" className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4" />
            </svg>
            Implement
          </button>
          {canForceImplement && (
            <button
              onClick={tracked('Implement', () => implement(true))}
              disabled={loading}
              className="btn btn-success btn-outline btn-square"
              aria-label="Force re-run implementation"
            >
              <svg aria-hidden="true" className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
            </button>
          )}
        </div>

        {/* Simplify — optional code clarity pass */}
        <button
          onClick={tracked('Simplify', () => simplify())}
          disabled={!canSimplify || loading}
          className="btn btn-info w-full"
        >
          <svg aria-hidden="true" className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h8m-8 6h16" />
          </svg>
          Simplify
        </button>

        {/* Optimize — optional performance/quality pass */}
        <button
          onClick={tracked('Optimize', () => optimize())}
          disabled={!canOptimize || loading}
          className="btn btn-secondary w-full"
        >
          <svg aria-hidden="true" className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
          </svg>
          Optimize
        </button>

        {/* Review controls: Review + Fix & Continue */}
        {canReview ? (
          <>
            <div className="flex gap-1">
              <button
                onClick={tracked('Review', () => review({ approve: true }))}
                disabled={loading}
                className="btn btn-warning flex-1"
              >
                <svg aria-hidden="true" className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                </svg>
                Review
              </button>
              <button
                onClick={tracked('Review', () => review({ fix: true }))}
                disabled={loading}
                className="btn btn-warning btn-outline btn-square"
                aria-label="Fix and Continue — review and automatically apply fixes"
              >
                <svg aria-hidden="true" className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                </svg>
              </button>
            </div>
            <p className="text-xs text-base-content/50 text-right -mt-1">
              Pencil icon: Fix &amp; Continue
            </p>
          </>
        ) : (
          <button
            onClick={tracked('Review', () => review({ approve: true }))}
            disabled={!canReview || loading}
            className="btn btn-warning w-full"
          >
            <svg aria-hidden="true" className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
            </svg>
            Review
          </button>
        )}

        {/* Submit — opens modal for delete-branch option */}
        <button
          onClick={() => canSubmit ? setShowSubmitModal(true) : undefined}
          disabled={!canSubmit || loading}
          className="btn btn-ghost w-full"
        >
          <svg aria-hidden="true" className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
          </svg>
          Submit PR
        </button>

        {/* Post-submit actions: Refresh & Finish */}
        {canFinish && (
          <>
            <div className="divider text-xs text-base-content/50 my-2">PR Submitted</div>

            {/* Refresh PR Status */}
            <button
              onClick={tracked('Check PR Status', handleRefresh)}
              disabled={loading}
              className="btn btn-info btn-outline w-full"
            >
              <svg aria-hidden="true" className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
              Check PR Status
            </button>

            {/* Refresh result notice */}
            {refreshResult && (
              <div className={`alert py-2 text-sm ${refreshResult.action === 'merged' ? 'alert-success' : 'alert-info'}`}>
                <svg className="w-4 h-4 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                <span>{refreshResult.message}</span>
              </div>
            )}

            {/* Approve PR */}
            <button
              onClick={tracked('Approve PR', () => approveRemote())}
              disabled={loading}
              className="btn btn-primary btn-outline w-full"
            >
              <svg aria-hidden="true" className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              Approve PR
            </button>

            {/* Merge PR */}
            <button
              onClick={tracked('Merge PR', () => mergeRemote('rebase'))}
              disabled={loading}
              className="btn btn-secondary w-full"
            >
              <svg aria-hidden="true" className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4" />
              </svg>
              Merge PR
            </button>

            {/* Finish — cleanup after PR merge */}
            <button
              onClick={() => setShowFinishModal(true)}
              disabled={loading}
              className="btn btn-success w-full"
            >
              <svg aria-hidden="true" className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
              Finish &amp; Cleanup
            </button>
          </>
        )}
      </div>

      {/* Undo/Redo - only show when there are checkpoints or redo items */}
      {(canUndo || canRedo) && (
        <div className="flex gap-2 pt-4 border-t border-base-300 mt-4">
          {canUndo && (
            <button
              onClick={() => undo()}
              disabled={loading}
              className="btn btn-ghost flex-1"
              aria-label={`Undo (${checkpoints.length} checkpoint${checkpoints.length !== 1 ? 's' : ''})`}
            >
              <svg aria-hidden="true" className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 10h10a8 8 0 018 8v2M3 10l6 6m-6-6l6-6" />
              </svg>
              Undo
            </button>
          )}
          {canRedo && (
            <button
              onClick={() => redo()}
              disabled={loading}
              className="btn btn-ghost flex-1"
              aria-label={`Redo (${redoStack.length} in redo stack)`}
            >
              Redo
              <svg aria-hidden="true" className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 10h-10a8 8 0 00-8 8v2m18-10l-6 6m6-6l-6-6" />
              </svg>
            </button>
          )}
        </div>
      )}

      {/* Abort */}
      <button
        onClick={tracked('Abort', abort)}
        disabled={!canAbort || loading}
        className="btn btn-error btn-outline w-full mt-2"
      >
        <svg aria-hidden="true" className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
        </svg>
        Abort Task
      </button>

      {/* Abandon Task */}
      {isActive && (
        <>
          <button
            onClick={() => setShowAbandonModal(true)}
            disabled={loading}
            className="btn btn-error btn-sm w-full mt-1"
          >
            <svg aria-hidden="true" className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
            </svg>
            Abandon Task
          </button>
          <button
            onClick={() => setShowDeleteModal(true)}
            disabled={loading}
            className="btn btn-error btn-outline btn-sm w-full mt-1"
          >
            <svg aria-hidden="true" className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
            </svg>
            Delete Task
          </button>
        </>
      )}

      {/* Error display with retry */}
      {error && lastAction && (
        <div className="alert alert-error py-2 text-sm mt-2">
          <svg className="w-4 h-4 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
          <div className="flex-1 min-w-0">
            <p className="font-medium">{lastAction.label} failed</p>
            <p className="text-xs opacity-80 break-words">{error}</p>
          </div>
          <button
            onClick={() => lastAction.fn()}
            disabled={loading}
            className="btn btn-xs btn-outline btn-error flex-shrink-0"
          >
            Retry
          </button>
        </div>
      )}

      {/* Loading Indicator */}
      {loading && (
        <div className="flex items-center justify-center gap-2 py-2 text-primary">
          <span className="loading loading-spinner loading-sm"></span>
          <span className="text-sm">Working...</span>
        </div>
      )}

      {/* Current State */}
      <div className="pt-4 border-t border-base-300 mt-4">
        <div className="stat-card">
          <span className="stat-label">Current State</span>
          <span className="stat-value text-base capitalize">{state}</span>
        </div>
      </div>

      {/* Abandon confirmation modal */}
      {showAbandonModal && (
        <div className="modal modal-open">
          <div role="dialog" aria-modal="true" aria-labelledby="abandon-modal-title" className="modal-box bg-base-100 max-w-sm">
            <h3 id="abandon-modal-title" className="font-bold text-lg text-error mb-2">Abandon Task?</h3>
            <p className="text-sm text-base-content/80 mb-4">
              This will discard the current task and reset the worktree. This action cannot be undone.
            </p>
            <label className="flex items-center gap-3 cursor-pointer mb-6">
              <input
                type="checkbox"
                className="checkbox checkbox-sm"
                checked={abandonKeepBranch}
                onChange={e => setAbandonKeepBranch(e.target.checked)}
              />
              <span className="text-sm">Keep branch after abandoning</span>
            </label>
            <div className="modal-action">
              <button
                onClick={() => { setShowAbandonModal(false); setAbandonKeepBranch(false) }}
                className="btn btn-ghost"
              >
                Cancel
              </button>
              <button
                onClick={handleAbandon}
                className="btn btn-error"
              >
                Abandon
              </button>
            </div>
          </div>
          <button type="button" className="modal-backdrop bg-black/60" onClick={() => setShowAbandonModal(false)} aria-label="Close dialog" tabIndex={-1} />
        </div>
      )}

      {/* Submit confirmation modal */}
      {showSubmitModal && (
        <div className="modal modal-open">
          <div role="dialog" aria-modal="true" aria-labelledby="submit-modal-title" className="modal-box bg-base-100 max-w-sm">
            <h3 id="submit-modal-title" className="font-bold text-lg mb-2">Submit Pull Request</h3>
            <p className="text-sm text-base-content/80 mb-4">
              This will create a pull request for the current task.
            </p>
            <label className="flex items-center gap-3 cursor-pointer mb-6">
              <input
                type="checkbox"
                className="checkbox checkbox-sm"
                checked={submitDeleteBranch}
                onChange={e => setSubmitDeleteBranch(e.target.checked)}
              />
              <span className="text-sm">Delete branch after submitting</span>
            </label>
            <div className="modal-action">
              <button
                onClick={() => { setShowSubmitModal(false); setSubmitDeleteBranch(false) }}
                className="btn btn-ghost"
              >
                Cancel
              </button>
              <button
                onClick={handleSubmit}
                className="btn btn-primary"
              >
                <svg aria-hidden="true" className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
                </svg>
                Submit PR
              </button>
            </div>
          </div>
          <button type="button" className="modal-backdrop bg-black/60" onClick={() => setShowSubmitModal(false)} aria-label="Close dialog" tabIndex={-1} />
        </div>
      )}

      {/* Finish confirmation modal */}
      {showFinishModal && (
        <div className="modal modal-open">
          <div role="dialog" aria-modal="true" aria-labelledby="finish-modal-title" className="modal-box bg-base-100 max-w-sm">
            <h3 id="finish-modal-title" className="font-bold text-lg text-success mb-2">Finish Task</h3>
            <p className="text-sm text-base-content/80 mb-4">
              This will switch to the base branch, pull latest changes, and delete the feature branch.
            </p>
            <label className="flex items-center gap-3 cursor-pointer mb-6">
              <input
                type="checkbox"
                className="checkbox checkbox-sm"
                checked={finishDeleteRemote}
                onChange={e => setFinishDeleteRemote(e.target.checked)}
              />
              <span className="text-sm">Also delete remote branch</span>
            </label>
            <div className="modal-action">
              <button
                onClick={() => { setShowFinishModal(false); setFinishDeleteRemote(false) }}
                className="btn btn-ghost"
              >
                Cancel
              </button>
              <button
                onClick={handleFinish}
                className="btn btn-success"
              >
                <svg aria-hidden="true" className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                Finish
              </button>
            </div>
          </div>
          <button type="button" className="modal-backdrop bg-black/60" onClick={() => setShowFinishModal(false)} aria-label="Close dialog" tabIndex={-1} />
        </div>
      )}

      {/* Delete confirmation modal */}
      {showDeleteModal && (
        <div className="modal modal-open">
          <div role="dialog" aria-modal="true" aria-labelledby="delete-modal-title" className="modal-box bg-base-100 max-w-sm">
            <h3 id="delete-modal-title" className="font-bold text-lg text-error mb-2">Delete Task?</h3>
            <p className="text-sm text-base-content/80 mb-4">
              This will permanently delete the task data. This action cannot be undone.
            </p>
            <div className="modal-action">
              <button
                onClick={() => setShowDeleteModal(false)}
                className="btn btn-ghost"
              >
                Cancel
              </button>
              <button
                onClick={handleDelete}
                className="btn btn-error"
              >
                Delete
              </button>
            </div>
          </div>
          <button type="button" className="modal-backdrop bg-black/60" onClick={() => setShowDeleteModal(false)} aria-label="Close dialog" tabIndex={-1} />
        </div>
      )}
    </>
  )

  if (embedded) {
    return <div className="space-y-2">{content}</div>
  }

  return (
    <section className="card bg-base-200">
      <div className="card-body">
        <h2 className="card-title text-base-content flex items-center gap-2">
          <svg aria-hidden="true" className="w-5 h-5 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
          </svg>
          Actions
        </h2>
        {content}
      </div>
    </section>
  )
}
