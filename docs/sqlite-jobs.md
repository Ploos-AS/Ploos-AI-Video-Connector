# SQLite job persistence (experimental)

Set `PLOOS_VIDEO_SQLITE_PATH=/var/lib/ploos-video/jobs.sqlite` for the HTTP MCP server. The directory must exist and be writable by the service account. SQLite uses WAL and a single connection. With this variable unset, the connector keeps its original in-memory behavior.

The store saves job creation, running state, completion, cancellation and result metadata. On restart, terminal jobs are restored; queued/running jobs are **marked failed**, not resumed. This is intentional: the connector does not yet have enough remote-job identifiers to safely reconcile an in-progress provider job.

Terminal jobs older than seven days are removed **on startup**. This is not a continuous retention worker. Back up the SQLite database with SQLite-aware backup tools rather than copying a live WAL database. Do not share one SQLite file across multiple connector processes.

**Current limitations:** Persistence is best-effort for background transitions; failures to write completion updates are not yet propagated to callers. A full durable-job implementation needs transactional state changes, ongoing pruning, provider job IDs and restart reconciliation. The existing 100-job in-memory cap still applies to loaded records. This is an opt-in first step, not yet production-grade durable job orchestration.
