# OAuth integration groundwork

**Status: discovery metadata only, not OAuth authentication.** The MCP server still authenticates using one static `PLOOS_MCP_BEARER_TOKEN`. Do **not** expose it publicly or use this implementation as a ChatGPT OAuth connector yet.

Optional environment variables:

- `PLOOS_MCP_PUBLIC_URL=https://video.example` — canonical HTTPS origin used in discovery metadata.
- `PLOOS_OAUTH_ISSUER=https://auth.example` — intended authorization server issuer.

When both values are valid HTTPS origins, the server publishes `GET /.well-known/oauth-protected-resource` and includes a `resource_metadata` hint in unauthorized challenges. This advertises discovery information but does **not** verify tokens issued by that authorization server.

## Required before production

1. Verify access tokens against trusted issuer/JWKS or introspection, including signature, issuer, audience/resource, expiry and scopes.
2. Implement authorization code + PKCE through a trusted authorization server, with correct redirect handling.
3. Enforce per-user authorization and consent, and require explicit confirmation before paid video jobs.
4. Run end-to-end MCP interoperability tests against ChatGPT, LibreChat and Open WebUI.
5. Deploy behind HTTPS with rate limits, audit logs and key rotation.

Do not use a self-supplied JWT claim without cryptographic verification.
