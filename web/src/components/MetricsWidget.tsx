import { useGlobalStore } from '../stores/globalStore'

export function MetricsWidget() {
  const metrics = useGlobalStore(s => s.metrics)
  const loadMetrics = useGlobalStore(s => s.loadMetrics)

  if (!metrics) {
    return (
      <div className="card bg-base-200 p-4">
        <h3 className="font-semibold mb-2">System Metrics</h3>
        <p className="text-sm opacity-60">No metrics available</p>
        <button className="btn btn-xs btn-ghost mt-2" onClick={loadMetrics}>Refresh</button>
      </div>
    )
  }

  return (
    <div className="card bg-base-200 p-4">
      <div className="flex items-center justify-between mb-3">
        <h3 className="font-semibold">System Metrics</h3>
        <button className="btn btn-xs btn-ghost" onClick={loadMetrics}>Refresh</button>
      </div>
      <div className="grid grid-cols-2 gap-2 text-sm">
        <div>
          <span className="opacity-60">Jobs Submitted</span>
          <span className="block font-mono">{metrics.jobs_submitted}</span>
        </div>
        <div>
          <span className="opacity-60">Jobs Completed</span>
          <span className="block font-mono">{metrics.jobs_completed}</span>
        </div>
        <div>
          <span className="opacity-60">Jobs Failed</span>
          <span className="block font-mono text-error">{metrics.jobs_failed || 0}</span>
        </div>
        <div>
          <span className="opacity-60">In Progress</span>
          <span className="block font-mono">{metrics.jobs_in_progress}</span>
        </div>
        <div>
          <span className="opacity-60">RPC Requests</span>
          <span className="block font-mono">{metrics.rpc_requests}</span>
        </div>
        <div>
          <span className="opacity-60">P99 Latency</span>
          <span className="block font-mono">{(metrics.p99_latency_ms ?? 0).toFixed(1)}ms</span>
        </div>
        <div>
          <span className="opacity-60">Agent Connects</span>
          <span className="block font-mono">{metrics.agent_connects}</span>
        </div>
        <div>
          <span className="opacity-60">Events Dropped</span>
          <span className="block font-mono">{metrics.events_dropped || 0}</span>
        </div>
      </div>
    </div>
  )
}
