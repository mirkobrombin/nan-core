package engine

import (
	"sync"
	"sync/atomic"
	"time"
)

type Metrics struct {
	IngestCount  atomic.Int64
	QueryCount   atomic.Int64
	ResolveCount atomic.Int64
	DeriveCount  atomic.Int64
	ClarifyCount atomic.Int64
	ReplayCount  atomic.Int64
	CompactCount atomic.Int64
	ErrorCount   atomic.Int64

	mu          sync.Mutex
	traceLog    []TraceEntry
	traceLogMax int
}

type TraceEntry struct {
	TraceID   string
	Op        string
	Duration  time.Duration
	Timestamp time.Time
	Details   map[string]string
}

func NewMetrics(traceLogMax int) *Metrics {
	if traceLogMax <= 0 {
		traceLogMax = 10000
	}
	return &Metrics{traceLogMax: traceLogMax}
}

func (m *Metrics) Record(traceID, op string, d time.Duration, details map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.traceLog) >= m.traceLogMax {
		m.traceLog = m.traceLog[1:]
	}
	m.traceLog = append(m.traceLog, TraceEntry{
		TraceID:   traceID,
		Op:        op,
		Duration:  d,
		Timestamp: time.Now(),
		Details:   details,
	})
}

func (m *Metrics) Traces(last int) []TraceEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	if last <= 0 || last > len(m.traceLog) {
		last = len(m.traceLog)
	}
	out := make([]TraceEntry, last)
	copy(out, m.traceLog[len(m.traceLog)-last:])
	return out
}

type MetricsSnapshot struct {
	Ingest  int64
	Query   int64
	Resolve int64
	Derive  int64
	Clarify int64
	Replay  int64
	Compact int64
	Errors  int64
}

func (m *Metrics) Snapshot() MetricsSnapshot {
	return MetricsSnapshot{
		Ingest:  m.IngestCount.Load(),
		Query:   m.QueryCount.Load(),
		Resolve: m.ResolveCount.Load(),
		Derive:  m.DeriveCount.Load(),
		Clarify: m.ClarifyCount.Load(),
		Replay:  m.ReplayCount.Load(),
		Compact: m.CompactCount.Load(),
		Errors:  m.ErrorCount.Load(),
	}
}
