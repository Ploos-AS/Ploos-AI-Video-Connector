# NextChat integration (experimental)

NextChat's MCP implementation launches a stdio child process. Enable MCP and configure the NextChat deployment access code. Build the Go binary on the NextChat server host (or inside its container).

```json
{"mcpServers":{"ploos-video":{"command":"/usr/local/bin/ploos-video-connector","args":["--transport","stdio"]}}}
```

This initial diagnostic MCP server does not generate videos yet. Never commit provider keys.
