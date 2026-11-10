# Authelia integration (experimental)

Authelia is the **first candidate** OAuth 2.0 / OpenID Connect authorization server. It is an external service, not a dependency of the Go connector.

## Architecture

- Authelia handles login, user consent, authorization code + PKCE and token issuance.
- The MCP connector is an OAuth **resource server** and validates bearer tokens using Authelia's RFC 7662 introspection endpoint.
- ChatGPT, LibreChat and Open WebUI are OAuth **clients**. Each needs a client registration and exact redirect URI(s). Do not guess or reuse callback URLs.
- NextChat's local stdio path does not require remote OAuth.

## Authelia endpoints

For an issuer at `https://auth.example.com`:

- Discovery: `https://auth.example.com/.well-known/openid-configuration`
- Authorization: `https://auth.example.com/api/oidc/authorization`
- Token: `https://auth.example.com/api/oidc/token`
- Introspection: `https://auth.example.com/api/oidc/introspection`

Confirm these using the live discovery document before deployment.

## Connector environment (placeholders, not deploy-ready)

```sh
PLOOS_MCP_PUBLIC_URL=https://video.example.com
PLOOS_OAUTH_ISSUER=https://auth.example.com
PLOOS_OAUTH_INTROSPECTION_URL=https://auth.example.com/api/oidc/introspection
PLOOS_OAUTH_CLIENT_ID=video-connector-resource
PLOOS_OAUTH_CLIENT_SECRET=<load-from-secret-manager>
PLOOS_OAUTH_REQUIRED_SCOPE=mcp:read
```

The connector requires introspection JSON containing `active`, `iss`, `aud`, `exp` and `scope`. **Verify that Authelia returns these exact fields for the selected token format and client configuration**. If not, adjust claim mapping or the connector's verified validation logic; do not weaken checks or assume compatibility.

## Authelia configuration checklist

1. Deploy Authelia privately first with persistent storage, strong secrets, HTTPS via Caddy, and MFA.
2. Configure an OIDC issuer and its signing key, and verify discovery.
3. Register separate clients with exact redirect URIs; require Authorization Code and PKCE S256.
4. Configure a resource audience matching the connector public URL and an appropriate access scope.
5. Provision a **confidential introspection client** with minimal permissions; store its secret outside git.
6. Confirm Authelia introspection returns the expected issuer, audience, expiry and scope.
7. Run `go test ./...` and `go test -race ./cmd/video-http`.
8. Perform a manual authorization-code/PKCE login and a negative test with a wrong audience and expired token.
9. Only then test remote MCP connections from individual clients.

## Current blockers

- Authelia configuration and live issuer have not been deployed.
- The Go connector's OAuth support is not yet verified against Authelia.
- OAuth client registration/redirect URLs and MCP client interoperability are untested.
- No public deployment, paid calls or video generation should be enabled yet.

Official documentation:
- https://www.authelia.com/integration/openid-connect/introduction/
- https://www.authelia.com/configuration/identity-providers/openid-connect/clients/
