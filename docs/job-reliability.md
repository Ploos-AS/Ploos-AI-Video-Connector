# Job reliability (PR #12)

- Terminal jobs (completed, failed, cancelled) may be evicted, oldest first, when the in-memory job limit is reached. Active jobs are never evicted for capacity.
- SQLite eviction is performed before removing a job from memory.
- Failed persistence on submission or cancellation is surfaced as a storage error, not an invalid request. A failed background state write fails closed in the in-memory status.
- Terminal jobs older than seven days are pruned on startup and hourly, with database and in-memory state kept consistent. Pruning errors are logged.

## Not yet implemented

**Remote job recovery is not yet safe.** The asynchronous provider does not persist the remote job identifier or request parameters. Interrupted jobs remain marked failed on restart; automatically resubmitting them could duplicate paid generation. A later PR must add durable provider IDs and explicit reconciliation semantics before resume is enabled.

SQLite persistence uses CGO and remains a single-process deployment option. Background write failures can still leave the database with an earlier status; use operational monitoring and backups.
