package jobs

// MetricsSnapshot returns bounded, aggregate, non-identifying job counts.
// Counts are current retained jobs, not lifetime counters (pruning removes jobs).
func(m *Manager) MetricsSnapshot()map[string]int{
 m.mu.Lock()
 defer m.mu.Unlock()
 counts:=map[string]int{"queued":0,"running":0,"completed":0,"failed":0,"cancelled":0}
 for _,job:=range m.jobs{
  if _,ok:=counts[job.Status];ok{counts[job.Status]++}
 }
 return counts
}
