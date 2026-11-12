# Async video provider (experimental)

PR #10 adds a provider-neutral asynchronous job API. Configure **one** upstream adapter:

- `PLOOS_VIDEO_ASYNC_ENDPOINT=http://127.0.0.1:9000/jobs`
- `PLOOS_VIDEO_ASYNC_TOKEN=<random-secret>`

Do not configure `PLOOS_VIDEO_HTTP_ENDPOINT` at the same time. The asynchronous adapter is opt-in; it does not call OpenAI/Sora or any paid provider by default.

Upstream contract:

- `POST /jobs` with bearer token and JSON `{"model":"demo","prompt":"..."}` returns HTTP 200/202 and `{"id":"job-1","status":"queued"}`.
- `GET /jobs/job-1` returns `{"id":"job-1","status":"running"}`, or `{"id":"job-1","status":"completed","url":"https://.../video.mp4","mime_type":"video/mp4"}`.
- `DELETE /jobs/job-1` is sent on local context cancellation as best-effort cleanup.

Status values: queued, running, in_progress, completed, failed, cancelled. The connector polls every two seconds and respects the parent job deadline. Results contain only an HTTPS MP4 URL, not the video bytes. The provider must be on a private/loopback IP and must not redirect. Credentials stay server-side.

**Important:** The Sora API was marked deprecated with a September 2026 shutdown date in OpenAI's Go SDK. This PR intentionally implements a provider-neutral interface instead of binding to a deprecated video API.

Known limitations: In-memory jobs are lost on restart; provider-side cancellation is best-effort; persistence, access control for result URLs, and durable job reconciliation are follow-up work.
