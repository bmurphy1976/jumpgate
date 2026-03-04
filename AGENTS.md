# Agents Guide

This is the primary reference for any AI agent working on this codebase. Read this before making changes.

## Working Principles

1. **Do EXACTLY what is asked — nothing more**
   - Don't add "best practices" unless requested
   - Don't add features "for later" or "just in case"
   - Don't make assumptions about what the user "probably wants"
   - The user is capable of adding things when they're actually needed

2. **Think critically before implementing**
   - If uncertain about an approach, search the web first — don't guess
   - Be honest about limitations — if something won't work, say so immediately
   - Test assumptions before claiming something will work
   - Don't add dependencies, features, or concepts unless the user requests them

3. **Simplicity over complexity**
   - Favor straightforward code over clever abstractions
   - If a more complex approach has strong justification, present the trade-offs and let the user decide
   - Remove unnecessary code and files proactively
   - Question whether features are truly needed
   - **Simple solutions first** — Don't iterate through multiple complex approaches

4. **Listen to explicit constraints**
   - When user says "don't use X", NEVER use X again in that task
   - When user says "only do Y", don't add Z "just in case"
   - When user asks "why", provide explanation — DON'T make changes unless explicitly asked
   - Read the full request before responding — don't act on partial understanding

5. **Be direct and honest**
   - If you don't know something, search for it or admit it
   - Don't promise features that have limitations you're aware of
   - Provide real options, not theoretical "best practices"

## Common Pitfalls

- Don't add preemptive features (favicon link tags, extra error handling, etc.)
- Don't add JavaScript beyond the existing policy (see [ARCHITECTURE.md](ARCHITECTURE.md) for when JS is allowed)
- Don't add "best practice" code that isn't explicitly requested
- Don't make changes to code you haven't read first
- Don't assume local hosting of fonts/icons is better — CDN is intentional
- Don't add fallback/alternative mechanisms — implement what was asked, not "just in case" alternatives
- Don't bypass HTMX with raw `fetch`/`XMLHttpRequest` — if `htmx.ajax()` doesn't seem to support what you need, check the docs (e.g. `source`, `handler`, event hooks) before reaching for vanilla JS alternatives
- Don't make changes when user asks "why" — explain first, only change if explicitly requested
- Don't keep trying rejected approaches — if an approach is declined, move on
- Don't hardcode counts in docs (e.g., "19 themes", "4 tables") — they go stale. Describe what something is, not how many there are
- Don't put implementation details in user-facing docs — describe features by what they do, not how (no library names, storage mechanisms, or API providers in the README)

## Project Overview

Jumpgate is a self-hosted bookmark dashboard. Single Go binary, pluggable storage via the `Datasource` interface, color themes.

**Tech stack:** Go, Echo, Templ, `Datasource` interface for pluggable storage. Admin panel uses HTMX, SortableJS, and MDI web font as third-party dependencies. Dashboard uses vanilla JS only (no third-party dependencies).

For design decisions and rationale, see [PHILOSOPHY.md](PHILOSOPHY.md). For technical architecture, see [ARCHITECTURE.md](ARCHITECTURE.md).

## Key File Paths

| Path | Purpose |
|------|---------|
| `cmd/jumpgate/main.go` | Entry point, CLI dispatch (server/import/export) |
| `cmd/jumpgate/cli.go` | Import/export command implementations |
| `handlers/server.go` | Echo server setup, route registration |
| `handlers/dashboard.go` | Public dashboard handler, weather, SVG icons, theme discovery |
| `handlers/admin.go` | Admin CRUD handlers (categories, bookmarks, settings, toggles) |
| `handlers/middleware.go` | Auth middleware (X-Authorized-User header check) |
| `views/*.templ` | Templ templates (dashboard, admin, layout, category, bookmark, toggle, icons) |
| `model/model.go` | Data structures (Settings, Category, Bookmark, typed IDs, update structs) |
| `storage/db.go` | Datasource interface definition |
| `storage/sqlite.go` | SQLite implementation (schema, CRUD, migrations) |
| `config/config.go` | YAML config types for import/export |
| `icons/icons.go` | MDI icon list loader (CDN fetch + file cache) |
| `static/fs.go` | `embed.FS` for CSS, JS, themes, favicon |
| `static/css/style.css` | Dashboard stylesheet |
| `static/css/admin.css` | Admin stylesheet |
| `static/js/app.js` | Dashboard JS (search, theme switching, weather geolocation) |
| `static/themes/*.css` | Color themes (CSS custom properties) |
| `Makefile` | Build/run/test/docker commands |
| `Dockerfile` | Multi-stage Go build |

## How Things Connect

1. `cmd/jumpgate/main.go` parses CLI args, dispatches to `runServer`, `runImport`, or `runExport`
2. `runServer` opens SQLite DB via `storage.NewSQLiteDB()`, creates `icons.Loader`, starts Echo server via `handlers.NewServer()`
3. `handlers.NewServer()` registers dashboard routes (`/`, `/set-theme`) and admin routes (`/admin/*` with auth middleware)
4. Dashboard handler reads from Datasource, filters by privacy/enabled/mobile, renders Templ components
5. Admin handlers perform CRUD via Datasource, return full pages or HTMX HTML fragments
6. All templates in `views/` are `.templ` files compiled to Go functions

## Development Workflow

```bash
make deps       # Install templ CLI + download Go modules
make generate   # Regenerate templ templates
make build      # Full build → bin/jumpgate
make run        # Build + start server (:8080)
make debug      # Build + start with --no-auth (skips auth check)
make test       # Generate templ + run Go tests
make clean      # Remove bin/
```

### Making Changes

- Edit `.templ` files in `views/` — `make generate` regenerates the compiled Go files
- `make test` runs `templ generate` before testing, so regeneration is automatic
- Tests use `t.TempDir()` for isolated SQLite databases — each test gets a fresh DB
- Use stylesheets, not inline styles (except `html{display:none;}` which prevents a flash of unstyled content)
- Cache busting is automatic — `fileHash()` computes MD5 from embedded file content. Theme hashes are pre-computed and embedded as `window.THEME_HASHES` for client-side switching. No manual version incrementing needed.
- No obvious comments — explain WHY, not WHAT. The code should be self-documenting.
- Remove unused code and files immediately after changes
- Remove unnecessary trailing whitespace when editing files
- Follow the conventions in [CODESTYLE.md](CODESTYLE.md)

## Code Constraints

- **No web fonts** — Uses system fonts only
- **No npm, no TypeScript, no JS frameworks** — Vanilla JS only, where server round-trips are impractical
- **Third-party libraries load from CDN** — HTMX, SortableJS, and MDI font. Do not self-host.
- **CSS files, not inline styles** — All styling goes in stylesheets
- **No ORMs** — Templates consume Go structs directly. View model structs that bundle template parameters are encouraged.
- **Private by default** — New categories and bookmarks inherit `private=true` from settings
- **Icon reference** — https://pictogrammers.com/library/mdi/
