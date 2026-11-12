# Ploos AI Video Connector monitoring

## Grafana

Import `grafana/ploos-video-connector.json` into Grafana and select your Prometheus datasource for the `DS_PROMETHEUS` UID placeholder. Dashboard panels show provider health (0/1), probe latency, current active jobs and retained completed/failed/cancelled jobs.

**Interpretation:** `ploos_video_jobs` is a gauge of jobs currently retained in memory/SQLite, **not** a lifetime counter. Pruning, eviction and restarts without SQLite can lower it. Do not use it to calculate historical billing, throughput or success rates.

## Prometheus on Mimir

The connector deliberately permits `/metrics` **only from loopback clients with a bearer token**. Therefore, Mimir cannot scrape the connector directly across WireGuard. The example `prometheus/video-connector-scrape.yml` must run in a Prometheus instance on the connector host, or on Mimir through an SSH tunnel bound to loopback on Mimir. For a same-host collector, use Prometheus Agent mode or an OpenTelemetry/Prometheus-compatible local collector forwarding via secure remote_write to Mimir's Prometheus receiver (if enabled). Configure the receiver and authentication separately.

1. Keep video-http listening on `127.0.0.1:8787` and do not expose `/metrics` via Caddy.
2. Place the metrics bearer token in `/etc/prometheus/secrets/ploos-video-metrics-token`, accessible only to the collector account. This token must match `PLOOS_MCP_BEARER_TOKEN`. Protect it from logs and backups.
3. Add the example scrape job to the collector's `scrape_configs`; it is a list item, **not** a complete Prometheus configuration.
4. Configure secure `remote_write` from the local collector to Mimir if using a separate collector, or tunnel the scrape securely.
5. Import the dashboard, select datasource, verify `up{job="ploos-video-connector"}` and the provider gauges.
6. Optionally include `prometheus/video-connector-alerts.yml` in Prometheus `rule_files` and connect Alertmanager.

The example alert rules warn about provider unavailability, slow health checks and failed scrapes. An absent time series caused by a disabled scrape is not the same as an unavailable provider; monitor collector health separately.

Scraping health endpoints is read-only and never triggers video generation or paid API requests. Metrics contain no prompts, job IDs, tokens or video URLs. The dashboard is a starting point; adjust thresholds and notification channels to your environment.
