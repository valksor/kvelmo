// Package metrics provides simple in-memory metrics for kvelmo.
package metrics

import (
	"math"
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics holds application-wide metrics.
type Metrics struct {
	// Job counters
	JobsSubmitted atomic.Int64
	JobsCompleted atomic.Int64
	JobsFailed    atomic.Int64

	// RPC counters
	RPCRequests atomic.Int64
	RPCErrors   atomic.Int64

	// Agent counters
	AgentConnects       atomic.Int64
	AgentDisconnects    atomic.Int64
	EventsDropped       atomic.Int64
	PermissionsApproved atomic.Int64
	PermissionsDenied   atomic.Int64

	// Latency tracking (simple moving average)
	mu              sync.RWMutex
	rpcLatencies    []time.Duration
	maxLatencySamps int
}

// New creates a new Metrics instance.
func New() *Metrics {
	return &Metrics{
		maxLatencySamps: 100, // Keep last 100 latency samples
	}
}

// RecordJobSubmitted increments the jobs submitted counter.
func (m *Metrics) RecordJobSubmitted() {
	m.JobsSubmitted.Add(1)
}

// RecordJobCompleted increments the jobs completed counter.
func (m *Metrics) RecordJobCompleted() {
	m.JobsCompleted.Add(1)
}

// RecordJobFailed increments the jobs failed counter.
func (m *Metrics) RecordJobFailed() {
	m.JobsFailed.Add(1)
}

// RecordAgentConnect increments the agent connects counter.
func (m *Metrics) RecordAgentConnect() {
	m.AgentConnects.Add(1)
}

// RecordAgentDisconnect increments the agent disconnects counter.
func (m *Metrics) RecordAgentDisconnect() {
	m.AgentDisconnects.Add(1)
}

// RecordEventDropped increments the dropped events counter.
func (m *Metrics) RecordEventDropped() {
	m.EventsDropped.Add(1)
}

// RecordPermissionApproved increments the approved permissions counter.
func (m *Metrics) RecordPermissionApproved() {
	m.PermissionsApproved.Add(1)
}

// RecordPermissionDenied increments the denied permissions counter.
func (m *Metrics) RecordPermissionDenied() {
	m.PermissionsDenied.Add(1)
}

// RecordRPCRequest records an RPC request with its latency.
func (m *Metrics) RecordRPCRequest(latency time.Duration, err error) {
	m.RPCRequests.Add(1)
	if err != nil {
		m.RPCErrors.Add(1)
	}

	m.mu.Lock()
	m.rpcLatencies = append(m.rpcLatencies, latency)
	if len(m.rpcLatencies) > m.maxLatencySamps {
		m.rpcLatencies = m.rpcLatencies[1:]
	}
	m.mu.Unlock()
}

// Snapshot returns a point-in-time snapshot of all metrics.
type Snapshot struct {
	JobsSubmitted  int64   `json:"jobs_submitted"`
	JobsCompleted  int64   `json:"jobs_completed"`
	JobsFailed     int64   `json:"jobs_failed"`
	JobsInProgress int64   `json:"jobs_in_progress"`
	RPCRequests    int64   `json:"rpc_requests"`
	RPCErrors      int64   `json:"rpc_errors"`
	AvgLatencyMs   float64 `json:"avg_latency_ms"`
	P99LatencyMs   float64 `json:"p99_latency_ms"`

	// Agent metrics
	AgentConnects       int64 `json:"agent_connects"`
	AgentDisconnects    int64 `json:"agent_disconnects"`
	EventsDropped       int64 `json:"events_dropped"`
	PermissionsApproved int64 `json:"permissions_approved"`
	PermissionsDenied   int64 `json:"permissions_denied"`
}

// Snapshot returns current metrics values.
func (m *Metrics) Snapshot() Snapshot {
	s := Snapshot{
		JobsSubmitted: m.JobsSubmitted.Load(),
		JobsCompleted: m.JobsCompleted.Load(),
		JobsFailed:    m.JobsFailed.Load(),
		RPCRequests:   m.RPCRequests.Load(),
		RPCErrors:     m.RPCErrors.Load(),
	}
	s.JobsInProgress = max(0, s.JobsSubmitted-s.JobsCompleted-s.JobsFailed)
	s.AgentConnects = m.AgentConnects.Load()
	s.AgentDisconnects = m.AgentDisconnects.Load()
	s.EventsDropped = m.EventsDropped.Load()
	s.PermissionsApproved = m.PermissionsApproved.Load()
	s.PermissionsDenied = m.PermissionsDenied.Load()

	m.mu.RLock()
	if len(m.rpcLatencies) > 0 {
		var total time.Duration
		for _, l := range m.rpcLatencies {
			total += l
		}
		s.AvgLatencyMs = float64(total.Milliseconds()) / float64(len(m.rpcLatencies))

		// P99: 99th percentile using ceiling-based index
		if len(m.rpcLatencies) > 1 {
			sorted := make([]time.Duration, len(m.rpcLatencies))
			copy(sorted, m.rpcLatencies)
			slices.Sort(sorted)

			n := len(sorted)
			p99Idx := int(math.Ceil(float64(n)*0.99)) - 1
			if p99Idx < 0 {
				p99Idx = 0
			}
			if p99Idx >= n {
				p99Idx = n - 1
			}
			s.P99LatencyMs = float64(sorted[p99Idx].Milliseconds())
		}
	}
	m.mu.RUnlock()

	return s
}

// RestoreFrom sets counter values from a previously saved snapshot.
// Used to restore metrics after a process restart.
func (m *Metrics) RestoreFrom(snap Snapshot) {
	m.JobsSubmitted.Store(snap.JobsSubmitted)
	m.JobsCompleted.Store(snap.JobsCompleted)
	m.JobsFailed.Store(snap.JobsFailed)
	m.RPCRequests.Store(snap.RPCRequests)
	m.RPCErrors.Store(snap.RPCErrors)
	m.AgentConnects.Store(snap.AgentConnects)
	m.AgentDisconnects.Store(snap.AgentDisconnects)
	m.EventsDropped.Store(snap.EventsDropped)
	m.PermissionsApproved.Store(snap.PermissionsApproved)
	m.PermissionsDenied.Store(snap.PermissionsDenied)
}

// Global metrics instance.
var global = New()

// Global returns the global metrics instance.
func Global() *Metrics {
	return global
}
