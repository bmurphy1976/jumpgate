# MCP Server

Server-side MCP over Streamable HTTP at `/mcp` on the existing Jumpgate server.

## Configuration

Enabled via `mcp.enabled` in `jumpgate.yaml`. Requires `api.tokens` to be configured (shares the same tokens for auth).

## Tool Naming

`noun_verb` pattern: `bookmark_create`, `category_list`, `icon_list`.

Param naming matches API field names: `bookmark_id`, `category_id`.

## Tools

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

## Auth

Read-only tokens can only call read tools (`category_list`, `category_get`, `bookmark_get`, `icon_list`). Write tools return an error for read-only tokens.

## Architecture

Direct Datasource access (same pattern as admin handlers). No HTTP round-trips.

## Claude Desktop Config

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
