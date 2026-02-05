import { useState } from 'react'
import { Loader2, AlertCircle, Search, FileText, AlertTriangle, Info, CheckCircle } from 'lucide-react'
import { useStandaloneReview, type StandaloneMode, type ReviewIssue } from '@/api/standalone'
import { useStatus } from '@/api/workflow'
import { ProjectSelector } from '@/components/project/ProjectSelector'

export default function Review() {
  const { data: status, isLoading: statusLoading } = useStatus()
  const [mode, setMode] = useState<StandaloneMode>('uncommitted')
  const [baseBranch, setBaseBranch] = useState('main')
  const [range, setRange] = useState('')
  const [files, setFiles] = useState('')
  const [agent, setAgent] = useState('')
  const [createCheckpoint, setCreateCheckpoint] = useState(true)

  const reviewMutation = useStandaloneReview()

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
        <h1 className="text-2xl font-bold">Standalone Review</h1>
        <ProjectSelector />
      </div>
    )
  }

  const handleRun = async () => {
    const request: Parameters<typeof reviewMutation.mutateAsync>[0] = {
      mode,
      create_checkpoint: createCheckpoint,
    }

    if (mode === 'branch') request.base_branch = baseBranch
    if (mode === 'range') request.range = range
    if (mode === 'files') request.files = files.split(',').map((f) => f.trim()).filter(Boolean)
    if (agent) request.agent = agent

    await reviewMutation.mutateAsync(request)
  }

  const getSeverityIcon = (severity: ReviewIssue['severity']) => {
    switch (severity) {
      case 'error':
        return <AlertCircle size={16} className="text-error" />
      case 'warning':
        return <AlertTriangle size={16} className="text-warning" />
      case 'info':
        return <Info size={16} className="text-info" />
    }
  }

  const getSeverityBadge = (severity: ReviewIssue['severity']) => {
    switch (severity) {
      case 'error':
        return 'badge-error'
      case 'warning':
        return 'badge-warning'
      case 'info':
        return 'badge-info'
    }
  }

  const issues = reviewMutation.data?.issues || []
  const errorCount = issues.filter((i) => i.severity === 'error').length
  const warningCount = issues.filter((i) => i.severity === 'warning').length
  const infoCount = issues.filter((i) => i.severity === 'info').length

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold">Standalone Review</h1>
        <p className="text-base-content/60 mt-1">
          Run AI-powered code review without an active task
        </p>
      </div>

      {/* Configuration */}
      <div className="card bg-base-100 shadow-sm">
        <div className="card-body">
          <h3 className="font-medium mb-4">Review Configuration</h3>

          {/* Mode Selection */}
          <div className="form-control">
            <label className="label">
              <span className="label-text">Review Mode</span>
            </label>
            <select
              value={mode}
              onChange={(e) => setMode(e.target.value as StandaloneMode)}
              className="select select-bordered"
            >
              <option value="uncommitted">Uncommitted Changes</option>
              <option value="branch">Branch Comparison</option>
              <option value="range">Commit Range</option>
              <option value="files">Specific Files</option>
            </select>
          </div>

          {/* Mode-specific inputs */}
          {mode === 'branch' && (
            <div className="form-control">
              <label className="label">
                <span className="label-text">Base Branch</span>
              </label>
              <input
                type="text"
                value={baseBranch}
                onChange={(e) => setBaseBranch(e.target.value)}
                placeholder="main"
                className="input input-bordered"
              />
            </div>
          )}

          {mode === 'range' && (
            <div className="form-control">
              <label className="label">
                <span className="label-text">Commit Range</span>
              </label>
              <input
                type="text"
                value={range}
                onChange={(e) => setRange(e.target.value)}
                placeholder="HEAD~3..HEAD"
                className="input input-bordered"
              />
            </div>
          )}

          {mode === 'files' && (
            <div className="form-control">
              <label className="label">
                <span className="label-text">File Paths (comma-separated)</span>
              </label>
              <textarea
                value={files}
                onChange={(e) => setFiles(e.target.value)}
                placeholder="src/main.go, internal/handler.go"
                className="textarea textarea-bordered h-24"
              />
            </div>
          )}

          {/* Agent override */}
          <div className="form-control">
            <label className="label">
              <span className="label-text">Agent (optional)</span>
            </label>
            <input
              type="text"
              value={agent}
              onChange={(e) => setAgent(e.target.value)}
              placeholder="Leave empty for default"
              className="input input-bordered"
            />
          </div>

          {/* Checkpoint toggle */}
          <div className="form-control">
            <label className="label cursor-pointer justify-start gap-3">
              <input
                type="checkbox"
                checked={createCheckpoint}
                onChange={(e) => setCreateCheckpoint(e.target.checked)}
                className="checkbox checkbox-sm"
              />
              <span className="label-text">Create checkpoint before review</span>
            </label>
          </div>

          {/* Run button */}
          <button
            onClick={handleRun}
            disabled={reviewMutation.isPending}
            className="btn btn-primary mt-4"
          >
            {reviewMutation.isPending ? (
              <>
                <Loader2 className="w-4 h-4 animate-spin" />
                Running Review...
              </>
            ) : (
              <>
                <Search size={18} />
                Run Review
              </>
            )}
          </button>
        </div>
      </div>

      {/* Error */}
      {reviewMutation.isError && (
        <div className="alert alert-error">
          <AlertCircle size={18} />
          <span>{reviewMutation.error.message}</span>
        </div>
      )}

      {/* Results */}
      {reviewMutation.isSuccess && (
        <div className="space-y-4">
          {/* Summary */}
          <div className="stats shadow w-full">
            <div className="stat">
              <div className="stat-title">Total Issues</div>
              <div className="stat-value text-primary">{reviewMutation.data.total_issues}</div>
            </div>
            <div className="stat">
              <div className="stat-title">Errors</div>
              <div className="stat-value text-error">{errorCount}</div>
            </div>
            <div className="stat">
              <div className="stat-title">Warnings</div>
              <div className="stat-value text-warning">{warningCount}</div>
            </div>
            <div className="stat">
              <div className="stat-title">Info</div>
              <div className="stat-value text-info">{infoCount}</div>
            </div>
          </div>

          {/* Summary text */}
          {reviewMutation.data.summary && (
            <div className="card bg-base-100 shadow-sm">
              <div className="card-body">
                <h3 className="font-medium">Summary</h3>
                <p className="text-base-content/80 whitespace-pre-wrap">{reviewMutation.data.summary}</p>
              </div>
            </div>
          )}

          {/* Issues list */}
          {issues.length === 0 ? (
            <div className="card bg-base-100 shadow-sm">
              <div className="card-body text-center py-12">
                <CheckCircle className="w-12 h-12 mx-auto text-success mb-4" />
                <h2 className="text-lg font-medium">No Issues Found</h2>
                <p className="text-base-content/60 mt-2">
                  The review completed without finding any issues.
                </p>
              </div>
            </div>
          ) : (
            <div className="card bg-base-100 shadow-sm">
              <div className="card-body">
                <h3 className="font-medium mb-4">Issues ({issues.length})</h3>
                <div className="space-y-3">
                  {issues.map((issue, idx) => (
                    <div key={idx} className="flex items-start gap-3 p-3 bg-base-200 rounded-lg">
                      {getSeverityIcon(issue.severity)}
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 flex-wrap">
                          <span className={`badge badge-sm ${getSeverityBadge(issue.severity)}`}>
                            {issue.severity}
                          </span>
                          {issue.rule && (
                            <span className="badge badge-sm badge-ghost font-mono">{issue.rule}</span>
                          )}
                        </div>
                        <p className="mt-1 text-sm">{issue.message}</p>
                        <div className="flex items-center gap-1 text-xs text-base-content/50 mt-2">
                          <FileText size={12} />
                          <span className="font-mono">{issue.file}</span>
                          {issue.line && <span>:{issue.line}</span>}
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
