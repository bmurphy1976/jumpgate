# HTTP JSON API

The JSON API lives at `/api/*` on the existing server. Bearer token auth with read-write and read-only access levels.

## Endpoints

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
| GET | /api/bookmarks/search | read | Search bookmarks (`url` exact match, `q` substring) |
| GET | /api/icons | read | List/search MDI icon names |
| GET | /api/openapi.json | read | OpenAPI 3.0 spec |
| GET | /api/docs | read | Swagger UI |

## Auth

Two-layer middleware:

1. `requireAPIToken` — validates `Authorization: Bearer <token>` against configured tokens, stores access level in context. Returns JSON 401 if invalid.
2. `requireWriteAccess` — checks that the token's access level is `read-write`. Returns JSON 403 if read-only.

## Conventions

- All responses are JSON
- Errors: `{"error": "description"}` with appropriate status code
- Plural resource paths (`/api/categories`, `/api/bookmarks`)
- Keywords are `[]string` in JSON — converted to/from space-separated TEXT column internally
- `POST /api/bookmarks` accepts all fields in one call (category_id, name, url, icon, keywords)
- Partial updates: only sent fields change on PUT

## Bookmark Create

```json
{"category_id": 1, "name": "Google", "url": "https://google.com", "icon": "google", "keywords": ["search", "engine"]}
```

Internally: `CreateBookmark(categoryID)` then `UpdateBookmark(id, fields)` then returns the final bookmark.

## Bookmark Search

`GET /api/bookmarks/search` — params: `url` (exact URL match), `q` (substring match on name/URL/keywords). Returns `[]Bookmark` (empty array if none).

## Icons

`GET /api/icons` — params: `q` (substring filter), `limit` (max results, default all), `offset` (pagination start, default 0). Response includes `total` count for pagination.

## Expected Agent Workflow

1. `GET /api/categories` to understand existing structure
2. `GET /api/icons?q=` to find an appropriate icon
3. `POST /api/bookmarks` with category, name, URL, icon, and keywords in one call
