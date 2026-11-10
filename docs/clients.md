# Client compatibility plan

| Client | Planned transport | Status |
| --- | --- | --- |
| NextChat | stdio | Experimental diagnostic server in PR #1 |
| LibreChat | stdio / Streamable HTTP | HTTP interoperability not verified |
| Open WebUI | Streamable HTTP | HTTP interoperability not verified |
| ChatGPT | Remote HTTPS MCP | OAuth and ChatGPT integration not implemented |

The experimental `video-http` server currently implements only JSON-response POST requests for a small set of MCP methods. It is **not yet a fully conformant Streamable HTTP MCP server**, and must not be advertised as production-ready.

## Local diagnostic test

```sh
export PLOOS_MCP_BEARER_TOKEN="$(openssl rand -hex 32)"
go run ./cmd/video-http --listen 127.0.0.1:8787
```

In a second terminal:

```sh
curl -sS -X POST http://127.0.0.1:8787/mcp \
  -H "Authorization: Bearer $PLOOS_MCP_BEARER_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

For remote clients, use HTTPS with a reverse proxy, proper OAuth authorization, and MCP protocol conformance tests. Do not publish this experimental endpoint to the internet.

## Next milestones

1. Complete Streamable HTTP spec compliance and transport tests.
2. Implement OAuth 2.0 / protected resource metadata and per-user authorization.
3. Validate each client with integration tests, including tool invocation.
4. Add provider adapters and mandatory cost approval before any paid job.
