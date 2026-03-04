# WIP — EARLY DRAFT

> **This plan is a work in progress.** It captures initial requirements discovery and architectural direction only. Details are subject to change as implementation progresses and configuration refactoring lands.

---

# AI Bookmark Auto-Categorization

## Context

Adding bookmarks currently requires manual entry: create a blank bookmark, then fill in name, URL, icon, keywords, and place it in the right category. This feature lets a user paste a URL anywhere on the admin page and have an AI agent automatically suggest the name, category, icon, and keywords — shown in an editable preview dialog before saving.

## Architecture

### New `ai/` package — pluggable provider interface

Mirrors the `Datasource` interface pattern from `storage/db.go`.

**`ai/ai.go`** — Interface, types, shared prompt builder:
- `Provider` interface with single method: `Suggest(ctx, SuggestInput) (Suggestion, error)`
- `SuggestInput` struct: URL, Title, Description, SiteName, Categories (existing names)
- `Suggestion` struct: Name, Category (string name), Icon (semantic term), Keywords
- `buildPrompt(SuggestInput) string` — shared prompt construction. Instructs the AI to prefer existing categories, return concise names, suggest a semantic icon concept, and respond with JSON only

**`ai/metadata.go`** — URL metadata fetcher:
- `FetchMeta(ctx, url) (PageMeta, error)` — HTTP GET, parse HTML for `<title>`, `<meta name="description">`, OG tags
- Reads only first 64KB of response body
- Uses `golang.org/x/net/html` tokenizer (new dependency — standard library extension, lightweight)
- Failure is non-fatal; AI still gets the raw URL

**`ai/anthropic.go`** — Anthropic Claude provider:
- Direct HTTP POST to `https://api.anthropic.com/v1/messages` (no SDK dependency)
- Default model: `claude-sonnet-4-20250514`

**`ai/openai.go`** — OpenAI provider:
- Direct HTTP POST to `https://api.openai.com/v1/chat/completions`
- Default model: `gpt-4o-mini`

**`ai/ollama.go`** — Ollama local provider:
- HTTP POST to `{baseURL}/api/chat`
- Default model: `llama3.2`, default URL: `http://localhost:11434`

All providers use the same shared `buildPrompt()` output, formatted into their respective API structures. No SDKs — raw HTTP calls with `encoding/json` for parsing. Separate HTTP client with 30s timeout for AI calls (vs the shared 10s client).

### Icon fuzzy matching

**Add `BestMatch(term string) string` to `icons/icons.go`**:
- The AI suggests a semantic icon concept (e.g., "github", "newspaper")
- `BestMatch` resolves it against the real MDI icon list: exact match -> prefix match -> contains match -> fallback to `"bookmark-outline"`
- Avoids sending 7000+ icon names to the AI

### Handler changes

**Modify `handlers/admin.go`**:
- Add `ai ai.Provider` field to `AdminHandler` (nil when AI not configured)
- Update `SetupAdminRoutes` to accept `ai.Provider` parameter
- New route: `POST /admin/bookmarks/suggest` -> `suggestBookmark` handler
- New route: `POST /admin/bookmarks/create-suggested` -> `createSuggestedBookmark` handler

`suggestBookmark` flow:
1. Receive pasted URL from form data
2. Fetch page metadata via `ai.FetchMeta`
3. Get existing category names from datasource
4. Call `ai.Suggest` with URL + metadata + categories
5. Resolve suggested icon via `icons.BestMatch`
6. Resolve suggested category to existing ID or mark as new
7. Render `SuggestPreview` templ component as HTML fragment

`createSuggestedBookmark` flow:
1. Create new category if suggested (via `ds.CreateCategory`)
2. Create bookmark in the target category (via `ds.CreateBookmark` + `ds.UpdateBookmark`)
3. Return `HX-Redirect: /admin` to reload the page

**Modify `handlers/server.go`**:
- Update `NewServer` signature to accept `ai.Provider`
- Pass through to `SetupAdminRoutes`

### Template changes

**New file: `views/suggest.templ`** — `SuggestPreview` component:
- Modal overlay with editable form: name, icon (with icon picker), category dropdown, keywords
- Category dropdown includes all existing categories; if AI suggested a new one, it appears as `"+ NewName"` option
- Form submits via `hx-post="/admin/bookmarks/create-suggested"`
- Cancel button calls `hideSuggestPreview()` JS function

**Modify `views/layout.templ`**:
- Add `<div id="suggestPreviewContainer"></div>` as HTMX swap target
- Add `data-ai-enabled` attribute to `<body>` when AI is configured

**Modify `views/viewdata.go`**:
- Add `AIEnabled bool` to `AdminLayoutData`

### JavaScript changes

**Modify `static/js/admin.js`**:
- Global paste listener on `document`:
  - Bail if `data-ai-enabled` not on body
  - Bail if paste target is an input/textarea/select/contenteditable
  - Validate pasted text is an HTTP(S) URL via `new URL()`
  - Show loading indicator in `#suggestPreviewContainer`
  - `htmx.ajax('POST', '/admin/bookmarks/suggest', ...)` targeting the container
- `hideSuggestPreview()` — clears the container
- Escape key closes the preview (add check in existing keydown handler)

### CSS changes

**Modify `static/css/admin.css`**:
- `.suggest-preview` — fixed overlay (same pattern as `.confirm-overlay`)
- `.suggest-preview-dialog` — centered card with form grid
- `.suggest-loading` — loading indicator with spinning icon
- Responsive: single-column form grid on mobile

### Wiring

**Modify `cmd/jumpgate/main.go`**:
- Import `ai` package
- Read env vars: `AI_PROVIDER`, `AI_API_KEY`, `AI_MODEL`, `AI_BASE_URL`
- Instantiate the matching provider (or nil if `AI_PROVIDER` unset)
- Pass to `NewServer`

### Environment variables

| Variable | Description | Required |
|----------|-------------|----------|
| `AI_PROVIDER` | `anthropic`, `openai`, or `ollama` | No (feature disabled if unset) |
| `AI_API_KEY` | API key for Anthropic or OpenAI | Yes for anthropic/openai |
| `AI_MODEL` | Model override (defaults per provider) | No |
| `AI_BASE_URL` | Ollama server URL | No (default `http://localhost:11434`) |

## Files to create
- `ai/ai.go` — Provider interface, types, prompt builder
- `ai/metadata.go` — URL metadata fetcher
- `ai/anthropic.go` — Anthropic provider
- `ai/openai.go` — OpenAI provider
- `ai/ollama.go` — Ollama provider
- `views/suggest.templ` — Preview dialog component

## Files to modify
- `handlers/admin.go` — AI field, new handlers, route registration
- `handlers/server.go` — Updated NewServer signature
- `cmd/jumpgate/main.go` — Provider wiring from env vars
- `views/layout.templ` — Preview container div, data-ai-enabled attribute
- `views/viewdata.go` — AIEnabled field on AdminLayoutData
- `icons/icons.go` — BestMatch method
- `static/js/admin.js` — Paste listener, preview management
- `static/css/admin.css` — Preview dialog styles

## New dependency
- `golang.org/x/net` — HTML tokenizer for metadata extraction

## Implementation order
1. `ai/ai.go` — interface, types, prompt
2. `ai/metadata.go` — URL metadata fetching
3. `ai/anthropic.go` — first provider (test the full flow)
4. `icons/icons.go` — BestMatch method
5. `views/viewdata.go` + `views/suggest.templ` — preview template
6. `handlers/admin.go` + `handlers/server.go` — handlers and routes
7. `cmd/jumpgate/main.go` — wiring
8. `static/js/admin.js` — paste listener
9. `static/css/admin.css` — preview styles
10. `views/layout.templ` — container div and data attribute
11. `ai/openai.go` + `ai/ollama.go` — remaining providers
12. `make generate && make test`

## Verification
1. `make generate` — regenerate templ templates
2. `make build` — ensure compilation succeeds
3. `make test` — run existing tests (should not break)
4. Manual testing: `AI_PROVIDER=anthropic AI_API_KEY=... make debug`
   - Open admin page
   - Paste a URL (e.g., `https://github.com`) outside of any input field
   - Verify loading indicator appears
   - Verify preview dialog shows with populated fields
   - Edit a field, click Save
   - Verify bookmark created in the correct category
   - Verify page reloads showing the new bookmark
5. Test without AI configured: `make debug` — paste should be ignored (no `data-ai-enabled`)
