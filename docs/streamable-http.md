# Stateless Streamable HTTP (experimental)

The diagnostic server implements JSON-response POST handling for MCP protocol version `2025-03-26`. It supports `initialize`, `ping`, `tools/list`, `tools/call`, and notification acknowledgement (`202 Accepted`). It does not issue session IDs, and GET/DELETE are unsupported in stateless mode.

## Local test

```sh
export PLOOS_MCP_BEARER_TOKEN="$(openssl rand -hex 32)"
go run ./cmd/video-http
```

In another shell with the same token:

```sh
curl -sS http://127.0.0.1:8787/mcp \
  -H "Authorization: Bearer $PLOOS_MCP_BEARER_TOKEN" \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json, text/event-stream' \
  -H 'MCP-Protocol-Version: 2025-03-26' \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

## Security and compatibility

- Binds to loopback by default. Do not expose it to the public internet.
- A static bearer token is **not OAuth**. ChatGPT remote MCP compatibility and client interoperability have **not** been verified.
- Browser-origin requests are rejected rather than implementing a CORS allowlist.
- The endpoint does not implement streaming SSE responses, server-initiated requests, resumability, or OAuth protected-resource metadata.
- A production implementation needs an MCP conformance suite, HTTPS, OAuth authorization and per-user scopes before public deployment.
- No video providers or paid operations are implemented.
