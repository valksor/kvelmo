package metrics

import "fmt"

// RenderPrometheus renders a Snapshot in Prometheus text exposition format.
func RenderPrometheus(snap Snapshot) string {
	var b []byte

	// Helper to write a metric
	write := func(name, help, mtype string, value any) {
		b = fmt.Appendf(b, "# HELP %s %s\n# TYPE %s %s\n%s %v\n", name, help, name, mtype, name, value)
	}

	// Job metrics (counters)
	write("kvelmo_jobs_submitted_total", "Total jobs submitted.", "counter", snap.JobsSubmitted)
	write("kvelmo_jobs_completed_total", "Total jobs completed.", "counter", snap.JobsCompleted)
	write("kvelmo_jobs_failed_total", "Total jobs failed.", "counter", snap.JobsFailed)
	// In-progress is a gauge (derived)
	write("kvelmo_jobs_in_progress", "Jobs currently in progress.", "gauge", snap.JobsInProgress)

	// RPC metrics
	write("kvelmo_rpc_requests_total", "Total RPC requests.", "counter", snap.RPCRequests)
	write("kvelmo_rpc_errors_total", "Total RPC errors.", "counter", snap.RPCErrors)

	// Latency (as gauge for avg/p99)
	write("kvelmo_rpc_latency_avg_ms", "Average RPC latency in milliseconds.", "gauge", snap.AvgLatencyMs)
	write("kvelmo_rpc_latency_p99_ms", "P99 RPC latency in milliseconds.", "gauge", snap.P99LatencyMs)

	// Agent metrics
	write("kvelmo_agent_connects_total", "Total agent connections.", "counter", snap.AgentConnects)
	write("kvelmo_agent_disconnects_total", "Total agent disconnections.", "counter", snap.AgentDisconnects)
	write("kvelmo_events_dropped_total", "Total events dropped.", "counter", snap.EventsDropped)
	write("kvelmo_permissions_approved_total", "Total permissions approved.", "counter", snap.PermissionsApproved)
	write("kvelmo_permissions_denied_total", "Total permissions denied.", "counter", snap.PermissionsDenied)

	return string(b)
}
