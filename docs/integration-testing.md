# MCP HTTP integration testing

Run from the repository root:

```sh
go test ./...
go test -race ./cmd/video-http
go vet ./...
```

`TestMCPHTTPIntegrationWithOAuth` starts an in-process MCP HTTP server and a TLS test authorization server. It verifies protected-resource discovery, `401` challenges, OAuth token introspection, rejection of invalid tokens and static-token fallback, MCP initialization, tool listing/calling, and notification acknowledgement. The TLS server is a **test fixture only**, not a production authorization server.

## External client compatibility checklist (not yet verified)

- [ ] ChatGPT remote MCP: authorization flow and tool discovery
- [ ] LibreChat: remote MCP connection and tool calls
- [ ] Open WebUI: remote MCP connection and tool calls
- [ ] NextChat: local stdio MCP connection
- [ ] OAuth provider: authorization-code/PKCE, discovery and claim mapping

These tests verify the server's internal HTTP/OAuth flow only; they do **not** establish MCP conformance or compatibility with those clients. Public deployment remains out of scope.
