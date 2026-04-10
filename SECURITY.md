# Security

Conventions and threat model for Jumpgate.

## Private by Default

`default_private=true` out of the box. All new categories and bookmarks inherit this setting. Admin access requires authentication.

## Admin Auth

Delegated to reverse proxy via `X-Authorized-User` (or `X-User`, `X-Remote-User`) header. Jumpgate does not implement its own user accounts — the reverse proxy is responsible for authentication.

## API Auth

Bearer tokens configured in `jumpgate.yaml` under `api.tokens`. Two access levels:

- **read-write** — full CRUD access
- **read-only** — GET endpoints only, write attempts return 403

Tokens are stored in plaintext in the config file. Users are responsible for securing the config file (file permissions, not checking into version control).

## MCP Auth

The MCP server at `/mcp` shares `api.tokens` for authentication. Same read-write / read-only access levels apply.

## Demo Mode Session Isolation

Sessions are keyed by (session cookie + client IP) for browser requests, (Bearer token + client IP) for API/MCP requests. IP binding prevents session/token hijacking — a stolen cookie or leaked token from a different IP gets a separate database, not access to the original session's data.

- **Session cap**: demo mode enforces a maximum number of concurrent sessions to prevent resource exhaustion
- **Session TTL**: demo sessions expire after a configurable duration
- **Allowed proxies**: when `demo.allowed_proxies` is set, demo mode uses Echo's built-in `X-Forwarded-For` extraction with those IPs and CIDRs added as trusted ranges; `demo.disable_proxy_headers: true` disables forwarded-header handling entirely
- **Header safety**: upstream proxies must overwrite `X-Forwarded-For`, not pass through user-supplied values

## Delegated Concerns

- **Rate limiting** — not built in, delegated to reverse proxy
- **TLS** — not built in, delegated to reverse proxy
- **CORS** — not configured, delegated to reverse proxy if needed
