# Prometheus metrics (PR #18)

The video HTTP process exposes `GET /metrics` on its existing listener. Access requires **both** a loopback TCP peer and an `Authorization: Bearer <PLOOS_MCP_BEARER_TOKEN>` header. Do not publish this path through Caddy or any public reverse proxy. For remote Prometheus, use a private SSH tunnel or a local authenticated collector/agent on the same host. A WireGuard address alone is **not** loopback and is intentionally denied.

Metrics:
- `ploos_video_jobs{status="queued|running|completed|failed|cancelled"}` — current retained job count gauge. This is **not** a lifetime total; retention and pruning affect it.
- `ploos_video_provider_up{provider="..."}` — 1 when read-only health probe succeeds, 0 otherwise.
- `ploos_video_provider_latency_seconds{provider="..."}` — last health probe latency.

Each scrape probes configured private providers using read-only `GET /health`, with a three-second per-provider deadline. Configure an appropriate scrape interval and timeout (for example, 30s interval and 10s timeout). Scrapes never trigger video generation, paid requests, or fallback.

**Important:** In OAuth-introspection-only deployments, configure a separate nonempty `PLOOS_MCP_BEARER_TOKEN` for metrics access. It is not an OAuth token. Store it securely and do not log it. Avoid exposing `/metrics` through a public reverse proxy, including one that forwards localhost addresses.

Metrics contain no prompts, job IDs, video URLs, tokens or remote IDs. Provider labels are from administrator-controlled configuration.
