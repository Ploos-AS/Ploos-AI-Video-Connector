# Remote job recovery (PR #13)

When SQLite persistence and the asynchronous HTTP provider are both enabled, the connector stores the upstream job ID immediately after the provider accepts a video request. On restart, jobs with known remote IDs resume via GET /jobs/{id}; **they are never submitted again via POST**.

Jobs without an upstream ID are marked failed on restart. This includes the unavoidable crash window between remote acceptance and local persistence. A later version should use provider-supported idempotency keys to close this gap.

Recovery is limited by the connector's concurrency cap. Excess jobs fail closed rather than being resubmitted. Remote provider credentials and endpoint configuration must remain consistent across restarts.

The synchronous HTTP provider cannot resume work. Results are still metadata URLs, not downloaded assets. Database failures, expired upstream jobs, or unavailable providers may cause recovery to fail; monitor these statuses. A full production queue needs durable timeout accounting, idempotency keys, multi-instance locking and retries.
