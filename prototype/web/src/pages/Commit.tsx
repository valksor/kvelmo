import { useState } from 'react'
import { Loader2, GitCommit, AlertCircle, FileText, Plus, Minus, Check, RefreshCw } from 'lucide-react'
import { useChanges, useAnalyzeChanges, useApplyCommit } from '@/api/commit'
import { useStatus } from '@/api/workflow'
import { ProjectSelector } from '@/components/project/ProjectSelector'

export default function Commit() {
  const { data: status, isLoading: statusLoading } = useStatus()
  const [includeUnstaged, setIncludeUnstaged] = useState(false)
  const [commitMessage, setCommitMessage] = useState('')
  const [isEditing, setIsEditing] = useState(false)

  const { data: changesData, isLoading: changesLoading, error: changesError, refetch } = useChanges(includeUnstaged)
  const analyzeMutation = useAnalyzeChanges()
  const applyMutation = useApplyCommit()

  if (statusLoading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    )
  }

  // Global mode: show project selector
  if (status?.mode === 'global') {
    return (
      <div className="space-y-4">
        <h1 className="text-2xl font-bold">Commit Generation</h1>
        <ProjectSelector />
      </div>
    )
  }

  const handleAnalyze = async () => {
    try {
      const result = await analyzeMutation.mutateAsync({ include_unstaged: includeUnstaged })
      setCommitMessage(result.message)
      setIsEditing(true)
    } catch {
      // Error handled by mutation state
    }
  }

  const handleApply = async () => {
    if (!commitMessage.trim()) return
    try {
      await applyMutation.mutateAsync({ message: commitMessage })
      setCommitMessage('')
      setIsEditing(false)
      refetch()
    } catch {
      // Error handled by mutation state
    }
  }

  const files = changesData?.files || []
  const hasChanges = files.length > 0

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Commit Generation</h1>
          <p className="text-base-content/60 mt-1">
            AI-powered commit message generation for your changes
          </p>
        </div>
        <button onClick={() => refetch()} className="btn btn-ghost btn-sm">
          <RefreshCw size={16} />
          Refresh
        </button>
      </div>

      {changesError && (
        <div className="alert alert-error">
          <AlertCircle size={18} />
          <span>Failed to load changes: {changesError.message}</span>
        </div>
      )}

      {/* Options */}
      <div className="card bg-base-100 shadow-sm">
        <div className="card-body py-4">
          <label className="flex items-center gap-3 cursor-pointer">
            <input
              type="checkbox"
              checked={includeUnstaged}
              onChange={(e) => setIncludeUnstaged(e.target.checked)}
              className="checkbox checkbox-sm"
            />
            <span className="text-sm">Include unstaged changes</span>
          </label>
        </div>
      </div>

      {/* Changes Summary */}
      {changesLoading ? (
        <div className="flex justify-center py-12">
          <Loader2 className="w-6 h-6 animate-spin text-primary" />
        </div>
      ) : !hasChanges ? (
        <div className="card bg-base-100 shadow-sm">
          <div className="card-body text-center py-12">
            <GitCommit className="w-12 h-12 mx-auto text-base-content/30 mb-4" />
            <h2 className="text-lg font-medium">No Changes</h2>
            <p className="text-base-content/60 mt-2">
              {includeUnstaged
                ? 'No uncommitted changes in your working directory.'
                : 'No staged changes. Try enabling "Include unstaged changes" or stage some files.'}
            </p>
          </div>
        </div>
      ) : (
        <>
          {/* Stats */}
          <div className="stats shadow w-full">
            <div className="stat">
              <div className="stat-title">Files Changed</div>
              <div className="stat-value text-primary">{files.length}</div>
            </div>
            <div className="stat">
              <div className="stat-title">Additions</div>
              <div className="stat-value text-success flex items-center gap-2">
                <Plus size={24} />
                {changesData?.total_additions || 0}
              </div>
            </div>
            <div className="stat">
              <div className="stat-title">Deletions</div>
              <div className="stat-value text-error flex items-center gap-2">
                <Minus size={24} />
                {changesData?.total_deletions || 0}
              </div>
            </div>
          </div>

          {/* File List */}
          <div className="card bg-base-100 shadow-sm">
            <div className="card-body">
              <h3 className="font-medium mb-3">Changed Files</h3>
              <div className="space-y-2 max-h-64 overflow-y-auto">
                {files.map((file) => (
                  <div
                    key={file.path}
                    className="flex items-center justify-between py-2 px-3 bg-base-200 rounded-lg"
                  >
                    <div className="flex items-center gap-2">
                      <FileText size={16} className="text-base-content/50" />
                      <span className="font-mono text-sm">{file.path}</span>
                      <span
                        className={`badge badge-sm ${
                          file.status === 'added'
                            ? 'badge-success'
                            : file.status === 'deleted'
                              ? 'badge-error'
                              : file.status === 'renamed'
                                ? 'badge-warning'
                                : 'badge-info'
                        }`}
                      >
                        {file.status}
                      </span>
                    </div>
                    <div className="flex items-center gap-2 text-sm">
                      <span className="text-success">+{file.additions}</span>
                      <span className="text-error">-{file.deletions}</span>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>

          {/* Analyze Button */}
          {!isEditing && (
            <button
              onClick={handleAnalyze}
              disabled={analyzeMutation.isPending}
              className="btn btn-primary w-full"
            >
              {analyzeMutation.isPending ? (
                <>
                  <Loader2 className="w-4 h-4 animate-spin" />
                  Analyzing...
                </>
              ) : (
                <>
                  <GitCommit size={18} />
                  Analyze Changes
                </>
              )}
            </button>
          )}

          {analyzeMutation.isError && (
            <div className="alert alert-error">
              <AlertCircle size={18} />
              <span>Failed to analyze: {analyzeMutation.error.message}</span>
            </div>
          )}

          {/* Commit Message Editor */}
          {isEditing && (
            <div className="card bg-base-100 shadow-sm">
              <div className="card-body">
                <h3 className="font-medium mb-3">Commit Message</h3>
                <textarea
                  value={commitMessage}
                  onChange={(e) => setCommitMessage(e.target.value)}
                  className="textarea textarea-bordered w-full h-32 font-mono"
                  placeholder="Enter commit message..."
                />
                <div className="flex gap-2 mt-4">
                  <button
                    onClick={handleApply}
                    disabled={applyMutation.isPending || !commitMessage.trim()}
                    className="btn btn-success flex-1"
                  >
                    {applyMutation.isPending ? (
                      <>
                        <Loader2 className="w-4 h-4 animate-spin" />
                        Committing...
                      </>
                    ) : (
                      <>
                        <Check size={18} />
                        Apply Commit
                      </>
                    )}
                  </button>
                  <button
                    onClick={() => {
                      setIsEditing(false)
                      setCommitMessage('')
                    }}
                    className="btn btn-ghost"
                  >
                    Cancel
                  </button>
                </div>
              </div>
            </div>
          )}

          {applyMutation.isError && (
            <div className="alert alert-error">
              <AlertCircle size={18} />
              <span>Failed to commit: {applyMutation.error.message}</span>
            </div>
          )}

          {applyMutation.isSuccess && (
            <div className="alert alert-success">
              <Check size={18} />
              <span>
                Commit created: <code className="font-mono">{applyMutation.data.commit_hash.slice(0, 7)}</code>
              </span>
            </div>
          )}
        </>
      )}
    </div>
  )
}
