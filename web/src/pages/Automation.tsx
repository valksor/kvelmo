import { useState } from 'react'
import {
  Loader2,
  AlertCircle,
  Bot,
  RefreshCw,
  XCircle,
  Clock,
  CheckCircle,
  Play,
  Pause,
  ChevronDown,
  ChevronUp,
  ExternalLink,
  Settings,
  GitPullRequest,
  AlertTriangle,
} from 'lucide-react'
import {
  useAutomationStatus,
  useAutomationJobs,
  useAutomationConfig,
  useCancelJob,
  useRetryJob,
  type AutomationJob,
  type JobStatus,
} from '@/api/automation'

export default function Automation() {
  const [statusFilter, setStatusFilter] = useState<JobStatus | undefined>(undefined)
  const [showConfig, setShowConfig] = useState(false)

  const { data: status, isLoading: statusLoading, refetch: refetchStatus } = useAutomationStatus()
  const { data: jobsData, isLoading: jobsLoading, error: jobsError, refetch: refetchJobs } = useAutomationJobs(statusFilter)
  const { data: config } = useAutomationConfig()

  const cancelMutation = useCancelJob()
  const retryMutation = useRetryJob()

  if (statusLoading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    )
  }

  const handleCancel = async (jobId: string) => {
    if (!confirm('Cancel this job?')) return
    await cancelMutation.mutateAsync(jobId)
  }

  const handleRetry = async (jobId: string) => {
    await retryMutation.mutateAsync(jobId)
  }

  const getStatusBadge = (s: JobStatus) => {
    switch (s) {
      case 'pending':
        return (
          <span className="badge badge-ghost badge-sm gap-1">
            <Clock size={12} />
            Pending
          </span>
        )
      case 'running':
        return (
          <span className="badge badge-info badge-sm gap-1">
            <Play size={12} />
            Running
          </span>
        )
      case 'completed':
        return (
          <span className="badge badge-success badge-sm gap-1">
            <CheckCircle size={12} />
            Completed
          </span>
        )
      case 'failed':
        return (
          <span className="badge badge-error badge-sm gap-1">
            <XCircle size={12} />
            Failed
          </span>
        )
      case 'cancelled':
        return (
          <span className="badge badge-warning badge-sm gap-1">
            <Pause size={12} />
            Cancelled
          </span>
        )
      default:
        return <span className="badge badge-ghost badge-sm">{s}</span>
    }
  }

  const formatDuration = (job: AutomationJob) => {
    if (!job.started_at) return '-'
    const start = new Date(job.started_at)
    const end = job.completed_at ? new Date(job.completed_at) : new Date()
    const ms = end.getTime() - start.getTime()
    if (ms < 1000) return `${ms}ms`
    if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`
    return `${Math.floor(ms / 60000)}m ${Math.floor((ms % 60000) / 1000)}s`
  }

  const formatTime = (iso: string) => {
    const date = new Date(iso)
    return date.toLocaleString()
  }

  const jobs = jobsData?.jobs || []

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Automation</h1>
          <p className="text-base-content/60 mt-1">
            Webhook-based automation for GitHub and GitLab
          </p>
        </div>
        <button
          className="btn btn-ghost btn-sm"
          onClick={() => {
            refetchStatus()
            refetchJobs()
          }}
        >
          <RefreshCw size={16} />
          Refresh
        </button>
      </div>

      {/* Status Card */}
      <div className="card bg-base-100 shadow-sm">
        <div className="card-body">
          <div className="flex items-center justify-between mb-4">
            <h3 className="font-medium flex items-center gap-2">
              <Bot size={18} />
              Status
            </h3>
            {status?.enabled ? (
              <span className="badge badge-success gap-1">
                <CheckCircle size={12} />
                Enabled
              </span>
            ) : (
              <span className="badge badge-ghost gap-1">
                <XCircle size={12} />
                Disabled
              </span>
            )}
          </div>

          {!status?.enabled ? (
            <div className="text-center py-8">
              <Bot size={48} className="mx-auto text-base-content/30 mb-4" />
              <p className="text-base-content/60">Automation is not enabled</p>
              <p className="text-sm text-base-content/40 mt-1">
                Configure automation in <code>.mehrhof/config.yaml</code> under{' '}
                <code>automation:</code>
              </p>
            </div>
          ) : (
            <div className="stats stats-horizontal shadow-sm bg-base-200/50 w-full">
              <div className="stat">
                <div className="stat-title">Pending</div>
                <div className="stat-value text-lg">{status.queue?.pending || 0}</div>
              </div>
              <div className="stat">
                <div className="stat-title">Running</div>
                <div className="stat-value text-lg text-info">{status.queue?.running || 0}</div>
              </div>
              <div className="stat">
                <div className="stat-title">Completed</div>
                <div className="stat-value text-lg text-success">{status.queue?.completed || 0}</div>
              </div>
              <div className="stat">
                <div className="stat-title">Failed</div>
                <div className="stat-value text-lg text-error">{status.queue?.failed || 0}</div>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Configuration Card (Collapsible) */}
      {status?.enabled && config && (
        <div className="card bg-base-100 shadow-sm">
          <div className="card-body">
            <button
              type="button"
              className="flex items-center justify-between w-full text-left"
              onClick={() => setShowConfig(!showConfig)}
            >
              <h3 className="font-medium flex items-center gap-2">
                <Settings size={18} />
                Configuration
              </h3>
              {showConfig ? <ChevronUp size={18} /> : <ChevronDown size={18} />}
            </button>

            {showConfig && (
              <div className="mt-4 space-y-4">
                {/* Providers */}
                {config.providers && Object.keys(config.providers).length > 0 && (
                  <div>
                    <h4 className="text-sm font-medium text-base-content/70 mb-2">Providers</h4>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-2">
                      {Object.entries(config.providers).map(([name, provider]) => (
                        <div
                          key={name}
                          className="p-3 rounded-lg bg-base-200/50 flex items-center justify-between"
                        >
                          <div>
                            <span className="font-medium capitalize">{name}</span>
                            {provider.command_prefix && (
                              <span className="text-xs text-base-content/50 ml-2">
                                prefix: {provider.command_prefix}
                              </span>
                            )}
                          </div>
                          {provider.enabled ? (
                            <span className="badge badge-success badge-xs">enabled</span>
                          ) : (
                            <span className="badge badge-ghost badge-xs">disabled</span>
                          )}
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                {/* Access Control */}
                {config.access_control && (
                  <div>
                    <h4 className="text-sm font-medium text-base-content/70 mb-2">Access Control</h4>
                    <div className="p-3 rounded-lg bg-base-200/50">
                      <div className="flex items-center gap-4 flex-wrap">
                        <span>
                          Mode: <code className="text-sm">{config.access_control.mode}</code>
                        </span>
                        {config.access_control.require_org_membership && (
                          <span className="badge badge-sm">Requires org membership</span>
                        )}
                        {config.access_control.allow_bots && (
                          <span className="badge badge-sm">Allows bots</span>
                        )}
                      </div>
                    </div>
                  </div>
                )}

                {/* Queue Settings */}
                {config.queue && (
                  <div>
                    <h4 className="text-sm font-medium text-base-content/70 mb-2">Queue Settings</h4>
                    <div className="p-3 rounded-lg bg-base-200/50 flex gap-4 flex-wrap text-sm">
                      {config.queue.max_concurrent && (
                        <span>Max concurrent: {config.queue.max_concurrent}</span>
                      )}
                      {config.queue.job_timeout && (
                        <span>Timeout: {config.queue.job_timeout}</span>
                      )}
                      {config.queue.retry_attempts && (
                        <span>Retries: {config.queue.retry_attempts}</span>
                      )}
                    </div>
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      )}

      {/* Jobs Table */}
      {status?.enabled && (
        <div className="card bg-base-100 shadow-sm">
          <div className="card-body">
            <div className="flex items-center justify-between mb-4">
              <h3 className="font-medium flex items-center gap-2">
                <GitPullRequest size={18} />
                Jobs ({jobs.length})
              </h3>

              {/* Status Filter */}
              <select
                className="select select-bordered select-sm"
                value={statusFilter || ''}
                onChange={(e) => setStatusFilter((e.target.value || undefined) as JobStatus | undefined)}
              >
                <option value="">All statuses</option>
                <option value="pending">Pending</option>
                <option value="running">Running</option>
                <option value="completed">Completed</option>
                <option value="failed">Failed</option>
                <option value="cancelled">Cancelled</option>
              </select>
            </div>

            {jobsLoading ? (
              <div className="flex justify-center py-8">
                <Loader2 className="w-6 h-6 animate-spin text-primary" />
              </div>
            ) : jobsError ? (
              <div className="alert alert-error">
                <AlertCircle size={18} />
                <span>{jobsError instanceof Error ? jobsError.message : 'Failed to load jobs'}</span>
              </div>
            ) : jobs.length === 0 ? (
              <div className="text-center py-12">
                <GitPullRequest size={48} className="mx-auto text-base-content/30 mb-4" />
                <p className="text-base-content/60">No automation jobs</p>
                <p className="text-sm text-base-content/40 mt-1">
                  Jobs appear when webhooks trigger automation
                </p>
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="table table-sm">
                  <thead>
                    <tr>
                      <th>ID</th>
                      <th>Type</th>
                      <th>Status</th>
                      <th>Event</th>
                      <th>Duration</th>
                      <th>Created</th>
                      <th>Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {jobs.map((job: AutomationJob) => (
                      <tr key={job.id} className="hover">
                        <td className="font-mono text-xs">{job.id.substring(0, 8)}</td>
                        <td>
                          <span className="badge badge-outline badge-sm">{job.workflow_type}</span>
                        </td>
                        <td>{getStatusBadge(job.status)}</td>
                        <td>
                          {job.event ? (
                            <div className="text-sm">
                              <span className="text-base-content/70">{job.event.provider}</span>
                              <span className="mx-1">·</span>
                              <span>{job.event.type}</span>
                              {job.event.pr_number && (
                                <span className="ml-1">#{job.event.pr_number}</span>
                              )}
                              {job.event.issue_number && (
                                <span className="ml-1">#{job.event.issue_number}</span>
                              )}
                            </div>
                          ) : (
                            <span className="text-base-content/40">-</span>
                          )}
                        </td>
                        <td className="text-sm">{formatDuration(job)}</td>
                        <td className="text-xs text-base-content/60">{formatTime(job.created_at)}</td>
                        <td>
                          <div className="flex gap-1">
                            {job.status === 'running' || job.status === 'pending' ? (
                              <button
                                className="btn btn-ghost btn-xs text-warning"
                                onClick={() => handleCancel(job.id)}
                                disabled={cancelMutation.isPending}
                                title="Cancel"
                              >
                                <XCircle size={14} />
                              </button>
                            ) : null}
                            {job.status === 'failed' && (
                              <button
                                className="btn btn-ghost btn-xs text-info"
                                onClick={() => handleRetry(job.id)}
                                disabled={retryMutation.isPending}
                                title="Retry"
                              >
                                <RefreshCw size={14} />
                              </button>
                            )}
                            {job.result?.pr_url && (
                              <a
                                href={job.result.pr_url}
                                target="_blank"
                                rel="noopener noreferrer"
                                className="btn btn-ghost btn-xs"
                                title="View PR"
                              >
                                <ExternalLink size={14} />
                              </a>
                            )}
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}

            {/* Job errors */}
            {jobs.some((j: AutomationJob) => j.error) && (
              <div className="mt-4 space-y-2">
                <h4 className="text-sm font-medium text-error flex items-center gap-1">
                  <AlertTriangle size={14} />
                  Recent Errors
                </h4>
                {jobs
                  .filter((j: AutomationJob) => j.error)
                  .slice(0, 3)
                  .map((job: AutomationJob) => (
                    <div key={job.id} className="p-2 rounded bg-error/10 text-sm">
                      <span className="font-mono text-xs">{job.id.substring(0, 8)}</span>
                      <span className="mx-2">·</span>
                      <span className="text-error">{job.error}</span>
                    </div>
                  ))}
              </div>
            )}
          </div>
        </div>
      )}

      {/* Mutation errors */}
      {cancelMutation.isError && (
        <div className="alert alert-error">
          <AlertCircle size={18} />
          <span>Failed to cancel: {cancelMutation.error.message}</span>
        </div>
      )}
      {retryMutation.isError && (
        <div className="alert alert-error">
          <AlertCircle size={18} />
          <span>Failed to retry: {retryMutation.error.message}</span>
        </div>
      )}
    </div>
  )
}
