# Provider health (PR #17)

The authenticated MCP tool `video_providers` checks each configured registry provider using a read-only `GET /health` on the same private host and port as its existing adapter endpoint. The request carries the configured bearer token. The tool never calls `POST /generate` or `POST /jobs`.

Each provider is checked in parallel with a three-second deadline. Status is `healthy` for successful HTTP 200 responses under one second, `degraded` for successful responses taking at least one second, and `unavailable` for errors, non-200 responses or timeouts. The response includes latency in milliseconds and check time. This is an on-demand snapshot, not persistent monitoring or a guarantee that generation will succeed.

In legacy single-provider mode the tool returns an empty list. The provider registry is required for named health reporting. Health status is informational and **does not trigger failover or automatically select another paid provider**.

Upstream adapters must implement `GET /health` and return HTTP 200 to report healthy. The check is intentionally not a video-generation test and therefore does not incur video API charges.
