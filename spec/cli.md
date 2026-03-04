# CLI Client

Separate binary: `jumpgate-cli`. Thin HTTP client wrapping the JSON API.

## Command Pattern

`<singular-resource> <verb> [--named-params]`

All parameters use named flags (no positional args). ID flags name the resource type (`--bookmark-id`, `--category-id`).

## Commands

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
| `keyword add --bookmark-id ID WORD [WORD ...]` | GET + PUT /api/bookmarks/:id |
| `keyword delete --bookmark-id ID WORD [WORD ...]` | GET + PUT /api/bookmarks/:id |
| `keyword set --bookmark-id ID WORD [WORD ...]` | PUT /api/bookmarks/:id |
| `keyword clear --bookmark-id ID` | PUT /api/bookmarks/:id |
| `icon list [--query QUERY] [--limit N] [--offset N]` | GET /api/icons |

## Configuration

```yaml
# jumpgate-cli.yaml
url: http://localhost:8080
token: "my-token-here"
```

Resolution order (last wins): defaults → config file → env vars.

| Source | URL | Token |
|--------|-----|-------|
| Default | `http://localhost:8080` | (none) |
| Config file | `url` | `token` |
| Env var | `JUMPGATE_API_URL` | `JUMPGATE_API_TOKEN` |

`--config` specifies the path to a YAML config file. If not provided, uses defaults and env var overrides.

## Output

JSON to stdout, errors to stderr.
