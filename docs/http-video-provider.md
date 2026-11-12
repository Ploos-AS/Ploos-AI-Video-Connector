# Private HTTP video provider (experimental)

PR #9 adds an opt-in synchronous HTTP provider adapter. It is disabled unless both environment variables are set:

- `PLOOS_VIDEO_HTTP_ENDPOINT=http://127.0.0.1:9000/generate`
- `PLOOS_VIDEO_HTTP_TOKEN=<long random secret>`

The provider must listen on a **private or loopback IP literal**, with an explicit port and the exact `/generate` path. The connector sends `POST /generate` with bearer authentication and JSON:

```json
{"model":"demo","prompt":"A lighthouse in the fog"}
```

The provider must respond with HTTP 200 and:

```json
{"url":"https://example.invalid/generated/video.mp4","mime_type":"video/mp4"}
```

The URL is metadata, not a downloaded or proxied asset. The provider must serve the result separately. Redirects, environment proxies, public-IP endpoints and oversized responses are rejected. The HTTP client uses a two-minute timeout; jobs have a two-minute deadline. Existing MCP tools `video_create`, `video_status`, and `video_cancel` remain unchanged.

**Limitations:** This is a synchronous adapter, not yet a Sora API adapter. Jobs are held in memory and are lost on restart. Only HTTPS MP4 result metadata is accepted. Deploy the connector and the provider on the same host or a trusted private network, and never expose the upstream HTTP endpoint directly to the public internet.
