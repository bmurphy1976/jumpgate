# Configuration

## Server Config (`jumpgate.yaml`)

Single `--config` flag. If not specified, uses built-in defaults.

```yaml
db: data/jumpgate.db
addr: :8080
auth: true
slow: 0

demo:
  enabled: false
  source: bookmarks.yaml.example

api:
  swagger: true
  tokens:
    read-write:
      - "rw-secret-here"
    read-only:
      - "ro-secret-here"

mcp:
  enabled: false
```

| Field | Type | Default | Purpose |
|-------|------|---------|---------|
| `db` | string | `data/jumpgate.db` | SQLite database path |
| `addr` | string | `:8080` | Listen address |
| `auth` | bool | `true` | Enable admin auth check |
| `slow` | int | `0` | Artificial delay in seconds (0 = disabled) |
| `demo.enabled` | bool | `false` | Enable demo mode |
| `demo.source` | string | required if enabled | Path to YAML data file |
| `api.swagger` | bool | `false` | Serve Swagger UI at `/api/docs` |
| `api.tokens.read-write` | []string | `[]` | Tokens with full access |
| `api.tokens.read-only` | []string | `[]` | Tokens with read-only access |
| `mcp.enabled` | bool | `false` | Serve MCP endpoint at `/mcp` |

API is enabled when tokens are configured or swagger is enabled. When no tokens are configured, all API endpoints are unauthenticated. Docs (`/api/docs`), OpenAPI spec (`/api/openapi.json`, `/api/openapi.yaml`), and the API index (`/api`) are always public. MCP requires `mcp.enabled: true`; if no tokens are configured, MCP is unauthenticated.

## Conventions

- Positive names for booleans (`auth: true`, `enabled: true`)
- `slow` is an integer (seconds), 0 = disabled
- Demo mode: `demo.source` for data file path
- When `demo.enabled` is true, `db` is ignored and `auth` is forced to false

## CLI Config (`jumpgate-cli.yaml`)

```yaml
url: http://localhost:8080
token: "my-token-here"
```

Single `--config` flag. If not specified, uses defaults and env var overrides.
