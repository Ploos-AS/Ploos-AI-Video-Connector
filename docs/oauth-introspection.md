# OAuth token introspection (experimental)

PR #5 adds optional **RFC 7662 introspection** for incoming MCP bearer tokens. The connector is a resource server, **not an OAuth authorization server**.

Configure an existing trusted OAuth authorization server that provides HTTPS token introspection:

```sh
export PLOOS_MCP_PUBLIC_URL=https://video.example
export PLOOS_OAUTH_ISSUER=https://auth.example
export PLOOS_OAUTH_INTROSPECTION_URL=https://auth.example/oauth2/introspect
export PLOOS_OAUTH_CLIENT_ID=connector-resource-server
export PLOOS_OAUTH_CLIENT_SECRET='read-from-secret-manager'
export PLOOS_OAUTH_REQUIRED_SCOPE=mcp:read
```

The endpoint validates `active`, `iss`, `aud`, `exp`, and required `scope` on every request. Introspection uses HTTP Basic client authentication and rejects redirects. If introspection is configured, the old static bearer token is **not accepted as fallback**. Missing or malformed configuration fails startup.

**Provider compatibility:** The introspection response must include issuer, audience, expiry and scopes. Not every authorization server returns all these fields by default; configure its claims mapping accordingly.

**Still required before production:** real authorization server configuration, OAuth authorization-code/PKCE integration, per-user identity and scope policy, ChatGPT/LibreChat/Open WebUI interoperability tests, HTTPS reverse proxy and abuse controls. Do not expose the service publicly yet. No video provider or billing functionality is implemented.
