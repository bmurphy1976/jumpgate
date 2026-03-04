# Plan: AI Agent Integration for Jumpgate

## Context

Jumpgate needs a machine-readable interface so AI agents (n8n workflows, Claude tool use, scripts) can manage bookmarks programmatically. The key use case: given a URL, an external agent categorizes it, picks an icon, and adds it to the right category. The "smart" logic lives in the agent — Jumpgate provides CRUD primitives and icon search.

Four layers: HTTP JSON API (foundation), OpenAPI + Swagger (documentation/discovery), CLI client (separate binary), MCP server (server-side Streamable HTTP).

## Server Config File

Replace all CLI flags with a YAML config file. The only CLI flag is `--config`.

```
jumpgate server                      # uses jumpgate.yaml from CWD
jumpgate server --config path.yaml   # explicit config path
```

If no config file found and no `--config` specified, uses defaults (db: `data/jumpgate.db`, addr: `:8080`, no API).

### Full config structure (`jumpgate.yaml`)

```yaml
# Server
db: data/jumpgate.db
addr: :8080
auth: true
slow: 0

# Demo mode — in-memory per-session databases loaded from a data file
demo:
  enabled: false
  source: bookmarks.yaml.example

# JSON API for agents, scripts, and automation
api:
  swagger: true
  tokens:
    read-write:
      - "rw-secret-here"
    read-only:
      - "ro-secret-here"

# MCP server (Streamable HTTP at /mcp, uses api.tokens for auth)
mcp:
  enabled: false
```

| Field | Type | Default | Purpose |
|-------|------|---------|---------|
| `db` | string | `data/jumpgate.db` | SQLite database path |
| `addr` | string | `:8080` | Listen address |
| `auth` | bool | `true` | Enable admin auth check |
| `slow` | int | `0` | Artificial delay in seconds added to non-static requests (0 = disabled) |
| `demo.enabled` | bool | `false` | Enable demo mode (in-memory, per-session) |
| `demo.source` | string | required if enabled | Path to YAML data file |
| `api.swagger` | bool | `false` | Serve Swagger UI at `/api/docs` |
| `api.tokens.read-write` | []string | `[]` | Tokens with full access |
| `api.tokens.read-only` | []string | `[]` | Tokens with read-only access |
| `mcp.enabled` | bool | `false` | Serve MCP endpoint at `/mcp` |

API is disabled when both token lists are empty. MCP requires API tokens to be configured (shares `api.tokens` for auth).

When `demo.enabled` is true, `db` is ignored (in-memory databases) and `auth` is forced to false (existing behavior). In demo mode, browser requests get per-session databases (keyed by session cookie + client IP), API/MCP requests get per-token databases (keyed by Bearer token + client IP). Each is independently seeded from `demo.source`. See `SECURITY.md` for rationale on the IP binding.

### Go types

```go
type ServerConfig struct {
    DB     string     `yaml:"db"`
    Addr   string     `yaml:"addr"`
    Auth   *bool      `yaml:"auth"`  // nil = true (default)
    Slow   int        `yaml:"slow"`
    Demo   DemoConfig `yaml:"demo"`
    API    APIConfig  `yaml:"api"`
    MCP    MCPConfig  `yaml:"mcp"`
}

type DemoConfig struct {
    Enabled bool   `yaml:"enabled"`
    Source  string `yaml:"source"`
}

type APIConfig struct {
    Swagger bool      `yaml:"swagger"`
    Tokens  APITokens `yaml:"tokens"`
}

type APITokens struct {
    ReadWrite []string `yaml:"read-write"`
    ReadOnly  []string `yaml:"read-only"`
}

type MCPConfig struct {
    Enabled bool `yaml:"enabled"`
}
```

### Files

**`config/server.go`** (new) — `ServerConfig`, `DemoConfig`, `APIConfig`, `APITokens`, `MCPConfig` types + `LoadServerConfig(path)` function + `ApplyDefaults()` method. Uses existing `gopkg.in/yaml.v3` dependency. `Auth` uses `*bool` internally so we can distinguish "not set" (nil → default true) from "explicitly set to false". `ApplyDefaults()` is called after unmarshal to fill in zero-value defaults (`db`, `addr`, `auth`).

**`cmd/jumpgate/main.go`** — Refactor `runServer()` to load config file. Replace all server flags with single `--config` flag. Apply defaults for missing fields.

## Layer 1: HTTP JSON API

`/api/*` routes on the existing Echo server. Bearer token auth with multiple tokens supporting read-write and read-only access.

### Endpoints

| Method | Path | Access | Purpose |
|--------|------|--------|---------|
| GET | /api/categories | read | List all categories with bookmarks |
| POST | /api/categories | write | Create category `{"name":"Work"}` |
| GET | /api/categories/:id | read | Get single category with bookmarks |
| PUT | /api/categories/:id | write | Update category `{"name":"New Name"}` |
| DELETE | /api/categories/:id | write | Delete category |
| POST | /api/bookmarks | write | Create bookmark with all fields in one call |
| GET | /api/bookmarks/:id | read | Get single bookmark |
| PUT | /api/bookmarks/:id | write | Update bookmark (partial — only sent fields change) |
| DELETE | /api/bookmarks/:id | write | Delete bookmark |
| POST | /api/bookmarks/:id/move | write | Move to different category `{"category_id":2,"position":0}` |
| GET | /api/bookmarks/search | read | Search bookmarks. Params: `url` (exact URL match), `q` (substring match on name/URL/keywords). Returns `[]Bookmark` (empty array if none). |
| GET | /api/icons | read | List MDI icon names. Params: `q` (substring filter), `limit` (max results, default all), `offset` (pagination start, default 0) |
| GET | /api/openapi.json | read | OpenAPI 3.0 spec (machine-readable API reference) |
| GET | /api/docs | read | Swagger UI (interactive API explorer, loads from CDN) |

`POST /api/bookmarks` accepts all fields in one call:
```json
{"category_id": 1, "name": "Google", "url": "https://google.com", "icon": "google", "keywords": ["search", "engine"]}
```
Internally: `CreateBookmark(categoryID)` then `UpdateBookmark(id, fields)` then returns the final bookmark. Keywords are `[]string` everywhere — API, model, MCP. The storage layer joins with spaces on write to the TEXT column (`strings.Join`) and splits on read (`strings.Fields`). `BookmarkUpdate.Keywords` changes from `*string` to `*[]string` so no conversion is needed in handlers.

All errors: `{"error": "description"}` with appropriate status code.

### Auth middleware

Two-layer approach:
1. `requireAPIToken(tokens)` — validates Bearer token against configured tokens, stores access level in context. Returns JSON 401 if invalid.
2. `requireWriteAccess` — checks that the token's access level is `read-write`. Returns JSON 403 if read-only.

Route registration:
```go
apiRO := api.Group("", /* no extra middleware */)
apiRW := api.Group("", requireWriteAccess)

apiRO.GET("/categories", h.listCategories)
apiRW.POST("/categories", h.createCategory)
// etc.
```

### Files

**`model/model.go`** — Add `json:"snake_case"` tags to `Settings`, `Category`, `Bookmark` structs. Change `BookmarkUpdate.Keywords` from `*string` to `*[]string`.

**`handlers/middleware.go`** — Add `requireAPIToken(tokens config.APITokens) echo.MiddlewareFunc` and `requireWriteAccess` middleware.

**`handlers/api.go`** (new) — `APIHandler` struct, `SetupAPIRoutes()`, handler methods, OpenAPI spec + Swagger UI handlers, helpers (`apiError`, `apiCategoryID`, `apiBookmarkID`).

**`handlers/server.go`** — Update `NewServer` to accept `config.APIConfig` instead of `apiKey string`. Conditionally call `SetupAPIRoutes` when tokens are configured.

**`handlers/api_test.go`** (new) — Reuses `setupTestDB` and `serve` from `admin_test.go` (same package). Covers: both token types, read-only rejection on write endpoints, all CRUD operations, icon search, invalid input.

## Layer 2: OpenAPI + Swagger

Built into the binary, served at `GET /api/openapi.json`. Defined as Go data structures alongside route definitions in `handlers/api.go`. No separate file, no drift.

Swagger UI at `GET /api/docs` — HTML page loading Swagger UI JS from CDN, pointing at `/api/openapi.json`. Controlled by `api.swagger` in config. Both endpoints behind the same token auth.

## Layer 3: CLI Client (separate binary)

Separate binary: `jumpgate-cli`. Lives in `cmd/jumpgate-cli/`.

### Configuration

```
jumpgate-cli [--config PATH] <command>
```

Config file (`jumpgate-cli.yaml`):
```yaml
url: http://localhost:8080
token: "my-token-here"
```

Resolution order (last wins): defaults → config file → env vars.

| Source | URL | Token |
|--------|-----|-------|
| Default | `http://localhost:8080` | (none) |
| Config file | `url` | `token` |
| Env var | `JUMPGATE_API_URL` | `JUMPGATE_API_TOKEN` |

`--config` defaults to `jumpgate-cli.yaml` in the working directory (ignored if not found).

| Command | Maps to |
|---------|---------|
| `category list` | GET /api/categories |
| `category get --category-id ID` | GET /api/categories/:id |
| `category create --name NAME` | POST /api/categories |
| `category update --category-id ID [--name NAME]` | PUT /api/categories/:id |
| `category delete --category-id ID` | DELETE /api/categories/:id |
| `bookmark create --category-id ID [--name --url --icon --keyword ...]` | POST /api/bookmarks |
| `bookmark get --bookmark-id ID` | GET /api/bookmarks/:id |
| `bookmark update --bookmark-id ID [--name --url --mobile-url --icon]` | PUT /api/bookmarks/:id |
| `bookmark delete --bookmark-id ID` | DELETE /api/bookmarks/:id |
| `bookmark move --bookmark-id ID --category-id ID [--position N]` | POST /api/bookmarks/:id/move |
| `bookmark search [--url URL] [--query QUERY]` | GET /api/bookmarks/search |
| `keyword list --bookmark-id ID` | GET /api/bookmarks/:id (shows keywords) |
| `keyword add --bookmark-id ID WORD [WORD ...]` | GET + PUT /api/bookmarks/:id (read-modify-write) |
| `keyword delete --bookmark-id ID WORD [WORD ...]` | GET + PUT /api/bookmarks/:id (read-modify-write) |
| `keyword set --bookmark-id ID WORD [WORD ...]` | PUT /api/bookmarks/:id (replace all) |
| `keyword clear --bookmark-id ID` | PUT /api/bookmarks/:id (set to []) |
| `icon list [--query QUERY] [--limit N] [--offset N]` | GET /api/icons |

JSON to stdout, errors to stderr.

### Files

**`cmd/jumpgate-cli/main.go`** (new) — Entry point, flag parsing, command dispatch.

**`cmd/jumpgate-cli/client.go`** (new) — Shared HTTP client with Bearer auth, request/response helpers.

**`cmd/jumpgate-cli/commands.go`** (new) — All command implementations.

**`Makefile`** — Add `jumpgate-cli` build target.

## Layer 4: MCP Server

Server-side MCP over Streamable HTTP, served at `/mcp` on the existing Jumpgate server. Uses the [official Go MCP SDK](https://github.com/modelcontextprotocol/go-sdk). Handlers access the Datasource directly (same pattern as admin handlers).

Enabled via `mcp.enabled` in config. Uses the same token auth as the API (`api.tokens`). Respects read-only vs read-write: read-only tokens can only call read tools (`category_list`, `category_get`, `bookmark_get`, `icon_list`). Write tools return an error for read-only tokens.

### Claude Desktop config

```json
{
  "mcpServers": {
    "jumpgate": {
      "url": "https://my-jumpgate.example.com/mcp",
      "headers": {
        "Authorization": "Bearer my-rw-token"
      }
    }
  }
}
```

No local binary needed.

### MCP Tools

| Tool | Params | Datasource calls |
|------|--------|-----------------|
| `bookmark_create` | `category_id`, `name?`, `url?`, `icon?`, `keywords?` | `CreateBookmark()` + `UpdateBookmark()` |
| `bookmark_delete` | `bookmark_id` | `DeleteBookmark()` |
| `bookmark_get` | `bookmark_id` | `GetBookmark(id)` |
| `bookmark_search` | `url?`, `query?` | `SearchBookmarks()` |
| `bookmark_move` | `bookmark_id`, `category_id`, `position?` | `MoveBookmark()` |
| `bookmark_update` | `bookmark_id`, `name?`, `url?`, `mobile_url?`, `icon?`, `keywords?` | `UpdateBookmark()` |
| `category_create` | `name` | `CreateCategory()` + `UpdateCategory()` |
| `category_delete` | `category_id` | `DeleteCategory()` |
| `category_get` | `category_id` | `GetCategory(id)` |
| `category_list` | (none) | `GetCategories()` |
| `category_update` | `category_id`, `name` | `UpdateCategory()` |
| `icon_list` | `query?`, `limit?`, `offset?` | `icons.Icons` / `icons.Search()` |

### Files

**`handlers/mcp.go`** (new) — `MCPHandler` struct with `ds DSResolver`, `icons *icons.Loader`. `SetupMCPRoutes()` registers `/mcp` endpoint. All tool definitions and handlers.

**`handlers/server.go`** — Conditionally call `SetupMCPRoutes` when `mcp.enabled` is true.

### Architecture

```
Claude Desktop/Code     jumpgate-cli        n8n / scripts
        │ (HTTP)              │ (HTTP)           │ (HTTP)
        ▼                     ▼                  ▼
   Jumpgate /mcp         Jumpgate /api/* ◀───────┘
   (Streamable HTTP)     (token auth)
        │                     │
        ▼                     ▼
   Datasource (SQLite)   Datasource (SQLite)
```

MCP and API are both server-side routes on the same binary, both using direct Datasource access.

## Documentation Updates

**`PHILOSOPHY.md`** — Two changes:

1. Add consistency principle under Design Principles:
   > **Prefer consistency** — When two approaches work equally well, choose the one that matches existing patterns. Consistency makes systems predictable and easier to reason about.

2. Update Non-Goals: remove "JSON API for external consumers (HTML-first, HTMX fragments for interactivity)" — the JSON API is now a feature, not a non-goal.

## Spec Files

Design decisions captured in `spec/` for durable reference. These are the source of truth for conventions and contracts — implementation must conform to them.

**`spec/api.md`** — HTTP JSON API contract
- Endpoint table (method, path, access level, purpose)
- Auth model: Bearer tokens, two access levels (read-write, read-only), two-layer middleware
- Request/response conventions: JSON bodies, `{"error": "..."}` for errors, status codes
- Naming: plural resource paths (`/api/categories`, `/api/bookmarks`)
- Keywords: JSON arrays in API, converted to/from space-separated strings internally
- `POST /api/bookmarks` accepts all fields in one call (no multi-step creation)
- Partial updates: only sent fields change on PUT
- Icons endpoint: `GET /api/icons` — single endpoint, three optional params: `q` (substring filter), `limit` (max results), `offset` (pagination start). No `q` = all icons. Response includes `total` count for pagination. Agent chooses strategy — full list for best selection, search for quick lookup. Server-side auto-selection is a future concern.
- Expected agent workflow for "add this URL": (1) `GET /api/categories` to understand existing structure, (2) `GET /api/icons?q=` to find an appropriate icon, (3) `POST /api/bookmarks` with category, name, URL, icon, and keywords in one call

**`spec/cli.md`** — CLI client conventions
- Command pattern: `<singular-resource> <verb> [--named-params]`
- All parameters use named flags (no positional args)
- ID flags name the resource type (`--bookmark-id`, `--category-id`)
- Verbs: `list`, `get`, `create`, `update`, `delete`, `move`, `search`
- `keyword` subcommand for granular keyword management (`add`, `delete`, `set`, `clear`, `list`)
- `--keyword` on `bookmark create` only (no ambiguity on creation)
- Config: `jumpgate-cli.yaml` + env vars (`JUMPGATE_API_URL`, `JUMPGATE_API_TOKEN`), no CLI flags for connection
- JSON to stdout, errors to stderr

**`spec/mcp.md`** — MCP server conventions
- Server-side Streamable HTTP at `/mcp`, not a local binary
- Tool naming: `noun_verb` pattern (`bookmark_create`, `category_list`, `icon_search`)
- Param naming matches API field names (`bookmark_id`, `category_id`)
- Direct Datasource access (same pattern as admin handlers)
- Shares `api.tokens` for auth
- Enabled via `mcp.enabled` in config

**`spec/config.md`** — Configuration conventions
- Server config: `jumpgate.yaml` (single `--config` flag, defaults from CWD)
- CLI config: `jumpgate-cli.yaml` (single `--config` flag, defaults from CWD)
- Full `jumpgate.yaml` schema with field table
- Positive names for booleans (`auth: true`, `enabled: true`, not `no_auth`, `disabled`)
- `slow` is an integer (seconds), 0 = disabled
- API disabled when no tokens configured
- MCP requires tokens to be configured
- Demo mode: `demo.source` for data file path

**`SECURITY.md`** (new, project root) — Security conventions and threat model
- Private by default: `private=true` out of the box, auth required on admin
- Admin auth: delegated to reverse proxy via `X-Authorized-User` header, not built-in
- API auth: Bearer tokens with read-write and read-only access levels, configured in `jumpgate.yaml`
- Demo mode session isolation: keyed by (session cookie + client IP) for browsers, (Bearer token + client IP) for API/MCP. IP binding prevents session/token hijacking — a stolen cookie or leaked token from a different IP gets a separate database, not access to the original session's data
- Session cap: demo mode enforces a maximum number of concurrent sessions to prevent resource exhaustion
- Session TTL: demo sessions expire after a configurable duration
- Tokens are stored in plaintext in the config file — users are responsible for securing the config file (file permissions, not checking into version control)
- No rate limiting built in — delegated to reverse proxy

## Implementation Order

1. `spec/` — Create spec files (`api.md`, `cli.md`, `mcp.md`, `config.md`)
2. `SECURITY.md` — Security conventions and threat model
3. `PHILOSOPHY.md` — Add consistency principle, update non-goals
4. Rename `config.yaml.example` → `bookmarks.yaml.example`, update all references (`README.md`, `Makefile`, `Dockerfile.demo`). Create `jumpgate.yaml.example` with commented example of all config fields.
5. `config/server.go` — Server config types + loader (with `ApplyDefaults()` for zero-value handling)
6. `cmd/jumpgate/main.go` — Refactor to load config file (`--config` only)
7. `model/model.go` — JSON tags + change `BookmarkUpdate.Keywords` from `*string` to `*[]string`
8. `storage/db.go` — Add `SearchBookmarks(url, query string) ([]model.Bookmark, error)` to `Datasource` interface
9. `storage/sqlite.go` — Implement `SearchBookmarks` (exact URL match and/or substring match on name/URL/keywords). Update `UpdateBookmark` to join `*[]string` with spaces on write.
10. `handlers/admin.go` — Update `formStr(c, "keywords")` caller to split form value into `*[]string`
11. `handlers/middleware.go` — `requireAPIToken` + `requireWriteAccess` middleware
12. `handlers/api.go` — All API handlers + OpenAPI spec + Swagger UI
13. `handlers/server.go` — Wire API/MCP config + `SetupAPIRoutes` + `SetupMCPRoutes`. Update slow middleware to use `int` duration.
14. `handlers/api_test.go` — API tests (auth, permissions, CRUD, errors)
15. `cmd/jumpgate-cli/` — CLI client binary
16. `handlers/mcp.go` — MCP server (Streamable HTTP, direct Datasource access)
17. `Makefile` — Build both binaries
18. `Dockerfile` — Build both binaries

## New Dependencies

- `github.com/modelcontextprotocol/go-sdk` — Official MCP SDK (Apache 2.0/MIT, maintained by MCP team + Google)

## Verification

1. `make test` — all existing + new tests pass
2. `make build` — both binaries compile
3. HTTP smoke test:
   ```bash
   # Create jumpgate.yaml with a test token, then:
   bin/jumpgate server
   curl -H "Authorization: Bearer <rw-token>" localhost:8080/api/categories
   curl -H "Authorization: Bearer <ro-token>" -X POST localhost:8080/api/categories
   # → should get 403
   ```
4. CLI smoke test:
   ```bash
   # Create jumpgate-cli.yaml with url + token, then:
   bin/jumpgate-cli category list
   bin/jumpgate-cli bookmark create \
     --category-id 1 --name "Test" --url "https://example.com" --icon "home"
   ```
5. MCP smoke test:
   ```bash
   # With mcp.enabled: true in jumpgate.yaml
   curl -X POST http://localhost:8080/mcp \
     -H "Authorization: Bearer <rw-token>" \
     -H "Content-Type: application/json" \
     -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
   ```
6. Swagger UI: open `http://localhost:8080/api/docs` in browser with token
